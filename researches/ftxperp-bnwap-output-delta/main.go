package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/ftx-usdfuture"
	"github.com/geometrybase/hft-micro/influx/client"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/geometrybase/hft-micro/tdigest"
	"os"
	"sort"
	"strings"
	"time"
)

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
	for ftxMarket := range ftx_usdfuture.PriceIncrements {
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
	symbols = matchSymbols[:]
	dateStrs := "20210509,20210510,20210511,20210512,20210513"
	deltaQuantiles := make(map[string]string)
	saveToInflux := true
	for _, symbol := range symbols {
		var lastBnTrade *bnswap.Trade
		var lastFtxTrade *ftx_usdfuture.Trade
		longDeltaTD, _ := tdigest.New()
		shortDeltaTD, _ := tdigest.New()
		lookback := time.Second
		ftxBuyMean := common.NewTimedMean(lookback)
		ftxSellMean := common.NewTimedMean(lookback)
		bnBuyMean := common.NewTimedMean(lookback)
		bnSellMean := common.NewTimedMean(lookback)
		for _, dateStr := range strings.Split(dateStrs, ",") {
			ftxFile, err := os.Open(
				fmt.Sprintf("/home/clu/MarketData/ftx-usdfuture-trade/%s-%s.ftx-usdfuture.trade.jl.gz", dateStr, symbol),
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
				fmt.Sprintf("/home/clu/MarketData/bnswap-trade/%s-%s.bnswap.trade.jl.gz", dateStr, pairMaps[symbol]),
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
				tradeData := ftx_usdfuture.TradesData{}
				err = json.Unmarshal(ftxScanner.Bytes(), &tradeData)
				if err != nil {
					continue
				}
				for _, ftxTrade := range tradeData.Data {
					if ftxTrade.Side == ftx_usdfuture.TradeSideBuy {
						ftxBuyMean.Insert(ftxTrade.Time, ftxTrade.Price)
					} else {
						ftxSellMean.Insert(ftxTrade.Time, ftxTrade.Price)
					}
					if lastFtxTrade == nil ||
						ftxTrade.Time.Truncate(lookback).Sub(lastFtxTrade.Time.Truncate(lookback)) <= 0 {
						ftxTrade := ftxTrade
						lastFtxTrade = &ftxTrade
						continue
					}
					ftxTrade := ftxTrade
					lastFtxTrade = &ftxTrade
					for bnScanner.Scan() {
						if lastBnTrade != nil {
							if lastBnTrade.IsTheBuyerTheMarketMaker {
								bnSellMean.Insert(lastBnTrade.EventTime, lastBnTrade.Price)
							} else {
								bnBuyMean.Insert(lastBnTrade.EventTime, lastBnTrade.Price)
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

					if ftxBuyMean.Len() > 0 &&
						ftxSellMean.Len() > 0 &&
						bnBuyMean.Len() > 0 &&
						bnSellMean.Len() > 0 {
						ftxBuyPrice := ftxBuyMean.Mean()
						ftxSellPrice := ftxSellMean.Mean()
						bnBuyPrice := bnBuyMean.Mean()
						bnSellPrice := bnSellMean.Mean()
						longDelta := (bnBuyPrice - ftxBuyPrice) / ftxBuyPrice
						shortDelta := (bnSellPrice - ftxSellPrice) / ftxSellPrice
						_ = longDeltaTD.Add(longDelta)
						_ = shortDeltaTD.Add(shortDelta)
						if saveToInflux && ftxTrade.Time.Sub(ftxTrade.Time.Truncate(time.Minute)) < lookback {
							fields := make(map[string]interface{})
							fields["longDelta"] = longDelta
							fields["shortDelta"] = shortDelta
							fields["longDeltaQ0.05"] = longDeltaTD.Quantile(0.05)
							fields["longDeltaQ0.1"] = longDeltaTD.Quantile(0.1)
							fields["longDeltaQ0.2"] = longDeltaTD.Quantile(0.2)
							fields["longDeltaQ0.5"] = longDeltaTD.Quantile(0.5)
							fields["longDeltaQ0.8"] = longDeltaTD.Quantile(0.8)
							fields["longDeltaQ0.9"] = longDeltaTD.Quantile(0.9)
							fields["longDeltaQ0.95"] = longDeltaTD.Quantile(0.95)
							fields["longDeltaQ0.995"] = longDeltaTD.Quantile(0.995)
							fields["shortDeltaQ0.005"] = shortDeltaTD.Quantile(0.005)
							fields["shortDeltaQ0.05"] = shortDeltaTD.Quantile(0.05)
							fields["shortDeltaQ0.1"] = shortDeltaTD.Quantile(0.1)
							fields["shortDeltaQ0.2"] = shortDeltaTD.Quantile(0.2)
							fields["shortDeltaQ0.5"] = shortDeltaTD.Quantile(0.5)
							fields["shortDeltaQ0.8"] = shortDeltaTD.Quantile(0.8)
							fields["shortDeltaQ0.9"] = shortDeltaTD.Quantile(0.9)
							fields["shortDeltaQ0.95"] = shortDeltaTD.Quantile(0.95)

							fields["ftxBuyPrice"] = ftxBuyMean.Mean()
							fields["ftxSellPrice"] = ftxSellMean.Mean()
							fields["bnBuyPrice"] = bnBuyMean.Mean()
							fields["bnSellPrice"] = bnSellMean.Mean()
							pt, err := client.NewPoint(
								"ftx-usdfuture-bnswap-delta",
								map[string]string{
									"symbol": symbol,
								},
								fields,
								ftxTrade.Time,
							)
							if err != nil {
								logger.Fatal(err)
							}
							iw.PointCh <- pt
						}
					}

				}
			}
			_ = ftxGr.Close()
			_ = ftxFile.Close()
		}
		deltaQuantiles[symbol] = fmt.Sprintf(
			"%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f",
			longDeltaTD.Quantile(0.005),
			longDeltaTD.Quantile(0.01),
			longDeltaTD.Quantile(0.05),
			longDeltaTD.Quantile(0.2),
			shortDeltaTD.Quantile(0.8),
			shortDeltaTD.Quantile(0.95),
			shortDeltaTD.Quantile(0.99),
			shortDeltaTD.Quantile(0.995),
		)
		//deltaQuantiles[symbol] = fmt.Sprintf(
		//	"  \"%s\": {%.6f,%.6f},\n",
		//	symbol,
		//	longDeltaTD.Quantile(0.1),
		//	shortDeltaTD.Quantile(0.9),
		//)
	}
	fmt.Printf("\n\n")
	for _, symbol := range symbols {
		fmt.Printf("%s: %s\n", symbol, deltaQuantiles[symbol])
	}
	fmt.Printf("\n\n")

	//fmt.Printf("\n\nvar topBots = map[string][2]float64{\n")
	//for _, symbol := range symbols {
	//	fmt.Printf("%s", deltaQuantiles[symbol])
	//}
	//fmt.Printf("}\n\n")
}
