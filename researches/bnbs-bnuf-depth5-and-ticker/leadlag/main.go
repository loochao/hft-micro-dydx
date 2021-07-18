package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	binance_busdspot "github.com/geometrybase/hft-micro/binance-busdspot"
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/influx/client"
	"github.com/geometrybase/hft-micro/logger"
	stream_stats "github.com/geometrybase/hft-micro/stream-stats"
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
		//"BTCBUSD": "BTCUSDT",
		"ETHBUSD": "ETHUSDT",
		//"ICPBUSD": "ICPUSDT",
	}
	symbols := make([]string, 0)
	for bSymbol := range pairs {
		symbols = append(symbols, bSymbol)
	}
	sort.Strings(symbols)
	logger.Debugf("%d", len(symbols))
	symbols = symbols[:]

	startTime, err := time.Parse("20060102", "20210712")
	if err != nil {
		logger.Fatal(err)
	}
	endTime, err := time.Parse("20060102", "20210716")
	if err != nil {
		logger.Fatal(err)
	}
	dateStrs := ""
	dayCount := 0.0
	for i := startTime; i.Sub(endTime) <= 0; i = i.Add(time.Hour * 24) {
		dateStrs += i.Format("20060102,")
		dayCount += 1.0
	}
	dateStrs = dateStrs[:len(dateStrs)-1]

	for _, xSymbol := range symbols {
		ySymbol := pairs[xSymbol]


		xPositionSize := 0.0
		xPositionPrice := 0.0
		yPositionSize := 0.0
		yPositionPrice := 0.0

		netWorth := 1.0
		enterSilentTime := time.Time{}
		enterSilent := time.Minute
		enterValue := 0.25
		commission := -0.000
		//longEnter := -0.0005
		//shortEnter := 0.0005
		breakTop := 0.95
		breakBot := 0.05
		hedgeDelay := time.Second*1

		longSpreadMean := common.NewTimedMean(time.Second * 3)
		shortSpreadMean := common.NewTimedMean(time.Second * 3)
		timedSpreadTD := stream_stats.NewTimedTDigest(time.Hour*24, time.Hour)
		xEnterTime := time.Time{}

		xDepth := &binance_busdspot.Depth5{}
		yDepth := &binance_usdtfuture.Depth5{}

		xBookTicker := &binance_busdspot.BookTicker{}
		yBookTicker := &binance_usdtfuture.BookTicker{}

		var xTicker common.Ticker
		var yTicker common.Ticker

		var shortEnter, longEnter float64
		var msg []byte
		counter := 0
		for _, dateStr := range strings.Split(dateStrs, ",") {
			//logger.Debugf("%s %s", xSymbol, dateStr)
			file, err := os.Open(
				fmt.Sprintf("/Users/chenjilin/MarketData/bnbs-bnuf-depth5-and-ticker/%s/%s-%s,%s.jl.gz", dateStr, dateStr, xSymbol, ySymbol),
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
			lastCollectTime := time.Time{}
			for scanner.Scan() {
				msg = scanner.Bytes()
				if msg[0] == 'S' && msg[1] == 'D' {
					err = binance_busdspot.ParseDepth5(msg[21:], xDepth)
					if err != nil {
						logger.Debugf("binance_busdspot.ParseDepth5 error %v", err)
						continue
					}
					t, _ := common.ParseInt(msg[2:21])
					xDepth.ParseTime = time.Unix(0, t)
					//if counter == 0 {
					//	logger.Debugf("%v, %s", xDepth.ParseTime, msg[2:21])
					//}
					xTicker = xDepth
				} else if msg[0] == 'S' && msg[1] == 'T' {
					err = binance_busdspot.ParseBookTicker(msg[21:], xBookTicker)
					if err != nil {
						logger.Debugf("binance_busdspot.ParseBookTicker error %v", err)
						continue
					}
					t, _ := common.ParseInt(msg[2:21])
					xBookTicker.ParseTime = time.Unix(0, t)
					//if counter == 0 {
					//	logger.Debugf("%v", xBookTicker.ParseTime)
					//}
					xTicker = xBookTicker
				} else if msg[0] == 'F' && msg[1] == 'D' {
					err = binance_usdtfuture.ParseDepth5(msg[21:], yDepth)
					if err != nil {
						logger.Debugf("binance_usdtfuture.ParseDepth5 error %v", err)
						continue
					}
					yTicker = yDepth
				} else if msg[0] == 'F' && msg[1] == 'T' {
					err = binance_usdtfuture.ParseBookTicker(msg[21:], yBookTicker)
					if err != nil {
						logger.Debugf("binance_usdtfuture.ParseBookTicker error %v", err)
						continue
					}
					yTicker = yBookTicker
				} else {
					continue
				}

				if xTicker == nil || yTicker == nil {
					continue
				}

				if yTicker.GetTime().Sub(yTicker.GetTime().Truncate(time.Hour*4)) < time.Minute {
					continue
				}
				if xTicker.GetTime().Sub(xTicker.GetTime().Truncate(time.Hour*4)) < time.Minute {
					continue
				}

				counter ++

				if xTicker.GetTime().Sub(yTicker.GetTime()) < time.Millisecond &&
					xTicker.GetTime().Sub(yTicker.GetTime()) > -time.Millisecond &&
					yTicker.GetTime().Sub(lastCollectTime) > 0 {
					lastCollectTime = yTicker.GetTime().Add(time.Millisecond)

					longSpread := (yTicker.GetAskPrice() - xTicker.GetBidPrice()) / xTicker.GetBidPrice()
					shortSpread := (yTicker.GetBidPrice() - xTicker.GetAskPrice()) / xTicker.GetAskPrice()
					longSpreadMean.Insert(yTicker.GetTime(), longSpread)
					shortSpreadMean.Insert(yTicker.GetTime(), shortSpread)
					_ = timedSpreadTD.Insert(yTicker.GetTime(), (longSpread+shortSpread)*0.5)

					shortEnter = timedSpreadTD.Quantile(breakTop)
					longEnter = timedSpreadTD.Quantile(breakBot)

					if yTicker.GetTime().Sub(enterSilentTime) > 0 && counter > 100000{
						if shortSpreadMean.Mean() > shortEnter && shortSpread >= shortSpreadMean.Mean() {
							enterSilentTime = yTicker.GetTime().Add(enterSilent+hedgeDelay)
							size := enterValue / yTicker.GetAskPrice()
							if xPositionSize >= 0 {
								if xPositionSize == 0 || xPositionPrice < xTicker.GetAskPrice() {
									xPositionPrice = (xPositionSize*xPositionPrice + enterValue) / (xPositionSize + size)
									netWorth += commission * enterValue
									xPositionSize += size
									xEnterTime = yTicker.GetTime()
								}
							} else {
								//先平仓
								netWorth += xPositionSize * (xTicker.GetAskPrice() - xPositionPrice)
								netWorth += -xPositionSize * xTicker.GetAskPrice() * commission
								//再加仓
								netWorth += commission * enterValue
								xPositionPrice = xTicker.GetAskPrice()
								xPositionSize = size
								xEnterTime = yTicker.GetTime()
							}
						} else if longSpreadMean.Mean() < longEnter && longSpread <= longSpreadMean.Mean() {
							enterSilentTime = yTicker.GetTime().Add(enterSilent+hedgeDelay)
							size := -enterValue / xTicker.GetBidPrice()
							if xPositionSize <= 0 {
								if xPositionSize == 0 || xPositionPrice > xTicker.GetBidPrice() {
									xPositionPrice = (xPositionSize*xPositionPrice - enterValue) / (xPositionSize + size)
									netWorth += commission * enterValue
									xPositionSize += size
									xEnterTime = yTicker.GetTime()
								}
							} else {
								//先平仓
								netWorth += xPositionSize * (xTicker.GetBidPrice() - xPositionPrice)
								netWorth += xPositionSize * xTicker.GetBidPrice() * commission
								//再加仓
								netWorth += commission * enterValue
								xPositionPrice = xTicker.GetBidPrice()
								xPositionSize = size
								xEnterTime = yTicker.GetTime()
							}
						}
					}

					ySize := -xPositionSize - yPositionSize
					if ySize != 0 && yTicker.GetTime().Sub(xEnterTime) > hedgeDelay {
						if ySize*yPositionSize > 0 {
							//同向加仓
							if ySize > 0 {
								yPositionPrice = (yPositionSize*yPositionPrice + ySize*yTicker.GetAskPrice()) / (yPositionSize + ySize)
								netWorth += ySize * yTicker.GetAskPrice() * commission
							} else {
								yPositionPrice = (yPositionSize*yPositionPrice + ySize*yTicker.GetBidPrice()) / (yPositionSize + ySize)
								netWorth += -ySize * yTicker.GetBidPrice() * commission
							}
						} else if math.Abs(ySize) >= math.Abs(yPositionSize) {
							//换仓
							if yPositionSize > 0 {
								netWorth += math.Abs(ySize) * yTicker.GetBidPrice() * commission
								netWorth += yPositionSize * (yTicker.GetBidPrice() - yPositionPrice)
								yPositionPrice = yTicker.GetBidPrice()
							} else {
								netWorth += math.Abs(ySize) * yTicker.GetAskPrice() * commission
								netWorth += yPositionSize * (yTicker.GetAskPrice() - yPositionPrice)
								yPositionPrice = yTicker.GetAskPrice()
							}
						} else {
							//减仓
							if ySize > 0 {
								netWorth += ySize * yTicker.GetBidPrice() * commission
								netWorth += -ySize * (yTicker.GetBidPrice() - yPositionPrice)
							} else {
								netWorth += -ySize * yTicker.GetAskPrice() * commission
								netWorth += -ySize * (yTicker.GetAskPrice() - yPositionPrice)
							}
						}
						yPositionSize += ySize
					}

					counter++
					if counter%100 == 0 {
						fields := make(map[string]interface{})
						fields["bidPrice"] = xTicker.GetBidPrice()
						fields["askPrice"] = xTicker.GetAskPrice()
						fields["shortEnter"] = shortEnter
						fields["longEnter"] = longEnter
						fields["shortSpread"] = shortSpread
						fields["shortSpreadMean"] = shortSpreadMean.Mean()
						fields["longSpread"] = longSpread
						fields["longSpreadMean"] = longSpreadMean.Mean()
						if xPositionSize != 0 {
							fields["xPositionSize"] = xPositionSize
							fields["xPositionPrice"] = xPositionPrice
						}
						if yPositionSize != 0 {
							fields["yPositionSize"] = yPositionSize
							fields["yPositionPrice"] = yPositionPrice
						}
						kcUnPnl := 0.0
						bnUnPnl := 0.0
						if xPositionSize > 0 {
							kcUnPnl = xPositionSize * (xTicker.GetBidPrice() - xPositionPrice)
						} else if xPositionSize < 0 {
							kcUnPnl = xPositionSize * (xTicker.GetAskPrice() - xPositionPrice)
						}
						if yPositionSize > 0 {
							bnUnPnl = yPositionSize * (yTicker.GetBidPrice() - yPositionPrice)
						} else if yPositionSize < 0 {
							bnUnPnl = yPositionSize * (yTicker.GetAskPrice() - yPositionPrice)
						}
						fields["netWorth"] = netWorth + kcUnPnl + bnUnPnl
						pt, err := client.NewPoint(
							"bnbs-bnuf-depth5-and-ticker-leadlag",
							map[string]string{
								"xSymbol": xSymbol,
								"ySymbol": ySymbol,
								"delay":   fmt.Sprintf("%v", hedgeDelay),
							},
							fields,
							yTicker.GetTime(),
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
