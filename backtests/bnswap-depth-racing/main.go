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
	symbols := "ETHUSDT"
	//dateStrs := "20210501,20210502,20210503,20210505,20210506,20210507,20210508,20210509,20210510,20210511"
	dateStrs := "20210511"
	for _, symbol := range strings.Split(symbols, ",") {

		depthSize := 0.0
		depthDeltaRatio := 0.0
		depthDeltaRatioTD, _ := tdigest.New()
		counter := 0
		bestBidPrice := 0.0
		bestAskPrice := 0.0
		var lastDepth *bnswap.Depth20
		lastDepthSize := 0.0
		depthDirTD, _ := tdigest.New()
		depthDir := 0.0
		depthTimedDir := common.NewTimedSum(time.Hour * 4)
		depthTimedSizeDelta := common.NewTimedSum(time.Second * 15)

		netWorth := 1.0
		commission := -0.0004
		nextTradeTime := time.Unix(0, 0)
		tradeInterval := time.Minute * 60
		addOffset := 0.001
		addValue := 0.1
		lastFilledPrice := 0.0
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
			for depthScanner.Scan() {
				d, err := bnswap.ParseDepth20(depthScanner.Bytes())
				if err != nil {
					logger.Debugf("bnswap.ParseDepth20 error %v", err)
					continue
				}
				depthSize = 0
				for i := 0; i < 20; i++ {
					depthSize += d.Bids[i][1] * d.Bids[i][0]
					depthSize += d.Asks[i][1] * d.Asks[i][0]
				}
				if lastDepth != nil {
					dir := (d.Bids[0][0] - lastDepth.Bids[0][0] + d.Asks[0][0] - lastDepth.Asks[0][0]) / (lastDepth.Asks[0][0] + lastDepth.Bids[0][0])
					depthTimedDir.Insert(d.EventTime, dir)
					depthDir = depthTimedDir.Sum()
					_ = depthDirTD.Add(depthDir)
					if depthSize-lastDepthSize > 0 {
						depthTimedSizeDelta.Insert(d.EventTime, depthSize-lastDepthSize)
						if depthSize > 0 {
							depthDeltaRatio = depthTimedSizeDelta.Sum() / depthSize
							_ = depthDeltaRatioTD.Add(depthDeltaRatio)
						}
					//}else {
					//	depthTimedSizeDelta.Insert(d.EventTime, 0)
					//	depthDeltaRatio = 0
					//	_ = depthDeltaRatioTD.Add(depthDeltaRatio)
					}
				}
				lastDepth = d
				lastDepthSize = depthSize
				bestBidPrice = d.Bids[0][0]
				bestAskPrice = d.Asks[0][0]

				if d.EventTime.Sub(nextTradeTime) > 0 {
					netWorth, positionSize, positionCost, lastFilledPrice, nextTradeTime = strategy1(
						depthDir,
						0.005,
						-0.005,
						addOffset,
						addValue,
						commission,
						netWorth,
						bestBidPrice,
						bestAskPrice,
						positionSize,
						positionCost,
						lastFilledPrice,
						d.EventTime,
						tradeInterval,
					)
				}

				fields := make(map[string]interface{})
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
				if lastFilledPrice != 0 {
					fields["lastFilledPrice"] = lastFilledPrice
				}
				fields["bestBidPrice"] = bestBidPrice
				fields["bestAskPrice"] = bestAskPrice
				fields["depthSize"] = depthSize
				fields["depthTimedSizeDelta"] = depthTimedSizeDelta.Sum()
				fields["depthDeltaRatio"] = depthDeltaRatio
				fields["depthDeltaRatioQ9995"] = depthDeltaRatioTD.Quantile(0.9995)
				fields["depthDeltaRatioQ995"] = depthDeltaRatioTD.Quantile(0.995)
				fields["depthDeltaRatioQ99"] = depthDeltaRatioTD.Quantile(0.99)
				fields["depthDeltaRatioQ95"] = depthDeltaRatioTD.Quantile(0.95)
				fields["depthDeltaRatioQ80"] = depthDeltaRatioTD.Quantile(0.80)

				fields["depthDir"] = depthDir
				fields["depthDirQ9995"] = depthDirTD.Quantile(0.9995)
				fields["depthDirQ995"] = depthDirTD.Quantile(0.995)
				fields["depthDirQ80"] = depthDirTD.Quantile(0.80)
				fields["depthDirQ0005"] = depthDirTD.Quantile(0.0005)
				fields["depthDirQ005"] = depthDirTD.Quantile(0.005)
				fields["depthDirQ20"] = depthDirTD.Quantile(0.20)
				pt, err := client.NewPoint(
					"bnswap-depth-racing",
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
			logger.Debugf("%s %d %f", symbol, counter, netWorth)
			time.Sleep(time.Second)
		}
	}
}

