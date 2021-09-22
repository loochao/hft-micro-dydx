package main

import (
	"bufio"
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"fmt"
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
	"github.com/geometrybase/hft-micro/common"
	kucoin_usdtfuture "github.com/geometrybase/hft-micro/kucoin-usdtfuture"
	"github.com/geometrybase/hft-micro/logger"
	stream_stats "github.com/geometrybase/hft-micro/stream-stats"
	"os"
	"sort"
	"strings"
	"time"
)

func main() {
	symbolsMap := map[string]string{
		"XBTUSDTM": "BTCUSDT",
	}
	symbols := []string{"XBTUSDTM"}
	for symbol := range kucoin_usdtfuture.TickSizes {
		if _, ok := binance_usdtfuture.TickSizes[strings.Replace(symbol, "USDTM", "USDT", -1)]; ok {
			symbols = append(symbols, symbol)
			symbolsMap[symbol] = strings.Replace(symbol, "USDTM", "USDT", -1)
		}
	}
	sort.Strings(symbols)
	//symbols = symbols[:1]
	symbols = []string{"ADAUSDTM"}
	startDateStr := "20210820"
	endDateStr := "20210919"
	startTime, err := time.Parse("20060102", startDateStr)
	if err != nil {
		logger.Fatal(err)
	}
	endTime, err := time.Parse("20060102", endDateStr)
	if err != nil {
		logger.Fatal(err)
	}
	dateStrs := ""
	for i := startTime; i.Sub(endTime) <= 0; i = i.Add(time.Hour * 24) {
		dateStrs += i.Format("20060102,")
	}
	dateStrs = dateStrs[:len(dateStrs)-1]

	quantileLookback := time.Hour * 72
	quantileSubInterval := time.Minute * 5
	maxTimeDiff := time.Millisecond * 100
	quantileAddInterval := time.Second
	spreadLookback := time.Second * 3
	outputInterval := time.Millisecond

	workerCh := make(chan interface{}, 8)
	doneSymbolCh := make(chan string, 100)

	for _, xSymbol := range symbols {
		ySymbol := symbolsMap[xSymbol]

		go func(xSymbol, ySymbol string) {

			workerCh <- nil
			defer func() {
				<-workerCh
				doneSymbolCh <- xSymbol
			}()

			timedTDigest := stream_stats.NewTimedTDigest(quantileLookback, quantileSubInterval)

			shortLastSpread := 0.0
			longLastSpread := 0.0
			shortMedianSpread := common.NewTimedMedian(spreadLookback)
			longMedianSpread := common.NewTimedMedian(spreadLookback)

			xFundingRate := &kucoin_usdtfuture.CurrentFundingRate{}
			xDepth := &kucoin_usdtfuture.Depth5{}
			xTicker := &kucoin_usdtfuture.Ticker{}

			yFundingRate := &binance_usdtfuture.PremiumIndex{}
			yTicker := &binance_usdtfuture.BookTicker{}
			yDepth := &binance_usdtfuture.Depth5{}

			var xTD, yTD common.Ticker
			var xFr, yFr common.FundingRate
			lastAddTime := time.Time{}
			matchedSpread := &common.MatchedSpread{}
			dayCounter := 0
			serverTime := int64(0)
			lastOutputTime := int64(0)
			outputIntervalNum := int64(outputInterval)

			//counter := 0
			outputFileName := fmt.Sprintf("/Users/chenjilin/Downloads/%s-%s-%s-%s-%v-%v-%v.gz", startDateStr, endDateStr, xSymbol, ySymbol, quantileLookback, spreadLookback, outputInterval)
			if _, err := os.Stat(outputFileName); err == nil {
				logger.Debugf("%s exists, ignore", outputFileName)
				return
			}
			outputTmpName := outputFileName + ".tmp"
			outputFile, err := os.Create(outputTmpName)
			if err != nil {
				logger.Debugf("os.Create error %v", err)
				return
			}
			outputWriter, err := gzip.NewWriterLevel(outputFile, gzip.BestCompression)
			if err != nil {
				logger.Debugf("gzip.NewWriterLevel error %v, stop ws", err)
				return
			}

			for _, dateStr := range strings.Split(dateStrs, ",") {
				logger.Debugf("%s %s %s", xSymbol, dateStr, fmt.Sprintf("/Users/chenjilin/MarketData/kcuf-bnuf-depth5-and-ticker/%s/%s-%s,%s.jl.gz", dateStr, dateStr, xSymbol, ySymbol))
				dayCounter++
				file, err := os.Open(
					fmt.Sprintf("/Users/chenjilin/MarketData/kcuf-bnuf-depth5-and-ticker/%s/%s-%s,%s.jl.gz", dateStr, dateStr, xSymbol, ySymbol),
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
					if msg[0] == 'K' && msg[1] == 'T' {
						err = kucoin_usdtfuture.ParseTicker(msg[21:], xTicker)
						if err != nil {
							logger.Debugf("%v", err)
							continue
						}
						if xTicker.Symbol != xSymbol {
							//logger.Debugf("bad msg: %s", msg)
							continue

						}
						serverTime, err = common.ParseInt(msg[2:21])
						if err != nil {
							logger.Debugf("%v", err)
							continue
						}
						xTD = xTicker
					} else if msg[0] == 'K' && msg[1] == 'D' {
						err = kucoin_usdtfuture.ParseDepth5(msg[21:], xDepth)
						if err != nil {
							logger.Debugf("%v", err)
							continue
						}
						if xDepth.Symbol != xSymbol {
							//logger.Debugf("bad msg: %s", msg)
							continue
						}
						serverTime, err = common.ParseInt(msg[2:21])
						if err != nil {
							logger.Debugf("%v", err)
							continue
						}
						xTD = xDepth
					} else if msg[0] == 'K' && msg[1] == 'F' {
						err = json.Unmarshal(msg[21:], xFundingRate)
						if err != nil {
							logger.Debugf("%v", err)
							continue
						}
						if !strings.Contains(xFundingRate.Symbol, xSymbol) {
							logger.Debugf("bad msg: %s %s %s", msg, xSymbol, xFundingRate.Symbol)
							continue
						}
						xFr = xFundingRate
					} else if msg[0] == 'B' && msg[1] == 'D' {
						err = binance_usdtfuture.ParseDepth5(msg[21:], yDepth)
						if err != nil {
							logger.Debugf("%v", err)
							continue
						}
						if yDepth.Symbol != ySymbol {
							//logger.Debugf("bad msg: %s", msg)
							continue
						}
						serverTime, err = common.ParseInt(msg[2:21])
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
						if yTicker.Symbol != ySymbol {
							//logger.Debugf("bad msg: %s", msg)
							continue
						}
						serverTime, err = common.ParseInt(msg[2:21])
						if err != nil {
							logger.Debugf("%v", err)
							continue
						}
						yTD = yTicker
						continue
					} else if msg[0] == 'B' && msg[1] == 'F' {
						err = json.Unmarshal(msg[21:], yFundingRate)
						if err != nil {
							logger.Debugf("%v", err)
							continue
						}
						if yFundingRate.Symbol != ySymbol {
							//logger.Debugf("bad msg: %s", msg)
							continue
						}
						yFr = yFundingRate
						continue
					} else {
						continue
					}

					if yTD != nil && xTD != nil {
						tDiff := xTD.GetTime().Sub(yTD.GetTime())
						if tDiff < maxTimeDiff &&
							tDiff > -maxTimeDiff {
							shortLastSpread = (yTD.GetBidPrice() - xTD.GetAskPrice()) / xTD.GetAskPrice()
							longLastSpread = (yTD.GetAskPrice() - xTD.GetBidPrice()) / xTD.GetBidPrice()
							if tDiff > 0 {
								longMedianSpread.Insert(xTD.GetTime(), longLastSpread)
								shortMedianSpread.Insert(xTD.GetTime(), shortLastSpread)
							} else {
								longMedianSpread.Insert(yTD.GetTime(), longLastSpread)
								shortMedianSpread.Insert(yTD.GetTime(), shortLastSpread)
							}
							if xTD.GetTime().Sub(lastAddTime) >= quantileAddInterval {
								lastAddTime = xTD.GetTime()
								if tDiff > 0 {
									_ = timedTDigest.Insert(xTD.GetTime(), (shortLastSpread+longLastSpread)*0.5)
								} else {
									_ = timedTDigest.Insert(yTD.GetTime(), (shortLastSpread+longLastSpread)*0.5)
								}
							}

							if xFr != nil && yFr != nil && dayCounter > 2 && serverTime-lastOutputTime >= outputIntervalNum {

								matchedSpread.ServerTime = serverTime
								lastOutputTime = serverTime
								if tDiff > 0 {
									matchedSpread.EventTime = xTD.GetTime().UnixNano()
								} else {
									matchedSpread.EventTime = yTD.GetTime().UnixNano()
								}

								matchedSpread.XBidPrice = xTD.GetBidPrice()
								matchedSpread.XBidSize = xTD.GetBidSize() * kucoin_usdtfuture.Multipliers[xSymbol]
								matchedSpread.XAskPrice = xTD.GetAskPrice()
								matchedSpread.XAskSize = xTD.GetAskSize() * kucoin_usdtfuture.Multipliers[xSymbol]

								matchedSpread.YBidPrice = yTD.GetBidPrice()
								matchedSpread.YBidSize = yTD.GetBidSize()
								matchedSpread.YAskPrice = yTD.GetAskPrice()
								matchedSpread.YAskSize = yTD.GetAskSize()

								matchedSpread.XFundingRate = xFr.GetFundingRate()
								matchedSpread.YFundingRate = yFr.GetFundingRate()

								matchedSpread.ShortLastSpread = shortLastSpread
								matchedSpread.ShortMedianSpread = shortMedianSpread.Median()
								matchedSpread.LongLastSpread = longLastSpread
								matchedSpread.LongMedianSpread = longMedianSpread.Median()

								matchedSpread.SpreadQuantile995 = timedTDigest.Quantile(0.995)
								matchedSpread.SpreadQuantile95 = timedTDigest.Quantile(0.95)
								matchedSpread.SpreadQuantile80 = timedTDigest.Quantile(0.80)
								matchedSpread.SpreadQuantile50 = timedTDigest.Quantile(0.50)
								matchedSpread.SpreadQuantile20 = timedTDigest.Quantile(0.20)
								matchedSpread.SpreadQuantile05 = timedTDigest.Quantile(0.05)
								matchedSpread.SpreadQuantile005 = timedTDigest.Quantile(0.005)

								//counter++
								//
								//if counter < 100 {
								//	logger.Debugf("%v", matchedSpread)
								//}

								err = binary.Write(outputWriter, binary.BigEndian, matchedSpread)
								if err != nil {
									logger.Debugf("binary.Write %v", err)
								}
							}
						}
					}

				}
				_ = gr.Close()
				_ = file.Close()
			}

			err = outputWriter.Close()
			if err != nil {
				logger.Debugf("outputWriter.Close error %v", err)
			}

			err = outputFile.Close()
			if err != nil {
				logger.Debugf("outputFile.Close error %v", err)
			}

			err = os.Rename(outputTmpName, outputFileName)
			if err != nil {
				logger.Debugf("os.Rename error %v", err)
			}
		}(xSymbol, ySymbol)

	}

	doneCounter := 0
	for doneCounter < len(symbols) {
		select {
		case <-doneSymbolCh:
			doneCounter++
		}
	}
}
