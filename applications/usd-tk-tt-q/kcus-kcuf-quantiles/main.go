package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/influx/client"
	kucoin_usdtfuture "github.com/geometrybase/hft-micro/kucoin-usdtfuture"
	kucoin_usdtspot "github.com/geometrybase/hft-micro/kucoin-usdtspot"
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
	symbolsMap := map[string]string{"BTC-USDT": "XBTUSDTM"}
	symbols := []string{"BTC-USDT"}
	for symbol := range kucoin_usdtspot.TickSizes {
		if _, ok := kucoin_usdtfuture.TickSizes[strings.Replace(symbol, "-USDT", "USDTM", -1)]; ok {
			symbols = append(symbols, symbol)
			symbolsMap[symbol] = strings.Replace(symbol, "-USDT", "USDTM", -1)
		}
	}
	sort.Strings(symbols)
	//symbols = symbols[:1]
	fmt.Printf("\n\nxyPairs:\n")
	for _, xSymbol := range symbols {
		fmt.Printf("  %s: %s\n", xSymbol, symbolsMap[xSymbol])
	}
	startTime, err := time.Parse("20060102", "20210831")
	if err != nil {
		logger.Fatal(err)
	}
	endTime, err := time.Parse("20060102", "20210904")
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
	quantilePath := "/home/clu/Projects/hft-micro/applications/usd-tk-tt-q/configs/kcus-kcuf-quantiles"
	dataPath := "/home/clu/MarketData/kcus-kcuf-depth5-and-ticker"
	maxTimeDiff := time.Millisecond * 1000
	quantileAddInterval := time.Second

	if _, err := os.Stat(quantilePath); err != nil && os.IsNotExist(err) {
		err = os.MkdirAll(quantilePath, 0775)
		if err != nil {
			logger.Fatal(err)
		}
	} else if err != nil {
		logger.Fatal(err)
	}

	sizeTDs := make(map[string]*tdigest.TDigest)

	for _, xSymbol := range symbols {
		ySymbol := symbolsMap[xSymbol]
		logger.Debugf("%s %s", xSymbol, ySymbol)
		timedTDigest := stream_stats.NewTimedTDigest(quantileLookback, quantileSubInterval)

		shortLastEnter := 0.0
		longLastEnter := 0.0

		xDepth := &kucoin_usdtspot.Depth5{}
		xTicker := &kucoin_usdtspot.Ticker{}
		yDepth := &kucoin_usdtfuture.Depth5{}
		yTicker := &kucoin_usdtfuture.Ticker{}

		sizeTD, _ := tdigest.New()

		var xTD, yTD common.Ticker
		var lastAddTime = time.Time{}
		for _, dateStr := range strings.Split(dateStrs, ",") {
			//logger.Debugf(
			//	"%s/%s/%s-%s,%s.jl.gz",
			//	dataPath, dateStr, dateStr,
			//	common.SymbolSanitize(xSymbol),
			//	common.SymbolSanitize(ySymbol),
			//)
			file, err := os.Open(
				fmt.Sprintf(
					"%s/%s/%s-%s,%s.jl.gz",
					dataPath, dateStr, dateStr,
					common.SymbolSanitize(xSymbol),
					common.SymbolSanitize(ySymbol),
				),
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
				msg = scanner.Bytes()
				if msg[0] == 'X' && msg[1] == 'D' {
					err = kucoin_usdtspot.ParseDepth5(msg[21:], xDepth)
					if err != nil {
						logger.Debugf("kucoin_usdtspot.ParseDepth5 error %v", err)
						continue
					}
					xTD = xDepth
				} else if msg[0] == 'X' && msg[1] == 'T' {
					err = kucoin_usdtspot.ParseTicker(msg[21:], xTicker)
					if err != nil {
						logger.Debugf("kucoin_usdtspot.ParseTicker error %v", err)
						continue
					}
					xTD = xTicker
				} else if msg[0] == 'Y' && msg[1] == 'D' {
					err = kucoin_usdtfuture.ParseDepth5(msg[21:], yDepth)
					if err != nil {
						logger.Debugf("kucoin_usdtfuture.ParseDepth5 error %v", err)
						continue
					}
					yTD = yDepth
				} else if msg[0] == 'Y' && msg[1] == 'T' {
					err = kucoin_usdtfuture.ParseTicker(msg[21:], yTicker)
					if err != nil {
						logger.Debugf("kucoin_usdtfuture.ParseTicker error %v", err)
						continue
					}
					yTD = yTicker
				} else {
					continue
				}

				if yTD != nil && xTD != nil {
					tDiff := xTD.GetEventTime().Sub(yTD.GetEventTime())
					if tDiff < maxTimeDiff &&
						tDiff > -maxTimeDiff &&
						xTD.GetEventTime().Sub(lastAddTime) >= quantileAddInterval {
						_ = sizeTD.Add(
							math.Min(
								math.Min(xTD.GetBidSize()*xTD.GetBidPrice(), xTD.GetAskSize()*xTD.GetAskPrice()),
								math.Min(yTD.GetBidSize()*yTD.GetBidPrice()*kucoin_usdtfuture.Multipliers[ySymbol], yTD.GetAskSize()*yTD.GetAskPrice()*kucoin_usdtfuture.Multipliers[ySymbol]),
							),
						)
						lastAddTime = xTD.GetEventTime()
						shortLastEnter = (yTD.GetBidPrice() - xTD.GetAskPrice()) / xTD.GetAskPrice()
						longLastEnter = (yTD.GetAskPrice() - xTD.GetBidPrice()) / xTD.GetBidPrice()
						_ = timedTDigest.Insert(yDepth.EventTime, (shortLastEnter+longLastEnter)*0.5)
						fields := make(map[string]interface{})
						fields["enterMiddle"] = timedTDigest.Quantile(0.5)
						fields["shortLastEnter"] = shortLastEnter
						fields["longLastEnter"] = longLastEnter
						pt, err := client.NewPoint(
							"kcus-kcuf-quantiles",
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

		if math.IsNaN(sizeTD.Quantile(0.5)) ||
			math.IsNaN(timedTDigest.Quantile(0.5)) {
			logger.Debugf("bad quantile for %s", xSymbol)
			continue
		}
		sizeTDs[xSymbol] = sizeTD
		data, err := json.Marshal(timedTDigest)
		if err != nil {
			logger.Debugf("%v", err)
			continue
		}
		file, err := os.OpenFile(
			path.Join(quantilePath, common.SymbolSanitize(xSymbol)+"-"+common.SymbolSanitize(ySymbol)+".json"),
			os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755,
		)
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

	symbols = make([]string, 0)
	for symbol := range sizeTDs {
		symbols = append(symbols, symbol)
	}
	sort.Strings(symbols)

	fmt.Printf("\n\nxyPairs:\n")
	for _, xSymbol := range symbols {
		fmt.Printf("  %s: %s\n", xSymbol, symbolsMap[xSymbol])
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
