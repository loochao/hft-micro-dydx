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

	symbols := `BTCUSDT,LTCUSDT,ETHUSDT,NEOUSDT,QTUMUSDT,EOSUSDT,ZRXUSDT,OMGUSDT,LRCUSDT,TRXUSDT,KNCUSDT,IOTAUSDT,LINKUSDT,CVCUSDT,ETCUSDT,ZECUSDT,BATUSDT,DASHUSDT,XMRUSDT,ENJUSDT,XRPUSDT,STORJUSDT,BTSUSDT,ADAUSDT,XLMUSDT,WAVESUSDT,ICXUSDT,RLCUSDT,IOSTUSDT,BLZUSDT,ONTUSDT,ZILUSDT,ZENUSDT,THETAUSDT,VETUSDT,RENUSDT,MATICUSDT,ATOMUSDT,FTMUSDT,CHZUSDT,ALGOUSDT,DOGEUSDT,ANKRUSDT,TOMOUSDT,BANDUSDT,XTZUSDT,KAVAUSDT,BCHUSDT,SOLUSDT,HNTUSDT,COMPUSDT,MKRUSDT,SXPUSDT,SNXUSDT,DOTUSDT,RUNEUSDT,BALUSDT,YFIUSDT,SRMUSDT,CRVUSDT,SANDUSDT,OCEANUSDT,LUNAUSDT,RSRUSDT,TRBUSDT,EGLDUSDT,BZRXUSDT,KSMUSDT,SUSHIUSDT,YFIIUSDT,BELUSDT,UNIUSDT,AVAXUSDT,FLMUSDT,ALPHAUSDT,NEARUSDT,AAVEUSDT,FILUSDT,CTKUSDT,AXSUSDT,AKROUSDT,SKLUSDT,GRTUSDT,1INCHUSDT,LITUSDT,RVNUSDT,SFPUSDT,REEFUSDT,DODOUSDT,COTIUSDT,CHRUSDT,ALICEUSDT,HBARUSDT,MANAUSDT,STMXUSDT,UNFIUSDT,XEMUSDT,CELRUSDT,HOTUSDT,ONEUSDT,LINAUSDT,DENTUSDT,MTLUSDT,OGNUSDT,NKNUSDT,DGBUSDT`
	//symbols := "COMPUSDT"
	dateStrs := "20210501,20210502,20210503,20210505,20210506,20210507,20210508,20210509,20210510,20210511"
	//dateStrs := "20210511"
	for _, symbol := range strings.Split(symbols, ",") {

		depthSize := 0.0
		depthBidSize := 0.0
		depthAskSize := 0.0
		depthSizeMean := common.NewTimedMean(time.Hour)
		depthSizeMean2 := common.NewTimedMean(time.Hour*4)
		depthImbalance := 0.0
		depthImbalanceMean := common.NewTimedMean(time.Hour)
		depthImbalanceMean2 := common.NewTimedMean(time.Hour*4)

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
				depth, err := bnswap.ParseDepth20(depthScanner.Bytes())
				if err != nil {
					logger.Debugf("bnswap.ParseDepth20 error %v", err)
					continue
				}
				depthSize = 0
				depthBidSize = 0
				depthAskSize = 0
				for i := 0; i < 20; i++ {
					depthBidSize += depth.Bids[i][1] * depth.Bids[i][0]
					depthAskSize += depth.Asks[i][1] * depth.Asks[i][0]
				}
				depthSize = depthBidSize + depthAskSize
				depthImbalance = (depthBidSize - depthAskSize) / depthSize
				depthImbalanceMean.Insert(depth.EventTime, depthImbalance)
				depthImbalanceMean2.Insert(depth.EventTime, depthImbalance)
				depthSizeMean.Insert(depth.EventTime, depthSize)
				depthSizeMean2.Insert(depth.EventTime, depthSize)

				fields := make(map[string]interface{})
				fields["depthSize"] = depthSize
				fields["depthBidSize"] = depthBidSize
				fields["depthAskSize"] = depthAskSize
				fields["depthSizeMean"] = depthSizeMean.Mean()
				fields["depthSizeMean2"] = depthSizeMean2.Mean()
				fields["depthImbalance"] = depthImbalance
				fields["depthImbalanceMean"] = depthImbalanceMean.Mean()
				fields["depthImbalanceMean2"] = depthImbalanceMean2.Mean()
				fields["bestBidPrice"] = depth.Bids[0][0]
				fields["bestAskPrice"] = depth.Asks[0][0]
				pt, err := client.NewPoint(
					"bnswap-depth-factors",
					map[string]string{
						"symbol": symbol,
					},
					fields,
					depth.EventTime,
				)
				iw.PointCh <- pt
			}
			_ = depthGzReader.Close()
			_ = depthFile.Close()
			time.Sleep(time.Second)
		}
	}
}
