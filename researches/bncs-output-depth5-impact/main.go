package main

import (
	"bufio"
	"compress/gzip"
	"fmt"
	binance_usdcspot "github.com/geometrybase/hft-micro/binance-usdcspot"
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/geometrybase/hft-micro/tdigest"
	"math"
	"os"
	"sort"
	"strings"
	"time"
)

func main() {

	symbols := make([]string, 0)
	for symbol := range binance_usdcspot.TickSizes {
		symbols = append(symbols, symbol)
	}
	sort.Strings(symbols)
	logger.Debugf("%d %v", len(symbols), symbols)
	startTime, err := time.Parse("20060102", "20210626")
	if err != nil {
		logger.Fatal(err)
	}
	endTime, err := time.Parse("20060102", "20210626")
	if err != nil {
		logger.Fatal(err)
	}
	dateStrs := ""
	for i := startTime; i.Sub(endTime) <= 0; i = i.Add(time.Hour * 24) {
		dateStrs += i.Format("20060102,")
	}
	dateStrs = dateStrs[:len(dateStrs)-1]

	quantiles := make(map[string]string)
	maxOrderValues := make(map[string]float64)
	for _, symbol := range symbols {
		impactTD, _ := tdigest.New()
		bookTD, _ := tdigest.New()
		for _, dateStr := range strings.Split(dateStrs, ",") {
			file, err := os.Open(
				fmt.Sprintf("/Users/chenjilin/MarketData/bncs-depth5/%s/%s-%s.depth5.jl.gz", dateStr, dateStr, symbol),
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
			var depth5 = binance_usdcspot.Depth5{}
			var lastDepth5 = binance_usdcspot.Depth5{}
			var counter = 0
			var step = 1
			for scanner.Scan() {
				counter++
				msg = scanner.Bytes()
				if counter%step != 0 {
					continue
				}
				err = binance_usdcspot.ParseDepth5(msg, &depth5)
				if err != nil {
					logger.Debugf("binance_busdspot.ParseDepth5 error %v", err)
					continue
				}
				if lastDepth5.Symbol != "" {
					if lastDepth5.Bids[0][0] >= depth5.Bids[0][0] {
						_ = impactTD.Add((depth5.Bids[0][0] - lastDepth5.Bids[0][0]) / lastDepth5.Bids[0][0])
					}
					if lastDepth5.Asks[0][0] <= depth5.Asks[0][0] {
						_ = impactTD.Add((depth5.Asks[0][0] - lastDepth5.Asks[0][0]) / lastDepth5.Asks[0][0])
					}
					bookSize := 0.0
					for i := 0; i < 5; i++ {
						bookSize += depth5.Bids[i][0]*depth5.Bids[i][1]
						bookSize += depth5.Asks[i][0]*depth5.Asks[i][1]
					}
					_ = bookTD.Add(bookSize)
				}
				lastDepth5 = depth5
			}
			_ = gr.Close()
			_ = file.Close()
		}
		quantiles[symbol] = fmt.Sprintf(
			"%.6f,%.6f,%.6f,%.6f,%.6f,%.6f",
			impactTD.Quantile(0.00005),
			impactTD.Quantile(0.005),
			impactTD.Quantile(0.05),
			impactTD.Quantile(0.95),
			impactTD.Quantile(0.995),
			impactTD.Quantile(0.99995),
		)
		maxOrderValues[symbol] =bookTD.Quantile(0.8)*0.1
	}

	fmt.Printf("\n\n\n")
	for _, symbol := range symbols {
		fmt.Printf(
			"%s:\t%s\n",
			symbol,
			quantiles[symbol],
		)
	}
	fmt.Printf("\n\n\n")

	fmt.Printf("\n\n\n")
	for _, symbol := range symbols {
		fmt.Printf(
			"%s:\t%.0f\n",
			symbol,
			maxOrderValues[symbol],
		)
	}
	fmt.Printf("\n\n\n")


	weights := make(map[string]float64)
	totalSize := 0.0
	for _, size := range maxOrderValues {
		totalSize += size
	}
	meanSize := totalSize/float64(len(maxOrderValues))
	for symbol, size := range maxOrderValues {
		weights[symbol] = math.Sqrt(size/meanSize)
		if weights[symbol] > 1 {
			weights[symbol] = 1.0
		}
	}
	for _, symbol := range symbols {
		fmt.Printf("%s: %.2f\n", symbol, weights[symbol])
	}
	sort.Strings(symbols)
	for _, symbol := range symbols {
		if _, ok := binance_usdtfuture.TickSizes[strings.Replace(symbol, "USDC", "USDT", -1)]; ok {
			fmt.Printf("  %s: %s\n", symbol, strings.Replace(symbol, "BUSD", "USDT", -1))
		}
	}
}
