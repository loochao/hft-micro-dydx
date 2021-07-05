package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/influx/client"
	kucoin_usdtfuture "github.com/geometrybase/hft-micro/kucoin-usdtfuture"
	"github.com/geometrybase/hft-micro/logger"
	stream_stats "github.com/geometrybase/hft-micro/stream-stats"
	"os"
	"path"
	"strings"
	"time"
)

var symbolsMap = map[string]string{
	"BTCUSDT":   "XBTUSDTM",
	"IOSTUSDT":  "IOSTUSDTM",
	"UNIUSDT":   "UNIUSDTM",
	"ICPUSDT":   "ICPUSDTM",
	"THETAUSDT": "THETAUSDTM",
	"YFIUSDT":   "YFIUSDTM",
	"OCEANUSDT": "OCEANUSDTM",
	"XMRUSDT":   "XMRUSDTM",
	"SXPUSDT":   "SXPUSDTM",
	"BCHUSDT":   "BCHUSDTM",
	"TRXUSDT":   "TRXUSDTM",
	"XEMUSDT":   "XEMUSDTM",
	"ETHUSDT":   "ETHUSDTM",
	"MKRUSDT":   "MKRUSDTM",
	"FTMUSDT":   "FTMUSDTM",
	"ATOMUSDT":  "ATOMUSDTM",
	"BANDUSDT":  "BANDUSDTM",
	"DOTUSDT":   "DOTUSDTM",
	"FILUSDT":   "FILUSDTM",
	"AVAXUSDT":  "AVAXUSDTM",
	"QTUMUSDT":  "QTUMUSDTM",
	"COMPUSDT":  "COMPUSDTM",
	"ZECUSDT":   "ZECUSDTM",
	"ADAUSDT":   "ADAUSDTM",
	"DOGEUSDT":  "DOGEUSDTM",
	"XLMUSDT":   "XLMUSDTM",
	"EOSUSDT":   "EOSUSDTM",
	"LTCUSDT":   "LTCUSDTM",
	"VETUSDT":   "VETUSDTM",
	"ONTUSDT":   "ONTUSDTM",
	"RVNUSDT":   "RVNUSDTM",
	"MATICUSDT": "MATICUSDTM",
	"1INCHUSDT": "1INCHUSDTM",
	"XRPUSDT":   "XRPUSDTM",
	"NEOUSDT":   "NEOUSDTM",
	"ALGOUSDT":  "ALGOUSDTM",
	"MANAUSDT":  "MANAUSDTM",
	"WAVESUSDT": "WAVESUSDTM",
	"KSMUSDT":   "KSMUSDTM",
	"AAVEUSDT":  "AAVEUSDTM",
	"LINKUSDT":  "LINKUSDTM",
	"BATUSDT":   "BATUSDTM",
	"DENTUSDT":  "DENTUSDTM",
	"LUNAUSDT":  "LUNAUSDTM",
	"ETCUSDT":   "ETCUSDTM",
	"CHZUSDT":   "CHZUSDTM",
	"CRVUSDT":   "CRVUSDTM",
	"DASHUSDT":  "DASHUSDTM",
	"SNXUSDT":   "SNXUSDTM",
	"GRTUSDT":   "GRTUSDTM",
	"BTTUSDT":   "BTTUSDTM",
	"SUSHIUSDT": "SUSHIUSDTM",
	"ENJUSDT":   "ENJUSDTM",
	"XTZUSDT":   "XTZUSDTM",
	"DGBUSDT":   "DGBUSDTM",
	"SOLUSDT":   "SOLUSDTM",
	"BNBUSDT":   "BNBUSDTM",
}

func main() {
	ctx := context.Background()
	iw, err := common.NewInfluxWriter(
		ctx,
		"http://localhost:8086",
		"",
		"",
		"hft",
		500,
	)
	if err != nil {
		panic(err)
	}
	defer iw.Stop()

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
	quantilePath := "/Users/chenjilin/Projects/hft-micro/applications/usd-q-mt/configs/kc-quantiles"

	for ySymbol, xSymbol := range symbolsMap {
		if _, err := os.Stat(path.Join(quantilePath, xSymbol+"-"+ySymbol+"-long-td.json")); err == nil {
			logger.Debugf("Exists %s %s %v", ySymbol, xSymbol, err)
			continue
		} else if !os.IsNotExist(err) {
			logger.Debugf("Error %s %s %v", ySymbol, xSymbol, err)
			continue
		}
		counter := 0
		longTimedTDigest := stream_stats.NewTimedTDigest(quantileLookback, quantileSubInterval)
		shortTimedTDigest := stream_stats.NewTimedTDigest(quantileLookback, quantileSubInterval)

		shortLastEnter := 0.0
		longLastEnter := 0.0

		xDepth := &kucoin_usdtfuture.Depth5{}
		yDepth := &binance_usdtfuture.Depth5{}
		xWalkedDepth := &common.WalkedDepthBBMAA{}
		yWalkedDepth := &common.WalkedDepthBBMAA{}
		for _, dateStr := range strings.Split(dateStrs, ",") {
			logger.Debugf("/Users/chenjilin/MarketData/bnuf-kcuf-depth5/%s/%s-%s,%s.depth5.jl.gz", dateStr, dateStr, ySymbol, xSymbol)
			file, err := os.Open(
				fmt.Sprintf("/Users/chenjilin/MarketData/bnuf-kcuf-depth5/%s/%s-%s,%s.depth5.jl.gz", dateStr, dateStr, ySymbol, xSymbol),
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
				if msg[0] == 'K' {
					err = kucoin_usdtfuture.ParseDepth5(msg[1:], xDepth)
					if err != nil {
						//logger.Debugf("%v", err)
						continue
					}
					err = common.WalkDepthBBMAA(xDepth, xMultiplier, depthTakerImpact, xWalkedDepth)
					if err != nil {
						logger.Debugf("%v", err)
						continue
					}
				} else if msg[0] == 'B' {
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
						fields["longQuantileBot"] = longQuantileBot
						fields["shortQuantileTop80"] = shortTimedTDigest.Quantile(0.8)
						fields["shortQuantileTop50"] = shortTimedTDigest.Quantile(0.5)
						fields["longQuantileBot20"] = longTimedTDigest.Quantile(0.2)
						fields["longQuantileBot50"] = longTimedTDigest.Quantile(0.5)
						fields["shortLastEnter"] = shortLastEnter
						fields["longLastEnter"] = longLastEnter
						pt, err := client.NewPoint(
							"usd-q-mt-kc-bn",
							map[string]string{
								"xSymbol": xSymbol,
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
		data, err := json.Marshal(longTimedTDigest)
		if err != nil {
			logger.Debugf("%v", err)
			continue
		}
		file, err := os.OpenFile(path.Join(quantilePath, xSymbol+"-"+ySymbol+"-long-td.json"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
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
		data, err = json.Marshal(shortTimedTDigest)
		if err != nil {
			logger.Debugf("%v", err)
			continue
		}
		file, err = os.OpenFile(path.Join(quantilePath, xSymbol+"-"+ySymbol+"-short-td.json"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0775)
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
