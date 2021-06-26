package main

import (
	"bufio"
	"compress/gzip"
	"fmt"
	binance_busdspot "github.com/geometrybase/hft-micro/binance-busdspot"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/geometrybase/hft-micro/tdigest"
	"os"
	"strings"
	"time"
)

func main() {

	symbols := strings.Split(
		`1INCHBUSD,AAVEBUSD,ADABUSD,ALGOBUSD,ALICEBUSD,ALPHABUSD,ATOMBUSD,AVAXBUSD,AXSBUSD,BAKEBUSD,BALBUSD,BANDBUSD,BATBUSD,BCHBUSD,BELBUSD,BNBBUSD,BTCBUSD,BTTBUSD,BZRXBUSD,CELRBUSD,CHRBUSD,CHZBUSD,COMPBUSD,COTIBUSD,CRVBUSD,CTKBUSD,DASHBUSD,DGBBUSD,DODOBUSD,DOGEBUSD,DOTBUSD,EGLDBUSD,ENJBUSD,EOSBUSD,ETCBUSD,ETHBUSD,FILBUSD,FLMBUSD,FTMBUSD,GRTBUSD,GTCBUSD,HBARBUSD,HNTBUSD,HOTBUSD,ICPBUSD,ICXBUSD,IOSTBUSD,IOTABUSD,KAVABUSD,KEEPBUSD,KNCBUSD,KSMBUSD,LINABUSD,LINKBUSD,LITBUSD,LRCBUSD,LTCBUSD,LUNABUSD,MANABUSD,MATICBUSD,MKRBUSD,NEARBUSD,NEOBUSD,OCEANBUSD,OMGBUSD,ONEBUSD,ONTBUSD,QTUMBUSD,REEFBUSD,RLCBUSD,RSRBUSD,RUNEBUSD,RVNBUSD,SANDBUSD,SCBUSD,SFPBUSD,SKLBUSD,SNXBUSD,SOLBUSD,SRMBUSD,STMXBUSD,SUSHIBUSD,SXPBUSD,THETABUSD,TOMOBUSD,TRBBUSD,TRXBUSD,UNFIBUSD,UNIBUSD,VETBUSD,WAVESBUSD,XEMBUSD,XLMBUSD,XMRBUSD,XRPBUSD,XTZBUSD,YFIBUSD,YFIIBUSD,ZECBUSD,ZENBUSD,ZILBUSD,ZRXBUSD`,
		",",
	)
	logger.Debugf("%d", len(symbols))
	startTime, err := time.Parse("20060102", "20210623")
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
	maxOrderSizes := make(map[string]string)
	for _, symbol := range symbols {
		impactTD, _ := tdigest.New()
		bookTD, _ := tdigest.New()
		for _, dateStr := range strings.Split(dateStrs, ",") {
			file, err := os.Open(
				fmt.Sprintf("/Users/chenjilin/MarketData/bnbs-depth5/%s/%s-%s.depth5.jl.gz", dateStr, dateStr, symbol),
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
			var depth5 = binance_busdspot.Depth5{}
			var lastDepth5 = binance_busdspot.Depth5{}
			var counter = 0
			var step = 1
			for scanner.Scan() {
				counter++
				msg = scanner.Bytes()
				if counter%step != 0 {
					continue
				}
				err = binance_busdspot.ParseDepth5(msg, &depth5)
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
		maxOrderSizes[symbol] = fmt.Sprintf(
			"%.0f",
			bookTD.Quantile(0.8)*0.1,
		)
		//fmt.Printf("%s %s\n", symbol, quantiles[symbol])
		fmt.Printf("%s %s\n", symbol, maxOrderSizes[symbol])
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
