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
	"github.com/geometrybase/hft-micro/tdigest"
	"math"
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
	sort.Strings(symbols)
	logger.Debugf("%d", len(symbols))
	symbols = symbols[:]

	startTime, err := time.Parse("20060102", "20210715")
	if err != nil {
		logger.Fatal(err)
	}
	endTime, err := time.Parse("20060102", "20210715")
	if err != nil {
		logger.Fatal(err)
	}
	dateStrs := ""
	for i := startTime; i.Sub(endTime) <= 0; i = i.Add(time.Hour * 24) {
		dateStrs += i.Format("20060102,")
	}
	dateStrs = dateStrs[:len(dateStrs)-1]
	for _, xSymbol := range symbols {
		ySymbol := pairs[xSymbol]

		kcPositionSize := 0.0
		kcPositionPrice := 0.0
		bnPositionSize := 0.0
		bnPositionPrice := 0.0

		netWorth := 1.0
		enterSilentTime := time.Time{}
		enterSilent := time.Minute
		enterValue := 0.1
		commission := -0.0004
		xDepth := &kucoin_usdtfuture.Depth5{}
		yDepth := &binance_usdtfuture.Depth5{}

		xTicker := &kucoin_usdtfuture.Ticker{}
		yTicker := &binance_usdtfuture.BookTicker{}

		var xT common.Ticker
		var yT common.Ticker

		longSpreadMean := common.NewTimedMean(time.Second * 3)
		shortSpreadMean := common.NewTimedMean(time.Second * 3)
		hedgeDelay := time.Second * 5
		kcEnterTime := time.Time{}

		longTD, _ := tdigest.New()
		shortTD, _ := tdigest.New()

		var msg []byte
		for _, dateStr := range strings.Split(dateStrs, ",") {
			//logger.Debugf("%s %s", xSymbol, dateStr)
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
			counter := 0
			for scanner.Scan() {
				msg = scanner.Bytes()
				if msg[0] == 'B' && msg[1] == 'D' {
					err = binance_usdtfuture.ParseDepth5(msg[21:], yDepth)
					if err != nil {
						logger.Debugf("binance_usdtfuture.ParseDepth5 error %v", err)
						continue
					}
					yT = yDepth
				} else if msg[0] == 'B' && msg[1] == 'T' {
					err = binance_usdtfuture.ParseBookTicker(msg[21:], yTicker)
					if err != nil {
						logger.Debugf("binance_usdtfuture.ParseBookTicker error %v", err)
						continue
					}
					yT = yTicker
				} else if msg[0] == 'K' && msg[1] == 'D' {
					err = kucoin_usdtfuture.ParseDepth5(msg[21:], xDepth)
					if err != nil {
						logger.Debugf("kucoin_usdtfuture.ParseDepth5 error %v", err)
						continue
					}
					xT = xDepth
				} else if msg[0] == 'K' && msg[1] == 'T' {
					err = kucoin_usdtfuture.ParseTicker(msg[21:], xTicker)
					if err != nil {
						logger.Debugf("kucoin_usdtfuture.ParseDepth5 error %v", err)
						continue
					}
					xT = xTicker
				} else {
					continue
				}

				if xT == nil || yT == nil {
					continue
				}

				if yT.GetEventTime().Sub(yT.GetEventTime().Truncate(time.Hour*4)) < time.Minute {
					continue
				}
				if xT.GetEventTime().Sub(xT.GetEventTime().Truncate(time.Hour*4)) < time.Minute {
					continue
				}

				if xT.GetEventTime().Sub(yT.GetEventTime()) < time.Second &&
					xT.GetEventTime().Sub(yT.GetEventTime()) > -time.Second {

					longSpread := (yT.GetAskPrice() - xT.GetBidPrice()) / xT.GetBidPrice()
					shortSpread := (yT.GetBidPrice() - xT.GetAskPrice()) / xT.GetAskPrice()
					longSpreadMean.Insert(yT.GetEventTime(), longSpread)
					shortSpreadMean.Insert(yT.GetEventTime(), shortSpread)
					_ = longTD.Add(longSpread)
					_ = shortTD.Add(shortSpread)

					if yT.GetEventTime().Sub(enterSilentTime) > 0 {
						if shortSpreadMean.Mean() > 0.001 && shortSpread >= shortSpreadMean.Mean() {
							enterSilentTime = yT.GetEventTime().Add(enterSilent)
							size := enterValue / xT.GetAskPrice()
							if kcPositionSize >= 0 {
								if kcPositionSize == 0 || kcPositionPrice < xT.GetAskPrice() {
									kcPositionPrice = (kcPositionSize*kcPositionPrice + enterValue) / (kcPositionSize + size)
									netWorth += commission * enterValue
									kcPositionSize += size
									kcEnterTime = yT.GetEventTime()
								}
							} else {
								//先平仓
								netWorth += kcPositionSize * (xT.GetAskPrice() - kcPositionPrice)
								netWorth += -kcPositionSize * xT.GetAskPrice() * commission
								//再加仓
								netWorth += commission * enterValue
								kcPositionPrice = xT.GetAskPrice()
								kcPositionSize = size
								kcEnterTime = yT.GetEventTime()
							}
						} else if longSpreadMean.Mean() < -0.001 && longSpread <= longSpreadMean.Mean() {
							enterSilentTime = yT.GetEventTime().Add(enterSilent)
							size := -enterValue / xT.GetBidPrice()
							if kcPositionSize <= 0 {
								if kcPositionSize == 0 || kcPositionPrice > xT.GetBidPrice() {
									kcPositionPrice = (kcPositionSize*kcPositionPrice - enterValue) / (kcPositionSize + size)
									netWorth += commission * enterValue
									kcPositionSize += size
									kcEnterTime = yT.GetEventTime()
								}
							} else {
								//先平仓
								netWorth += kcPositionSize * (xT.GetBidPrice() - kcPositionPrice)
								netWorth += kcPositionSize * xT.GetBidPrice() * commission
								//再加仓
								netWorth += commission * enterValue
								kcPositionPrice = xT.GetBidPrice()
								kcPositionSize = size
								kcEnterTime = yT.GetEventTime()
							}
						}
					}

					bnSize := -kcPositionSize - bnPositionSize
					if bnSize != 0 && yT.GetEventTime().Sub(kcEnterTime) > hedgeDelay {
						if bnSize*bnPositionSize > 0 {
							//同向加仓
							if bnSize > 0 {
								bnPositionPrice = (bnPositionSize*bnPositionPrice + bnSize*yT.GetAskPrice()) / (bnPositionSize + bnSize)
								netWorth += bnSize * yT.GetAskPrice() * commission
							} else {
								bnPositionPrice = (bnPositionSize*bnPositionPrice + bnSize*yT.GetBidPrice()) / (bnPositionSize + bnSize)
								netWorth += -bnSize * yT.GetBidPrice() * commission
							}
						} else if math.Abs(bnSize) >= math.Abs(bnPositionSize) {
							//换仓
							if bnPositionSize > 0 {
								netWorth += math.Abs(bnSize) * yT.GetBidPrice() * commission
								netWorth += bnPositionSize * (yT.GetBidPrice() - bnPositionPrice)
								bnPositionPrice = yT.GetBidPrice()
							} else {
								netWorth += math.Abs(bnSize) * yT.GetAskPrice() * commission
								netWorth += bnPositionSize * (yT.GetAskPrice() - bnPositionPrice)
								bnPositionPrice = yT.GetAskPrice()
							}
						} else {
							//减仓
							if bnSize > 0 {
								netWorth += bnSize * yT.GetBidPrice() * commission
								netWorth += -bnSize * (yT.GetBidPrice() - bnPositionPrice)
							} else {
								netWorth += -bnSize * yT.GetAskPrice() * commission
								netWorth += -bnSize * (yT.GetAskPrice() - bnPositionPrice)
							}
						}
						bnPositionSize += bnSize
					}

					counter++
					if counter%100 == 0 {
						fields := make(map[string]interface{})
						fields["bidPrice"] = xT.GetBidPrice()
						fields["askPrice"] = xT.GetAskPrice()
						fields["shortSpread"] = shortSpread
						fields["shortSpreadMean"] = shortSpreadMean.Mean()
						fields["longSpread"] = longSpread
						fields["longSpreadMean"] = longSpreadMean.Mean()
						if kcPositionSize != 0 {
							fields["kcPositionSize"] = kcPositionSize
							fields["kcPositionPrice"] = kcPositionPrice
						}
						if bnPositionSize != 0 {
							fields["bnPositionSize"] = bnPositionSize
							fields["bnPositionPrice"] = bnPositionPrice
						}
						kcUnPnl := 0.0
						bnUnPnl := 0.0
						if kcPositionSize > 0 {
							kcUnPnl = kcPositionSize * (xT.GetBidPrice() - kcPositionPrice)
						} else if kcPositionSize < 0 {
							kcUnPnl = kcPositionSize * (xT.GetAskPrice() - kcPositionPrice)
						}
						if bnPositionSize > 0 {
							bnUnPnl = bnPositionSize * (yT.GetBidPrice() - bnPositionPrice)
						} else if bnPositionSize < 0 {
							bnUnPnl = bnPositionSize * (yT.GetAskPrice() - bnPositionPrice)
						}
						fields["netWorth"] = netWorth + kcUnPnl + bnUnPnl
						pt, err := client.NewPoint(
							"kcuf-bnuf-leadlag-depth5-and-ticker",
							map[string]string{
								"xSymbol": xSymbol,
								"ySymbol": ySymbol,
								"delay":   fmt.Sprintf("%v", hedgeDelay),
							},
							fields,
							yT.GetEventTime(),
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

		fmt.Printf("\n\n%s Long Spread CDF:\n", xSymbol)
		fmt.Printf("  %.6f:%.6f\n", -0.000999, longTD.CDF(-0.000999))
		fmt.Printf("  %.6f:%.6f\n", -0.001111, longTD.CDF(-0.001111))
		fmt.Printf("  %.6f:%.6f\n", -0.001222, longTD.CDF(-0.001222))
		fmt.Printf("%s Short Spread CDF:\n", xSymbol)
		fmt.Printf("  %.6f:%.6f\n", 0.000999, 1.0 - shortTD.CDF(0.000999))
		fmt.Printf("  %.6f:%.6f\n", 0.001111, 1.0 - shortTD.CDF(0.001111))
		fmt.Printf("  %.6f:%.6f\n", 0.001222, 1.0 - shortTD.CDF(0.001222))
	}
}
