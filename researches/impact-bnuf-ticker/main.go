package main

import (
	"bufio"
	"compress/gzip"
	"flag"
	"fmt"
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/geometrybase/hft-micro/tdigest"
	"math"
	"os"
	"strings"
	"time"
)

func main() {

	symbolsStr := flag.String("symbols", "1000SHIBUSDT,1INCHUSDT,AAVEUSDT,ADAUSDT,AKROUSDT,ALGOUSDT,ALICEUSDT,ALPHAUSDT,ANKRUSDT,ATOMUSDT,AVAXUSDT,AXSUSDT,BAKEUSDT,BALUSDT,BANDUSDT,BATUSDT,BCHUSDT,BELUSDT,BLZUSDT,BNBUSDT,BTCDOMUSDT,BTCUSDT,BTSUSDT,BTTUSDT,BZRXUSDT,CELRUSDT,CHRUSDT,CHZUSDT,COMPUSDT,COTIUSDT,CRVUSDT,CTKUSDT,CVCUSDT,DASHUSDT,DEFIUSDT,DENTUSDT,DGBUSDT,DODOUSDT,DOGEUSDT,DOTUSDT,EGLDUSDT,ENJUSDT,EOSUSDT,ETCUSDT,ETHUSDT,FILUSDT,FLMUSDT,FTMUSDT,GRTUSDT,GTCUSDT,HBARUSDT,HNTUSDT,HOTUSDT,ICPUSDT,ICXUSDT,IOSTUSDT,IOTAUSDT,KAVAUSDT,KEEPUSDT,KNCUSDT,KSMUSDT,LINAUSDT,LINKUSDT,LITUSDT,LRCUSDT,LTCUSDT,LUNAUSDT,MANAUSDT,MATICUSDT,MKRUSDT,MTLUSDT,NEARUSDT,NEOUSDT,NKNUSDT,OCEANUSDT,OGNUSDT,OMGUSDT,ONEUSDT,ONTUSDT,QTUMUSDT,REEFUSDT,RENUSDT,RLCUSDT,RSRUSDT,RUNEUSDT,RVNUSDT,SANDUSDT,SCUSDT,SFPUSDT,SKLUSDT,SNXUSDT,SOLUSDT,SRMUSDT,STMXUSDT,STORJUSDT,SUSHIUSDT,SXPUSDT,THETAUSDT,TOMOUSDT,TRBUSDT,TRXUSDT,UNFIUSDT,UNIUSDT,VETUSDT,WAVESUSDT,XEMUSDT,XLMUSDT,XMRUSDT,XRPUSDT,XTZUSDT,YFIIUSDT,YFIUSDT,ZECUSDT,ZENUSDT,ZILUSDT,ZRXUSDT", "symbols, separate by comma")
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
				fmt.Sprintf("/home/clu/MarketData/bnuf-ticker/%s/%s-%s.ticker.jl.gz", dateStr, dateStr, symbol),
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
			var bookTicker = binance_usdtfuture.BookTicker{}
			var lastBookTicker = binance_usdtfuture.BookTicker{}
			var counter = 0
			var step = 1
			for scanner.Scan() {
				counter++
				msg = scanner.Bytes()
				if counter%step != 0 {
					continue
				}
				err = binance_usdtfuture.ParseBookTicker(msg, &bookTicker)
				if err != nil {
					logger.Debugf("binance_busdspot.ParseDepth5 error %v", err)
					continue
				}
				if lastBookTicker.Symbol != "" {
					if lastBookTicker.BestBidPrice >= bookTicker.BestBidPrice {
						_ = impactTD.Add((bookTicker.BestBidPrice - lastBookTicker.BestBidPrice) / lastBookTicker.BestBidPrice)
					}
					if lastBookTicker.BestAskPrice <= bookTicker.BestAskPrice {
						_ = impactTD.Add((bookTicker.BestAskPrice - lastBookTicker.BestAskPrice) / lastBookTicker.BestAskPrice)
					}
					_ = bookTD.Add(bookTicker.BestBidPrice*bookTicker.BestAskQty + bookTicker.BestAskPrice*bookTicker.BestAskQty)
				}
				lastBookTicker = bookTicker
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
