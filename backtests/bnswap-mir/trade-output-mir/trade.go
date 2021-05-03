package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/influx/client"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"strings"
	"time"
)

var tradeMinSizes = map[string]float64{
	"ETHUSDT":   55100,
	"BTCUSDT":   27700,
	"XRPUSDT":   23000,
	"LTCUSDT":   15900,
	"EOSUSDT":   15600,
	"LINKUSDT":  7200,
	"CRVUSDT":   6000,
	"BCHUSDT":   6000,
	"AAVEUSDT":  5700,
	"DOTUSDT":   5500,
	"FILUSDT":   5000,
	"ADAUSDT":   4300,
	"XTZUSDT":   4200,
	"AVAXUSDT":  3900,
	"COMPUSDT":  3900,
	"MATICUSDT": 3900,
	"TRXUSDT":   3800,
	"UNIUSDT":   3600,
	"ONTUSDT":   3600,
	"XEMUSDT":   3600,
	"SXPUSDT":   3600,
	"DOGEUSDT":  3500,
	"LUNAUSDT":  3400,
	"SOLUSDT":   3200,
	"ETCUSDT":   3100,
	"FLMUSDT":   3000,
	"NEOUSDT":   3000,
	"CHZUSDT":   2900,
	"YFIUSDT":   2800,
	"ANKRUSDT":  2800,
	"FTMUSDT":   2800,
	"CHRUSDT":   2700,
	"KAVAUSDT":  2700,
	"ATOMUSDT":  2700,
	"XLMUSDT":   2700,
	"REEFUSDT":  2700,
	"ENJUSDT":   2600,
	"DENTUSDT":  2600,
	"MANAUSDT":  2600,
	"SUSHIUSDT": 2500,
	"DODOUSDT":  2400,
	"MKRUSDT":   2400,
	"HOTUSDT":   2400,
	"AXSUSDT":   2300,
	"CTKUSDT":   2300,
	"XMRUSDT":   2200,
	"ZRXUSDT":   2100,
	"ALGOUSDT":  2100,
	"KSMUSDT":   2100,
	"QTUMUSDT":  2100,
	"ALICEUSDT": 2100,
	"VETUSDT":   2000,
	"GRTUSDT":   2000,
	"SRMUSDT":   2000,
	"ALPHAUSDT": 2000,
	"1INCHUSDT": 2000,
	"THETAUSDT": 2000,
	"BZRXUSDT":  2000,
	"DASHUSDT":  2000,
	"YFIIUSDT":  1900,
	"ONEUSDT":   1900,
	"ZILUSDT":   1800,
	"ZECUSDT":   1800,
	"IOTAUSDT":  1800,
	"KNCUSDT":   1800,
	"LINAUSDT":  1700,
	"STORJUSDT": 1700,
	"OMGUSDT":   1700,
	"BATUSDT":   1700,
	"LITUSDT":   1700,
	"CVCUSDT":   1600,
	"OCEANUSDT": 1600,
	"SNXUSDT":   1600,
	"BTSUSDT":   1500,
	"ZENUSDT":   1500,
	"BALUSDT":   1500,
	"AKROUSDT":  1500,
	"SANDUSDT":  1500,
	"OGNUSDT":   1400,
	"ICXUSDT":   1400,
	"NKNUSDT":   1400,
	"TOMOUSDT":  1400,
	"IOSTUSDT":  1400,
	"RENUSDT":   1400,
	"NEARUSDT":  1400,
	"SFPUSDT":   1300,
	"CELRUSDT":  1300,
	"EGLDUSDT":  1200,
	"RVNUSDT":   1200,
	"RUNEUSDT":  1200,
	"SKLUSDT":   1100,
	"UNFIUSDT":  1100,
	"RLCUSDT":   1100,
	"WAVESUSDT": 1100,
	"RSRUSDT":   1000,
	"HNTUSDT":   1000,
	"STMXUSDT":  1000,
	"BELUSDT":   900,
	"BANDUSDT":  900,
	"DGBUSDT":   900,
	"MTLUSDT":   900,
	"TRBUSDT":   900,
	"COTIUSDT":  800,
	"HBARUSDT":  800,
	"LRCUSDT":   600,
	"BLZUSDT":   600,
}

