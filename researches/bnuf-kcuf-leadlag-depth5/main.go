package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/influx/client"
	kucoin_usdtfuture "github.com/geometrybase/hft-micro/kucoin-usdtfuture"
	"github.com/geometrybase/hft-micro/logger"
	"os"
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
		500,
	)
	if err != nil {
		panic(err)
	}
	defer iw.Stop()

	pairs := map[string]string{
		"XBTUSDTM":   "BTCUSDT",
		"UNIUSDTM":   "UNIUSDT",
		"DGBUSDTM":   "DGBUSDT",
		"IOSTUSDTM":  "IOSTUSDT",
		"RVNUSDTM":   "RVNUSDT",
		"THETAUSDTM": "THETAUSDT",
		"WAVESUSDTM": "WAVESUSDT",
		"DENTUSDTM":  "DENTUSDT",
		"DOTUSDTM":   "DOTUSDT",
		"XMRUSDTM":   "XMRUSDT",
		"FILUSDTM":   "FILUSDT",
		"ICPUSDTM":   "ICPUSDT",
		"MANAUSDTM":  "MANAUSDT",
		"MATICUSDTM": "MATICUSDT",
		"ALGOUSDTM":  "ALGOUSDT",
		"KSMUSDTM":   "KSMUSDT",
		"LUNAUSDTM":  "LUNAUSDT",
		"DASHUSDTM":  "DASHUSDT",
		"LTCUSDTM":   "LTCUSDT",
		"CHZUSDTM":   "CHZUSDT",
		"MKRUSDTM":   "MKRUSDT",
		"ADAUSDTM":   "ADAUSDT",
		"BCHUSDTM":   "BCHUSDT",
		"COMPUSDTM":  "COMPUSDT",
		"FTMUSDTM":   "FTMUSDT",
		"NEOUSDTM":   "NEOUSDT",
		"SXPUSDTM":   "SXPUSDT",
		"XRPUSDTM":   "XRPUSDT",
		"BNBUSDTM":   "BNBUSDT",
		"ETHUSDTM":   "ETHUSDT",
		"LINKUSDTM":  "LINKUSDT",
		"GRTUSDTM":   "GRTUSDT",
		"YFIUSDTM":   "YFIUSDT",
		"AAVEUSDTM":  "AAVEUSDT",
		"AVAXUSDTM":  "AVAXUSDT",
		"ETCUSDTM":   "ETCUSDT",
		"QTUMUSDTM":  "QTUMUSDT",
		"XLMUSDTM":   "XLMUSDT",
		"ZECUSDTM":   "ZECUSDT",
		"BTTUSDTM":   "BTTUSDT",
		"ENJUSDTM":   "ENJUSDT",
		"ONTUSDTM":   "ONTUSDT",
		"SUSHIUSDTM": "SUSHIUSDT",
		"XEMUSDTM":   "XEMUSDT",
		"DOGEUSDTM":  "DOGEUSDT",
		"OCEANUSDTM": "OCEANUSDT",
		"BATUSDTM":   "BATUSDT",
		"CRVUSDTM":   "CRVUSDT",
		"EOSUSDTM":   "EOSUSDT",
		"SNXUSDTM":   "SNXUSDT",
		"ATOMUSDTM":  "ATOMUSDT",
		"BANDUSDTM":  "BANDUSDT",
		"XTZUSDTM":   "XTZUSDT",
		"1INCHUSDTM": "1INCHUSDT",
		"TRXUSDTM":   "TRXUSDT",
		"SOLUSDTM":   "SOLUSDT",
		"VETUSDTM":   "VETUSDT",
	}
	symbols := make([]string, 0)
	for bSymbol := range pairs {
		symbols = append(symbols, bSymbol)
	}
	//sort.Strings(symbols)
	logger.Debugf("%d", len(symbols))
	//symbols = symbols[:1]

	startTime, err := time.Parse("20060102", "20210701")
	if err != nil {
		logger.Fatal(err)
	}
	endTime, err := time.Parse("20060102", "20210706")
	if err != nil {
		logger.Fatal(err)
	}
	dateStrs := ""
	for i := startTime; i.Sub(endTime) <= 0; i = i.Add(time.Hour * 24) {
		dateStrs += i.Format("20060102,")
	}
	dateStrs = dateStrs[:len(dateStrs)-1]
	for _, kSymbol := range symbols {
		bSymbol := pairs[kSymbol]

		positionSize := 0.0
		positionPrice := 0.0
		netWorth := 1.0
		enterSilentTime := time.Time{}
		enterSilent := time.Minute
		enterValue := 0.1
		commission := -0.0004
		kcDepth := &kucoin_usdtfuture.Depth5{}
		bnDepth := &binance_usdtfuture.Depth5{}
		longSpreadMean := common.NewTimedMean(time.Second * 3)
		shortSpreadMean := common.NewTimedMean(time.Second * 3)
		var msg []byte
		for _, dateStr := range strings.Split(dateStrs, ",") {
			logger.Debugf("%s %s", kSymbol, dateStr)
			file, err := os.Open(
				fmt.Sprintf("/home/clu/MarketData/bnuf-kcuf-depth5/%s/%s-%s,%s.depth5.jl.gz", dateStr, dateStr, bSymbol, kSymbol),
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
			counter := 0
			for scanner.Scan() {
				msg = scanner.Bytes()
				if msg[0] == 'B' {
					err = binance_usdtfuture.ParseDepth5(msg[1:], bnDepth)
					if err != nil {
						logger.Debugf("binance_usdtfuture.ParseDepth5 error %v", err)
						continue
					}
				} else if msg[0] == 'K' {
					err = kucoin_usdtfuture.ParseDepth5(msg[1:], kcDepth)
					if err != nil {
						logger.Debugf("kucoin_usdtfuture.ParseDepth5 error %v", err)
						continue
					}
				} else {
					continue
				}
				if kcDepth.Symbol == "" || bnDepth.Symbol == "" {
					continue
				}

				if bnDepth.EventTime.Sub(bnDepth.EventTime.Truncate(time.Hour*4)) < time.Minute {
					continue
				}
				if bnDepth.EventTime.Truncate(time.Hour*4).Add(time.Hour*4).Sub(bnDepth.EventTime) < time.Minute {
					continue
				}

				if kcDepth.EventTime.Sub(bnDepth.EventTime) < time.Second &&
					kcDepth.EventTime.Sub(bnDepth.EventTime) > -time.Second {

					longSpread := (bnDepth.Asks[0][0] - kcDepth.Asks[0][0]) / kcDepth.Asks[0][0]
					shortSpread := (bnDepth.Bids[0][0] - kcDepth.Bids[0][0]) / kcDepth.Bids[0][0]
					longSpreadMean.Insert(bnDepth.EventTime, longSpread)
					shortSpreadMean.Insert(bnDepth.EventTime, shortSpread)

					if bnDepth.EventTime.Sub(enterSilentTime) > 0 {
						if shortSpreadMean.Mean() > 0.001 && shortSpread >= shortSpreadMean.Mean() {
							enterSilentTime = bnDepth.EventTime.Add(enterSilent)
							size := enterValue / kcDepth.Asks[0][0]
							if positionSize >= 0 {
								if positionSize == 0 || positionPrice < kcDepth.Asks[0][0] {
									positionPrice = (positionSize*positionPrice + enterValue) / (positionSize + size)
									netWorth += commission * enterValue
									positionSize += size
								}
							} else {
								//先平仓
								netWorth += positionSize * (kcDepth.Asks[0][0] - positionPrice)
								netWorth += -positionSize * kcDepth.Asks[0][0] * commission
								//再加仓
								netWorth += commission * enterValue
								positionPrice = kcDepth.Asks[0][0]
								positionSize = size
							}
						} else if longSpreadMean.Mean() < -0.001 && longSpread <= longSpreadMean.Mean() {
							enterSilentTime = bnDepth.EventTime.Add(enterSilent)
							size := -enterValue / kcDepth.Bids[0][0]
							if positionSize <= 0 {
								if positionSize == 0 || positionPrice > kcDepth.Bids[0][0] {
									positionPrice = (positionSize*positionPrice - enterValue) / (positionSize + size)
									netWorth += commission * enterValue
									positionSize += size
								}
							} else {
								//先平仓
								netWorth += positionSize * (kcDepth.Bids[0][0] - positionPrice)
								netWorth += positionSize * kcDepth.Bids[0][0] * commission
								//再加仓
								netWorth += commission * enterValue
								positionPrice = kcDepth.Bids[0][0]
								positionSize = size
							}
						}
					}

					counter++
					if counter%100 == 0 {
						//logger.Debugf("%v", currentLagBookTicker)
						//logger.Debugf("%v", lastLagBookTicker)
						//return
						fields := make(map[string]interface{})
						fields["bidPrice"] = kcDepth.Bids[0][0]
						fields["askPrice"] = kcDepth.Asks[0][0]
						fields["shortSpread"] = shortSpread
						fields["shortSpreadMean"] = shortSpreadMean.Mean()
						fields["longSpread"] = longSpread
						fields["longSpreadMean"] = longSpreadMean.Mean()
						if positionPrice > 0 {
							fields["positionSize"] = positionSize
							fields["positionPrice"] = positionPrice
						}
						if positionSize > 0 {
							fields["netWorth"] = netWorth + positionSize*(kcDepth.Bids[0][0]-positionPrice)
						} else if positionSize < 0 {
							fields["netWorth"] = netWorth + positionSize*(kcDepth.Asks[0][0]-positionPrice)
						} else {
							fields["netWorth"] = netWorth
						}
						pt, err := client.NewPoint(
							"bnuf-kcuf-leadlag-depth5",
							map[string]string{
								"kSymbol": kSymbol,
							},
							fields,
							bnDepth.EventTime,
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
	}
}


