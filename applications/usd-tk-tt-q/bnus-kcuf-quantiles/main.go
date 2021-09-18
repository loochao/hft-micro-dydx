package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	binance_usdtspot "github.com/geometrybase/hft-micro/binance-usdtspot"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/influx/client"
	kucoin_usdtfuture "github.com/geometrybase/hft-micro/kucoin-usdtfuture"
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
	symbolsMap := map[string]string{
		"BTCUSDT": "XBTUSDTM",
	}
	symbols := []string{"BTCUSDT"}
	for symbol := range binance_usdtspot.TickSizes {
		if _, ok := kucoin_usdtfuture.TickSizes[strings.Replace(symbol, "USDT", "USDTM", -1)]; ok {
			symbols = append(symbols, symbol)
			symbolsMap[symbol] = strings.Replace(symbol, "USDT", "USDTM", -1)
		}
	}
	sort.Strings(symbols)
	//symbols = symbols[:1]
	startTime, err := time.Parse("20060102", "20210910")
	if err != nil {
		logger.Fatal(err)
	}
	endTime, err := time.Parse("20060102", "20210914")
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
	quantilePath := "/Users/chenjilin/Projects/hft-micro/applications/usd-tk-tt-q/configs/bnus-kcuf-quantiles"
	maxTimeDiff := time.Millisecond * 100
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

		yTicker := &kucoin_usdtfuture.Ticker{}
		yDepth := &kucoin_usdtfuture.Depth5{}
		xDepth := &binance_usdtspot.Depth5{}
		xTicker := &binance_usdtspot.BookTicker{}

		sizeTD, _ := tdigest.New()

		var xTD, yTD common.Ticker
		var t int64
		var lastAddTime = time.Time{}
		for _, dateStr := range strings.Split(dateStrs, ",") {
			logger.Debugf("%s %s %s", xSymbol, dateStr, fmt.Sprintf("/Users/chenjilin/MarketData/kcuf-bnus-depth5-and-ticker/%s/%s-%s,%s.jl.gz", dateStr, dateStr, ySymbol, xSymbol))
			file, err := os.Open(
				fmt.Sprintf("/Users/chenjilin/MarketData/kcuf-bnus-depth5-and-ticker/%s/%s-%s,%s.jl.gz", dateStr, dateStr, ySymbol, xSymbol),
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
				if msg[0] == 'X' && msg[1] == 'T' {
					err = kucoin_usdtfuture.ParseTicker(msg[21:], yTicker)
					if err != nil {
						logger.Debugf("%v", err)
						continue
					}
					if yTicker.Symbol != ySymbol {
						//logger.Debugf("bad msg: %s", msg)
						continue

					}
					yTD = yTicker
				} else if msg[0] == 'X' && msg[1] == 'D' {
					err = kucoin_usdtfuture.ParseDepth5(msg[21:], yDepth)
					if err != nil {
						logger.Debugf("%v", err)
						continue
					}
					if yDepth.Symbol != ySymbol {
						//logger.Debugf("bad msg: %s", msg)
						continue
					}
					yTD = yDepth
				} else if msg[0] == 'Y' && msg[1] == 'D' {
					err = binance_usdtspot.ParseDepth5(msg[21:], xDepth)
					if err != nil {
						logger.Debugf("%v", err)
						continue
					}
					if xDepth.Symbol != xSymbol {
						//logger.Debugf("bad msg: %s", msg)
						continue
					}
					t, err = common.ParseInt(msg[2:21])
					if err != nil {
						logger.Debugf("%v", err)
						continue
					}
					xDepth.ParseTime = time.Unix(0, t)
					xTD = xDepth
				} else if msg[0] == 'Y' && msg[1] == 'T' {
					err = binance_usdtspot.ParseTicker(msg[21:], xTicker)
					if err != nil {
						logger.Debugf("%v", err)
						continue
					}
					if xTicker.Symbol != xSymbol {
						//logger.Debugf("bad msg: %s", msg)
						continue
					}
					t, err = common.ParseInt(msg[2:21])
					if err != nil {
						logger.Debugf("%v", err)
						continue
					}
					xTicker.ParseTime = time.Unix(0, t)
					xTD = xTicker
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
								4*math.Min(xTD.GetBidSize()*xTD.GetBidPrice(), 4*xTD.GetAskSize()*xTD.GetAskPrice()),
								math.Min(yTD.GetBidSize()*yTD.GetBidPrice()*kucoin_usdtfuture.Multipliers[ySymbol], yTD.GetAskSize()*yTD.GetAskPrice()*kucoin_usdtfuture.Multipliers[ySymbol]),
							),
						)
						_ = timedTDigest.Insert(xTD.GetTime(), (shortLastEnter+longLastEnter)*0.5)
						fields := make(map[string]interface{})
						fields["enterMiddle"] = timedTDigest.Quantile(0.5)
						fields["shortLastEnter"] = shortLastEnter
						fields["longLastEnter"] = longLastEnter
						pt, err := client.NewPoint(
							"bnus-kcuf-depth5-and-ticker",
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
