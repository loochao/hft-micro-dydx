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
	"github.com/geometrybase/hft-micro/tdigest"
	"os"
	"path"
	"strings"
	"time"
)

var symbolsMap = map[string]string{
	//"BTCUSDT":   "XBTUSDTM",
	//"IOSTUSDT":  "IOSTUSDTM",
	//"UNIUSDT":   "UNIUSDTM",
	"ICPUSDT": "ICPUSDTM",
	//"THETAUSDT": "THETAUSDTM",
	//"YFIUSDT":   "YFIUSDTM",
	//"OCEANUSDT": "OCEANUSDTM",
	//"XMRUSDT":   "XMRUSDTM",
	//"SXPUSDT":   "SXPUSDTM",
	//"BCHUSDT":   "BCHUSDTM",
	//"TRXUSDT":   "TRXUSDTM",
	//"XEMUSDT":   "XEMUSDTM",
	//"ETHUSDT":   "ETHUSDTM",
	//"MKRUSDT":   "MKRUSDTM",
	//"FTMUSDT":   "FTMUSDTM",
	//"ATOMUSDT":  "ATOMUSDTM",
	//"BANDUSDT":  "BANDUSDTM",
	//"DOTUSDT":   "DOTUSDTM",
	//"FILUSDT":   "FILUSDTM",
	//"AVAXUSDT":  "AVAXUSDTM",
	//"QTUMUSDT":  "QTUMUSDTM",
	//"COMPUSDT":  "COMPUSDTM",
	//"ZECUSDT":   "ZECUSDTM",
	//"ADAUSDT":   "ADAUSDTM",
	//"DOGEUSDT":  "DOGEUSDTM",
	//"XLMUSDT":   "XLMUSDTM",
	//"EOSUSDT":   "EOSUSDTM",
	//"LTCUSDT":   "LTCUSDTM",
	//"VETUSDT":   "VETUSDTM",
	//"ONTUSDT":   "ONTUSDTM",
	//"RVNUSDT":   "RVNUSDTM",
	//"MATICUSDT": "MATICUSDTM",
	//"1INCHUSDT": "1INCHUSDTM",
	//"XRPUSDT":   "XRPUSDTM",
	//"NEOUSDT":   "NEOUSDTM",
	//"ALGOUSDT":  "ALGOUSDTM",
	//"MANAUSDT":  "MANAUSDTM",
	//"WAVESUSDT": "WAVESUSDTM",
	//"KSMUSDT":   "KSMUSDTM",
	//"AAVEUSDT":  "AAVEUSDTM",
	//"LINKUSDT":  "LINKUSDTM",
	//"BATUSDT":   "BATUSDTM",
	//"DENTUSDT":  "DENTUSDTM",
	//"LUNAUSDT":  "LUNAUSDTM",
	//"ETCUSDT":   "ETCUSDTM",
	//"CHZUSDT":   "CHZUSDTM",
	//"CRVUSDT":   "CRVUSDTM",
	//"DASHUSDT":  "DASHUSDTM",
	//"SNXUSDT":   "SNXUSDTM",
	//"GRTUSDT":   "GRTUSDTM",
	//"BTTUSDT":   "BTTUSDTM",
	//"SUSHIUSDT": "SUSHIUSDTM",
	//"ENJUSDT":   "ENJUSDTM",
	//"XTZUSDT":   "XTZUSDTM",
	//"DGBUSDT":   "DGBUSDTM",
	//"SOLUSDT":   "SOLUSDTM",
	"BNBUSDT": "BNBUSDTM",
}

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
	startTime, err := time.Parse("20060102", "20210715")
	if err != nil {
		logger.Fatal(err)
	}
	endTime, err := time.Parse("20060102", "20210724")
	if err != nil {
		logger.Fatal(err)
	}
	dateStrs := ""
	for i := startTime; i.Sub(endTime) <= 0; i = i.Add(time.Hour * 24) {
		dateStrs += i.Format("20060102,")
	}
	dateStrs = dateStrs[:len(dateStrs)-1]

	quantileLookback := time.Hour
	quantileSubInterval := time.Minute*5
	quantilePath := "/home/clu/Projects/hft-micro/applications/usd-tk-tt-q/kcuf-bnuf-quantiles/spread-signal/configs"

	sizeTDs := make(map[string]*tdigest.TDigest)

	for ySymbol, xSymbol := range symbolsMap {
		counter := 0
		timedTDigest := stream_stats.NewTimedTDigest(quantileLookback, quantileSubInterval)

		timedX := common.NewTimedMean(time.Hour)
		timedY := common.NewTimedMean(time.Hour)

		shortLastEnter := 0.0
		longLastEnter := 0.0

		xTicker := &kucoin_usdtfuture.Ticker{}
		xDepth := &kucoin_usdtfuture.Depth5{}
		yDepth := &binance_usdtfuture.Depth5{}
		yTicker := &binance_usdtfuture.BookTicker{}

		sizeTD, _ := tdigest.New()

		var xTD, yTD common.Ticker
		for _, dateStr := range strings.Split(dateStrs, ",") {
			logger.Debugf("%s %s %s", xSymbol, dateStr, fmt.Sprintf("/home/clu/MarketData/kcuf-bnuf-depth5-and-ticker/%s/%s-%s,%s.jl.gz", dateStr, dateStr, xSymbol, ySymbol))
			file, err := os.Open(
				fmt.Sprintf("/home/clu/MarketData/kcuf-bnuf-depth5-and-ticker/%s/%s-%s,%s.jl.gz", dateStr, dateStr, xSymbol, ySymbol),
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
			var msg []byte
			for scanner.Scan() {
				counter++
				msg = scanner.Bytes()
				if msg[0] == 'K' && msg[1] == 'T' {
					err = kucoin_usdtfuture.ParseTicker(msg[21:], xTicker)
					if err != nil {
						logger.Debugf("%v", err)
						continue
					}
					xTD = xTicker
				} else if msg[0] == 'K' && msg[1] == 'D' {
					err = kucoin_usdtfuture.ParseDepth5(msg[21:], xDepth)
					if err != nil {
						logger.Debugf("%v", err)
						continue
					}
					xTD = xDepth
				} else if msg[0] == 'B' && msg[1] == 'D' {
					err = binance_usdtfuture.ParseDepth5(msg[21:], yDepth)
					if err != nil {
						logger.Debugf("%v", err)
						continue
					}
					yTD = yDepth
				} else if msg[0] == 'B' && msg[1] == 'T' {
					err = binance_usdtfuture.ParseBookTicker(msg[21:], yTicker)
					if err != nil {
						logger.Debugf("%v", err)
						continue
					}
					yTD = yTicker
				} else {
					continue
				}

				if yTD != nil && xTD != nil {
					shortLastEnter = (yTD.GetBidPrice() - xTD.GetAskPrice()) / xTD.GetAskPrice()
					longLastEnter = (yTD.GetAskPrice() - xTD.GetBidPrice()) / xTD.GetBidPrice()
					_ = timedY.Insert(yDepth.EventTime, (yDepth.GetAskPrice()+yDepth.GetBidPrice())*0.5)
					_ = timedX.Insert(yDepth.EventTime, (xDepth.GetAskPrice()+xDepth.GetBidPrice())*0.5)
					_ = timedTDigest.Insert(yDepth.EventTime, (shortLastEnter+longLastEnter)*0.5)
					if counter%1000 == 0 {
						fields := make(map[string]interface{})
						fields["spread"] = timedTDigest.Quantile(0.5)
						fields["spread995"] = timedTDigest.Quantile(0.995)
						fields["spread95"] = timedTDigest.Quantile(0.95)
						fields["spread50"] = timedTDigest.Quantile(0.5)
						fields["spread05"] = timedTDigest.Quantile(0.05)
						fields["spread005"] = timedTDigest.Quantile(0.005)
						fields["fastY"] = timedY.Mean()
						fields["fastX"] = timedX.Mean()
						fields["xBidPrice"] = xTD.GetBidPrice()
						fields["xAskPrice"] = xTD.GetAskPrice()
						pt, err := client.NewPoint(
							"kcbn-signal",
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
		sizeTDs[xSymbol] = sizeTD
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
