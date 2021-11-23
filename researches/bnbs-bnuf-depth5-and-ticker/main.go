package main

import (
	"bufio"
	"compress/gzip"
	"fmt"
	binance_busdfuture "github.com/geometrybase/hft-micro/binance-busdfuture"
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
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
		"BTCBUSD": "BTCUSDT",
		"ETHBUSD": "ETHUSDT",
	}
	symbols := make([]string, 0)
	for bSymbol := range pairs {
		symbols = append(symbols, bSymbol)
	}
	sort.Strings(symbols)
	logger.Debugf("%d", len(symbols))
	symbols = symbols[:]

	startTime, err := time.Parse("20060102", "20210716")
	if err != nil {
		logger.Fatal(err)
	}
	endTime, err := time.Parse("20060102", "20210716")
	if err != nil {
		logger.Fatal(err)
	}
	dateStrs := ""
	dayCount := 0.0
	for i := startTime; i.Sub(endTime) <= 0; i = i.Add(time.Hour * 24) {
		dateStrs += i.Format("20060102,")
		dayCount += 1.0
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

		xDepth := &binance_busdfuture.Depth5{}
		yDepth := &binance_usdtfuture.Depth5{}

		xBookTicker := &binance_busdfuture.BookTicker{}
		yBookTicker := &binance_usdtfuture.BookTicker{}

		var xTicker common.Ticker
		var yTicker common.Ticker

		timedTD := stream_stats.NewTimedTDigest(time.Hour*72, time.Minute*15)

		maxBidTD, _ := tdigest.New()
		maxAskTD, _ := tdigest.New()

		var msg []byte
		var dayCounter = 0
		for _, dateStr := range strings.Split(dateStrs, ",") {
			dayCounter++
			//logger.Debugf("%s %s", xSymbol, dateStr)
			file, err := os.Open(
				fmt.Sprintf("/Users/chenjilin/MarketData/bnbs-bnuf-depth5-and-ticker/%s/%s-%s,%s.jl.gz", dateStr, dateStr, xSymbol, ySymbol),
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
					err = binance_busdfuture.ParseDepth5(msg[21:], xDepth)
					if err != nil {
						logger.Debugf("binance_busdfuture.ParseDepth5 error %v", err)
						continue
					}
					xTicker = xDepth
				} else if msg[0] == 'X' && msg[1] == 'T' {
					err = binance_busdfuture.ParseBookTicker(msg[21:], xBookTicker)
					if err != nil {
						logger.Debugf("binance_busdfuture.ParseBookTicker error %v", err)
						continue
					}
					xTicker = xBookTicker
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

				if yTicker.GetEventTime().Sub(yTicker.GetEventTime().Truncate(time.Hour*4)) < time.Minute {
					continue
				}
				if xTicker.GetEventTime().Sub(xTicker.GetEventTime().Truncate(time.Hour*4)) < time.Minute {
					continue
				}

				if xTicker.GetEventTime().Sub(yTicker.GetEventTime()) < time.Millisecond &&
					xTicker.GetEventTime().Sub(yTicker.GetEventTime()) > -time.Millisecond &&
					yTicker.GetEventTime().Sub(lastCollectTime) > 0 {
					lastCollectTime = yTicker.GetEventTime().Add(time.Millisecond)
					longSpread := (yTicker.GetAskPrice() - xTicker.GetBidPrice()) / xTicker.GetBidPrice()
					shortSpread := (yTicker.GetBidPrice() - xTicker.GetAskPrice()) / xTicker.GetAskPrice()
					_ = timedTD.Insert(yTicker.GetEventTime(), (longSpread+shortSpread)/2)
					//_ = maxBidTD.Add(math.Min(xTicker.GetBidSize() * xTicker.GetBidPrice(), yTicker.GetBidSize() * yTicker.GetBidPrice()))
					//_ = maxAskTD.Add(math.Min(xTicker.GetAskSize() * xTicker.GetAskPrice(), yTicker.GetAskSize() * yTicker.GetAskPrice()))
					_ = maxBidTD.Add(yTicker.GetBidSize() * yTicker.GetBidPrice())
					_ = maxAskTD.Add(yTicker.GetAskSize() * yTicker.GetAskPrice())
					//if dayCounter >= 3 {
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
					//}
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
		err = ioutil.WriteFile("/Users/chenjilin/Projects/hft-micro/researches/bnbs-bnuf-depth5-and-ticker/configs/longTD", longBytes, 0755)
		if err != nil {
			logger.Debugf("%v", err)
		}
	}
	shortBytes, err := shortTD.AsBytes()
	if err == nil {
		err = ioutil.WriteFile("/Users/chenjilin/Projects/hft-micro/researches/bnbs-bnuf-depth5-and-ticker/configs/shortTD", shortBytes, 0755)
		if err != nil {
			logger.Debugf("%v", err)
		}
	}
	fmt.Printf("\n\n")
	for xSymbol := range maxBidQuantiles {
		fmt.Printf("%s BID %5.0f ASK %5.0f\n", xSymbol, maxBidQuantiles[xSymbol].Quantile(0.25), maxAskQuantiles[xSymbol].Quantile(0.25))
	}
}
