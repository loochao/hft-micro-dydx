package main

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/geometrybase/hft-micro/tdigest"
	"gopkg.in/yaml.v2"
	"os"
	"strings"
)

func main() {
	symbols := `BTCUSDT,LTCUSDT,ETHUSDT,NEOUSDT,QTUMUSDT,EOSUSDT,ZRXUSDT,OMGUSDT,LRCUSDT,TRXUSDT,KNCUSDT,IOTAUSDT,LINKUSDT,CVCUSDT,ETCUSDT,ZECUSDT,BATUSDT,DASHUSDT,XMRUSDT,ENJUSDT,XRPUSDT,STORJUSDT,BTSUSDT,ADAUSDT,XLMUSDT,WAVESUSDT,ICXUSDT,RLCUSDT,IOSTUSDT,BLZUSDT,ONTUSDT,ZILUSDT,ZENUSDT,THETAUSDT,VETUSDT,RENUSDT,MATICUSDT,ATOMUSDT,FTMUSDT,CHZUSDT,ALGOUSDT,DOGEUSDT,ANKRUSDT,TOMOUSDT,BANDUSDT,XTZUSDT,KAVAUSDT,BCHUSDT,SOLUSDT,HNTUSDT,COMPUSDT,MKRUSDT,SXPUSDT,SNXUSDT,DOTUSDT,RUNEUSDT,BALUSDT,YFIUSDT,SRMUSDT,CRVUSDT,SANDUSDT,OCEANUSDT,LUNAUSDT,RSRUSDT,TRBUSDT,EGLDUSDT,BZRXUSDT,KSMUSDT,SUSHIUSDT,YFIIUSDT,BELUSDT,UNIUSDT,AVAXUSDT,FLMUSDT,ALPHAUSDT,NEARUSDT,AAVEUSDT,FILUSDT,CTKUSDT,AXSUSDT,AKROUSDT,SKLUSDT,GRTUSDT,1INCHUSDT,LITUSDT,RVNUSDT,SFPUSDT,REEFUSDT,DODOUSDT,COTIUSDT,CHRUSDT,ALICEUSDT,HBARUSDT,MANAUSDT,STMXUSDT,UNFIUSDT,XEMUSDT,CELRUSDT,HOTUSDT,ONEUSDT,LINAUSDT,DENTUSDT,MTLUSDT,OGNUSDT,NKNUSDT,DGBUSDT`
	dateStrs := "20210501,20210502,20210503,20210504,20210505,20210506"
	quantiles := make(map[string]int)
	priceRatios := make(map[string]float64)
	var err error
	var d *bnswap.Trade
	for _, symbol := range strings.Split(symbols, ",") {
		tradeValueTD, _ := tdigest.New()
		for _, dateStr := range strings.Split(dateStrs, ",") {
			file, err := os.Open(
				fmt.Sprintf("/home/clu/MarketData/bnswap-trade/%s-%s.bnswap.trade.jl.gz", dateStr, symbol),
			)
			if err != nil {
				logger.Debugf("os.Open() error %v", err)
				return
			}
			gr, err := gzip.NewReader(file)
			if err != nil {
				logger.Debugf("gzip.NewReader(file) error %v", err)
				return
			}
			scanner := bufio.NewScanner(gr)
			for scanner.Scan() {
				d, err = bnswap.ParseTrade(scanner.Bytes())
				if err != nil {
					logger.Debugf("bnswap.ParseDepth20 error %v", err)
					logger.Debugf("%s", scanner.Bytes())
					continue
				}
				_ = tradeValueTD.Add(d.Price * d.Quantity)
			}
			_ = gr.Close()
			_ = file.Close()
			//time.Sleep(time.Second)
		}
		quantiles[symbol] = int(tradeValueTD.Quantile(0.8)/100) * 100
		if d != nil && d.Price != 0 {
			priceRatios[symbol] = tradeValueTD.Quantile(0.8) / d.Price
			logger.Debugf("%s TRADE_VALUE %d TRADE_VALUE/PRICE %.4f PRICE %.4f", symbol, quantiles[symbol], priceRatios[symbol], d.Price)
		}
	}
	output, err := yaml.Marshal(quantiles)
	if err != nil {
		logger.Debugf("yaml.Marshal error %v", err)
	} else {
		logger.Debugf("YAML OUTPUT:\n%s", output)
	}
	ss := make([]string, 0)
	vv := make([]float64, 0)
	for symbol, value := range quantiles {
		ss = append(ss, symbol)
		vv = append(vv, float64(value))
	}
	maps, err := common.RankSymbols(ss, vv)
	if err != nil {
		logger.Debugf("common.RankSymbols() error %v", err)
	}
	logger.Debugf("%v", maps)
	for i := len(ss) - 1; i >= 0; i-- {
		fmt.Printf("\"%s\": %d,\n", maps[i], quantiles[maps[i]])
	}
	//output, err = yaml.Marshal(priceRatios)
	//if err != nil {
	//	logger.Debugf("yaml.Marshal error %v", err)
	//}else{
	//	logger.Debugf("YAML OUTPUT:\n%s", output)
	//}
}
