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

	symbols := make([]string, 0)
	for symbol := range binance_usdtspot.TickSizes {
		if _, ok := binance_usdtfuture.TickSizes[symbol]; ok {
			symbols = append(symbols, symbol)
		}
	}
	sort.Strings(symbols)

	symbols = symbols[:1]

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
	quantilePath := "/Users/chenjilin/Projects/hft-micro/applications/usd-tk-tt-q/configs/bnus-bnuf-quantiles"
	dataPath := "/Volumes/MarketData/MarketData/bnus-bnuf-depth5-and-ticker"

	for _, xSymbol := range symbols {
		ySymbol := xSymbol
		counter := 0
		timedTDigest := stream_stats.NewTimedTDigest(quantileLookback, quantileSubInterval)

		shortLastEnter := 0.0
		longLastEnter := 0.0

		xDepth := &binance_usdtspot.Depth5{}
		yDepth := &binance_usdtfuture.Depth5{}
		for _, dateStr := range strings.Split(dateStrs, ",") {
			logger.Debugf("%s/%s/%s-%s,%s.jl.gz", dataPath, dateStr, dateStr, xSymbol, ySymbol)
			file, err := os.Open(
				fmt.Sprintf("%s/%s/%s-%s,%s.jl.gz", dataPath, dateStr, dateStr, xSymbol, ySymbol),
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
				if msg[0] == 'S' && msg[1] == 'T' {
					err = binance_usdtspot.ParseDepth5(msg[21:], xDepth)
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
				} else {
					continue
				}

				if xDepth.Symbol != "" && yDepth.Symbol != "" {

					shortLastEnter = (yDepth.Bids[0][0] - xDepth.Asks[0][0]) / xDepth.Asks[0][0]
					longLastEnter = (yDepth.Asks[0][0] - xDepth.Bids[0][0]) / xDepth.Bids[0][0]
					_ = timedTDigest.Insert(yDepth.EventTime, (shortLastEnter+longLastEnter)*0.5)
					if counter%1000 == 0 {
						fields := make(map[string]interface{})
						fields["enterMiddle"] = timedTDigest.Quantile(0.5)
						fields["shortLastEnter"] = shortLastEnter
						fields["longLastEnter"] = longLastEnter
						pt, err := client.NewPoint(
							"usd-tk-tt-q-bn-usuf",
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
