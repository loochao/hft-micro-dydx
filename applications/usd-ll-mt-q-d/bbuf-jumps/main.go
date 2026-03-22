package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
	bybit_usdtfuture "github.com/geometrybase/hft-micro/bybit-usdtfuture"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/influx/client"
	"github.com/geometrybase/hft-micro/logger"
	stream_stats "github.com/geometrybase/hft-micro/stream-stats"
	"github.com/geometrybase/hft-micro/tdigest"
	"math"
	"os"
	"path"
	"sort"
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
	symbols := make([]string, 0)
	for symbol := range bybit_usdtfuture.TickSizes {
		if _, ok := binance_usdtfuture.TickSizes[symbol]; ok {
			symbols = append(symbols, symbol)
		}
	}
	sort.Strings(symbols)
	symbols = symbols[:1]
	startTime, err := time.Parse("20060102", "20210723")
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

	quantileLookback := time.Hour * 72
	quantileSubInterval := time.Hour
	quantilePath := "/home/clu/Projects/hft-micro/applications/usd-tk-tt-q/configs/bbuf-bnuf-quantiles"

	sizeTDs := make(map[string]*tdigest.TDigest)

	for _, xSymbol := range symbols {
		ySymbol := xSymbol
		counter := 0

		timedTDigest := stream_stats.NewTimedTDigest(quantileLookback, quantileSubInterval)

		upJumps, _ := tdigest.New()
		downJumps, _ := tdigest.New()

		xDepth := &bybit_usdtfuture.OrderBook{}
		sizeTD, _ := tdigest.New()

		var xTD common.Ticker
		var lastTime time.Time
		var lastBidPrice, lastAskPrice *float64
		var isXDepthReady = false
		var symbolLen = len(xSymbol)
		for _, dateStr := range strings.Split(dateStrs, ",") {
			logger.Debugf("%s %s %s", xSymbol, dateStr, fmt.Sprintf("/home/clu/MarketData/bbuf-bnuf-depth-and-ticker/%s/%s-%s,%s.gz", dateStr, dateStr, xSymbol, ySymbol))
			file, err := os.Open(
				fmt.Sprintf("/home/clu/MarketData/bbuf-bnuf-depth-and-ticker/%s/%s-%s,%s.gz", dateStr, dateStr, xSymbol, ySymbol),
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
				if msg[0] == 'X' && msg[1] == 'D' {
					if xTD != nil && isXDepthReady{
						if lastAskPrice == nil {
							lastAskPrice = new(float64)
						}
						if lastBidPrice == nil {
							lastBidPrice = new(float64)
						}
						*lastAskPrice = xTD.GetAskPrice()
						*lastBidPrice = xTD.GetBidPrice()
						lastTime = xTD.GetEventTime()
					}
					err = bybit_usdtfuture.UpdateOrderBook(msg[21:], xDepth)
					if err != nil {
						logger.Debugf("bybit_usdtfuture.UpdateOrderBook error %v", err)
						continue
					}
					if msg[56+symbolLen] == 's' {
						isXDepthReady = true
						//logger.Debugf("true %v %v %d %d ", isXDepthReady,xDepth.EventTime, len(xDepth.Bids), len(xDepth.Asks))
					}
					if !xDepth.IsValidate() {
						isXDepthReady = false
					}
					xTD = xDepth
					//if msg[56+symbolLen] == 's' {
					//	logger.Debugf("new %v %v %v %v %v %v", isXDepthReady,xTD.GetEventTime(), xDepth.IsValidate(), lastTime, lastAskPrice, lastBidPrice)
					//}
				} else {
					continue
				}

				if lastAskPrice != nil && lastBidPrice != nil && isXDepthReady {
					//logger.Debugf("%v", xTD.GetEventTime().Sub(lastTime))
					//if xTD.GetEventTime().Sub(lastTime) != 0 && counter < 10000{
					//	logger.Debugf("%v %v %v", xTD.GetEventTime().Sub(lastTime), xTD.GetEventTime(), lastTime)
					//}
					if xTD.GetEventTime().Sub(lastTime) < time.Millisecond*100 {
						if xTD.GetAskPrice() > *lastAskPrice {
							_ = upJumps.Add(xTD.GetAskPrice() - *lastAskPrice)
						}
						if xTD.GetBidPrice() < *lastBidPrice {
							_ = downJumps.Add(xTD.GetBidPrice() - *lastBidPrice)
						}
					}
					if counter%1000 == 0 {
						fields := make(map[string]interface{})
						fields["td"] = xTD.GetEventTime().Sub(lastTime).Seconds()
						fields["up50"] = upJumps.Quantile(0.5)
						fields["up80"] = upJumps.Quantile(0.8)
						fields["up95"] = upJumps.Quantile(0.95)
						fields["up995"] = upJumps.Quantile(0.995)
						fields["up9995"] = upJumps.Quantile(0.9995)
						fields["down50"] = downJumps.Quantile(0.5)
						fields["down20"] = downJumps.Quantile(0.2)
						fields["down05"] = downJumps.Quantile(0.05)
						fields["down005"] = downJumps.Quantile(0.005)
						fields["down0005"] = downJumps.Quantile(0.0005)
						pt, err := client.NewPoint(
							"bbuf-jumps",
							map[string]string{
								"xSymbol": xSymbol,
							},
							fields,
							xTD.GetEventTime(),
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
	fmt.Printf("\n\nxyPairs:\n")
	for _, xSymbol := range symbols {
		fmt.Printf("  %s: %s\n", xSymbol, xSymbol)
	}
	qSum := 0.0
	fmt.Printf("\n\nmaxOrderValues:\n")
	for _, xSymbol := range symbols {
		td := sizeTDs[xSymbol]
		qSum += td.Quantile(0.8)
		fmt.Printf("  %s: %.0f\n", xSymbol, td.Quantile(0.8))
	}
	qMean := qSum / float64(len(sizeTDs))
	fmt.Printf("\ntargetWeights:\n")
	for _, xSymbol := range symbols {
		td := sizeTDs[xSymbol]
		weight := td.Quantile(0.8) / qMean
		weight = math.Sqrt(weight)
		if weight > 1.0 {
			weight = 1.0
		}
		fmt.Printf("  %s: %.5f\n", xSymbol, weight)
	}
	fmt.Printf("\n\n")
}
