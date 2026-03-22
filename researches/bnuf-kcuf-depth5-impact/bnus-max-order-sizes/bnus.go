package main

import (
	"bufio"
	"compress/gzip"
	"fmt"
	binance_usdtspot "github.com/geometrybase/hft-micro/binance-usdtspot"
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
	endTime, err := time.Parse("20060102", "20210622")
	if err != nil {
		logger.Fatal(err)
	}
	dateStrs := ""
	for i := startTime; i.Sub(endTime) <= 0; i = i.Add(time.Hour * 24) {
		dateStrs += i.Format("20060102,")
	}
	dateStrs = dateStrs[:len(dateStrs)-1]

	maxOrderSizes := make(map[string]string)
	for _, symbol := range symbols {
		booksizeTD, _ := tdigest.New()
		for _, dateStr := range strings.Split(dateStrs, ",") {
			file, err := os.Open(
				fmt.Sprintf("/home/clu/MarketData/bnspot-bnswap-depth5/%s/%s-%s.depth5.jl.gz", dateStr, dateStr, symbol),
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
			var depth5 = binance_usdtspot.Depth5{}
			var lastDepth5 = binance_usdtspot.Depth5{}
			var counter = 0
			for scanner.Scan() {
				counter++
				msg = scanner.Bytes()
				if msg[0] != 'S' {
					continue
				}
				err = binance_usdtspot.ParseDepth5(msg[1:], &depth5)
				if err != nil {
					logger.Debugf("binance_usdtspot.ParseDepth5 error %v", err)
					continue
				}
				if lastDepth5.Symbol != "" {
					bookSize := 0.0
					for i := 0; i < 5; i++ {
						bookSize += depth5.Bids[i][0]*depth5.Bids[i][1]
						bookSize += depth5.Asks[i][0]*depth5.Asks[i][1]
					}
					_ = booksizeTD.Add(bookSize)
				}
				lastDepth5 = depth5
			}
			_ = gr.Close()
			_ = file.Close()
		}
		maxOrderSizes[symbol] = fmt.Sprintf(
			"%.0f",
			booksizeTD.Quantile(0.8)*0.05,
		)
		fmt.Printf("%s %s\n", symbol, maxOrderSizes[symbol])
	}

	fmt.Printf("\n\n\n")
	for _, symbol := range symbols {
		fmt.Printf(
			"%s:\t%s\n",
			symbol,
			maxOrderSizes[symbol],
		)
	}
	fmt.Printf("\n\n\n")

	fmt.Printf("\n\n\n var maxOrderSizes = map[string]float{\n")
	for _, symbol := range symbols {
		fmt.Printf(
			"\"%s\":\t%s,\n",
			symbol,
			maxOrderSizes[symbol],
		)
	}
	fmt.Printf("}\n\n\n")

}
