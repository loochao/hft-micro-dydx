package main

import (
	"bufio"
	"compress/gzip"
	"fmt"
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
	"github.com/geometrybase/hft-micro/common"
	kucoin_usdtfuture "github.com/geometrybase/hft-micro/kucoin-usdtfuture"
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
		"XBTUSDTM": "BTCUSDT",
		"UNIUSDTM":   "UNIUSDT",
		"DGBUSDTM":   "DGBUSDT",
		"IOSTUSDTM":  "IOSTUSDT",
		"RVNUSDTM":   "RVNUSDT",
		"THETAUSDTM": "THETAUSDT",
		"WAVESUSDTM": "WAVESUSDT",
		"DENTUSDTM":  "DENTUSDT",
		"DOTUSDTM":   "DOTUSDT",
		"XMRUSDTM":   "XMRUSDT",
		"FILUSDTM":   "FILUSDT",
		"ICPUSDTM":   "ICPUSDT",
		"MANAUSDTM":  "MANAUSDT",
		"MATICUSDTM": "MATICUSDT",
		"ALGOUSDTM":  "ALGOUSDT",
		"KSMUSDTM":   "KSMUSDT",
		"LUNAUSDTM":  "LUNAUSDT",
		"DASHUSDTM":  "DASHUSDT",
		"LTCUSDTM":   "LTCUSDT",
		"CHZUSDTM":   "CHZUSDT",
		"MKRUSDTM":   "MKRUSDT",
		"ADAUSDTM":   "ADAUSDT",
		"BCHUSDTM":   "BCHUSDT",
		"COMPUSDTM":  "COMPUSDT",
		"FTMUSDTM":   "FTMUSDT",
		"NEOUSDTM":   "NEOUSDT",
		"SXPUSDTM":   "SXPUSDT",
		"XRPUSDTM":   "XRPUSDT",
		"BNBUSDTM":   "BNBUSDT",
		"ETHUSDTM":   "ETHUSDT",
		"LINKUSDTM":  "LINKUSDT",
		"GRTUSDTM":   "GRTUSDT",
		"YFIUSDTM":   "YFIUSDT",
		"AAVEUSDTM":  "AAVEUSDT",
		"AVAXUSDTM":  "AVAXUSDT",
		"ETCUSDTM":   "ETCUSDT",
		"QTUMUSDTM":  "QTUMUSDT",
		"XLMUSDTM":   "XLMUSDT",
		"ZECUSDTM":   "ZECUSDT",
		"BTTUSDTM":   "BTTUSDT",
		"ENJUSDTM":   "ENJUSDT",
		"ONTUSDTM":   "ONTUSDT",
		"SUSHIUSDTM": "SUSHIUSDT",
		"XEMUSDTM":   "XEMUSDT",
		"DOGEUSDTM":  "DOGEUSDT",
		"OCEANUSDTM": "OCEANUSDT",
		"BATUSDTM":   "BATUSDT",
		"CRVUSDTM":   "CRVUSDT",
		"EOSUSDTM":   "EOSUSDT",
		"SNXUSDTM":   "SNXUSDT",
		"ATOMUSDTM":  "ATOMUSDT",
		"BANDUSDTM":  "BANDUSDT",
		"XTZUSDTM":   "XTZUSDT",
		"1INCHUSDTM": "1INCHUSDT",
		"TRXUSDTM":   "TRXUSDT",
		"SOLUSDTM":   "SOLUSDT",
		"VETUSDTM":   "VETUSDT",
	}
	symbols := make([]string, 0)
	for bSymbol := range pairs {
		symbols = append(symbols, bSymbol)
	}
	sort.Strings(symbols)
	logger.Debugf("%d", len(symbols))
	symbols = symbols[:]

	startTime, err := time.Parse("20060102", "20210701")
	if err != nil {
		logger.Fatal(err)
	}
	endTime, err := time.Parse("20060102", "20210715")
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
	longTD, _ := tdigest.New(tdigest.Compression(1000))
	shortTD, _ := tdigest.New(tdigest.Compression(1000))
	totalBidWeight := 0.0
	totalAskWeight := 0.0
	weightStep := 10.0

	for _, xSymbol := range symbols {
		ySymbol := pairs[xSymbol]

		xDepth := &kucoin_usdtfuture.Depth5{}
		yDepth := &binance_usdtfuture.Depth5{}

		var xT common.Ticker
		var yT common.Ticker

		timedTD := stream_stats.NewTimedTDigest(time.Hour*72, time.Minute*15)

		var msg []byte
		var dayCounter = 0
		for _, dateStr := range strings.Split(dateStrs, ",") {
			dayCounter++
			//logger.Debugf("%s %s", xSymbol, dateStr)
			file, err := os.Open(
				fmt.Sprintf("/home/clu/MarketData/bnuf-kcuf-depth5/%s/%s-%s,%s.depth5.jl.gz", dateStr, dateStr, ySymbol, xSymbol),
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
				if msg[0] == 'B' {
					err = binance_usdtfuture.ParseDepth5(msg[1:], yDepth)
					if err != nil {
						logger.Debugf("binance_usdtfuture.ParseDepth5 error %v", err)
						continue
					}
					yT = yDepth
				} else if msg[0] == 'K' {
					err = kucoin_usdtfuture.ParseDepth5(msg[1:], xDepth)
					if err != nil {
						logger.Debugf("kucoin_usdtfuture.ParseDepth5 error %v", err)
						continue
					}
					xT = xDepth
				} else {
					continue
				}

				if xT == nil || yT == nil {
					continue
				}

				if yT.GetEventTime().Sub(yT.GetEventTime().Truncate(time.Hour*4)) < time.Minute {
					continue
				}
				if xT.GetEventTime().Sub(xT.GetEventTime().Truncate(time.Hour*4)) < time.Minute {
					continue
				}

				if xT.GetEventTime().Sub(yT.GetEventTime()) < time.Second &&
					xT.GetEventTime().Sub(yT.GetEventTime()) > -time.Second &&
					yT.GetEventTime().Sub(lastCollectTime) > 0 {
					lastCollectTime = yT.GetEventTime().Add(time.Second)
					longSpread := (yT.GetAskPrice() - xT.GetBidPrice()) / xT.GetBidPrice()
					shortSpread := (yT.GetBidPrice() - xT.GetAskPrice()) / xT.GetAskPrice()
					_ = timedTD.Insert(yT.GetEventTime(), (longSpread+shortSpread)/2)
					if dayCounter >= 3 {
						bidWeight := uint32(xT.GetBidSize() * kucoin_usdtfuture.Multipliers[xSymbol] * xT.GetBidPrice() / weightStep)
						askWeight := uint32(xT.GetAskSize() * kucoin_usdtfuture.Multipliers[xSymbol] * xT.GetAskPrice() / weightStep)
						if bidWeight > 0 {
							totalBidWeight += float64(bidWeight)
							_ = longTD.AddWeighted(longSpread - timedTD.Quantile(0.5), bidWeight)
						}
						if askWeight > 0 {
							totalAskWeight += float64(askWeight)
							_ = shortTD.AddWeighted(shortSpread - timedTD.Quantile(0.5), askWeight)
						}
					}
				}
			}
			_ = gr.Close()
			_ = file.Close()
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
		err = ioutil.WriteFile("/home/clu/Projects/hft-micro/researches/kcuf-bnuf-depth5/configs/longTD", longBytes, 0755)
		if err != nil {
			logger.Debugf("%v", err)
		}
	}
	shortBytes, err := shortTD.AsBytes()
	if err == nil {
		err = ioutil.WriteFile("/home/clu/Projects/hft-micro/researches/kcuf-bnuf-depth5/configs/shortTD", shortBytes, 0755)
		if err != nil {
			logger.Debugf("%v", err)
		}
	}
}
