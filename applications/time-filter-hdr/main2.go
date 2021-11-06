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
		5000,
	)
	if err != nil {
		panic(err)
	}
	defer iw.Stop()

	startTime, err := time.Parse("20060102", "20211101")
	if err != nil {
		logger.Fatal(err)
	}
	endTime, err := time.Parse("20060102", "20211104")
	if err != nil {
		logger.Fatal(err)
	}
	dateStrs := ""
	for i := startTime; i.Sub(endTime) <= 0; i = i.Add(time.Hour * 24) {
		dateStrs += i.Format("20060102,")
	}
	dateStrs = dateStrs[:len(dateStrs)-1]

	symbols := []string{"1INCHUSDT"}
	rootPath := "/Volumes/CryptoData/bnuf"
	//rootPath := "/Users/chenjilin/Downloads/bnuf"
	parallelCh := make(chan interface{}, 16)
	doneCh := make(chan string)

	for _, xSymbol := range symbols {

		go func(xSymbol string) {
			parallelCh <- nil
			defer func() {
				<-parallelCh
				doneCh <- xSymbol
			}()

			//maxSize := 10000000.0
			//minSize := 1.0
			//minSize := 1.0
			timedBidSizeTD := stream_stats.NewTimedTDigestWithCompression(time.Hour*4, time.Minute*5, 10)
			//timedBidSizeHDR := stream_stats.NewTimedHdrHistogram(int64(minSize), int64(maxSize), 1, time.Hour*4, time.Minute*5)
			timedBidSizeTD2 := stream_stats.NewTimedTDigestWithCompression(time.Hour*4, time.Minute*5, 100)

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
						//timeDiff = float64(trade.EventTime.UnixNano()-t) / timeScale
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
						//timeDiff = float64(ticker.EventTime.UnixNano()-t) / timeScale
						price = (ticker.BestBidPrice + ticker.BestAskPrice) / 2
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
						//timeDiff = float64(depth.EventTime.UnixNano()-t) / timeScale
						price = (depth.Bids[0][0] + depth.Asks[0][0]) / 2
					}
					if localTime.Sub(lastInsertTime) > insertInterval {
						size := ticker.BestBidQty*ticker.BestBidPrice + 1.0
						err = timedBidSizeTD.Insert(localTime, size)
						if err != nil {
							logger.Debugf("timedBidSizeTD.Insert error %v", err)
							continue
						}
						err = timedBidSizeTD2.Insert(localTime, size)
						if err != nil {
							logger.Debugf("timedBidSizeTD.Insert error %v", err)
							continue
						}
						//if size > maxSize {
						//	size = maxSize
						//}
						//err = timedBidSizeHDR.Insert(localTime, int64(size))
						//if err != nil {
						//	logger.Debugf("timedBidSizeHDR.Insert error %v", err)
						//	continue
						//}
						counter++
					}
					if counter > 1000 &&
						localTime.Sub(lastSaveTime) > saveInterval {
						fields := make(map[string]interface{})
						fields["price"] = price
						fields["timeDiff"] = timeDiff

						fields["bidSizeTD005"] = timedBidSizeTD.Quantile(0.005)
						fields["bidSizeTD05"] = timedBidSizeTD.Quantile(0.05)
						fields["bidSizeTD20"] = timedBidSizeTD.Quantile(0.20)
						fields["bidSizeTD50"] = timedBidSizeTD.Quantile(0.50)
						fields["bidSizeTD80"] = timedBidSizeTD.Quantile(0.80)
						fields["bidSizeTD95"] = timedBidSizeTD.Quantile(0.95)
						fields["bidSizeTD995"] = timedBidSizeTD.Quantile(0.995)

						fields["bidSizeHDR005"] = timedBidSizeTD2.Quantile(0.005)
						fields["bidSizeHDR05"] = timedBidSizeTD2.Quantile(0.05)
						fields["bidSizeHDR20"] = timedBidSizeTD2.Quantile(0.20)
						fields["bidSizeHDR50"] = timedBidSizeTD2.Quantile(0.50)
						fields["bidSizeHDR80"] = timedBidSizeTD2.Quantile(0.80)
						fields["bidSizeHDR95"] = timedBidSizeTD2.Quantile(0.95)
						fields["bidSizeHDR995"] = timedBidSizeTD2.Quantile(0.995)
						//
						//fields["bidSizeHDR005"] = timedBidSizeHDR.Quantile(0.5)
						//fields["bidSizeHDR05"] = timedBidSizeHDR.Quantile(5)
						//fields["bidSizeHDR20"] = timedBidSizeHDR.Quantile(20)
						//fields["bidSizeHDR50"] = timedBidSizeHDR.Quantile(50)
						//fields["bidSizeHDR80"] = timedBidSizeHDR.Quantile(80)
						//fields["bidSizeHDR95"] = timedBidSizeHDR.Quantile(95)
						//fields["bidSizeHDR995"] = timedBidSizeHDR.Quantile(99.5)


						pt, err := client.NewPoint(
							"timeFilterCompare",
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
