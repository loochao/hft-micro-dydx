package main

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/geometrybase/hft-micro/tdigest"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

func main() {

	symbols := strings.Split(
		`1INCHUSDT,AAVEUSDT,ADAUSDT,AKROUSDT,ALGOUSDT,ALICEUSDT,ALPHAUSDT,ANKRUSDT,ATOMUSDT,AVAXUSDT,AXSUSDT,BAKEUSDT,BALUSDT,BANDUSDT,BATUSDT,BCHUSDT,BELUSDT,BLZUSDT,BNBUSDT,BTCBUSD,BTCUSDT,BTSUSDT,BTTUSDT,BZRXUSDT,CELRUSDT,CHRUSDT,CHZUSDT,COMPUSDT,COTIUSDT,CRVUSDT,CTKUSDT,CVCUSDT,DASHUSDT,DENTUSDT,DGBUSDT,DODOUSDT,DOGEUSDT,DOTUSDT,EGLDUSDT,ENJUSDT,EOSUSDT,ETCUSDT,ETHUSDT,FILUSDT,FLMUSDT,FTMUSDT,GRTUSDT,HBARUSDT,HNTUSDT,HOTUSDT,ICPUSDT,ICXUSDT,IOSTUSDT,IOTAUSDT,KAVAUSDT,KNCUSDT,KSMUSDT,LINAUSDT,LINKUSDT,LITUSDT,LRCUSDT,LTCUSDT,LUNAUSDT,MANAUSDT,MATICUSDT,MKRUSDT,MTLUSDT,NEARUSDT,NEOUSDT,NKNUSDT,OCEANUSDT,OGNUSDT,OMGUSDT,ONEUSDT,ONTUSDT,QTUMUSDT,REEFUSDT,RENUSDT,RLCUSDT,RSRUSDT,RUNEUSDT,RVNUSDT,SANDUSDT,SCUSDT,SFPUSDT,SKLUSDT,SNXUSDT,SOLUSDT,SRMUSDT,STMXUSDT,STORJUSDT,SUSHIUSDT,SXPUSDT,THETAUSDT,TOMOUSDT,TRBUSDT,TRXUSDT,UNFIUSDT,UNIUSDT,VETUSDT,WAVESUSDT,XEMUSDT,XLMUSDT,XMRUSDT,XRPUSDT,XTZUSDT,YFIIUSDT,YFIUSDT,ZECUSDT,ZENUSDT,ZILUSDT,ZRXUSDT`,
		",",
	)[:1]
	startTime, err := time.Parse("20060102", "20210621")
	if err != nil {
		logger.Fatal(err)
	}
	endTime, err := time.Parse("20060102", "20210622")
	if err != nil {
		logger.Fatal(err)
	}
	dateStrs := ""
	for i := startTime; i.Sub(endTime) <= 0; i = i.Add(time.Hour*24) {
		dateStrs += i.Format("20060102,")
	}
	dateStrs = dateStrs[:len(dateStrs)-1]

	quantiles := make(map[string]string)
	tradeSizes := make(map[string]float64)

	for _, symbol := range symbols {
		//var lastBuyTrade *bnspot.Trade
		//var lastSellTrade *bnspot.Trade
		//var lastTrade *bnspot.Trade
		sellImpactTD, _ := tdigest.New()
		buyImpactTD, _ := tdigest.New()
		tradeSizeTD, _ := tdigest.New()
		for _, dateStr := range strings.Split(dateStrs, ",") {
			file, err := os.Open(
				fmt.Sprintf("/Users/chenjilin/MarketData/bnspot-bnswap-depth5/%s/%s-%s.depth5.jl.gz", dateStr, dateStr, symbol),
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
			b := make([]byte, 0, 512)
			_, err = gr.Read(b)
			if err != nil {
				logger.Debugf("gr.Read(b) error %v", err)
				continue
			}
			logger.Debugf("%v", gr.Header)
			logger.Debugf("%v", gr.OS)
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
				counter ++
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
		quantiles[symbol] = fmt.Sprintf(
			"%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f",
			sellImpactTD.Quantile(0.0005),
			sellImpactTD.Quantile(0.005),
			sellImpactTD.Quantile(0.05),
			sellImpactTD.Quantile(0.2),
			sellImpactTD.Quantile(0.5),
			buyImpactTD.Quantile(0.5),
			buyImpactTD.Quantile(0.8),
			buyImpactTD.Quantile(0.95),
			buyImpactTD.Quantile(0.995),
			buyImpactTD.Quantile(0.9995),
		)
		fmt.Printf(
			"%s:\t%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f\n",
			symbol,
			sellImpactTD.Quantile(0.0005),
			sellImpactTD.Quantile(0.005),
			sellImpactTD.Quantile(0.05),
			sellImpactTD.Quantile(0.2),
			sellImpactTD.Quantile(0.5),
			buyImpactTD.Quantile(0.5),
			buyImpactTD.Quantile(0.8),
			buyImpactTD.Quantile(0.95),
			buyImpactTD.Quantile(0.995),
			buyImpactTD.Quantile(0.9995),
		)
		tradeSizes[symbol] = tradeSizeTD.Quantile(0.8)
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

	fmt.Printf("\n\n var TradeQ80Sizes=map[string]float64{\n")
	for _, symbol := range symbols {
		fmt.Printf(
			"  \"%s\": %f\n",
			symbol, tradeSizes[symbol],
		)
	}
	fmt.Printf("}\n\n")
}
