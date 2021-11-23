package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
	"github.com/geometrybase/hft-micro/common"
	dydx_usdfuture "github.com/geometrybase/hft-micro/dydx-usdfuture"
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

	symbols := make([]string, 0)
	for symbol := range dydx_usdfuture.TickSizes {
		if _, ok := binance_usdtfuture.TickSizes[strings.Replace(symbol, "-USD", "USDT", -1)]; ok {
			symbols = append(symbols, symbol)
		}
	}
	sort.Strings(symbols)
	//symbols = symbols[:1]
	logger.Debugf("%s", symbols)

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

	startTime, err := time.Parse("20060102", "20211011")
	if err != nil {
		logger.Fatal(err)
	}
	endTime, err := time.Parse("20060102", "20211015")
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
	quantilePath := "/Users/chenjilin/Projects/hft-micro/applications/usd-tk-tt-q/dduf-bnuf-quantiles/outputs"
	dataPath := "/Users/chenjilin/MarketData/dduf-bnuf-depth-and-ticker"

	sizeTDs := make(map[string]*tdigest.TDigest)
	quantileMiddle := 0.0
	enterThreshold := 0.0002

	for _, xSymbol := range symbols {
		ySymbol := strings.Replace(xSymbol, "-USD", "USDT", -1)
		//if _, err := os.Stat(path.Join(quantilePath, xSymbol+"-"+ySymbol+"-long-td.json")); err == nil {
		//	logger.Debugf("Exists %s %s %v", ySymbol, xSymbol, err)
		//	continue
		//} else if !os.IsNotExist(err) {
		//	logger.Debugf("Error %s %s %v", ySymbol, xSymbol, err)
		//	continue
		//}
		counter := 0
		timedTDigest := stream_stats.NewTimedTDigest(quantileLookback, quantileSubInterval)

		shortLastEnter := 0.0
		longLastEnter := 0.0

		xDepth := &dydx_usdfuture.Depth{}
		yDepth := &binance_usdtfuture.Depth5{}
		yTicker := &binance_usdtfuture.BookTicker{}

		var xTD, yTD common.Ticker

		sizeTD, _ := tdigest.New()

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
				if msg[0] == 'X' && msg[1] == 'D' {
					if msg[30] == 'u' {
						continue
					}
					err = dydx_usdfuture.UpdateDepth(msg[21:], xDepth)
					if err != nil {
						logger.Debugf("%v", err)
						continue
					}
					t, err := common.ParseInt(msg[2:21])
					if err != nil {
						logger.Debugf("%v", err)
						continue
					}

					xDepth.ParseTime = time.Unix(0, t)
					if xDepth.IsValid() {
						xTD = xDepth
					} else {
						continue
					}
				} else if msg[0] == 'Y' && msg[1] == 'D' {
					err = binance_usdtfuture.ParseDepth5(msg[21:], yDepth)
					if err != nil {
						logger.Debugf("%v", err)
						continue
					}
					yTD = yDepth
				} else if msg[0] == 'Y' && msg[1] == 'T' {
					err = binance_usdtfuture.ParseBookTicker(msg[21:], yTicker)
					if err != nil {
						logger.Debugf("%v", err)
						continue
					}
					yTD = yTicker
				} else {
					continue
				}

				if yTD != nil &&
					xTD != nil {
					tDiff := yTD.GetEventTime().Sub(xTD.GetEventTime())

					if tDiff > -time.Second &&
						tDiff < time.Second {

						shortLastEnter = (yTD.GetBidPrice() - xTD.GetAskPrice()) / xTD.GetAskPrice()
						longLastEnter = (yTD.GetAskPrice() - xTD.GetBidPrice()) / xTD.GetBidPrice()
						_ = timedTDigest.Insert(yDepth.EventTime, (shortLastEnter+longLastEnter)*0.5)
						quantileMiddle = timedTDigest.Quantile(0.5)
						if shortLastEnter > quantileMiddle+enterThreshold {
							_ = sizeTD.Add(math.Min(yTD.GetBidPrice()*yTD.GetBidSize(), xTD.GetAskPrice()*xTD.GetAskSize()))
						} else if longLastEnter < quantileMiddle-enterThreshold {
							_ = sizeTD.Add(math.Min(yTD.GetAskPrice()*yTD.GetAskSize(), xTD.GetBidPrice()*xTD.GetBidSize()))
						}
						if counter%1000 == 0 {
							fields := make(map[string]interface{})
							fields["enterMiddle"] = timedTDigest.Quantile(0.5)
							fields["shortLastEnter"] = shortLastEnter
							fields["longLastEnter"] = longLastEnter
							pt, err := client.NewPoint(
								"dduf-bnuf-depth-and-ticker",
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
			}
			_ = gr.Close()
			_ = file.Close()
		}
		data, err := json.Marshal(timedTDigest)
		if err != nil {
			logger.Debugf("%v", err)
			continue
		}
		err = os.WriteFile(path.Join(quantilePath, xSymbol+"-"+ySymbol+".json"), data, 0755)
		if err != nil {
			logger.Debugf("%v", err)
			continue
		}
		sizeTDs[xSymbol] = sizeTD
		fmt.Printf("  %s: %.0f\n", xSymbol, sizeTD.Quantile(0.8))
	}
	fmt.Printf("\n\nxyPairs:\n")
	for _, xSymbol := range symbols {
		fmt.Printf("  %s: %s\n", xSymbol, strings.Replace(xSymbol, "-USD", "USDT", -1))
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
