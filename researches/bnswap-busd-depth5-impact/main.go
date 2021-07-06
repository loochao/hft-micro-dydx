package main

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"github.com/geometrybase/hft-micro/logger"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

func main() {

	busdSymbol := "BTCBUSD"
	usdtSymbol := "BTCUSDT"
	startTime, err := time.Parse("20060102", "20210621")
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

	for _, dateStr := range strings.Split(dateStrs, ",") {
		file, err := os.Open(
			fmt.Sprintf("/Users/chenjilin/MarketData/bnswap-depth5-leadlag/%s/%s-%s,%s.depth5.jl.gz", dateStr, dateStr, usdtSymbol, busdSymbol),
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
		content, err := ioutil.ReadAll(gr)
		if err != nil {
			logger.Debugf("ioutil.ReadAll(gr) error %v", err)
			continue
		}
		logger.Debugf("%s", content[:1000])
		return
		scanner := bufio.NewScanner(gr)
		counter := 0
		logger.Debugf("%d", counter)
		logger.Debugf("%v", scanner.Scan())
		for scanner.Scan() {
			counter++
			logger.Debugf("%d", counter)
			logger.Debugf("%s %s", dateStr, scanner.Bytes())
			if counter > 100 {
				break
			}
			//d, err := bnspot.ParseTrade(scanner.Bytes())
			//if err != nil {
			//	//logger.Debugf("bnspot.ParseDepth20 error %v", err)
			//	continue
			//}
		}
		_ = gr.Close()
		_ = file.Close()
	}
	//quantiles[symbol] = fmt.Sprintf(
	//	"%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f",
	//	sellImpactTD.Quantile(0.0005),
	//	sellImpactTD.Quantile(0.005),
	//	sellImpactTD.Quantile(0.05),
	//	sellImpactTD.Quantile(0.2),
	//	sellImpactTD.Quantile(0.5),
	//	buyImpactTD.Quantile(0.5),
	//	buyImpactTD.Quantile(0.8),
	//	buyImpactTD.Quantile(0.95),
	//	buyImpactTD.Quantile(0.995),
	//	buyImpactTD.Quantile(0.9995),
	//)
	//fmt.Printf(
	//	"%s:\t%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f\n",
	//	symbol,
	//	sellImpactTD.Quantile(0.0005),
	//	sellImpactTD.Quantile(0.005),
	//	sellImpactTD.Quantile(0.05),
	//	sellImpactTD.Quantile(0.2),
	//	sellImpactTD.Quantile(0.5),
	//	buyImpactTD.Quantile(0.5),
	//	buyImpactTD.Quantile(0.8),
	//	buyImpactTD.Quantile(0.95),
	//	buyImpactTD.Quantile(0.995),
	//	buyImpactTD.Quantile(0.9995),
	//)
	//tradeSizes[symbol] = tradeSizeTD.Quantile(0.8)
}
