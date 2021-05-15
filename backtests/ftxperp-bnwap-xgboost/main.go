package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/ftxperp"
	"github.com/geometrybase/hft-micro/logger"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"time"
)

var topBots = map[string][2]float64{
	"DOGE-PERP": {-0.003, 0.001},
}

func main() {

	ctx := context.Background()
	iw, err := common.NewInfluxWriter(
		ctx,
		"http://localhost:8086",
		"",
		"",
		"hft",
		100,
	)
	if err != nil {
		panic(err)
	}
	defer iw.Stop()

	matchSymbols := make([]string, 0)
	pairMaps := make(map[string]string)
	for ftxMarket := range ftxperp.PriceIncrements {
		bnSymbol := strings.Replace(ftxMarket, "-PERP", "USDT", -1)
		if _, ok := bnswap.TickSizes[bnSymbol]; ok && len(bnSymbol) >= 7 {
			pairMaps[ftxMarket] = bnSymbol
			matchSymbols = append(matchSymbols, ftxMarket)
		}
	}

	logger.Debugf("\n\n%s\n\n", matchSymbols)

	sort.Strings(matchSymbols)
	fmt.Printf("\n\n")
	for _, fs := range matchSymbols {
		bs := pairMaps[fs]
		fmt.Printf("%s: %s\n", fs, bs)
	}
	fmt.Printf("\n\n")

	symbols := strings.Split(
		`DOGE-PERP`,
		",",
	)
	//symbols = matchSymbols[:]
	dateStrs := "20210509,20210510,20210511,20210512,20210513"
	for _, symbol := range symbols {
		var lastBnTrade *bnswap.Trade
		var lastFtxTrade *ftxperp.Trade
		lookback := time.Second
		ftxBuyMean1s := common.NewTimedMean(lookback)
		ftxSellMean1s := common.NewTimedMean(lookback)

		bnBuyPrice1s := common.NewTimedMean(lookback)
		bnSellPrice1s := common.NewTimedMean(lookback)

		bnBuyPrice5s := common.NewTimedMean(time.Second * 5)
		bnSellPrice5s := common.NewTimedMean(time.Second * 5)
		bnBuyPrice15s := common.NewTimedMean(time.Second * 15)
		bnSellPrice15s := common.NewTimedMean(time.Second * 15)
		bnBuyPrice30s := common.NewTimedMean(time.Second * 30)
		bnSellPrice30s := common.NewTimedMean(time.Second * 30)
		bnBuyPrice1m := common.NewTimedMean(time.Minute)
		bnSellPrice1m := common.NewTimedMean(time.Minute)
		bnBuyPrice3m := common.NewTimedMean(time.Minute * 3)
		bnSellPrice3m := common.NewTimedMean(time.Minute * 3)
		bnBuyPrice5m := common.NewTimedMean(time.Minute * 5)
		bnSellPrice5m := common.NewTimedMean(time.Minute * 5)

		bnBuyVolume1s := common.NewTimedMean(lookback)
		bnSellVolume1s := common.NewTimedMean(lookback)
		bnBuyVolume5s := common.NewTimedMean(time.Second * 5)
		bnSellVolume5s := common.NewTimedMean(time.Second * 5)
		bnBuyVolume15s := common.NewTimedMean(time.Second * 15)
		bnSellVolume15s := common.NewTimedMean(time.Second * 15)
		bnBuyVolume30s := common.NewTimedMean(time.Second * 30)
		bnSellVolume30s := common.NewTimedMean(time.Second * 30)
		bnBuyVolume1m := common.NewTimedMean(time.Minute)
		bnSellVolume1m := common.NewTimedMean(time.Minute)
		bnBuyVolume3m := common.NewTimedMean(time.Minute * 3)
		bnSellVolume3m := common.NewTimedMean(time.Minute * 3)
		bnBuyVolume5m := common.NewTimedMean(time.Minute * 5)
		bnSellVolume5m := common.NewTimedMean(time.Minute * 5)

		const featureCount = 14
		const closeTime = 15000000000
		longLabels := make(map[int]float64)
		longFeatures := make(map[int][featureCount]float64)
		longMarkPrices := make(map[int]float64)
		shortLabels := make(map[int]float64)
		shortFeatures := make(map[int][featureCount]float64)
		shortMarkPrices := make(map[int]float64)
		for _, dateStr := range strings.Split(dateStrs, ",") {
			ftxFile, err := os.Open(
				fmt.Sprintf("/Users/chenjilin/MarketData/ftxperp-trade/%s-%s.ftxperp.trade.jl.gz", dateStr, symbol),
			)
			if err != nil {
				logger.Debugf("os.Open() error %v", err)
				continue
				//return
			}
			ftxGr, err := gzip.NewReader(ftxFile)
			if err != nil {
				logger.Debugf("gzip.NewReader(ftxFile) error %v", err)
				continue
				//return
			}
			ftxScanner := bufio.NewScanner(ftxGr)

			bnFile, err := os.Open(
				fmt.Sprintf("/Users/chenjilin/MarketData/bnswap-trade/%s-%s.bnswap.trade.jl.gz", dateStr, pairMaps[symbol]),
			)
			if err != nil {
				logger.Debugf("os.Open() error %v", err)
				continue
			}
			bnGr, err := gzip.NewReader(bnFile)
			if err != nil {
				logger.Debugf("gzip.NewReader(bnFile) error %v", err)
				continue
			}
			bnScanner := bufio.NewScanner(bnGr)

			for ftxScanner.Scan() {
				tradeData := ftxperp.TradesData{}
				err = json.Unmarshal(ftxScanner.Bytes(), &tradeData)
				if err != nil {
					continue
				}
				for _, ftxTrade := range tradeData.Data {
					if ftxTrade.Side == ftxperp.TradeSideBuy {
						ftxBuyMean1s.Insert(ftxTrade.Time, ftxTrade.Price)
					} else {
						ftxSellMean1s.Insert(ftxTrade.Time, ftxTrade.Price)
					}
					if lastFtxTrade == nil ||
						ftxTrade.Time.Truncate(lookback).Sub(lastFtxTrade.Time.Truncate(lookback)) <= 0 {
						ftxTrade := ftxTrade
						lastFtxTrade = &ftxTrade
						continue
					}
					for bnScanner.Scan() {
						if lastBnTrade != nil {
							if lastBnTrade.IsTheBuyerTheMarketMaker {

								bnSellPrice1s.Insert(lastBnTrade.EventTime, lastBnTrade.Price)

								bnSellPrice5s.Insert(lastBnTrade.EventTime, lastBnTrade.Price)
								bnSellPrice15s.Insert(lastBnTrade.EventTime, lastBnTrade.Price)
								bnSellPrice30s.Insert(lastBnTrade.EventTime, lastBnTrade.Price)
								bnSellPrice1m.Insert(lastBnTrade.EventTime, lastBnTrade.Price)
								bnSellPrice3m.Insert(lastBnTrade.EventTime, lastBnTrade.Price)
								bnSellPrice5m.Insert(lastBnTrade.EventTime, lastBnTrade.Price)

								bnSellVolume1s.Insert(lastBnTrade.EventTime, lastBnTrade.Quantity)
								bnSellVolume5s.Insert(lastBnTrade.EventTime, lastBnTrade.Quantity)
								bnSellVolume15s.Insert(lastBnTrade.EventTime, lastBnTrade.Quantity)
								bnSellVolume30s.Insert(lastBnTrade.EventTime, lastBnTrade.Quantity)
								bnSellVolume1m.Insert(lastBnTrade.EventTime, lastBnTrade.Quantity)
								bnSellVolume3m.Insert(lastBnTrade.EventTime, lastBnTrade.Quantity)
								bnSellVolume5m.Insert(lastBnTrade.EventTime, lastBnTrade.Quantity)

							} else {
								bnBuyPrice1s.Insert(lastBnTrade.EventTime, lastBnTrade.Price)
								bnBuyPrice5s.Insert(lastBnTrade.EventTime, lastBnTrade.Price)
								bnBuyPrice15s.Insert(lastBnTrade.EventTime, lastBnTrade.Price)
								bnBuyPrice30s.Insert(lastBnTrade.EventTime, lastBnTrade.Price)
								bnBuyPrice1m.Insert(lastBnTrade.EventTime, lastBnTrade.Price)
								bnBuyPrice3m.Insert(lastBnTrade.EventTime, lastBnTrade.Price)
								bnBuyPrice5m.Insert(lastBnTrade.EventTime, lastBnTrade.Price)

								bnBuyVolume1s.Insert(lastBnTrade.EventTime, lastBnTrade.Quantity)
								bnBuyVolume5s.Insert(lastBnTrade.EventTime, lastBnTrade.Quantity)
								bnBuyVolume15s.Insert(lastBnTrade.EventTime, lastBnTrade.Quantity)
								bnBuyVolume30s.Insert(lastBnTrade.EventTime, lastBnTrade.Quantity)
								bnBuyVolume1m.Insert(lastBnTrade.EventTime, lastBnTrade.Quantity)
								bnBuyVolume3m.Insert(lastBnTrade.EventTime, lastBnTrade.Quantity)
								bnBuyVolume5m.Insert(lastBnTrade.EventTime, lastBnTrade.Quantity)
							}
						}
						bnTrade, err := bnswap.ParseTrade(bnScanner.Bytes())
						if err != nil {
							continue
						}
						lastBnTrade = bnTrade
						if bnTrade.EventTime.Sub(ftxTrade.Time) > 0 {
							break
						}
					}

					if ftxBuyMean1s.Len() > 0 &&
						ftxSellMean1s.Len() > 0 &&
						bnBuyPrice1s.Len() > 0 &&
						bnSellPrice1s.Len() > 0 {
						ftxBuyPrice := ftxBuyMean1s.Mean()
						ftxSellPrice := ftxSellMean1s.Mean()
						bnBuyPrice := bnBuyPrice1s.Mean()
						bnSellPrice := bnSellPrice1s.Mean()
						longDelta := (bnBuyPrice - ftxSellPrice) / ftxSellPrice
						shortDelta := (bnSellPrice - ftxBuyPrice) / ftxBuyPrice
						if lastFtxTrade != nil {
							for t, markPrice := range longMarkPrices {
								if int(lastFtxTrade.Time.UnixNano()/1)-t < closeTime && int(ftxTrade.Time.UnixNano()/1)-t >= closeTime {
									longLabels[t] = (markPrice - bnBuyPrice) / markPrice
									delete(longMarkPrices, t)
								}
							}
							for t, markPrice := range shortMarkPrices {
								if int(lastFtxTrade.Time.UnixNano()/1)-t < closeTime && int(ftxTrade.Time.UnixNano()/1)-t >= closeTime {
									shortLabels[t] = (bnSellPrice - markPrice) / markPrice
									delete(shortMarkPrices, t)
								}
							}
						}
						if longDelta < topBots[symbol][0] {
							longMarkPrices[int(ftxTrade.Time.UnixNano()/1)] = bnBuyPrice
							longFeatures[int(ftxTrade.Time.UnixNano()/1)] = [featureCount]float64{
								//(bnBuyPrice1s.Mean() - bnBuyPrice5s.Mean()) / bnBuyPrice5s.Mean(),
								//(bnBuyPrice1s.Mean() - bnBuyPrice15s.Mean()) / bnBuyPrice15s.Mean(),
								//(bnBuyPrice1s.Mean() - bnBuyPrice30s.Mean()) / bnBuyPrice30s.Mean(),
								//(bnBuyPrice1s.Mean() - bnBuyPrice1m.Mean()) / bnBuyPrice1m.Mean(),
								//(bnBuyPrice1s.Mean() - bnBuyPrice3m.Mean()) / bnBuyPrice3m.Mean(),
								//(bnBuyPrice1s.Mean() - bnBuyPrice5m.Mean()) / bnBuyPrice5m.Mean(),
								//
								//(bnSellPrice1s.Mean() - bnSellPrice5s.Mean()) / bnSellPrice5s.Mean(),
								//(bnSellPrice1s.Mean() - bnSellPrice15s.Mean()) / bnSellPrice15s.Mean(),
								//(bnSellPrice1s.Mean() - bnSellPrice30s.Mean()) / bnSellPrice30s.Mean(),
								//(bnSellPrice1s.Mean() - bnSellPrice1m.Mean()) / bnSellPrice1m.Mean(),
								//(bnSellPrice1s.Mean() - bnSellPrice3m.Mean()) / bnSellPrice3m.Mean(),
								//(bnSellPrice1s.Mean() - bnSellPrice5m.Mean()) / bnSellPrice5m.Mean(),

								(bnBuyVolume1s.Mean() - bnSellPrice1s.Mean()) / (bnBuyVolume1s.Mean() + bnSellPrice1s.Mean()),
								(bnBuyVolume5s.Mean() - bnSellPrice5s.Mean()) / (bnBuyVolume5s.Mean() + bnSellPrice5s.Mean()),
								(bnBuyVolume15s.Mean() - bnSellPrice15s.Mean()) / (bnBuyVolume15s.Mean() + bnSellPrice15s.Mean()),
								(bnBuyVolume30s.Mean() - bnSellPrice30s.Mean()) / (bnBuyVolume30s.Mean() + bnSellPrice30s.Mean()),
								(bnBuyVolume1m.Mean() - bnSellPrice1m.Mean()) / (bnBuyVolume1m.Mean() + bnSellPrice1m.Mean()),
								(bnBuyVolume3m.Mean() - bnSellPrice3m.Mean()) / (bnBuyVolume3m.Mean() + bnSellPrice3m.Mean()),
								(bnBuyVolume5m.Mean() - bnSellPrice5m.Mean()) / (bnBuyVolume5m.Mean() + bnSellPrice5m.Mean()),

								float64(bnBuyVolume1s.Len()-bnSellPrice1s.Len()) / float64(bnBuyVolume1s.Len()+bnSellPrice1s.Len()),
								float64(bnBuyVolume5s.Len()-bnSellPrice5s.Len()) / float64(bnBuyVolume5s.Len()+bnSellPrice5s.Len()),
								float64(bnBuyVolume15s.Len()-bnSellPrice15s.Len()) / float64(bnBuyVolume15s.Len()+bnSellPrice15s.Len()),
								float64(bnBuyVolume30s.Len()-bnSellPrice30s.Len()) / float64(bnBuyVolume30s.Len()+bnSellPrice30s.Len()),
								float64(bnBuyVolume1m.Len()-bnSellPrice1m.Len()) / float64(bnBuyVolume1m.Len()+bnSellPrice1m.Len()),
								float64(bnBuyVolume3m.Len()-bnSellPrice3m.Len()) / float64(bnBuyVolume3m.Len()+bnSellPrice3m.Len()),
								float64(bnBuyVolume5m.Len()-bnSellPrice5m.Len()) / float64(bnBuyVolume5m.Len()+bnSellPrice5m.Len()),


								//(bnBuyPrice1s.Mean() - bnSellPrice1s.Mean()) / bnSellPrice1s.Mean(),
								//(bnBuyPrice5s.Mean() - bnSellPrice5s.Mean()) / bnSellPrice5s.Mean(),
								//(bnBuyPrice15s.Mean() - bnSellPrice15s.Mean()) / bnSellPrice15s.Mean(),
								//(bnBuyPrice30s.Mean() - bnSellPrice30s.Mean()) / bnSellPrice30s.Mean(),
								//(bnBuyPrice1m.Mean() - bnSellPrice1m.Mean()) / bnSellPrice1m.Mean(),
								//(bnBuyPrice3m.Mean() - bnSellPrice3m.Mean()) / bnSellPrice3m.Mean(),
								//(bnBuyPrice5m.Mean() - bnSellPrice5m.Mean()) / bnSellPrice5m.Mean(),
							}
						}
						if shortDelta > topBots[symbol][1] {
							shortMarkPrices[int(ftxTrade.Time.UnixNano()/1)] = bnSellPrice
							shortFeatures[int(ftxTrade.Time.UnixNano()/1)] = [featureCount]float64{

								//(bnBuyPrice1s.Mean() - bnBuyPrice5s.Mean()) / bnBuyPrice5s.Mean(),
								//(bnBuyPrice1s.Mean() - bnBuyPrice15s.Mean()) / bnBuyPrice15s.Mean(),
								//(bnBuyPrice1s.Mean() - bnBuyPrice30s.Mean()) / bnBuyPrice30s.Mean(),
								//(bnBuyPrice1s.Mean() - bnBuyPrice1m.Mean()) / bnBuyPrice1m.Mean(),
								//(bnBuyPrice1s.Mean() - bnBuyPrice3m.Mean()) / bnBuyPrice3m.Mean(),
								//(bnBuyPrice1s.Mean() - bnBuyPrice5m.Mean()) / bnBuyPrice5m.Mean(),
								//
								//(bnSellPrice1s.Mean() - bnSellPrice5s.Mean()) / bnSellPrice5s.Mean(),
								//(bnSellPrice1s.Mean() - bnSellPrice15s.Mean()) / bnSellPrice15s.Mean(),
								//(bnSellPrice1s.Mean() - bnSellPrice30s.Mean()) / bnSellPrice30s.Mean(),
								//(bnSellPrice1s.Mean() - bnSellPrice1m.Mean()) / bnSellPrice1m.Mean(),
								//(bnSellPrice1s.Mean() - bnSellPrice3m.Mean()) / bnSellPrice3m.Mean(),
								//(bnSellPrice1s.Mean() - bnSellPrice5m.Mean()) / bnSellPrice5m.Mean(),

								(bnBuyVolume1s.Mean() - bnSellVolume1s.Mean()) / (bnBuyVolume1s.Mean() + bnSellVolume1s.Mean()),
								(bnBuyVolume5s.Mean() - bnSellVolume5s.Mean()) / (bnBuyVolume5s.Mean() + bnSellVolume5s.Mean()),
								(bnBuyVolume15s.Mean() - bnSellVolume15s.Mean()) / (bnBuyVolume15s.Mean() + bnSellVolume15s.Mean()),
								(bnBuyVolume30s.Mean() - bnSellVolume30s.Mean()) / (bnBuyVolume30s.Mean() + bnSellVolume30s.Mean()),
								(bnBuyVolume1m.Mean() - bnSellVolume1m.Mean()) / (bnBuyVolume1m.Mean() + bnSellVolume1m.Mean()),
								(bnBuyVolume3m.Mean() - bnSellVolume3m.Mean()) / (bnBuyVolume3m.Mean() + bnSellVolume3m.Mean()),
								(bnBuyVolume5m.Mean() - bnSellVolume5m.Mean()) / (bnBuyVolume5m.Mean() + bnSellVolume5m.Mean()),

								float64(bnBuyPrice1s.Len()-bnSellPrice1s.Len()) / float64(bnBuyPrice1s.Len()+bnSellPrice1s.Len()),
								float64(bnBuyPrice5s.Len()-bnSellPrice5s.Len()) / float64(bnBuyPrice5s.Len()+bnSellPrice5s.Len()),
								float64(bnBuyPrice15s.Len()-bnSellPrice15s.Len()) / float64(bnBuyPrice15s.Len()+bnSellPrice15s.Len()),
								float64(bnBuyPrice30s.Len()-bnSellPrice30s.Len()) / float64(bnBuyPrice30s.Len()+bnSellPrice30s.Len()),
								float64(bnBuyPrice1m.Len()-bnSellPrice1m.Len()) / float64(bnBuyPrice1m.Len()+bnSellPrice1m.Len()),
								float64(bnBuyPrice3m.Len()-bnSellPrice3m.Len()) / float64(bnBuyPrice3m.Len()+bnSellPrice3m.Len()),
								float64(bnBuyPrice5m.Len()-bnSellPrice5m.Len()) / float64(bnBuyPrice5m.Len()+bnSellPrice5m.Len()),

								//(bnBuyPrice1s.Mean() - bnSellPrice1s.Mean()) / bnSellPrice1s.Mean(),
								//(bnBuyPrice5s.Mean() - bnSellPrice5s.Mean()) / bnSellPrice5s.Mean(),
								//(bnBuyPrice15s.Mean() - bnSellPrice15s.Mean()) / bnSellPrice15s.Mean(),
								//(bnBuyPrice30s.Mean() - bnSellPrice30s.Mean()) / bnSellPrice30s.Mean(),
								//(bnBuyPrice1m.Mean() - bnSellPrice1m.Mean()) / bnSellPrice1m.Mean(),
								//(bnBuyPrice3m.Mean() - bnSellPrice3m.Mean()) / bnSellPrice3m.Mean(),
								//(bnBuyPrice5m.Mean() - bnSellPrice5m.Mean()) / bnSellPrice5m.Mean(),
							}
						}

					}
					ftxTrade := ftxTrade
					lastFtxTrade = &ftxTrade
				}
			}
			_ = ftxGr.Close()
			_ = ftxFile.Close()
		}

		ts := make([]int,0, len(longLabels))
		for t := range longLabels {
			ts = append(ts, t)
		}
		sort.Ints(ts)
		longText := ""
		for _, t := range ts {
			label :=  longLabels[t]
			longText += fmt.Sprintf("%d,%.6f,%.6f\n", t, label, longFeatures[t])
		}
		longText = strings.Replace(longText, "[", "", -1)
		longText = strings.Replace(longText, "]", "", -1)
		longText = strings.Replace(longText, " ", ",", -1)

		ts = make([]int,0, len(shortLabels))
		for t := range shortLabels {
			ts = append(ts, t)
		}
		sort.Ints(ts)
		shortText := ""
		for _, t := range ts {
			label := shortLabels[t]
			shortText += fmt.Sprintf("%d,%.6f,%.6f\n", t, label, shortFeatures[t])
		}
		shortText = strings.Replace(shortText, "[", "", -1)
		shortText = strings.Replace(shortText, "]", "", -1)
		shortText = strings.Replace(shortText, " ", ",", -1)

		err := ioutil.WriteFile(fmt.Sprintf("/Users/chenjilin/Downloads/%s-long-features.txt", symbol), []byte(longText), 0644)
		if err != nil {
			logger.Fatal(err)
		}
		err = ioutil.WriteFile(fmt.Sprintf("/Users/chenjilin/Downloads/%s-short-features.txt", symbol), []byte(shortText), 0644)
		if err != nil {
			logger.Fatal(err)
		}

	}
}
