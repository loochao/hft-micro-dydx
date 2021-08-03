package main

import (
	"bufio"
	"compress/gzip"
	"fmt"
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
	binance_usdtspot "github.com/geometrybase/hft-micro/binance-usdtspot"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/geometrybase/hft-micro/tdigest"
	"io/ioutil"
	"math"
	"os"
	"sort"
	"strings"
	"time"
)

func main() {
	symbols := make([]string, 0)
	for symbol := range binance_usdtspot.TickSizes {
		if _, ok := binance_usdtfuture.TickSizes[strings.Replace(symbol, "USDT", "USDT", -1)]; ok {
			symbols = append(symbols, symbol)
		}
	}
	sort.Strings(symbols)

	//symbols = symbols[:1]
	startTime, err := time.Parse("20060102", "20210716")
	if err != nil {
		logger.Fatal(err)
	}
	endTime, err := time.Parse("20060102", "20210801")
	if err != nil {
		logger.Fatal(err)
	}
	dateStrs := ""
	for i := startTime; i.Sub(endTime) <= 0; i = i.Add(time.Hour * 24) {
		dateStrs += i.Format("20060102,")
	}
	dateStrs = dateStrs[:len(dateStrs)-1]
	quantileAddInterval := time.Millisecond * 100
	quantile := 0.8

	dataPath := "/Volumes/MarketData/MarketData/bnus-bnuf-depth5-and-ticker"
	quantilePath := "/Users/chenjilin/Projects/hft-micro/applications/usd-ll-mt-q/bnus-bnuf-quantiles/outputs/size-quantiles"
	sizeTDs := make(map[string]*tdigest.TDigest)

	for _, xSymbol := range symbols {
		ySymbol := strings.Replace(xSymbol, "USDT", "USDT", -1)
		sizeTD, _ := tdigest.New()
		lastQuantileAddTime := time.Time{}

		data, err := ioutil.ReadFile(quantilePath + "/" + xSymbol + "," + ySymbol)
		if err == nil {
			err = sizeTD.FromBytes(data)
			if err != nil {
				logger.Debugf("sizeTD.FromBytes error %v", err)
			}
		} else {
			yDepth := &binance_usdtfuture.Depth5{}
			yTicker := &binance_usdtfuture.BookTicker{}
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
					msg = scanner.Bytes()
					if len(msg) < 128 {
						continue
					}
					if msg[0] == 'F' && msg[1] == 'D' {
						err = binance_usdtfuture.ParseDepth5(msg[21:], yDepth)
						if err != nil {
							logger.Debugf("%v", err)
							continue
						}
						if yDepth.Symbol != ySymbol {
							continue
						}
						if yDepth.EventTime.Sub(lastQuantileAddTime) > quantileAddInterval {
							lastQuantileAddTime = yDepth.EventTime
							_ = sizeTD.Add(0.5 * (yDepth.GetBidPrice()*yDepth.GetBidSize() + yDepth.GetAskPrice()*yDepth.GetAskSize()))
						}
					} else if msg[0] == 'F' && msg[1] == 'T' {
						err = binance_usdtfuture.ParseBookTicker(msg[21:], yTicker)
						if err != nil {
							logger.Debugf("%v", err)
							continue
						}
						if yTicker.Symbol != ySymbol {
							continue
						}
						if yTicker.EventTime.Sub(lastQuantileAddTime) > quantileAddInterval {
							lastQuantileAddTime = yTicker.EventTime
							_ = sizeTD.Add(0.5 * (yTicker.GetBidPrice()*yTicker.GetBidSize() + yTicker.GetAskPrice()*yTicker.GetAskSize()))
						}
					} else {
						continue
					}
				}
				_ = gr.Close()
				_ = file.Close()
			}
		}
		sizeTDs[xSymbol] = sizeTD
		data, err = sizeTD.AsBytes()
		if err != nil {
			logger.Debugf("sizeTD.AsBytes() error %v", err)
		} else {
			err := ioutil.WriteFile(quantilePath+"/"+xSymbol+","+ySymbol, data, 0775)
			if err != nil {
				logger.Debugf("ioutil.WriteFile error %v", err)
			}
		}
		fmt.Printf("\n\n  %s: %.0f\n\n", xSymbol, sizeTD.Quantile(quantile))
	}
	fmt.Printf("\n\nmaxOrderValues:\n")
	sumValue := 0.0
	for _, xSymbol := range symbols {
		td := sizeTDs[xSymbol]
		sumValue += td.Quantile(quantile)
		fmt.Printf("  %s: %.0f\n", xSymbol, td.Quantile(quantile))
	}
	meanValue := sumValue / float64(len(sizeTDs))
	fmt.Printf("\n\ntargetWeights:\n")
	for _, xSymbol := range symbols {
		td := sizeTDs[xSymbol]
		weight := math.Sqrt(td.Quantile(quantile)  / meanValue)
		if weight > 1 {
			weight = 1
		}
		fmt.Printf("  %s: %.4f\n", xSymbol, weight)
	}
}
