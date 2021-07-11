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

	startTime, err := time.Parse("20060102", "20210708")
	if err != nil {
		logger.Fatal(err)
	}
	endTime, err := time.Parse("20060102", "20210710")
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
	quantilePath := "/Users/chenjilin/Projects/hft-micro/applications/usd-tk-tt-q/configs/kc-bn-quantiles"

	for ySymbol, xSymbol := range symbolsMap {
		if _, err := os.Stat(path.Join(quantilePath, xSymbol+"-"+ySymbol+"-long-td.json")); err == nil {
			logger.Debugf("Exists %s %s %v", ySymbol, xSymbol, err)
			continue
		} else if !os.IsNotExist(err) {
			logger.Debugf("Error %s %s %v", ySymbol, xSymbol, err)
			continue
		}
		counter := 0
		timedTDigest := stream_stats.NewTimedTDigest(quantileLookback, quantileSubInterval)

		shortLastEnter := 0.0
		longLastEnter := 0.0

		xDepth := &kucoin_usdtfuture.Depth5{}
		yDepth := &binance_usdtfuture.Depth5{}
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
				} else if msg[0] == 'B' {
					err = binance_usdtfuture.ParseDepth5(msg[1:], yDepth)
					if err != nil {
						logger.Debugf("%v", err)
						continue
					}
				} else {
					continue
				}

				if xDepth.Symbol != "" && yDepth.Symbol != "" {

					shortLastEnter = (yDepth.Bids[0][0] - xDepth.Asks[0][0]) / xDepth.Asks[0][0]
					longLastEnter = (yDepth.Asks[0][0] - xDepth.Bids[0][0]) / xDepth.Bids[0][0]
					if xDepth.EventTime.Sub(yDepth.EventTime) > 0 {
						_ = timedTDigest.Insert(xDepth.EventTime, (shortLastEnter+longLastEnter)*0.5)
					} else {
						_ = timedTDigest.Insert(yDepth.EventTime, (shortLastEnter+longLastEnter)*0.5)
					}
					if counter%1000 == 0 {
						fields := make(map[string]interface{})
						fields["enterMiddle"] = timedTDigest.Quantile(0.5)
						fields["shortLastEnter"] = shortLastEnter
						fields["longLastEnter"] = longLastEnter
						pt, err := client.NewPoint(
							"usd-tk-tt-q-kc-bn",
							map[string]string{
								"xSymbol": xSymbol,
							},
							fields,
							yDepth.EventTime,
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
		data, err := json.Marshal(timedTDigest)
		if err != nil {
			logger.Debugf("%v", err)
			continue
		}
		file, err := os.OpenFile(path.Join(quantilePath, xSymbol+"-"+ySymbol+".json"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
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