var tradeMinSizes80 = map[string]float64{
	"ETHUSDT":   10400,
	"BTCUSDT":   9900,
	"XRPUSDT":   7800,
	"LTCUSDT":   5500,
	"EOSUSDT":   3900,
	"LINKUSDT":  2300,
	"CRVUSDT":   2100,
	"BCHUSDT":   2000,
	"DOTUSDT":   1900,
	"ADAUSDT":   1600,
	"AAVEUSDT":  1600,
	"DOGEUSDT":  1600,
	"FILUSDT":   1500,
	"TRXUSDT":   1400,
	"COMPUSDT":  1400,
	"ONTUSDT":   1300,
	"LUNAUSDT":  1200,
	"XEMUSDT":   1200,
	"XTZUSDT":   1100,
	"SXPUSDT":   1100,
	"ENJUSDT":   1100,
	"CHZUSDT":   1100,
	"AXSUSDT":   1100,
	"ALPHAUSDT": 1000,
	"MATICUSDT": 1000,
	"ATOMUSDT":  1000,
	"UNIUSDT":   1000,
	"SOLUSDT":   1000,
	"SUSHIUSDT": 900,
	"FTMUSDT":   900,
	"MANAUSDT":  900,
	"DODOUSDT":  900,
	"KAVAUSDT":  900,
	"CTKUSDT":   900,
	"AVAXUSDT":  900,
	"XLMUSDT":   800,
	"ETCUSDT":   800,
	"DENTUSDT":  800,
	"ONEUSDT":   800,
	"ZILUSDT":   800,
	"BATUSDT":   800,
	"FLMUSDT":   800,
	"LITUSDT":   800,
	"BZRXUSDT":  800,
	"REEFUSDT":  800,
	"ANKRUSDT":  800,
	"MKRUSDT":   800,
	"ALICEUSDT": 800,
	"NEOUSDT":   800,
	"IOTAUSDT":  700,
	"OCEANUSDT": 700,
	"SNXUSDT":   700,
	"BALUSDT":   700,
	"TOMOUSDT":  700,
	"HOTUSDT":   700,
	"KNCUSDT":   700,
	"GRTUSDT":   700,
	"ZRXUSDT":   600,
	"IOSTUSDT":  600,
	"1INCHUSDT": 600,
	"CHRUSDT":   600,
	"ZECUSDT":   600,
	"VETUSDT":   600,
	"QTUMUSDT":  600,
	"AKROUSDT":  600,
	"DASHUSDT":  600,
	"THETAUSDT": 600,
	"YFIUSDT":   600,
	"YFIIUSDT":  500,
	"TRBUSDT":   500,
	"SFPUSDT":   500,
	"OMGUSDT":   500,
	"BTSUSDT":   500,
	"SRMUSDT":   500,
	"ALGOUSDT":  500,
	"NEARUSDT":  500,
	"LINAUSDT":  500,
	"ZENUSDT":   500,
	"CELRUSDT":  500,
	"KSMUSDT":   500,
	"RENUSDT":   500,
	"MTLUSDT":   400,
	"RVNUSDT":   400,
	"WAVESUSDT": 400,
	"STMXUSDT":  400,
	"BANDUSDT":  400,
	"OGNUSDT":   400,
	"STORJUSDT": 400,
	"CVCUSDT":   400,
	"NKNUSDT":   400,
	"RUNEUSDT":  400,
	"XMRUSDT":   400,
	"COTIUSDT":  300,
	"BELUSDT":   300,
	"EGLDUSDT":  300,
	"RSRUSDT":   300,
	"SKLUSDT":   300,
	"SANDUSDT":  300,
	"HBARUSDT":  300,
	"HNTUSDT":   300,
	"RLCUSDT":   300,
	"UNFIUSDT":  300,
	"ICXUSDT":   200,
	"BLZUSDT":   100,
	"DGBUSDT":   100,
	"LRCUSDT":   100,
}

