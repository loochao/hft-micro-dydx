package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
	binance_usdtspot "github.com/geometrybase/hft-micro/binance-usdtspot"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/influx/client"
	"github.com/geometrybase/hft-micro/logger"
	stream_stats "github.com/geometrybase/hft-micro/stream-stats"
	"os"
	"path"
	"strings"
	"time"
)

func main() {
	ctx := context.Background()
	iw, err := common.NewInfluxWriter(
		ctx,
		os.Getenv("INFLUX_URL"),
		os.Getenv("INFLUX_USER"),
		os.Getenv("INFLUX_PASS"),
		"hft",
		500,
	)
	if err != nil {
		panic(err)
	}
	defer iw.Stop()

	symbols := strings.Split(
		`1INCHUSDT,AAVEUSDT,ADAUSDT,AKROUSDT,ALGOUSDT,ALICEUSDT,ALPHAUSDT,ANKRUSDT,ATOMUSDT,AVAXUSDT,AXSUSDT,BAKEUSDT,BALUSDT,BANDUSDT,BATUSDT,BCHUSDT,BELUSDT,BLZUSDT,BNBUSDT,BTCUSDT,BTSUSDT,BTTUSDT,BZRXUSDT,CELRUSDT,CHRUSDT,CHZUSDT,COMPUSDT,COTIUSDT,CRVUSDT,CTKUSDT,CVCUSDT,DASHUSDT,DENTUSDT,DGBUSDT,DODOUSDT,DOGEUSDT,DOTUSDT,EGLDUSDT,ENJUSDT,EOSUSDT,ETCUSDT,ETHUSDT,FILUSDT,FLMUSDT,FTMUSDT,GRTUSDT,HBARUSDT,HNTUSDT,HOTUSDT,ICPUSDT,ICXUSDT,IOSTUSDT,IOTAUSDT,KAVAUSDT,KNCUSDT,KSMUSDT,LINAUSDT,LINKUSDT,LITUSDT,LRCUSDT,LTCUSDT,LUNAUSDT,MANAUSDT,MATICUSDT,MKRUSDT,MTLUSDT,NEARUSDT,NEOUSDT,NKNUSDT,OCEANUSDT,OGNUSDT,OMGUSDT,ONEUSDT,ONTUSDT,QTUMUSDT,REEFUSDT,RENUSDT,RLCUSDT,RSRUSDT,RUNEUSDT,RVNUSDT,SANDUSDT,SCUSDT,SFPUSDT,SKLUSDT,SNXUSDT,SOLUSDT,SRMUSDT,STMXUSDT,STORJUSDT,SUSHIUSDT,SXPUSDT,THETAUSDT,TOMOUSDT,TRBUSDT,TRXUSDT,UNFIUSDT,UNIUSDT,VETUSDT,WAVESUSDT,XEMUSDT,XLMUSDT,XMRUSDT,XRPUSDT,XTZUSDT,YFIIUSDT,YFIUSDT,ZECUSDT,ZENUSDT,ZILUSDT,ZRXUSDT`,
		",",
	)
	startTime, err := time.Parse("20060102", "20210622")
	if err != nil {
		logger.Fatal(err)
	}
	endTime, err := time.Parse("20060102", "20210704")
	if err != nil {
		logger.Fatal(err)
	}
	dateStrs := ""
	for i := startTime; i.Sub(endTime) <= 0; i = i.Add(time.Hour * 24) {
		dateStrs += i.Format("20060102,")
	}
	dateStrs = dateStrs[:len(dateStrs)-1]

	quantileLookback := time.Hour * 72
	quantileSubInterval := time.Hour
	xMultiplier := 1.0
	yMultiplier := 1.0
	depthTakerImpact := 300.0
	quantileTop := 0.95
	quantileBot := 0.05
	shortQuantileTop := 0.0
	longQuantileBot := 0.0
	quantilePath := "/home/clu/Projects/hft-micro/applications/usd-ll-mt-q/configs/quantiles"

	longTimedTDigests := make(map[string]*stream_stats.TimedTDigest)
	shortTimedTDigests := make(map[string]*stream_stats.TimedTDigest)
	for _, symbol := range symbols {
		counter := 0
		longTimedTDigest := stream_stats.NewTimedTDigest(quantileLookback, quantileSubInterval)
		shortTimedTDigest := stream_stats.NewTimedTDigest(quantileLookback, quantileSubInterval)

		shortLastEnter := 0.0
		longLastEnter := 0.0

		xDepth := &binance_usdtspot.Depth5{}
		yDepth := &binance_usdtfuture.Depth5{}
		xWalkedDepth := &common.WalkedDepthBBMAA{}
		yWalkedDepth := &common.WalkedDepthBBMAA{}
		for _, dateStr := range strings.Split(dateStrs, ",") {
			logger.Debugf("%s %s %s", symbol, dateStr, fmt.Sprintf("/home/clu/MarketData/bnspot-bnswap-depth5/%s/%s-%s.depth5.jl.gz", dateStr, dateStr, symbol))
			file, err := os.Open(
				fmt.Sprintf("/home/clu/MarketData/bnspot-bnswap-depth5/%s/%s-%s.depth5.jl.gz", dateStr, dateStr, symbol),
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
			for scanner.Scan() {
				counter++
				msg = scanner.Bytes()
				if msg[0] == 'S' {
					err = binance_usdtspot.ParseDepth5(msg[1:], xDepth)
					if err != nil {
						logger.Debugf("%v", err)
						continue
					}
					err = common.WalkDepthBBMAA(xDepth, xMultiplier, depthTakerImpact, xWalkedDepth)
					if err != nil {
						logger.Debugf("%v", err)
						continue
					}
				} else if msg[0] == 'F' {
					err = binance_usdtfuture.ParseDepth5(msg[1:], yDepth)
					if err != nil {
						logger.Debugf("%v", err)
						continue
					}
					err = common.WalkDepthBBMAA(yDepth, yMultiplier, depthTakerImpact, yWalkedDepth)
					if err != nil {
						logger.Debugf("%v", err)
						continue
					}
				}

				if xWalkedDepth.Symbol != "" && yWalkedDepth.Symbol != "" {
					shortLastEnter = (yWalkedDepth.BidPrice - xWalkedDepth.MidPrice) / xWalkedDepth.MidPrice
					longLastEnter = (yWalkedDepth.AskPrice - xWalkedDepth.MidPrice) / xWalkedDepth.MidPrice

					_ = shortTimedTDigest.Insert(yWalkedDepth.Time, shortLastEnter)
					_ = longTimedTDigest.Insert(yWalkedDepth.Time, longLastEnter)
					if counter%1000 == 0 {
						shortQuantileTop = shortTimedTDigest.Quantile(quantileTop)
						longQuantileBot = longTimedTDigest.Quantile(quantileBot)
						fields := make(map[string]interface{})
						fields["shortQuantileTop"] = shortQuantileTop
						fields["longQuantileBot"] =  longQuantileBot
						fields["shortQuantileTop80"] = shortTimedTDigest.Quantile(0.8)
						fields["shortQuantileTop50"] = shortTimedTDigest.Quantile(0.5)
						fields["longQuantileBot20"] =  longTimedTDigest.Quantile(0.2)
						fields["longQuantileBot50"] =  longTimedTDigest.Quantile(0.5)
						fields["shortLastEnter"] = shortLastEnter
						fields["longLastEnter"] = longLastEnter
						pt, err := client.NewPoint(
							"usd-ll-mt-q",
							map[string]string{
								"symbol": symbol,
							},
							fields,
							yWalkedDepth.Time,
						)
						if err == nil {
							iw.PointCh <- pt
						}
					}
				}
			}
			_ = gr.Close()
			_ = file.Close()
		}
		longTimedTDigests[symbol] = longTimedTDigest
		shortTimedTDigests[symbol] = shortTimedTDigest
	}
	for _, symbol := range symbols {
		data, err := json.Marshal(longTimedTDigests[symbol])
		if err != nil {
			logger.Debugf("%v", err)
			continue
		}
		file, err := os.OpenFile(path.Join(quantilePath, symbol+"-"+symbol+"-long-td.json"), os.O_CREATE|os.O_WRONLY, 0775)
		if err != nil {
			logger.Debugf("%v", err)
			continue
		}
		_, err = file.Write(data)
		if err != nil {
			logger.Debugf("%v", err)
			continue
		}
		err = file.Close()
		if err != nil {
			logger.Debugf("%v", err)
			continue
		}
		data, err = json.Marshal(shortTimedTDigests[symbol])
		if err != nil {
			logger.Debugf("%v", err)
			continue
		}
		file, err = os.OpenFile(path.Join(quantilePath, symbol+"-"+symbol+"-short-td.json"), os.O_CREATE|os.O_WRONLY, 0775)
		if err != nil {
			logger.Debugf("%v", err)
			continue
		}
		_, err = file.Write(data)
		if err != nil {
			logger.Debugf("%v", err)
			continue
		}
		err = file.Close()
		if err != nil {
			logger.Debugf("%v", err)
			continue
		}
	}
}



