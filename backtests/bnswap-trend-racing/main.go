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
	//dateStrs := "20210501,20210502,20210503,20210505,20210506,20210507,20210508,20210509,20210510,20210511"

	symbols := "EOSUSDT"
	dateStrs := "20210511"

	tradeTimedDir := common.NewTimedSum(time.Second*5)
	tradeTimedPrice := common.NewTimedWeightedMean(time.Second*60)
	for _, symbol := range strings.Split(symbols, ",") {
		bookVolume := 0.0
		tradeBookRatio := 0.0
		tradeBookRatioTD, _ := tdigest.New()
		tradeDir := 0.0
		counter := 0
		bestBidPrice := 0.0
		bestAskPrice := 0.0
		var lastTrade *bnswap.Trade
		var lastLastTrade *bnswap.Trade

		netWorth := 1.0
		commission := -0.0004
		nextTradeTime := time.Unix(0, 0)
		tradeInterval := time.Minute*3
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

			for depthScanner.Scan() {
				d, err := bnswap.ParseDepth20(depthScanner.Bytes())
				if err != nil {
					logger.Debugf("bnswap.ParseDepth20 error %v", err)
					continue
				}

				for tradeScanner.Scan() {
					trade, err := bnswap.ParseTrade(tradeScanner.Bytes())
					if err != nil {
						logger.Debugf(" bnswap.ParseTrade error %v", err)
						continue
					}
					tradeTimedPrice.Insert(trade.EventTime, trade.Quantity, trade.Price)
					if lastTrade != nil &&
						lastTrade.Price > 0 &&
						lastLastTrade != nil &&
						lastLastTrade.Price > 0 {
						tradeDir = (lastTrade.Price-lastLastTrade.Price)/lastLastTrade.Price
						tradeTimedDir.Insert(lastTrade.EventTime, lastTrade.Price-lastLastTrade.Price)
					}
					lastLastTrade = lastTrade
					lastTrade = trade
					if trade.EventTime.Sub(d.EventTime) > 0 {
						break
					}
				}

				bookVolume = 0
				//tradeDir = 0
				for i := 0; i < 20; i++ {
					bookVolume += d.Bids[i][1] * d.Bids[i][0]
					bookVolume += d.Asks[i][1] * d.Asks[i][0]
				}
				//lastDepth = d
				if bookVolume > 0 {
					tradeBookRatio = lastTrade.Price * lastTrade.Quantity / bookVolume
					_ = tradeBookRatioTD.Add(tradeBookRatio)
				}
				bestBidPrice = d.Bids[0][0]
				bestAskPrice = d.Asks[0][0]
				//tradeDir = tradeTimedDir.Sum()

				if d.EventTime.Sub(nextTradeTime) > 0 {
					netWorth, positionSize, positionCost, lastMarkedPrice, nextTradeTime = strategy1(
						tradeBookRatio,
						tradeDir,
						addOffset,
						addValue,
						commission,
						netWorth,
						bestBidPrice,
						bestAskPrice,
						tradeTimedPrice.Mean(),
						positionSize,
						positionCost,
						lastMarkedPrice,
						d.EventTime,
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
				fields["tradeVolume"] = lastTrade.Quantity * lastTrade.Price
				fields["tradeDir"] = tradeDir
				fields["tradeTimedPrice"] = tradeTimedPrice.Mean()
				fields["tradeBookRatio"] = tradeBookRatio
				fields["tradeBookRatioQ99995"] = tradeBookRatioTD.Quantile(0.99995)
				fields["tradeBookRatioQ9995"] = tradeBookRatioTD.Quantile(0.9995)
				fields["tradeBookRatioQ995"] = tradeBookRatioTD.Quantile(0.995)

				//fields["depthMotion"] = depthMotion
				//fields["depthMotionQ9995"] = depthMotionTD.Quantile(0.9995)
				//fields["depthMotionQ995"] = depthMotionTD.Quantile(0.995)
				//fields["depthMotionQ99"] = depthMotionTD.Quantile(0.99)
				//fields["depthMotionQ95"] = depthMotionTD.Quantile(0.95)
				//fields["depthMotionQ80"] = depthMotionTD.Quantile(0.80)
				pt, err := client.NewPoint(
					"bnswap-trend-racing",
					map[string]string{
						"symbol": symbol,
					},
					fields,
					d.EventTime,
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
