package main

//func main() {
//	ctx := context.Background()
//	iw, err := common.NewInfluxWriter(
//		ctx,
//		os.Getenv("INFLUX_URL"),
//		"",
//		"",
//		"hft",
//		5000,
//	)
//	if err != nil {
//		panic(err)
//	}
//	defer iw.Stop()
//
//	startTime, err := time.Parse("20060102", "20211103")
//	if err != nil {
//		logger.Fatal(err)
//	}
//	endTime, err := time.Parse("20060102", "20211103")
//	if err != nil {
//		logger.Fatal(err)
//	}
//	dateStrs := ""
//	for i := startTime; i.Sub(endTime) <= 0; i = i.Add(time.Hour * 24) {
//		dateStrs += i.Format("20060102,")
//	}
//	dateStrs = dateStrs[:len(dateStrs)-1]
//
//	symbols := []string{"1INCHUSDT"}
//	//rootPath := "/Volumes/CryptoData/bnuf"
//	rootPath := "/home/clu/Downloads/bnuf"
//	parallelCh := make(chan interface{}, 16)
//	doneCh := make(chan string)
//
//	for _, xSymbol := range symbols {
//
//		go func(xSymbol string) {
//			parallelCh <- nil
//			defer func() {
//				<-parallelCh
//				doneCh <- xSymbol
//			}()
//
//			maxDiff := int64(time.Second * 10)
//			minDiff := int64(1)
//			hhOffset := maxDiff / 2
//			timedDiffTD := stream_stats.NewTimedTDigest(time.Hour*4, time.Minute*5)
//			timedDiffHDR := stream_stats.NewTimedHdrHistogram(minDiff, maxDiff, 2, time.Hour*4, time.Minute*5)
//
//			lastSaveTime := time.Time{}
//			saveInterval := time.Second
//			counter := 0
//			lastInsertTime := time.Time{}
//			insertInterval := time.Second
//
//			for _, dateStr := range strings.Split(dateStrs, ",") {
//
//				dataPath := path.Join(rootPath, dateStr, fmt.Sprintf("%s-%s.jl.gz", dateStr, xSymbol))
//				logger.Debug(dataPath)
//				file, err := os.Open(dataPath)
//				if err != nil {
//					logger.Debugf("os.Open() error %v", err)
//					continue
//				}
//				gr, err := gzip.NewReader(file)
//				if err != nil {
//					logger.Debugf("gzip.NewReader(file) error %v", err)
//					continue
//				}
//				b := make([]byte, 0, 512)
//				_, err = gr.Read(b)
//				if err != nil {
//					logger.Debugf("gr.Read(b) error %v", err)
//					continue
//				}
//				scanner := bufio.NewScanner(gr)
//				var msg []byte
//				var t int64
//				var localTime time.Time
//				var trade *binance_usdtfuture.Trade
//				var depth = &binance_usdtfuture.Depth20{}
//				var ticker = &binance_usdtfuture.BookTicker{}
//				var timeDiff, timeDiffWithOffset int64
//				var price float64
//				for scanner.Scan() {
//					msg = scanner.Bytes()
//					if len(msg) < 128 {
//						continue
//					}
//					if msg[0] == 'T' {
//						t, err = common.ParseInt(msg[1:20])
//						if err != nil {
//							logger.Debugf("common.ParseInt error %v", err)
//							continue
//						}
//						localTime = time.Unix(0, t)
//						trade, err = binance_usdtfuture.ParseTrade(msg[20:])
//						if err != nil {
//							logger.Debugf("binance_usdtfuture.ParseTrade error %v", err)
//							continue
//						}
//						timeDiff = trade.EventTime.UnixNano() - t
//						timeDiffWithOffset = timeDiff + hhOffset
//						price = trade.Price
//					} else if msg[0] == 'B' {
//						t, err = common.ParseInt(msg[1:20])
//						if err != nil {
//							logger.Debugf("common.ParseInt error %v", err)
//							continue
//						}
//						localTime = time.Unix(0, t)
//						err = binance_usdtfuture.ParseBookTicker(msg[20:], ticker)
//						if err != nil {
//							logger.Debugf("binance_usdtfuture.ParseBookTicker error %v", err)
//							continue
//						}
//						timeDiff = ticker.EventTime.UnixNano() - t
//						timeDiffWithOffset = timeDiff + hhOffset
//						price = (ticker.BestBidPrice + ticker.BestAskPrice) / 2
//					} else if msg[0] == 'D' {
//						t, err = common.ParseInt(msg[1:20])
//						if err != nil {
//							logger.Debugf("common.ParseInt error %v", err)
//							continue
//						}
//						localTime = time.Unix(0, t)
//						err = binance_usdtfuture.ParseDepth20(msg[20:], depth)
//						if err != nil {
//							logger.Debugf("binance_usdtfuture.ParseBookTicker error %v", err)
//							continue
//						}
//						timeDiff = depth.EventTime.UnixNano() - t
//						timeDiffWithOffset = timeDiff + hhOffset
//						price = (depth.Bids[0][0] + depth.Asks[0][0]) / 2
//					}
//					if localTime.Sub(lastInsertTime) > insertInterval {
//						if timeDiffWithOffset > maxDiff {
//							timeDiffWithOffset = maxDiff
//						}
//						if timeDiffWithOffset < minDiff {
//							timeDiffWithOffset = minDiff
//						}
//						err = timedDiffHDR.Insert(localTime, timeDiffWithOffset)
//						if err != nil {
//							logger.Debugf("timedDiffHDR.Insert error %v", err)
//							continue
//						}
//						err = timedDiffTD.Insert(localTime, float64(timeDiffWithOffset-hhOffset))
//						if err != nil {
//							logger.Debugf("timedDiffTD.Insert error %v", err)
//							continue
//						}
//						counter++
//					}
//					if counter > 1000 &&
//						localTime.Sub(lastSaveTime) > saveInterval {
//						fields := make(map[string]interface{})
//						fields["price"] = price
//						fields["timeDiff"] = timeDiff
//
//						fields["diffTD005"] = timedDiffTD.Quantile(0.005)
//						fields["diffTD05"] = timedDiffTD.Quantile(0.05)
//						fields["diffTD20"] = timedDiffTD.Quantile(0.20)
//						fields["diffTD50"] = timedDiffTD.Quantile(0.50)
//						fields["diffTD80"] = timedDiffTD.Quantile(0.80)
//						fields["diffTD95"] = timedDiffTD.Quantile(0.95)
//						fields["diffTD995"] = timedDiffTD.Quantile(0.995)
//
//						fields["diffHDR005"] = timedDiffHDR.Quantile(0.5) - hhOffset
//						fields["diffHDR05"] = timedDiffHDR.Quantile(5) - hhOffset
//						fields["diffHDR20"] = timedDiffHDR.Quantile(20) - hhOffset
//						fields["diffHDR50"] = timedDiffHDR.Quantile(50) - hhOffset
//						fields["diffHDR80"] = timedDiffHDR.Quantile(80) - hhOffset
//						fields["diffHDR95"] = timedDiffHDR.Quantile(95) - hhOffset
//						fields["diffHDR995"] = timedDiffHDR.Quantile(99.5) - hhOffset
//
//						pt, err := client.NewPoint(
//							"timeFilterCompare2",
//							map[string]string{
//								"xSymbol": xSymbol,
//							},
//							fields,
//							localTime,
//						)
//						if err == nil {
//							iw.PointCh <- pt
//						}
//						lastSaveTime = localTime
//					}
//
//				}
//			}
//		}(xSymbol)
//	}
//	symbolCounter := 0
//outerLoop:
//	for {
//		select {
//		case _ = <-doneCh:
//			symbolCounter += 1
//			if len(symbols) == symbolCounter {
//				break outerLoop
//			}
//		case <-time.After(time.Hour):
//			logger.Debugf("timeout after 1h")
//			break outerLoop
//		}
//	}
//}
//
//