//`BTCUSDT,LTCUSDT,ETHUSDT,NEOUSDT,QTUMUSDT,EOSUSDT,ZRXUSDT,OMGUSDT,LRCUSDT,TRXUSDT,KNCUSDT,IOTAUSDT,LINKUSDT,CVCUSDT,ETCUSDT,ZECUSDT,BATUSDT,DASHUSDT,XMRUSDT,ENJUSDT,XRPUSDT,STORJUSDT,BTSUSDT,ADAUSDT,XLMUSDT,WAVESUSDT,ICXUSDT,RLCUSDT,IOSTUSDT,BLZUSDT,ONTUSDT,ZILUSDT,ZENUSDT,THETAUSDT,VETUSDT,RENUSDT,MATICUSDT,ATOMUSDT,FTMUSDT,CHZUSDT,ALGOUSDT,DOGEUSDT,ANKRUSDT,TOMOUSDT,BANDUSDT,XTZUSDT,KAVAUSDT,BCHUSDT,SOLUSDT,HNTUSDT,COMPUSDT,MKRUSDT,SXPUSDT,SNXUSDT,DOTUSDT,RUNEUSDT,BALUSDT,YFIUSDT,SRMUSDT,CRVUSDT,SANDUSDT,OCEANUSDT,LUNAUSDT,RSRUSDT,TRBUSDT,EGLDUSDT,BZRXUSDT,KSMUSDT,SUSHIUSDT,YFIIUSDT,BELUSDT,UNIUSDT,AVAXUSDT,FLMUSDT,ALPHAUSDT,NEARUSDT,AAVEUSDT,FILUSDT,CTKUSDT,AXSUSDT,AKROUSDT,SKLUSDT,GRTUSDT,1INCHUSDT,LITUSDT,RVNUSDT,SFPUSDT,REEFUSDT,DODOUSDT,COTIUSDT,CHRUSDT,ALICEUSDT,HBARUSDT,MANAUSDT,STMXUSDT,UNFIUSDT,XEMUSDT,CELRUSDT,HOTUSDT,ONEUSDT,LINAUSDT,DENTUSDT,MTLUSDT,OGNUSDT,NKNUSDT,DGBUSDT`
//BTCUSDT LTCUSDT ETHUSDT NEOUSDT QTUMUSDT EOSUSDT ZRXUSDT OMGUSDT LRCUSDT TRXUSDT KNCUSDT IOTAUSDT LINKUSDT CVCUSDT ETCUSDT ZECUSDT BATUSDT DASHUSDT XMRUSDT ENJUSDT XRPUSDT STORJUSDT BTSUSDT ADAUSDT XLMUSDT WAVESUSDT ICXUSDT RLCUSDT IOSTUSDT BLZUSDT ONTUSDT ZILUSDT ZENUSDT THETAUSDT VETUSDT RENUSDT MATICUSDT ATOMUSDT FTMUSDT CHZUSDT ALGOUSDT DOGEUSDT ANKRUSDT TOMOUSDT BANDUSDT XTZUSDT KAVAUSDT BCHUSDT SOLUSDT HNTUSDT COMPUSDT MKRUSDT SXPUSDT SNXUSDT DOTUSDT RUNEUSDT BALUSDT YFIUSDT SRMUSDT CRVUSDT SANDUSDT OCEANUSDT LUNAUSDT RSRUSDT TRBUSDT EGLDUSDT BZRXUSDT KSMUSDT SUSHIUSDT YFIIUSDT BELUSDT UNIUSDT AVAXUSDT FLMUSDT ALPHAUSDT NEARUSDT AAVEUSDT FILUSDT CTKUSDT AXSUSDT AKROUSDT SKLUSDT GRTUSDT 1INCHUSDT LITUSDT RVNUSDT SFPUSDT REEFUSDT DODOUSDT COTIUSDT CHRUSDT ALICEUSDT HBARUSDT MANAUSDT STMXUSDT UNFIUSDT XEMUSDT CELRUSDT HOTUSDT ONEUSDT LINAUSDT DENTUSDT MTLUSDT OGNUSDT NKNUSDT DGBUSDT

func main() {
	ctx := context.Background()
	iw, err := common.NewInfluxWriter(
		ctx,
		"http://localhost:8086",
		"",
		"",
		"hft",
		500,
	)
	if err != nil {
		panic(err)
	}
	defer iw.Stop()

	dateStrs := "20210428,20210429,20210430,20210501,20210502"
	for symbol, minTradeSize := range tradeMinSizes80 {
		lookback := time.Hour*4

		computeInterval := time.Minute * 5
		timedMean := common.NewTimedMean(lookback)
		var lastTradeTime *time.Time
		var firstPrice *float64
		for _, dateStr := range strings.Split(dateStrs, ",") {
			logger.Debugf("%s %s %f", symbol, dateStr, minTradeSize)
			file, err := os.Open(
				fmt.Sprintf("/Users/chenjilin/MarketData/bnswap-trade/%s-%s.bnswap.trade.jl.gz", dateStr, symbol),
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
				d, err := bnswap.ParseTrade(scanner.Bytes())
				if err != nil {
					logger.Debugf("bnswap.ParseDepth20 error %v", err)
					continue
				}
				if firstPrice == nil {
					firstPrice = &d.Price
				}
				if d.Price*d.Quantity > minTradeSize {
					timedMean.Insert(d.EventTime, d.Price)
				}
				if timedMean.Len() > 0 && lastTradeTime != nil && d.EventTime.Truncate(computeInterval).Sub(*lastTradeTime) > 0 {
					mir := common.ComputeMIR(timedMean.Values())
					fields := make(map[string]interface{})
					fields["mir"] = mir
					fields["price"] = d.Price
					fields["return"] = d.Price / *firstPrice
					pt, err := client.NewPoint(
						fmt.Sprintf("bnswap-trade-mir-80-%v", lookback),
						map[string]string{
							"symbol": symbol,
						},
						fields,
						d.EventTime,
					)
					if err != nil {
						logger.Debugf("client.NewPoint error %v", err)
					}
					iw.PointCh <- pt
				}
				lastTradeTime = &d.EventTime
			}
			_ = gr.Close()
			_ = file.Close()
		}
		time.Sleep(time.Second)
	}
}
