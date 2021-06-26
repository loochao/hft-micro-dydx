package main

import (
	"bufio"
	"compress/gzip"
	"fmt"
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/geometrybase/hft-micro/tdigest"
	"os"
	"strings"
	"time"
)

func main() {

	symbols := strings.Split(
		`1INCHUSDT,AAVEUSDT,ADAUSDT,AKROUSDT,ALGOUSDT,ALICEUSDT,ALPHAUSDT,ANKRUSDT,ATOMUSDT,AVAXUSDT,AXSUSDT,BAKEUSDT,BALUSDT,BANDUSDT,BATUSDT,BCHUSDT,BELUSDT,BLZUSDT,BNBUSDT,BTCUSDT,BTSUSDT,BTTUSDT,BZRXUSDT,CELRUSDT,CHRUSDT,CHZUSDT,COMPUSDT,COTIUSDT,CRVUSDT,CTKUSDT,CVCUSDT,DASHUSDT,DENTUSDT,DGBUSDT,DODOUSDT,DOGEUSDT,DOTUSDT,EGLDUSDT,ENJUSDT,EOSUSDT,ETCUSDT,ETHUSDT,FILUSDT,FLMUSDT,FTMUSDT,GRTUSDT,HBARUSDT,HNTUSDT,HOTUSDT,ICPUSDT,ICXUSDT,IOSTUSDT,IOTAUSDT,KAVAUSDT,KNCUSDT,KSMUSDT,LINAUSDT,LINKUSDT,LITUSDT,LRCUSDT,LTCUSDT,LUNAUSDT,MANAUSDT,MATICUSDT,MKRUSDT,MTLUSDT,NEARUSDT,NEOUSDT,NKNUSDT,OCEANUSDT,OGNUSDT,OMGUSDT,ONEUSDT,ONTUSDT,QTUMUSDT,REEFUSDT,RENUSDT,RLCUSDT,RSRUSDT,RUNEUSDT,RVNUSDT,SANDUSDT,SCUSDT,SFPUSDT,SKLUSDT,SNXUSDT,SOLUSDT,SRMUSDT,STMXUSDT,STORJUSDT,SUSHIUSDT,SXPUSDT,THETAUSDT,TOMOUSDT,TRBUSDT,TRXUSDT,UNFIUSDT,UNIUSDT,VETUSDT,WAVESUSDT,XEMUSDT,XLMUSDT,XMRUSDT,XRPUSDT,XTZUSDT,YFIIUSDT,YFIUSDT,ZECUSDT,ZENUSDT,ZILUSDT,ZRXUSDT`,
		",",
	)
	startTime, err := time.Parse("20060102", "20210622")
	if err != nil {
		logger.Fatal(err)
	}
	endTime, err := time.Parse("20060102", "20210625")
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
	for _, symbol := range symbols {
		//var lastBuyTrade *bnspot.Trade
		//var lastSellTrade *bnspot.Trade
		//var lastTrade *bnspot.Trade
		impactTD, _ := tdigest.New()
		bookTD, _ := tdigest.New()
		for _, dateStr := range strings.Split(dateStrs, ",") {
			file, err := os.Open(
				fmt.Sprintf("/Users/chenjilin/MarketData/bnspot-bnswap-depth5/%s/%s-%s.depth5.jl.gz", dateStr, dateStr, symbol),
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
			var depth5 = binance_usdtfuture.Depth5{}
			var lastDepth5 = binance_usdtfuture.Depth5{}
			//counter := 0
			for scanner.Scan() {
				msg = scanner.Bytes()
				if msg[0] != 'F' {
					continue
				}
				//counter++
				//if counter%2 != 0 {
				//	continue
				//}
				err = binance_usdtfuture.ParseDepth5(msg[1:], &depth5)
				if err != nil {
					//logger.Debugf("binance_usdtfuture.ParseDepth5 error %v", err)
					continue
				}
				if lastDepth5.Symbol != "" {
					if lastDepth5.Bids[0][0] >= depth5.Bids[0][0] {
						_ = impactTD.Add((depth5.Bids[0][0] - lastDepth5.Bids[0][0]) / lastDepth5.Bids[0][0] )
					}
					if lastDepth5.Asks[0][0] <= depth5.Asks[0][0] {
						_ = impactTD.Add((depth5.Asks[0][0] - lastDepth5.Asks[0][0]) / lastDepth5.Asks[0][0] )
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
		quantiles[symbol] = fmt.Sprintf(
			"%.6f,%.6f,%.6f,%.6f,%.6f,%.6f",
			impactTD.Quantile(0.0005),
			impactTD.Quantile(0.005),
			impactTD.Quantile(0.05),
			impactTD.Quantile(0.95),
			impactTD.Quantile(0.995),
			impactTD.Quantile(0.9995),
		)
		maxOrderValues[symbol] = fmt.Sprintf(
			"%.2f",
			bookTD.Quantile(0.8)*0.1,
		)
		fmt.Printf("%s %s\n", symbol, quantiles[symbol])
		fmt.Printf("%s %s\n", symbol, maxOrderValues[symbol])
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

	fmt.Printf("\n\n\nvar maxOrderValues = map[string]float64{")
	for _, symbol := range symbols {
		fmt.Printf(
			"\"%s\":\t%s,\n",
			symbol,
			maxOrderValues[symbol],
		)
	}
	fmt.Printf("}\n\n\n")
}
