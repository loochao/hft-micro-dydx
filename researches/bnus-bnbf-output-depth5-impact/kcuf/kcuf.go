package main

import (
	"bufio"
	"compress/gzip"
	"fmt"
	kucoin_usdtfuture "github.com/geometrybase/hft-micro/kucoin-usdtfuture"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/geometrybase/hft-micro/tdigest"
	"os"
	"sort"
	"strings"
	"time"
)

func main() {

	pairs := map[string]string{
		"XBTUSDTM":   "BTCUSDT",
		"UNIUSDTM":   "UNIUSDT",
		"DGBUSDTM":   "DGBUSDT",
		"IOSTUSDTM":  "IOSTUSDT",
		"RVNUSDTM":   "RVNUSDT",
		"THETAUSDTM": "THETAUSDT",
		"WAVESUSDTM": "WAVESUSDT",
		"DENTUSDTM":  "DENTUSDT",
		"DOTUSDTM":   "DOTUSDT",
		"XMRUSDTM":   "XMRUSDT",
		"FILUSDTM":   "FILUSDT",
		"ICPUSDTM":   "ICPUSDT",
		"MANAUSDTM":  "MANAUSDT",
		"MATICUSDTM": "MATICUSDT",
		"ALGOUSDTM":  "ALGOUSDT",
		"KSMUSDTM":   "KSMUSDT",
		"LUNAUSDTM":  "LUNAUSDT",
		"DASHUSDTM":  "DASHUSDT",
		"LTCUSDTM":   "LTCUSDT",
		"CHZUSDTM":   "CHZUSDT",
		"MKRUSDTM":   "MKRUSDT",
		"ADAUSDTM":   "ADAUSDT",
		"BCHUSDTM":   "BCHUSDT",
		"COMPUSDTM":  "COMPUSDT",
		"FTMUSDTM":   "FTMUSDT",
		"NEOUSDTM":   "NEOUSDT",
		"SXPUSDTM":   "SXPUSDT",
		"XRPUSDTM":   "XRPUSDT",
		"BNBUSDTM":   "BNBUSDT",
		"ETHUSDTM":   "ETHUSDT",
		"LINKUSDTM":  "LINKUSDT",
		"GRTUSDTM":   "GRTUSDT",
		"YFIUSDTM":   "YFIUSDT",
		"AAVEUSDTM":  "AAVEUSDT",
		"AVAXUSDTM":  "AVAXUSDT",
		"ETCUSDTM":   "ETCUSDT",
		"QTUMUSDTM":  "QTUMUSDT",
		"XLMUSDTM":   "XLMUSDT",
		"ZECUSDTM":   "ZECUSDT",
		"BTTUSDTM":   "BTTUSDT",
		"ENJUSDTM":   "ENJUSDT",
		"ONTUSDTM":   "ONTUSDT",
		"SUSHIUSDTM": "SUSHIUSDT",
		"XEMUSDTM":   "XEMUSDT",
		"DOGEUSDTM":  "DOGEUSDT",
		"OCEANUSDTM": "OCEANUSDT",
		"BATUSDTM":   "BATUSDT",
		"CRVUSDTM":   "CRVUSDT",
		"EOSUSDTM":   "EOSUSDT",
		"SNXUSDTM":   "SNXUSDT",
		"ATOMUSDTM":  "ATOMUSDT",
		"BANDUSDTM":  "BANDUSDT",
		"XTZUSDTM":   "XTZUSDT",
		"1INCHUSDTM": "1INCHUSDT",
		"TRXUSDTM":   "TRXUSDT",
		"SOLUSDTM":   "SOLUSDT",
		"VETUSDTM":   "VETUSDT",
	}
	symbols := make([]string, 0)
	for bSymbol := range pairs {
		symbols = append(symbols, bSymbol)
	}
	sort.Strings(symbols)
	startTime, err := time.Parse("20060102", "20210622")
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
	maxOrderValues := make(map[string]string)
	for _, kSymbol := range symbols {
		bSymbol := pairs[kSymbol]
		//var lastBuyTrade *bnspot.Trade
		//var lastSellTrade *bnspot.Trade
		//var lastTrade *bnspot.Trade
		impactTD, _ := tdigest.New()
		bookTD, _ := tdigest.New()
		for _, dateStr := range strings.Split(dateStrs, ",") {
			file, err := os.Open(
				fmt.Sprintf("/home/clu/MarketData/bnuf-kcuf-depth5/%s/%s-%s,%s.depth5.jl.gz", dateStr, dateStr, bSymbol, kSymbol),
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
			var depth5 = kucoin_usdtfuture.Depth5{}
			var lastDepth5 = kucoin_usdtfuture.Depth5{}
			//counter := 0
			for scanner.Scan() {
				msg = scanner.Bytes()
				if msg[0] != 'K' {
					continue
				}
				//counter++
				//if counter%2 != 0 {
				//	continue
				//}
				err = kucoin_usdtfuture.ParseDepth5(msg[1:], &depth5)
				if err != nil {
					//logger.Debugf("binance_usdtfuture.ParseDepth5 error %v", err)
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
						bookSize += depth5.Bids[i][0] * depth5.Bids[i][1]
						bookSize += depth5.Asks[i][0] * depth5.Asks[i][1]
					}
					_ = bookTD.Add(bookSize)
				}
				lastDepth5 = depth5
			}
			_ = gr.Close()
			_ = file.Close()
		}
		quantiles[kSymbol] = fmt.Sprintf(
			"%.6f,%.6f,%.6f,%.6f,%.6f,%.6f",
			impactTD.Quantile(0.00005),
			impactTD.Quantile(0.005),
			impactTD.Quantile(0.05),
			impactTD.Quantile(0.95),
			impactTD.Quantile(0.995),
			impactTD.Quantile(0.99995),
		)
		maxOrderValues[kSymbol] = fmt.Sprintf(
			"%.0f",
			bookTD.Quantile(0.8)*0.1,
		)
		fmt.Printf("%s %s\n", kSymbol, quantiles[kSymbol])
		fmt.Printf("%s %s\n", kSymbol, maxOrderValues[kSymbol])
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

	fmt.Printf("\n\n\nmaxOrderValues:")
	for _, symbol := range symbols {
		fmt.Printf(
			"%s:\t%s\n",
			symbol,
			maxOrderValues[symbol],
		)
	}
	fmt.Printf("\n\n\n")

	fmt.Printf("\n\n\nvar maxOrderValues = map[string]float64{\n")
	for _, symbol := range symbols {
		fmt.Printf(
			"\"%s\":\t%s,\n",
			symbol,
			maxOrderValues[symbol],
		)
	}
	fmt.Printf("}\n\n\n")
}
