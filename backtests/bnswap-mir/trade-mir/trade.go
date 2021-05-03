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
	"github.com/geometrybase/hft-micro/tdigest"
	"os"
	"strings"
	"time"
)

//func NewInfluxWriter(ctx context.Context, address, username, password, database string, batchSize int) (*InfluxWriter, error) {

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

	//symbols := `BTCUSDT,LTCUSDT,ETHUSDT,NEOUSDT,QTUMUSDT,EOSUSDT,ZRXUSDT,OMGUSDT,LRCUSDT,TRXUSDT,KNCUSDT,IOTAUSDT,LINKUSDT,CVCUSDT,ETCUSDT,ZECUSDT,BATUSDT,DASHUSDT,XMRUSDT,ENJUSDT,XRPUSDT,STORJUSDT,BTSUSDT,ADAUSDT,XLMUSDT,WAVESUSDT,ICXUSDT,RLCUSDT,IOSTUSDT,BLZUSDT,ONTUSDT,ZILUSDT,ZENUSDT,THETAUSDT,VETUSDT,RENUSDT,MATICUSDT,ATOMUSDT,FTMUSDT,CHZUSDT,ALGOUSDT,DOGEUSDT,ANKRUSDT,TOMOUSDT,BANDUSDT,XTZUSDT,KAVAUSDT,BCHUSDT,SOLUSDT,HNTUSDT,COMPUSDT,MKRUSDT,SXPUSDT,SNXUSDT,DOTUSDT,RUNEUSDT,BALUSDT,YFIUSDT,SRMUSDT,CRVUSDT,SANDUSDT,OCEANUSDT,LUNAUSDT,RSRUSDT,TRBUSDT,EGLDUSDT,BZRXUSDT,KSMUSDT,SUSHIUSDT,YFIIUSDT,BELUSDT,UNIUSDT,AVAXUSDT,FLMUSDT,ALPHAUSDT,NEARUSDT,AAVEUSDT,FILUSDT,CTKUSDT,AXSUSDT,AKROUSDT,SKLUSDT,GRTUSDT,1INCHUSDT,LITUSDT,RVNUSDT,SFPUSDT,REEFUSDT,DODOUSDT,COTIUSDT,CHRUSDT,ALICEUSDT,HBARUSDT,MANAUSDT,STMXUSDT,UNFIUSDT,XEMUSDT,CELRUSDT,HOTUSDT,ONEUSDT,LINAUSDT,DENTUSDT,MTLUSDT,OGNUSDT,NKNUSDT,DGBUSDT`
	symbols := "BTCUSDT,ETHUSDT,FILUSDT,DOGEUSDT,XRPUSDT,MATICUSDT,LTCUSDT,EOSUSDT,LINKUSDT,MKRUSDT"
	//symbols := "LINKUSDT,MKRUSDT,FILUSDT,EOSUSDT"

	//symbols := "LTCUSDT,EOSUSDT,LINKUSDT,MKRUSDT"
	dateStrs := "20210428,20210429,20210430,20210501"
	minTradeValues := map[string]float64{
		"BTCUSDT":   10000,
		"ETHUSDT":   6000,
		"XRPUSDT":   5600,
		"EOSUSDT":   8000,
		"LTCUSDT":   2000,
		"FILUSDT":   4000,
		"MATICUSDT": 2000,
		"DOGEUSDT":  2000,
		"LINKUSDT":  5000,
		"MKRUSDT":   9000,
	}
	quantileTop := 0.05
	quantileBot := -0.05
	commission := -0.0004
	for _, symbol := range strings.Split(symbols, ",") {
		lookback := time.Hour * 4
		computeInterval := time.Minute*5
		timedMean := common.NewTimedMean(lookback)
		mirTd, _ := tdigest.New()
		valueTd, _ := tdigest.New()
		enterSize := 0.0
		enterStep := 0.1
		enterTarget := 1.0
		entryPrice := 0.0
		netWorth := 1.0
		lastEnterPrice := 0.0
		lastEnterTime := time.Unix(0, 0)
		enterInterval := time.Minute
		stepPct := float64(lookback / time.Minute)
		minTradeValue, ok := minTradeValues[symbol]
		if !ok {
			logger.Debugf("minTradeValue not fund for %s", symbol)
		}
		for _, dateStr := range strings.Split(dateStrs, ",") {
			logger.Debugf("%s", dateStr)
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

			var lastTradeTime  *time.Time

			scanner := bufio.NewScanner(gr)
			for scanner.Scan() {
				d, err := bnswap.ParseTrade(scanner.Bytes())
				if err != nil {
					logger.Debugf("bnswap.ParseDepth20 error %v", err)
					continue
				}
				_ = valueTd.Add(d.Price * d.Quantity)
				if d.Price*d.Quantity < minTradeValue {
					continue
				}
				if lastTradeTime != nil && d.EventTime.Truncate(computeInterval).Sub(lastEnterTime) > 0 {
					mir := common.ComputeMIR(timedMean.Values())
					fields := make(map[string]interface{})
					if mir > quantileTop && d.EventTime.Sub(lastEnterTime) > enterInterval {
						if enterSize > 0 &&
							d.Price > lastEnterPrice*(1+mir/stepPct) &&
							enterSize < enterTarget {
							entryPrice = (d.Price*enterStep + entryPrice*enterSize) / (enterStep + enterSize)
							enterSize += enterStep
							netWorth += enterStep * commission
							lastEnterTime = d.EventTime
							lastEnterPrice = d.Price
						} else if enterSize == 0 {
							enterSize = enterStep
							netWorth += enterStep * commission
							entryPrice = d.Price
							lastEnterTime = d.EventTime
							lastEnterPrice = d.Price
						} else if enterSize < 0 && enterSize < -enterTarget/2 {
							netWorth += -enterSize/2*commission + enterSize/2*(d.Price-entryPrice)/entryPrice
							enterSize /= 2
							entryPrice = d.Price
							lastEnterTime = d.EventTime
							lastEnterPrice = d.Price
						} else if enterSize < 0 && enterSize > -enterTarget/2 {
							netWorth += -enterSize*commission + enterSize*(d.Price-entryPrice)/entryPrice
							netWorth += enterStep * commission
							enterSize = enterStep
							entryPrice = d.Price
							lastEnterTime = d.EventTime
							lastEnterPrice = d.Price
						}
					} else if mir < quantileBot && d.EventTime.Sub(lastEnterTime) > enterInterval {
						if enterSize < 0 &&
							d.Price < lastEnterPrice*(1+mir/stepPct) &&
							enterSize > -enterTarget {
							entryPrice = (d.Price*-enterStep + entryPrice*enterSize) / (-enterStep + enterSize)
							enterSize -= enterStep
							netWorth += enterStep * commission
							lastEnterTime = d.EventTime
							lastEnterPrice = d.Price
						} else if enterSize == 0 {
							enterSize = -enterStep
							netWorth += enterStep * commission
							entryPrice = d.Price
							lastEnterTime = d.EventTime
							lastEnterPrice = d.Price
						} else if enterSize > 0 && enterSize > enterTarget/2 {
							netWorth += enterSize/2*commission + enterSize/2*(d.Price-entryPrice)/entryPrice
							enterSize /= 2
							entryPrice = d.Price
							lastEnterTime = d.EventTime
							lastEnterPrice = d.Price
						} else if enterSize > 0 && enterSize < enterTarget/2 {
							netWorth += enterSize*commission + enterSize*(d.Price-entryPrice)/entryPrice
							netWorth += enterStep * commission
							enterSize = -enterStep
							entryPrice = d.Price
							lastEnterTime = d.EventTime
							lastEnterPrice = d.Price
						}
					} else if mir < quantileTop &&
						d.EventTime.Sub(lastEnterTime) > enterInterval &&
						enterSize > 0 {
						if enterSize > enterTarget/2 {
							netWorth += enterSize/2*commission + enterSize/2*(d.Price-entryPrice)/entryPrice
							enterSize /= 2
							entryPrice = d.Price
							lastEnterTime = d.EventTime
							lastEnterPrice = d.Price
						} else {
							netWorth += enterSize*commission + enterSize*(d.Price-entryPrice)/entryPrice
							enterSize = 0
							entryPrice = d.Price
							lastEnterTime = d.EventTime
							lastEnterPrice = d.Price
						}
					} else if mir > quantileBot &&
						d.EventTime.Sub(lastEnterTime) > enterInterval &&
						enterSize < 0 {
						if enterSize < -enterTarget/2 {
							netWorth += -enterSize/2*commission + enterSize/2*(d.Price-entryPrice)/entryPrice
							enterSize /= 2
							lastEnterTime = d.EventTime
							lastEnterPrice = d.Price
							entryPrice = d.Price
						} else {
							netWorth += -enterSize*commission + enterSize*(d.Price-entryPrice)/entryPrice
							enterSize = 0
							lastEnterTime = d.EventTime
							lastEnterPrice = d.Price
							entryPrice = d.Price
						}
					}
					_ = mirTd.Add(mir)
					fields["lastPrice"] = d.Price
					fields["entrySize"] = enterSize
					fields["netWorth"] = netWorth
					fields["mir"] = mir
					fields["qRefTop"] = quantileTop //mirTd.Quantile(0.05)
					fields["qRefBot"] = quantileBot //mirTd.Quantile(0.95)
					fields["qTop"] = mirTd.Quantile(0.05)
					fields["qBot"] = mirTd.Quantile(0.95)
					fields["value95"] = valueTd.Quantile(0.95)
					fields["value80"] = valueTd.Quantile(0.80)
					fields["value50"] = valueTd.Quantile(0.50)
					pt, err := client.NewPoint(
						fmt.Sprintf("bnswap-trade-mir-%v", lookback),
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
				timedMean.Insert(d.EventTime, d.Price)
				lastTradeTime = &d.EventTime
			}
			_ = gr.Close()
			_ = file.Close()
		}
		time.Sleep(time.Second)
	}
}
