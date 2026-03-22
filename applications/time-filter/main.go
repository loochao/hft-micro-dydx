package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/influx/client"
	"github.com/geometrybase/hft-micro/logger"
	stream_stats "github.com/geometrybase/hft-micro/stream-stats"
	"math"
	"os"
	"path"
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
		5000,
	)
	if err != nil {
		panic(err)
	}
	defer iw.Stop()

	startTime, err := time.Parse("20060102", "20211030")
	if err != nil {
		logger.Fatal(err)
	}
	endTime, err := time.Parse("20060102", "20211102")
	if err != nil {
		logger.Fatal(err)
	}
	dateStrs := ""
	for i := startTime; i.Sub(endTime) <= 0; i = i.Add(time.Hour * 24) {
		dateStrs += i.Format("20060102,")
	}
	dateStrs = dateStrs[:len(dateStrs)-1]

	symbols := []string{"BTCUSDT"}
	rootPath := "/Volumes/CryptoData/bnuf"
	parallelCh := make(chan interface{}, 16)
	doneCh := make(chan string)

	for _, xSymbol := range symbols {

		go func(xSymbol string) {
			parallelCh <- nil
			defer func() {
				<-parallelCh
				doneCh <- xSymbol
			}()

			//timeDiffTD := stream_stats.NewTimedHdrHistogram(
			//	timeBot, timeTop, 3,
			//	time.Hour*24, time.Minute*15,
			//)
			timeScale := float64(time.Millisecond)
			timeDiffMax := float64(time.Second * 5 / time.Millisecond)
			timedBidDelta := stream_stats.NewTimedDelta(time.Second * 5)
			timedAskDelta := stream_stats.NewTimedDelta(time.Second * 5)

			timeDiffTD := stream_stats.NewTimedTDigest(time.Hour*4, time.Minute*5)
			timedBidDeltaTD := stream_stats.NewTimedTDigest(time.Hour*4, time.Minute*5)
			timedAskDeltaTD := stream_stats.NewTimedTDigest(time.Hour*4, time.Minute*5)
			timedBidSizeTD := stream_stats.NewTimedTDigest(time.Hour*4, time.Minute*5)
			timedAskSizeTD := stream_stats.NewTimedTDigest(time.Hour*4, time.Minute*5)

			lastSaveTime := time.Time{}
			saveInterval := time.Second
			counter := 0
			lastInsertTime := time.Time{}
			insertInterval := time.Second

			for _, dateStr := range strings.Split(dateStrs, ",") {

				dataPath := path.Join(rootPath, dateStr, fmt.Sprintf("%s-%s.jl.gz", dateStr, xSymbol))
				logger.Debug(dataPath)
				file, err := os.Open(dataPath)
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
				var t int64
				var localTime time.Time
				var trade *binance_usdtfuture.Trade
				var depth = &binance_usdtfuture.Depth20{}
				var ticker = &binance_usdtfuture.BookTicker{}
				var timeDiff float64
				var price float64
				for scanner.Scan() {
					msg = scanner.Bytes()
					if len(msg) < 128 {
						continue
					}
					if msg[0] == 'T' {
						t, err = common.ParseInt(msg[1:20])
						if err != nil {
							logger.Debugf("common.ParseInt error %v", err)
							continue
						}
						localTime = time.Unix(0, t)
						trade, err = binance_usdtfuture.ParseTrade(msg[20:])
						if err != nil {
							logger.Debugf("binance_usdtfuture.ParseTrade error %v", err)
							continue
						}
						timeDiff = float64(trade.EventTime.UnixNano()-t) / timeScale
						price = trade.Price
					} else if msg[0] == 'B' {
						t, err = common.ParseInt(msg[1:20])
						if err != nil {
							logger.Debugf("common.ParseInt error %v", err)
							continue
						}
						localTime = time.Unix(0, t)
						err = binance_usdtfuture.ParseBookTicker(msg[20:], ticker)
						if err != nil {
							logger.Debugf("binance_usdtfuture.ParseBookTicker error %v", err)
							continue
						}
						timeDiff = float64(ticker.EventTime.UnixNano()-t) / timeScale
						price = (ticker.BestBidPrice + ticker.BestAskPrice) / 2
						timedBidDelta.Insert(localTime, ticker.BestBidPrice)
						timedAskDelta.Insert(localTime, ticker.BestAskPrice)
					} else if msg[0] == 'D' {
						t, err = common.ParseInt(msg[1:20])
						if err != nil {
							logger.Debugf("common.ParseInt error %v", err)
							continue
						}
						localTime = time.Unix(0, t)
						err = binance_usdtfuture.ParseDepth20(msg[20:], depth)
						if err != nil {
							logger.Debugf("binance_usdtfuture.ParseBookTicker error %v", err)
							continue
						}
						timeDiff = float64(depth.EventTime.UnixNano()-t) / timeScale
						price = (depth.Bids[0][0] + depth.Asks[0][0]) / 2
						timedBidDelta.Insert(localTime, depth.Bids[0][0])
						timedAskDelta.Insert(localTime, depth.Asks[0][0])
					}
					if localTime.Sub(lastInsertTime) > insertInterval &&
						math.Abs(timeDiff) < timeDiffMax {
						err = timeDiffTD.Insert(localTime, timeDiff)
						if err != nil {
							logger.Debugf("timeDiffTD.Insert error %v", err)
							continue
						}
						lastInsertTime = localTime
						if timedBidDelta.Delta() < 0 {
							err = timedBidDeltaTD.Insert(localTime, timedBidDelta.Delta()/price)
							if err != nil {
								logger.Debugf("timedBidDeltaTD.Insert error %v", err)
								continue
							}
						}
						if timedAskDelta.Delta() > 0 {
							err = timedAskDeltaTD.Insert(localTime, timedAskDelta.Delta()/price)
							if err != nil {
								logger.Debugf("timedAskDeltaTD.Insert error %v", err)
								continue
							}
						}
						err = timedBidSizeTD.Insert(localTime, ticker.BestBidQty*ticker.BestBidPrice)
						if err != nil {
							logger.Debugf("timedBidSizeTD.Insert error %v", err)
							continue
						}
						err = timedAskSizeTD.Insert(localTime, ticker.BestAskQty*ticker.BestAskPrice)
						if err != nil {
							logger.Debugf("timedAskSizeTD.Insert error %v", err)
							continue
						}
						counter++
					}
					if counter > 1000 &&
						math.Abs(timeDiff) < timeDiffMax &&
						localTime.Sub(lastSaveTime) > saveInterval {
						fields := make(map[string]interface{})
						fields["price"] = price
						fields["timeDiff"] = timeDiff
						fields["timeDiff"] = timeDiff
						fields["timeDiff005"] = timeDiffTD.Quantile(0.005)
						fields["timeDiff05"] = timeDiffTD.Quantile(0.05)
						fields["timeDiff20"] = timeDiffTD.Quantile(0.20)
						fields["timeDiff50"] = timeDiffTD.Quantile(0.50)
						fields["timeDiff80"] = timeDiffTD.Quantile(0.80)
						fields["timeDiff95"] = timeDiffTD.Quantile(0.95)
						fields["timeDiff995"] = timeDiffTD.Quantile(0.995)

						fields["bidSize005"] = timedBidSizeTD.Quantile(0.005)
						fields["bidSize05"] = timedBidSizeTD.Quantile(0.05)
						fields["bidSize20"] = timedBidSizeTD.Quantile(0.20)
						fields["bidSize50"] = timedBidSizeTD.Quantile(0.50)
						fields["bidSize80"] = timedBidSizeTD.Quantile(0.80)
						fields["bidSize95"] = timedBidSizeTD.Quantile(0.95)
						fields["bidSize995"] = timedBidSizeTD.Quantile(0.995)

						fields["askSize005"] = timedAskSizeTD.Quantile(0.005)
						fields["askSize05"] = timedAskSizeTD.Quantile(0.05)
						fields["askSize20"] = timedAskSizeTD.Quantile(0.20)
						fields["askSize50"] = timedAskSizeTD.Quantile(0.50)
						fields["askSize80"] = timedAskSizeTD.Quantile(0.80)
						fields["askSize95"] = timedAskSizeTD.Quantile(0.95)
						fields["askSize995"] = timedAskSizeTD.Quantile(0.995)

						fields["bidDelta005"] = timedBidDeltaTD.Quantile(0.005)
						fields["bidDelta05"] = timedBidDeltaTD.Quantile(0.05)
						fields["bidDelta20"] = timedBidDeltaTD.Quantile(0.20)
						fields["bidDelta50"] = timedBidDeltaTD.Quantile(0.50)
						fields["bidDelta80"] = timedBidDeltaTD.Quantile(0.80)
						fields["bidDelta95"] = timedBidDeltaTD.Quantile(0.95)
						fields["bidDelta995"] = timedBidDeltaTD.Quantile(0.995)

						fields["askDelta005"] = timedAskDeltaTD.Quantile(0.005)
						fields["askDelta05"] = timedAskDeltaTD.Quantile(0.05)
						fields["askDelta20"] = timedAskDeltaTD.Quantile(0.20)
						fields["askDelta50"] = timedAskDeltaTD.Quantile(0.50)
						fields["askDelta80"] = timedAskDeltaTD.Quantile(0.80)
						fields["askDelta95"] = timedAskDeltaTD.Quantile(0.95)
						fields["askDelta995"] = timedAskDeltaTD.Quantile(0.995)

						pt, err := client.NewPoint(
							"timeFilter",
							map[string]string{
								"xSymbol": xSymbol,
							},
							fields,
							localTime,
						)
						if err == nil {
							iw.PointCh <- pt
						}
						lastSaveTime = localTime
					}

				}
			}
		}(xSymbol)
	}
	symbolCounter := 0
outerLoop:
	for {
		select {
		case _ = <-doneCh:
			symbolCounter += 1
			if len(symbols) == symbolCounter {
				break outerLoop
			}
		case <-time.After(time.Hour):
			logger.Debugf("timeout after 1h")
			break outerLoop
		}
	}
}

