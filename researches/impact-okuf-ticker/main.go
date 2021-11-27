package main

import (
	"bufio"
	"compress/gzip"
	"flag"
	"fmt"
	"github.com/geometrybase/hft-micro/logger"
	//okex_usdtspot "github.com/geometrybase/hft-micro/okex-usdtspot"
	"github.com/geometrybase/hft-micro/tdigest"
	"math"
	"os"
	"strings"
	"time"
)



func main() {

	symbolsStr := flag.String("symbols", "1INCH-USDT,AAVE-USDT,ADA-USDT,ALGO-USDT,ATOM-USDT,AVAX-USDT,BAL-USDT,BAND-USDT,BAT-USDT,BCH-USDT,BTC-USDT,COMP-USDT,CRV-USDT,CVC-USDT,DASH-USDT,DOGE-USDT,DOT-USDT,EGLD-USDT,EOS-USDT,ETC-USDT,ETH-USDT,FIL-USDT,FLM-USDT,FTM-USDT,GRT-USDT,ICX-USDT,IOST-USDT,IOTA-USDT,KNC-USDT,KSM-USDT,LINK-USDT,LRC-USDT,LTC-USDT,MKR-USDT,NEAR-USDT,NEO-USDT,OMG-USDT,ONT-USDT,QTUM-USDT,RSR-USDT,SNX-USDT,SOL-USDT,SRM-USDT,STORJ-USDT,SUSHI-USDT,THETA-USDT,TRB-USDT,TRX-USDT,UNI-USDT,WAVES-USDT,XLM-USDT,XMR-USDT,XRP-USDT,XTZ-USDT,YFI-USDT,YFII-USDT,ZEC-USDT,ZEN-USDT,ZIL-USDT,ZRX-USDT", "symbols, separate by comma")
	flag.Parse()
	symbols := strings.Split(*symbolsStr, ",")
	logger.Debugf("%d", len(symbols))
	startTime, err := time.Parse("20060102", "20210707")
	if err != nil {
		logger.Fatal(err)
	}
	endTime, err := time.Parse("20060102", "20210707")
	if err != nil {
		logger.Fatal(err)
	}
	dateStrs := ""
	for i := startTime; i.Sub(endTime) <= 0; i = i.Add(time.Hour * 24) {
		dateStrs += i.Format("20060102,")
	}
	dateStrs = dateStrs[:len(dateStrs)-1]

	quantiles := make(map[string]string)
	maxOrderValues := make(map[string]float64)
	for _, symbol := range symbols {
		impactTD, _ := tdigest.New()
		bookTD, _ := tdigest.New()
		for _, dateStr := range strings.Split(dateStrs, ",") {
			file, err := os.Open(
				fmt.Sprintf("/Users/chenjilin/MarketData/okus-ticker/%s/%s-%s.ticker.jl.gz", dateStr, dateStr, symbol),
			)
			if err != nil {
				logger.Debugf("os.Open() error %v", err)
				continue
			}
			gr, err := gzip.NewReader(file)
			if err != nil {
				logger.Debugf("gzip.NewReader(file) error %v", err)
				continue
			}
			b := make([]byte, 0, 512)
			_, err = gr.Read(b)
			if err != nil {
				logger.Debugf("gr.Read(b) error %v", err)
				continue
			}
			scanner := bufio.NewScanner(gr)
			var msg []byte
			var depth5 = okex_usdtspot.Ticker{}
			var lastDepth5 = okex_usdtspot.Ticker{}
			var counter = 0
			var step = 1
			for scanner.Scan() {
				counter++
				msg = scanner.Bytes()
				if counter%step != 0 {
					continue
				}
				err = okex_usdtspot.ParseTicker(msg, &depth5)
				if err != nil {
					logger.Debugf("binance_busdspot.ParseDepth5 error %v", err)
					continue
				}
				if lastDepth5.InstrumentID != "" {
					if lastDepth5.BestBid >= depth5.BestBid {
						_ = impactTD.Add((depth5.BestBid - lastDepth5.BestBid) / lastDepth5.BestBid)
					}
					if lastDepth5.BestAsk <= depth5.BestAsk {
						_ = impactTD.Add((depth5.BestAsk - lastDepth5.BestAsk) / lastDepth5.BestAsk)
					}
					_ = bookTD.Add(depth5.BestBid*depth5.BestBidSize + depth5.BestAsk*depth5.BestAskSize)
				}
				lastDepth5 = depth5
			}
			_ = gr.Close()
			_ = file.Close()
		}
		quantiles[symbol] = fmt.Sprintf(
			"%.6f,%.6f,%.6f,%.6f,%.6f,%.6f",
			impactTD.Quantile(0.00005),
			impactTD.Quantile(0.005),
			impactTD.Quantile(0.05),
			impactTD.Quantile(0.95),
			impactTD.Quantile(0.995),
			impactTD.Quantile(0.99995),
		)
		maxOrderValues[symbol] = bookTD.Quantile(0.8) * 0.1
	}
	fmt.Printf("\n\n\nxyPairs:\n")
	for _, symbol := range symbols {
		fmt.Printf(
			"  %s:\t%s\n",
			symbol,
			strings.Replace(symbol, "-USDT", "USDT", -1),
		)
	}
	fmt.Printf("\n\n\norderOffsets:\n")
	for _, symbol := range symbols {
		fmt.Printf(
			"  %s:\t%s\n",
			symbol,
			quantiles[symbol],
		)
	}
	fmt.Printf("\n\n\nmaxOrderValues:\n")
	for _, symbol := range symbols {
		fmt.Printf(
			"  %s:\t%.0f\n",
			symbol,
			maxOrderValues[symbol],
		)
	}
	fmt.Printf("\n\n\n")

	weights := make(map[string]float64)
	totalSize := 0.0
	for _, size := range maxOrderValues {
		totalSize += size
	}
	meanSize := totalSize / float64(len(maxOrderValues))
	symbols = make([]string, 0)
	for symbol, size := range maxOrderValues {
		weights[symbol] = math.Sqrt(size / meanSize)
		if weights[symbol] > 1 {
			weights[symbol] = 1.0
		}
		symbols = append(symbols, symbol)
	}
	fmt.Printf("\n\n\ntargetWeights:\n")
	for symbol, weight := range weights {
		fmt.Printf("  %s: %.2f\n", symbol, weight)
	}
}
