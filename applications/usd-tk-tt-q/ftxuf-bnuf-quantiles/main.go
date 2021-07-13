package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
	"github.com/geometrybase/hft-micro/common"
	ftx_usdfuture "github.com/geometrybase/hft-micro/ftx-usdfuture"
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
	symbols := make([]string, 0)
	for symbol := range ftx_usdfuture.PriceIncrements {
		if _, ok := binance_usdtfuture.TickSizes[strings.Replace(symbol, "-PERP", "USDT", -1)]; ok {
			symbols = append(symbols, symbol)
		}
	}
	//logger.Debugf("%s", symbols)
	//return
	startTime, err := time.Parse("20060102", "20210712")
	if err != nil {
		logger.Fatal(err)
	}
	endTime, err := time.Parse("20060102", "20210712")
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
	quantilePath := "/Users/chenjilin/Projects/hft-micro/applications/usd-tk-tt-q/configs/ftxuf-bnuf-quantiles"

	for _, xSymbol := range symbols {
		ySymbol := strings.Replace(xSymbol, "-PERP", "USDT", -1)
		counter := 0
		timedTDigest := stream_stats.NewTimedTDigest(quantileLookback, quantileSubInterval)

		shortLastEnter := 0.0
		longLastEnter := 0.0

		xTicker := &ftx_usdfuture.Ticker{}
		yDepth := &binance_usdtfuture.Depth5{}
		yTicker := &binance_usdtfuture.BookTicker{}

		var xTD, yTD common.Ticker
		for _, dateStr := range strings.Split(dateStrs, ",") {
			logger.Debugf("%s %s %s", xSymbol, dateStr, fmt.Sprintf("/Users/chenjilin/MarketData/ftxuf-bnuf-depth5-and-ticker/%s/%s-%s,%s.jl.gz", dateStr, dateStr, xSymbol, ySymbol))
			file, err := os.Open(
				fmt.Sprintf("/Users/chenjilin/MarketData/ftxuf-bnuf-depth5-and-ticker/%s/%s-%s,%s.jl.gz", dateStr, dateStr, xSymbol, ySymbol),
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
				if msg[0] == 'F' && msg[1] == 'T' {
					err = ftx_usdfuture.ParseTicker(msg[21:], xTicker)
					if err != nil {
						logger.Debugf("%v", err)
						continue
					}
					xTD = xTicker
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
					_ = timedTDigest.Insert(yDepth.EventTime, (shortLastEnter+longLastEnter)*0.5)
					if counter%1000 == 0 {
						fields := make(map[string]interface{})
						fields["enterMiddle"] = timedTDigest.Quantile(0.5)
						fields["shortLastEnter"] = shortLastEnter
						fields["longLastEnter"] = longLastEnter
						pt, err := client.NewPoint(
							"ftxuf-bnuf-depth5-and-ticker",
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
