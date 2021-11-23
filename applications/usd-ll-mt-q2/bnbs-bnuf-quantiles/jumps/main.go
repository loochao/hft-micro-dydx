package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	binance_busdspot "github.com/geometrybase/hft-micro/binance-busdspot"
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
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
	"sync"
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
	for symbol := range binance_busdspot.TickSizes {
		//logger.Debugf("%s", strings.Replace(symbol, "BUSD", "USDT", -1))
		if _, ok := binance_usdtfuture.TickSizes[strings.Replace(symbol, "BUSD", "USDT", -1)]; ok {
			symbols = append(symbols, symbol)
		}
	}
	sort.Strings(symbols)

	//symbols = symbols[:10]

	startTime, err := time.Parse("20060102", "20210730")
	if err != nil {
		logger.Fatal(err)
	}
	endTime, err := time.Parse("20060102", "20210802")
	if err != nil {
		logger.Fatal(err)
	}
	dateStrs := ""
	for i := startTime; i.Sub(endTime) <= 0; i = i.Add(time.Hour * 24) {
		dateStrs += i.Format("20060102,")
	}
	dateStrs = dateStrs[:len(dateStrs)-1]

	enterThreshold := 0.001
	offsets := make(map[string]string)
	qAddInterval := time.Second
	jumpLookback := time.Second * 1
	quantileLookback := time.Hour * 72
	quantileSubInterval := time.Hour
	outputPath := fmt.Sprintf(
		"/Users/chenjilin/Projects/hft-micro/applications/usd-ll-mt-q2/bnbs-bnuf-quantiles/outputs/%v-%v-%v-%v-%.4f",
		jumpLookback,
		qAddInterval,
		quantileLookback,
		quantileSubInterval,
		enterThreshold,
	)
	bidJumpOutputPath := fmt.Sprintf("%s/bid-jumps", outputPath)
	askJumpOutputPath := fmt.Sprintf("%s/ask-jumps", outputPath)
	timedTDOutputPath := fmt.Sprintf("%s/timed-tds", outputPath)
	save := false
	parallelCh := make(chan interface{}, 16)
	doneCh := make(chan string)
	mu := sync.Mutex{}

	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		err = os.MkdirAll(outputPath, 0755)
		if err != nil {
			logger.Fatal(err)
		}
	} else if err != nil {
		logger.Fatal(err)
	}
	if _, err := os.Stat(bidJumpOutputPath); os.IsNotExist(err) {
		err = os.MkdirAll(bidJumpOutputPath, 0755)
		if err != nil {
			logger.Fatal(err)
		}
	} else if err != nil {
		logger.Fatal(err)
	}
	if _, err := os.Stat(askJumpOutputPath); os.IsNotExist(err) {
		err = os.MkdirAll(askJumpOutputPath, 0755)
		if err != nil {
			logger.Fatal(err)
		}
	} else if err != nil {
		logger.Fatal(err)
	}
	if _, err := os.Stat(timedTDOutputPath); os.IsNotExist(err) {
		err = os.MkdirAll(timedTDOutputPath, 0755)
		if err != nil {
			logger.Fatal(err)
		}
	} else if err != nil {
		logger.Fatal(err)
	}

	dataPath := "/Volumes/MarketData/MarketData/bnbs-bnuf-depth5-and-ticker"

	for _, xSymbol := range symbols {
		go func(xSymbol string) {
			parallelCh <- nil
			defer func() {
				<-parallelCh
				doneCh <- xSymbol
			}()

			lastAskAddTime := time.Time{}
			lastBidAddTime := time.Time{}

			ySymbol := strings.Replace(xSymbol, "BUSD", "USDT", -1)
			counter := 0

			timedTDigest := stream_stats.NewTimedTDigest(quantileLookback, quantileSubInterval)
			quantileMiddle := 0.0

			sizeTD, _ := tdigest.New()

			shortLastEnter := 0.0
			longLastEnter := 0.0

			xDepth := &binance_busdspot.Depth5{}
			xTicker := &binance_busdspot.BookTicker{}
			yDepth := &binance_usdtfuture.Depth5{}
			yTicker := &binance_usdtfuture.BookTicker{}

			bidDelta := stream_stats.NewTimedDelta(jumpLookback)
			askDelta := stream_stats.NewTimedDelta(jumpLookback)

			//bidJumpTD := stream_stats.NewTimedTDigest(quantileLookback, quantileSubInterval)
			//askJumpTD := stream_stats.NewTimedTDigest(quantileLookback, quantileSubInterval)

			bidJumpTD, _ := tdigest.New()
			askJumpTD, _ := tdigest.New()

			hasBidAskJumpAll := true

			data, err := os.ReadFile(fmt.Sprintf("%s/%s.td", bidJumpOutputPath, xSymbol))
			if err != nil {
				logger.Debugf("os.ReadFile error %v", err)
				hasBidAskJumpAll = false
			} else {
				err = bidJumpTD.FromBytes(data)
				if err != nil {
					logger.Debugf("bidJumpTD.FromBytes error %v", err)
					hasBidAskJumpAll = false
				}
			}
			data, err = os.ReadFile(fmt.Sprintf("%s/%s.td", askJumpOutputPath, xSymbol))
			if err != nil {
				logger.Debugf("os.ReadFile error %v", err)
				hasBidAskJumpAll = false
			} else {
				err = askJumpTD.FromBytes(data)
				if err != nil {
					logger.Debugf("askJumpTD.FromBytes error %v", err)
					hasBidAskJumpAll = false
				}
			}

			if !hasBidAskJumpAll {

				var t int64

				var xTD, yTD common.Ticker
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
						if len(msg) < 128 {
							continue
						}
						if msg[0] == 'S' && msg[1] == 'D' {
							err = binance_busdspot.ParseDepth5(msg[21:], xDepth)
							if err != nil {
								logger.Debugf("%v", err)
								continue
							}
							t, err = common.ParseInt(msg[3:21])
							if err != nil {
								logger.Debugf("%v", err)
								continue
							}
							if xDepth.Symbol != xSymbol {
								continue
							}
							xDepth.ParseTime = time.Unix(0, t)
							xTD = xDepth
							askDelta.Insert(xTD.GetEventTime(), xTD.GetAskPrice())
							bidDelta.Insert(xTD.GetEventTime(), xTD.GetBidPrice())
						} else if msg[0] == 'S' && msg[1] == 'T' {
							err = binance_busdspot.ParseBookTicker(msg[21:], xTicker)
							if err != nil {
								logger.Debugf("%v", err)
								continue
							}
							t, err = common.ParseInt(msg[3:21])
							if err != nil {
								logger.Debugf("%v", err)
								continue
							}
							if xTicker.Symbol != xSymbol {
								continue
							}
							xTicker.ParseTime = time.Unix(0, t)
							xTD = xTicker
							askDelta.Insert(xTD.GetEventTime(), xTD.GetAskPrice())
							bidDelta.Insert(xTD.GetEventTime(), xTD.GetBidPrice())
						} else if msg[0] == 'F' && msg[1] == 'D' {
							err = binance_usdtfuture.ParseDepth5(msg[21:], yDepth)
							if err != nil {
								logger.Debugf("%v", err)
								continue
							}
							if yDepth.Symbol != ySymbol {
								continue
							}
							yTD = yDepth
						} else if msg[0] == 'F' && msg[1] == 'T' {
							err = binance_usdtfuture.ParseBookTicker(msg[21:], yTicker)
							if err != nil {
								logger.Debugf("%v", err)
								continue
							}
							if yTicker.Symbol != ySymbol {
								continue
							}
							yTD = yTicker
						} else {
							continue
						}

						if yTD != nil && xTD != nil {
							_ = sizeTD.Add(math.Min(yTD.GetBidSize()*yTD.GetBidPrice(), yTD.GetAskSize()*yTD.GetAskPrice()))
							shortLastEnter = (yTD.GetBidPrice() - xTD.GetBidPrice()) / xTD.GetBidPrice()
							longLastEnter = (yTD.GetAskPrice() - xTD.GetAskPrice()) / xTD.GetAskPrice()
							_ = timedTDigest.Insert(yDepth.EventTime, (shortLastEnter+longLastEnter)*0.5)

							quantileMiddle = timedTDigest.Quantile(0.5)

							if longLastEnter < quantileMiddle-enterThreshold {
								if askDelta.Delta() > 0 && xTD.GetEventTime().Sub(lastAskAddTime) > qAddInterval {
									lastAskAddTime = xTD.GetEventTime()
									//_ = askJumpTD.Insert(xTD.GetEventTime(), askDelta.Delta()/xTD.GetAskPrice())
									_ = askJumpTD.Add(askDelta.Delta() / xTD.GetAskPrice())
								}
							} else if shortLastEnter > quantileMiddle+enterThreshold {
								if bidDelta.Delta() < 0 && xTD.GetEventTime().Sub(lastBidAddTime) > qAddInterval {
									lastBidAddTime = xTD.GetEventTime()
									//_ = bidJumpTD.Insert(xTD.GetEventTime(), bidDelta.Delta()/xTD.GetBidPrice())
									_ = bidJumpTD.Add(bidDelta.Delta() / xTD.GetBidPrice())
								}
							}

							if save && counter%1000 == 0 {
								fields := make(map[string]interface{})
								fields["middlePrice"] = (xTD.GetAskPrice() + xTD.GetBidPrice()) * 0.5
								fields["enterMiddle"] = timedTDigest.Quantile(0.5) * 10000
								fields["shortLastEnter"] = shortLastEnter * 10000
								fields["longLastEnter"] = longLastEnter * 10000
								fields["askJump50"] = askJumpTD.Quantile(0.5) * 10000
								fields["askJump80"] = askJumpTD.Quantile(0.8) * 10000
								fields["askJump95"] = askJumpTD.Quantile(0.95) * 10000
								fields["askJump995"] = askJumpTD.Quantile(0.995) * 10000
								fields["askJump9995"] = askJumpTD.Quantile(0.9995) * 10000
								fields["bidJump50"] = bidJumpTD.Quantile(0.5) * 10000
								fields["bidJump80"] = bidJumpTD.Quantile(0.2) * 10000
								fields["bidJump95"] = bidJumpTD.Quantile(0.05) * 10000
								fields["bidJump995"] = bidJumpTD.Quantile(0.005) * 10000
								fields["bidJump9995"] = bidJumpTD.Quantile(0.0005) * 10000
								pt, err := client.NewPoint(
									"bnbs-bnuf-depth5-and-ticker",
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
					logger.Debugf("json.Marshal error %v", err)
					return
				}
				err = os.WriteFile(path.Join(timedTDOutputPath, xSymbol+"-"+ySymbol+".json"), data, 0755)
				if err != nil {
					logger.Debugf("os.WriteFile error %v", err)
					return
				}

				data, err = bidJumpTD.AsBytes()
				err = os.WriteFile(path.Join(bidJumpOutputPath, xSymbol+".td"), data, 0755)
				if err != nil {
					logger.Debugf("os.WriteFile error %v", err)
					return
				}
				data, err = askJumpTD.AsBytes()
				err = os.WriteFile(path.Join(askJumpOutputPath, xSymbol+".td"), data, 0755)
				if err != nil {
					logger.Debugf("os.WriteFile error %v", err)
					return
				}
			}
			mu.Lock()
			offsets[xSymbol] = fmt.Sprintf(
				"%.6f,%.6f,%.6f,%.6f,%.6f,%.6f",
				bidJumpTD.Quantile(0.005),
				bidJumpTD.Quantile(0.20),
				bidJumpTD.Quantile(0.80),
				askJumpTD.Quantile(0.20),
				askJumpTD.Quantile(0.80),
				askJumpTD.Quantile(0.995),
			)
			fmt.Printf("\n\n%s %s\n\n", xSymbol, offsets[xSymbol])
			mu.Unlock()
		}(xSymbol)
	}

	checkMaps := make(map[string]string)
	for _, xSymbol := range symbols {
		checkMaps[xSymbol] = xSymbol
	}
outerLoop:
	for {
		select {
		case xSymbol := <-doneCh:
			delete(checkMaps, xSymbol)
			if len(checkMaps) == 0 {
				break outerLoop
			}
		case <-time.After(time.Hour):
			logger.Debugf("timeout after 1h")
			break outerLoop
		}
	}

	fmt.Printf("\n\norderOffsets:\n")
	for _, xSymbol := range symbols {
		fmt.Printf("  %s: %s\n", xSymbol, offsets[xSymbol])
	}
}
