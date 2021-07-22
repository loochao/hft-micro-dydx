package main

import (
	"bufio"
	"compress/gzip"
	"fmt"
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
	bybit_usdtfuture "github.com/geometrybase/hft-micro/bybit-usdtfuture"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	stream_stats "github.com/geometrybase/hft-micro/stream-stats"
	"github.com/geometrybase/hft-micro/tdigest"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"time"
)

func main() {

	pairs := map[string]string{
		"BTCUSDT": "BTCUSDT",
	}
	symbols := make([]string, 0)
	for bSymbol := range pairs {
		symbols = append(symbols, bSymbol)
	}
	sort.Strings(symbols)
	logger.Debugf("%d", len(symbols))
	symbols = symbols[:]

	startTime, err := time.Parse("20060102", "20210718")
	if err != nil {
		logger.Fatal(err)
	}
	endTime, err := time.Parse("20060102", "20210719")
	if err != nil {
		logger.Fatal(err)
	}
	dateStrs := ""
	dayCount := 0.0
	for i := startTime; i.Sub(endTime) <= 0; i = i.Add(time.Hour * 24) {
		dayCount++
		dateStrs += i.Format("20060102,")
	}
	dateStrs = dateStrs[:len(dateStrs)-1]
	longTD, _ := tdigest.New(tdigest.Compression(100))
	shortTD, _ := tdigest.New(tdigest.Compression(100))
	totalBidWeight := 0.0
	totalAskWeight := 0.0

	maxBidQuantiles := make(map[string]*tdigest.TDigest)
	maxAskQuantiles := make(map[string]*tdigest.TDigest)

	for _, xSymbol := range symbols {
		ySymbol := pairs[xSymbol]

		xDepth := &bybit_usdtfuture.OrderBook{}
		yDepth := &binance_usdtfuture.Depth5{}

		yBookTicker := &binance_usdtfuture.BookTicker{}

		var xTicker common.Ticker
		var yTicker common.Ticker

		timedTD := stream_stats.NewTimedTDigest(time.Hour*72, time.Minute*15)

		maxBidTD, _ := tdigest.New()
		maxAskTD, _ := tdigest.New()

		var msg []byte
		var dayCounter = 0
		var isXDepthReady = false
		var symbolLen = len(xSymbol)
		for _, dateStr := range strings.Split(dateStrs, ",") {
			dayCounter++
			//logger.Debugf("%s %s", xSymbol, dateStr)
			file, err := os.Open(
				fmt.Sprintf("/Users/chenjilin/MarketData/bbuf-bnuf-depth-and-ticker/%s/%s-%s,%s.gz", dateStr, dateStr, xSymbol, ySymbol),
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
			scanner := bufio.NewScanner(gr)
			lastCollectTime := time.Time{}
			for scanner.Scan() {
				msg = scanner.Bytes()
				if msg[0] == 'X' && msg[1] == 'D' {
					err = bybit_usdtfuture.UpdateOrderBook(msg[21:], xDepth)
					if err != nil {
						logger.Debugf("bybit_usdtfuture.UpdateOrderBook error %v", err)
						continue
					}
					if msg[56+symbolLen] == 's' {
						isXDepthReady = true
					}
					if !xDepth.IsValidate() {
						isXDepthReady = false
					}
					xTicker = xDepth
				} else if msg[0] == 'Y' && msg[1] == 'D' {
					err = binance_usdtfuture.ParseDepth5(msg[21:], yDepth)
					if err != nil {
						logger.Debugf("binance_usdtfuture.ParseDepth5 error %v", err)
						continue
					}
					yTicker = yDepth
				} else if msg[0] == 'Y' && msg[1] == 'T' {
					err = binance_usdtfuture.ParseBookTicker(msg[21:], yBookTicker)
					if err != nil {
						logger.Debugf("binance_usdtfuture.ParseBookTicker error %v", err)
						continue
					}
					yTicker = yBookTicker
				} else {
					continue
				}

				if xTicker == nil || yTicker == nil {
					continue
				}

				if yTicker.GetTime().Sub(yTicker.GetTime().Truncate(time.Hour*4)) < time.Minute {
					continue
				}
				if xTicker.GetTime().Sub(xTicker.GetTime().Truncate(time.Hour*4)) < time.Minute {
					continue
				}

				//if isXDepthReady {
				//	logger.Debugf("%v", xTicker.GetTime().Sub(yTicker.GetTime()))
				//}

				if isXDepthReady &&
					xTicker.GetTime().Sub(yTicker.GetTime()) < time.Second &&
					xTicker.GetTime().Sub(yTicker.GetTime()) > -time.Second &&
					yTicker.GetTime().Sub(lastCollectTime) > 0 {
					lastCollectTime = yTicker.GetTime().Add(time.Second)
					longSpread := (yTicker.GetAskPrice() - xTicker.GetBidPrice()) / xTicker.GetBidPrice()
					shortSpread := (yTicker.GetBidPrice() - xTicker.GetAskPrice()) / xTicker.GetAskPrice()
					_ = timedTD.Insert(yTicker.GetTime(), (longSpread+shortSpread)/2)
					_ = maxBidTD.Add(yTicker.GetBidSize() * yTicker.GetBidPrice())
					_ = maxAskTD.Add(yTicker.GetAskSize() * yTicker.GetAskPrice())
					bidWeight := uint32(xTicker.GetBidSize() * xTicker.GetBidPrice())
					askWeight := uint32(xTicker.GetAskSize() * xTicker.GetAskPrice())
					if bidWeight > 0 {
						totalBidWeight += float64(bidWeight)
						_ = longTD.AddWeighted(longSpread-timedTD.Quantile(0.5), bidWeight)
					}
					if askWeight > 0 {
						totalAskWeight += float64(askWeight)
						_ = shortTD.AddWeighted(shortSpread-timedTD.Quantile(0.5), askWeight)
					}
				}
			}
			_ = gr.Close()
			_ = file.Close()
			maxBidQuantiles[xSymbol] = maxBidTD
			maxAskQuantiles[xSymbol] = maxAskTD
		}
	}

	fmt.Printf("\n\nLONG SPREAD, DAY VOLUME %.0f:\n", totalBidWeight/dayCount)
	for i := 0.000111; i <= 0.0111; i += 0.000111 {
		fmt.Printf("  SPREAD %.10f QUANTILE %.10f TRADEABLE VALUE %.0f\n", -i, longTD.CDF(-i), longTD.CDF(-i)*totalBidWeight/dayCount)
	}
	fmt.Printf("SHORT SPREAD, DAY VOLUME %.0f:\n", totalAskWeight)
	for i := 0.000111; i <= 0.0111; i += 0.000111 {
		fmt.Printf("  SPREAD %.10f QUANTILE %.10f TRADEABLE VALUE %.0f\n", i, 1.0-shortTD.CDF(i), (1.0-shortTD.CDF(i))*totalAskWeight/dayCount)
	}
	longBytes, err := longTD.AsBytes()
	if err == nil {
		err = ioutil.WriteFile("/Users/chenjilin/Projects/hft-micro/researches/bbuf-bnuf-depth-and-ticker/configs/longTD", longBytes, 0755)
		if err != nil {
			logger.Debugf("%v", err)
		}
	}
	shortBytes, err := shortTD.AsBytes()
	if err == nil {
		err = ioutil.WriteFile("/Users/chenjilin/Projects/hft-micro/researches/bbuf-bnuf-depth-and-ticker/configs/shortTD", shortBytes, 0755)
		if err != nil {
			logger.Debugf("%v", err)
		}
	}
	fmt.Printf("\n\n")
	for xSymbol := range maxBidQuantiles {
		fmt.Printf("%s BID %5.0f ASK %5.0f\n", xSymbol, maxBidQuantiles[xSymbol].Quantile(0.25), maxAskQuantiles[xSymbol].Quantile(0.25))
	}
}
