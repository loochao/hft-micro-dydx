package main

import (
	"bufio"
	"compress/gzip"
	"flag"
	"fmt"
	ftx_usdfuture "github.com/geometrybase/hft-micro/ftx-usdfuture"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/geometrybase/hft-micro/tdigest"
	"math"
	"os"
	"strings"
	"time"
)

func main() {

	symbolsStr := flag.String("symbols", "1INCH-PERP,AAVE-PERP,ADA-PERP,ALGO-PERP,ALPHA-PERP,ATOM-PERP,AVAX-PERP,AXS-PERP,BAL-PERP,BAND-PERP,BAT-PERP,BCH-PERP,BNB-PERP,BTC-PERP,BTT-PERP,CHZ-PERP,COMP-PERP,CRV-PERP,DASH-PERP,DEFI-PERP,DENT-PERP,DODO-PERP,DOGE-PERP,DOT-PERP,EGLD-PERP,ENJ-PERP,EOS-PERP,ETC-PERP,ETH-PERP,FIL-PERP,FLM-PERP,FTM-PERP,GRT-PERP,HBAR-PERP,HNT-PERP,HOT-PERP,ICP-PERP,IOTA-PERP,KAVA-PERP,KNC-PERP,KSM-PERP,LINA-PERP,LINK-PERP,LRC-PERP,LTC-PERP,LUNA-PERP,MATIC-PERP,MKR-PERP,MTL-PERP,NEAR-PERP,NEO-PERP,OMG-PERP,ONT-PERP,QTUM-PERP,REEF-PERP,REN-PERP,RSR-PERP,RUNE-PERP,SAND-PERP,SC-PERP,SKL-PERP,SNX-PERP,SOL-PERP,SRM-PERP,STMX-PERP,STORJ-PERP,SUSHI-PERP,SXP-PERP,THETA-PERP,TOMO-PERP,TRX-PERP,UNI-PERP,VET-PERP,WAVES-PERP,XEM-PERP,XLM-PERP,XMR-PERP,XRP-PERP,XTZ-PERP,YFI-PERP,YFII-PERP,ZEC-PERP,ZIL-PERP,ZRX-PERP", "symbols, separate by comma")
	flag.Parse()
	symbols := strings.Split(*symbolsStr, ",")
	logger.Debugf("%d", len(symbols))
	startTime, err := time.Parse("20060102", "20210629")
	if err != nil {
		logger.Fatal(err)
	}
	endTime, err := time.Parse("20060102", "20210629")
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
				fmt.Sprintf("/home/clu/MarketData/ftxuf-ticker/%s/%s-%s.ticker.jl.gz", dateStr, dateStr, symbol),
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
			var depth5 = ftx_usdfuture.Ticker{}
			var lastDepth5 = ftx_usdfuture.Ticker{}
			var counter = 0
			var step = 1
			for scanner.Scan() {
				counter++
				msg = scanner.Bytes()
				if counter%step != 0 {
					continue
				}
				err = ftx_usdfuture.ParseTicker(msg, &depth5)
				if err != nil {
					logger.Debugf("binance_busdspot.ParseDepth5 error %v", err)
					continue
				}
				if lastDepth5.Symbol != "" {
					if lastDepth5.Bid >= depth5.Bid {
						_ = impactTD.Add((depth5.Bid - lastDepth5.Bid) / lastDepth5.Bid)
					}
					if lastDepth5.Ask <= depth5.Ask {
						_ = impactTD.Add((depth5.Ask - lastDepth5.Ask) / lastDepth5.Ask)
					}
					_ = bookTD.Add(depth5.Bid*depth5.BidSize + depth5.Ask*depth5.AskSize)
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
			strings.Replace(symbol, "-PERP", "USDT", -1),
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
