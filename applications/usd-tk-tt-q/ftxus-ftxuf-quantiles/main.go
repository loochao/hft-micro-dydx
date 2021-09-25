package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	ftxuf "github.com/geometrybase/hft-micro/ftx-usdfuture"
	ftxus "github.com/geometrybase/hft-micro/ftx-usdspot"
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
	symbolsMap := make(map[string]string, 0)
	symbols := make([]string, 0)
	for symbol := range ftxus.PriceIncrements {
		if _, ok := ftxuf.PriceIncrements[strings.Replace(symbol, "/USD", "-PERP", -1)]; ok {
			symbols = append(symbols, symbol)
			symbolsMap[symbol] = strings.Replace(symbol, "/USD", "-PERP", -1)
		}
	}
	sort.Strings(symbols)
	//symbols = symbols[:1]
	logger.Debugf("SYMBOLS %s", symbols)
	startTime, err := time.Parse("20060102", "20210918")
	if err != nil {
		logger.Fatal(err)
	}
	endTime, err := time.Parse("20060102", "20210923")
	if err != nil {
		logger.Fatal(err)
	}
	dateStrs := ""
	for i := startTime; i.Sub(endTime) <= 0; i = i.Add(time.Hour * 24) {
		dateStrs += i.Format("20060102,")
	}
	dateStrs = dateStrs[:len(dateStrs)-1]

	quantileLookback := time.Hour * 24
	quantileSubInterval := time.Hour
	quantilePath := "/Users/chenjilin/Projects/hft-micro/applications/usd-tk-tt-q/configs/ftxus-ftxuf-ticker"
	maxTimeDiff := time.Millisecond * 1000
	quantileAddInterval := time.Second

	err = os.MkdirAll(quantilePath, 0755)
	if err != nil {
		logger.Fatal(err)
	}

	sizeTDs := make(map[string]*tdigest.TDigest)

	for _, xSymbol := range symbols {
		ySymbol := symbolsMap[xSymbol]
		timedTDigest := stream_stats.NewTimedTDigest(quantileLookback, quantileSubInterval)

		shortLastEnter := 0.0
		longLastEnter := 0.0

		xTicker := &ftxus.Ticker{}
		yTicker := &ftxuf.Ticker{}

		sizeTD, _ := tdigest.New()

		var xTD, yTD common.Ticker
		var lastAddTime = time.Time{}
		for _, dateStr := range strings.Split(dateStrs, ",") {
			logger.Debugf(
				"/Users/chenjilin/MarketData/ftxus-ftxuf-ticker/%s/%s-%s,%s.jl.gz",
				dateStr, dateStr,
				common.SymbolSanitize(xSymbol),
				common.SymbolSanitize(ySymbol),
			)
			file, err := os.Open(
				fmt.Sprintf(
					"/Users/chenjilin/MarketData/ftxus-ftxuf-ticker/%s/%s-%s,%s.jl.gz",
					dateStr, dateStr,
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
				if msg[0] == 'X' && msg[1] == 'T' && len(msg) > 21{
					err = ftxus.ParseTicker(msg[21:], xTicker)
					if err != nil {
						logger.Debugf("%v", err)
						continue
					}
					if xTicker.Symbol != xSymbol {
						//logger.Debugf("bad msg: %s", msg)
						continue
					}
					xTD = xTicker
				} else if msg[0] == 'Y' && msg[1] == 'T'  && len(msg) > 21{
					err = ftxuf.ParseTicker(msg[21:], yTicker)
					if err != nil {
						logger.Debugf("%v", err)
						continue
					}
					if yTicker.Symbol != ySymbol {
						//logger.Debugf("bad msg: %s", msg)
						continue
					}
					yTD = yTicker
				} else {
					continue
				}

				if yTD != nil && xTD != nil {
					tDiff := xTD.GetTime().Sub(yTD.GetTime())
					if tDiff < maxTimeDiff &&
						tDiff > -maxTimeDiff &&
						xTD.GetTime().Sub(lastAddTime) >= quantileAddInterval {

						shortLastEnter = (yTD.GetBidPrice() - xTD.GetAskPrice()) / xTD.GetAskPrice()
						longLastEnter = (yTD.GetAskPrice() - xTD.GetBidPrice()) / xTD.GetBidPrice()
						lastAddTime = xTD.GetTime()
						_ = sizeTD.Add(
							math.Min(
								xTD.GetBidSize()*xTD.GetBidPrice(),
								yTD.GetAskSize()*yTD.GetAskPrice(),
							),
						)
						_ = timedTDigest.Insert(xTD.GetTime(), (shortLastEnter+longLastEnter)*0.5)
						fields := make(map[string]interface{})
						fields["enterMiddle"] = timedTDigest.Quantile(0.5)
						fields["shortLastEnter"] = shortLastEnter
						fields["longLastEnter"] = longLastEnter
						pt, err := client.NewPoint(
							"ftxus-ftxuf-ticker",
							map[string]string{
								"xSymbol": xSymbol,
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
		file, err := os.OpenFile(path.Join(quantilePath,
			common.SymbolSanitize(xSymbol)+"-"+common.SymbolSanitize(ySymbol)+".json",
		), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
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
		fmt.Printf("  %s: %.0f\n", xSymbol, td.Quantile(0.95))
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
