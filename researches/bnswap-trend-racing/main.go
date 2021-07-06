package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/influx/client"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/geometrybase/hft-micro/tdigest"
	"os"
	"strings"
	"time"
)

//func NewInfluxWriter(ctx context.Context, address, username, password, database string, batchSize int) (*InfluxWriter, error) {

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

	//symbols := `BTCUSDT,LTCUSDT,ETHUSDT,NEOUSDT,QTUMUSDT,EOSUSDT,ZRXUSDT,OMGUSDT,LRCUSDT,TRXUSDT,KNCUSDT,IOTAUSDT,LINKUSDT,CVCUSDT,ETCUSDT,ZECUSDT,BATUSDT,DASHUSDT,XMRUSDT,ENJUSDT,XRPUSDT,STORJUSDT,BTSUSDT,ADAUSDT,XLMUSDT,WAVESUSDT,ICXUSDT,RLCUSDT,IOSTUSDT,BLZUSDT,ONTUSDT,ZILUSDT,ZENUSDT,THETAUSDT,VETUSDT,RENUSDT,MATICUSDT,ATOMUSDT,FTMUSDT,CHZUSDT,ALGOUSDT,DOGEUSDT,ANKRUSDT,TOMOUSDT,BANDUSDT,XTZUSDT,KAVAUSDT,BCHUSDT,SOLUSDT,HNTUSDT,COMPUSDT,MKRUSDT,SXPUSDT,SNXUSDT,DOTUSDT,RUNEUSDT,BALUSDT,YFIUSDT,SRMUSDT,CRVUSDT,SANDUSDT,OCEANUSDT,LUNAUSDT,RSRUSDT,TRBUSDT,EGLDUSDT,BZRXUSDT,KSMUSDT,SUSHIUSDT,YFIIUSDT,BELUSDT,UNIUSDT,AVAXUSDT,FLMUSDT,ALPHAUSDT,NEARUSDT,AAVEUSDT,FILUSDT,CTKUSDT,AXSUSDT,AKROUSDT,SKLUSDT,GRTUSDT,1INCHUSDT,LITUSDT,RVNUSDT,SFPUSDT,REEFUSDT,DODOUSDT,COTIUSDT,CHRUSDT,ALICEUSDT,HBARUSDT,MANAUSDT,STMXUSDT,UNFIUSDT,XEMUSDT,CELRUSDT,HOTUSDT,ONEUSDT,LINAUSDT,DENTUSDT,MTLUSDT,OGNUSDT,NKNUSDT,DGBUSDT`

	symbols := "EOSUSDT"
	//dateStrs := "20210501,20210502,20210503,20210505,20210506,20210507,20210508,20210509,20210510,20210511"
	dateStrs := "20210511"

	tradeTimedDir := common.NewTimedSum(time.Second * 5)
	tradeTimedFastPrice := common.NewTimedWeightedMean(time.Second * 60)
	tradeTimedSlowPrice := common.NewTimedWeightedMean(time.Minute * 60)
	for _, symbol := range strings.Split(symbols, ",") {
		bookVolume := 0.0
		tradeBookRatio := 0.0
		tradeBookRatioTD, _ := tdigest.New()
		tradeDir := 0.0
		counter := 0
		bestBidPrice := 0.0
		bestAskPrice := 0.0
		var lastDepth *bnswap.Depth20
		var lastTrade *bnswap.Trade

		netWorth := 1.0
		commission := -0.0004
		nextTradeTime := time.Unix(0, 0)
		tradeInterval := time.Minute
		addOffset := 0.001
		addValue := 1.0
		lastMarkedPrice := 0.0
		positionSize := 0.0
		positionCost := 0.0

		for _, dateStr := range strings.Split(dateStrs, ",") {
			depthFile, err := os.Open(
				fmt.Sprintf("/Users/chenjilin/MarketData/bnswap-depth20/%s-%s.depth20.jl.gz", dateStr, symbol),
			)
			if err != nil {
				logger.Debugf("os.Open() error %v", err)
				return
			}
			depthGzReader, err := gzip.NewReader(depthFile)
			if err != nil {
				logger.Debugf("gzip.NewReader(depthFile) error %v", err)
				return
			}
			depthScanner := bufio.NewScanner(depthGzReader)
			tradeFile, err := os.Open(
				fmt.Sprintf("/Users/chenjilin/MarketData/bnswap-trade/%s-%s.bnswap.trade.jl.gz", dateStr, symbol),
			)
			if err != nil {
				logger.Debugf("os.Open() error %v", err)
				return
			}
			tradeGzReader, err := gzip.NewReader(tradeFile)
			if err != nil {
				logger.Debugf("gzip.NewReader(depthFile) error %v", err)
				return
			}
			tradeScanner := bufio.NewScanner(tradeGzReader)

			for tradeScanner.Scan() {
				trade, err := bnswap.ParseTrade(tradeScanner.Bytes())
				if err != nil {
					logger.Debugf(" bnswap.ParseTrade error %v", err)
					continue
				}
				if lastDepth == nil || trade.EventTime.Sub(lastDepth.EventTime) > time.Millisecond*100 {
					for depthScanner.Scan() {
						depth, err := bnswap.ParseDepth20(depthScanner.Bytes())
						if err != nil {
							logger.Debugf("bnswap.ParseDepth20 error %v", err)
							continue
						}
						if trade.EventTime.Sub(depth.EventTime) < time.Millisecond*100 {
							lastDepth = depth
							break
						}
					}
				}
				if lastDepth == nil || lastDepth.EventTime.Sub(trade.EventTime) > 0 {
					//如果depth比trade新，说明没有合适的depth
					lastTrade = trade
					continue
				}
				tradeTimedFastPrice.Insert(trade.EventTime, trade.Quantity, trade.Price)
				tradeTimedSlowPrice.Insert(trade.EventTime, trade.Quantity, trade.Price)
				if lastTrade != nil {
					tradeDir = (tradeTimedFastPrice.Mean() - tradeTimedSlowPrice.Mean())/trade.Price
					//tradeDir = (trade.Price - lastTrade.Price) / trade.Price
					tradeTimedDir.Insert(lastTrade.EventTime, tradeDir)
				}
				lastTrade = trade

				bookVolume = 0
				for i := 0; i < 20; i++ {
					bookVolume += lastDepth.Bids[i][1] * lastDepth.Bids[i][0]
					bookVolume += lastDepth.Asks[i][1] * lastDepth.Asks[i][0]
				}
				if bookVolume > 0 {
					tradeBookRatio = lastTrade.Price * lastTrade.Quantity / bookVolume
					_ = tradeBookRatioTD.Add(tradeBookRatio)
				}
				bestBidPrice = lastDepth.Bids[0][0]
				bestAskPrice = lastDepth.Asks[0][0]

				if trade.EventTime.Sub(nextTradeTime) > 0 {
					netWorth, positionSize, positionCost, lastMarkedPrice, nextTradeTime = strategy1(
						tradeBookRatio,
						tradeDir,
						addOffset,
						addValue,
						commission,
						netWorth,
						bestBidPrice,
						bestAskPrice,
						tradeTimedFastPrice.Mean(),
						positionSize,
						positionCost,
						lastMarkedPrice,
						trade.EventTime,
						tradeInterval,
					)
				}

				fields := make(map[string]interface{})
				if lastTrade != nil && lastTrade.Price > 0 {
					fields["lastPrice"] = lastTrade.Price
				}
				if positionSize > 0 {
					fields["netWorth"] = netWorth + positionSize*(bestBidPrice-positionCost)/positionCost
				} else if positionSize < 0 {
					fields["netWorth"] = netWorth + positionSize*(bestAskPrice-positionCost)/positionCost
				} else {
					fields["netWorth"] = netWorth
				}
				fields["positionSize"] = positionSize
				if positionCost != 0 {
					fields["positionCost"] = positionCost
				}
				fields["bookVolume"] = bookVolume
				if lastTrade != nil {
					fields["tradeVolume"] = lastTrade.Quantity * lastTrade.Price
				}
				fields["tradeDir"] = tradeDir
				fields["tradeTimedFastPrice"] = tradeTimedFastPrice.Mean()
				fields["tradeBookRatio"] = tradeBookRatio
				fields["bestBidPrice"] = bestBidPrice
				fields["bestAskPrice"] = bestAskPrice
				fields["tradeBookRatioQ99995"] = tradeBookRatioTD.Quantile(0.99995)
				fields["tradeBookRatioQ9995"] = tradeBookRatioTD.Quantile(0.9995)
				fields["tradeBookRatioQ995"] = tradeBookRatioTD.Quantile(0.995)

				pt, err := client.NewPoint(
					"bnswap-trend-racing",
					map[string]string{
						"symbol": symbol,
					},
					fields,
					trade.EventTime,
				)
				iw.PointCh <- pt
				counter++
			}
			_ = depthGzReader.Close()
			_ = depthFile.Close()
			_ = tradeGzReader.Close()
			_ = tradeFile.Close()
			logger.Debugf("%s %d %f", symbol, counter, netWorth)
			time.Sleep(time.Second)
		}
	}
}
