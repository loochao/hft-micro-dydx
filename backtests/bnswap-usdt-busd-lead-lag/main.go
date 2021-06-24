package main

import (
	"bufio"
	"compress/gzip"
	"fmt"
	binance_busdfuture "github.com/geometrybase/hft-micro/binance-busdfuture"
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/geometrybase/hft-micro/tdigest"
	"os"
	"strings"
	"time"
)

func main() {

	busdSymbol := "BTCBUSD"
	usdtSymbol := "BTCUSDT"
	startTime, err := time.Parse("20060102", "20210622")
	if err != nil {
		logger.Fatal(err)
	}
	endTime, err := time.Parse("20060102", "20210622")
	if err != nil {
		logger.Fatal(err)
	}
	dateStrs := ""
	for i := startTime; i.Sub(endTime) <= 0; i = i.Add(time.Hour * 24) {
		dateStrs += i.Format("20060102,")
	}
	dateStrs = dateStrs[:len(dateStrs)-1]

	var lastBusdDepth, busdDepth *binance_busdfuture.Depth5
	var lastUsdtDepth, usdtDepth *binance_usdtfuture.Depth5
	usdtDepth = &binance_usdtfuture.Depth5{}
	busdDepth = &binance_busdfuture.Depth5{}
	longTD, _ := tdigest.New()
	shortTD, _ := tdigest.New()
	for _, dateStr := range strings.Split(dateStrs, ",") {
		//file, err := os.Open(
		//	fmt.Sprintf("/Users/chenjilin/MarketData/bnswap-depth5-leadlag/%s/%s-%s,%s.depth5.jl.gz", dateStr, dateStr, usdtSymbol, busdSymbol),
		//)
		file, err := os.Open(
			fmt.Sprintf("/Users/chenjilin/MarketData/bnswap-depth5-leadlag/%s-%s,%s.depth5.jl.gz", dateStr,  usdtSymbol, busdSymbol),
		)
		logger.Debugf("%s", dateStr)
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
		counter := 0
		for scanner.Scan() {
			msg := scanner.Bytes()
			if msg[14] == 'b' {
				err = binance_busdfuture.ParseDepth5(msg, busdDepth)
				if err == nil {
					if lastUsdtDepth != nil {
						_ = longTD.Add((lastUsdtDepth.Asks[0][0] - busdDepth.Bids[0][0])/busdDepth.Bids[0][0])
						_ = shortTD.Add((lastUsdtDepth.Bids[0][0] - busdDepth.Asks[0][0])/busdDepth.Bids[0][0])
					}
					lastBusdDepth = busdDepth
				}
			}else if msg[14] == 'u' {
				err = binance_usdtfuture.ParseDepth5(msg, usdtDepth)
				if err == nil {
					if lastBusdDepth != nil {
						_ = longTD.Add((usdtDepth.Asks[0][0] - lastBusdDepth.Bids[0][0])/lastBusdDepth.Bids[0][0])
						_ = shortTD.Add((usdtDepth.Bids[0][0] - lastBusdDepth.Asks[0][0])/lastBusdDepth.Bids[0][0])
					}
					lastUsdtDepth = usdtDepth
				}
			}
			counter++
		}
		_ = gr.Close()
		_ = file.Close()
	}

	fmt.Printf(
		"longTD: \t%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f\n",
		longTD.Quantile(0.0005),
		longTD.Quantile(0.005),
		longTD.Quantile(0.05),
		longTD.Quantile(0.2),
		longTD.Quantile(0.5),
		longTD.Quantile(0.8),
		longTD.Quantile(0.95),
		longTD.Quantile(0.995),
		longTD.Quantile(0.9995),
	)
	fmt.Printf(
		"shortTD: \t%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f\n",
		shortTD.Quantile(0.0005),
		shortTD.Quantile(0.005),
		shortTD.Quantile(0.05),
		shortTD.Quantile(0.2),
		shortTD.Quantile(0.5),
		shortTD.Quantile(0.8),
		shortTD.Quantile(0.95),
		shortTD.Quantile(0.995),
		shortTD.Quantile(0.9995),
	)
}
