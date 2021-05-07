package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/influx/client"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"strings"
	"time"
)

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
	//symbolsStr := `BTCUSDT,LTCUSDT,ETHUSDT,NEOUSDT,QTUMUSDT,EOSUSDT,ZRXUSDT,OMGUSDT,LRCUSDT,TRXUSDT,KNCUSDT,IOTAUSDT,LINKUSDT,CVCUSDT,ETCUSDT,ZECUSDT,BATUSDT,DASHUSDT,XMRUSDT,ENJUSDT,XRPUSDT,STORJUSDT,BTSUSDT,ADAUSDT,XLMUSDT,WAVESUSDT,ICXUSDT,RLCUSDT,IOSTUSDT,BLZUSDT,ONTUSDT,ZILUSDT,ZENUSDT,THETAUSDT,VETUSDT,RENUSDT,MATICUSDT,ATOMUSDT,FTMUSDT,CHZUSDT,ALGOUSDT,DOGEUSDT,ANKRUSDT,TOMOUSDT,BANDUSDT,XTZUSDT,KAVAUSDT,BCHUSDT,SOLUSDT,HNTUSDT,COMPUSDT,MKRUSDT,SXPUSDT,SNXUSDT,DOTUSDT,RUNEUSDT,BALUSDT,YFIUSDT,SRMUSDT,CRVUSDT,SANDUSDT,OCEANUSDT,LUNAUSDT,RSRUSDT,TRBUSDT,EGLDUSDT,BZRXUSDT,KSMUSDT,SUSHIUSDT,YFIIUSDT,BELUSDT,UNIUSDT,AVAXUSDT,FLMUSDT,ALPHAUSDT,NEARUSDT,AAVEUSDT,FILUSDT,CTKUSDT,AXSUSDT,AKROUSDT,SKLUSDT,GRTUSDT,1INCHUSDT,LITUSDT,RVNUSDT,SFPUSDT,REEFUSDT,DODOUSDT,COTIUSDT,CHRUSDT,ALICEUSDT,HBARUSDT,MANAUSDT,STMXUSDT,UNFIUSDT,XEMUSDT,CELRUSDT,HOTUSDT,ONEUSDT,LINAUSDT,DENTUSDT,MTLUSDT,OGNUSDT,NKNUSDT,DGBUSDT`
	//symbolsStr := "LINKUSDT,FILUSDT"

	//symbols := strings.Split(symbolsStr, ",")[:20]

	symbols := []string{
		"BTCUSDT",
		"ETHUSDT",
		"XRPUSDT",
		"EOSUSDT",
		"LTCUSDT",
		"LINKUSDT",
		"BCHUSDT",
		"DOTUSDT",
		"TRXUSDT",
		"ETCUSDT",
		"ADAUSDT",
		"CRVUSDT",
		"DOGEUSDT",
		"SOLUSDT",
		"UNIUSDT",
		"XTZUSDT",
		"FILUSDT",
		"LUNAUSDT",
		"AVAXUSDT",
		"XEMUSDT",
		"NEOUSDT",
		"CTKUSDT",
		"DODOUSDT",
		"VETUSDT",
		"CHRUSDT",
		"AAVEUSDT",
		"CHZUSDT",
		"MKRUSDT",
		"SUSHIUSDT",
		"KNCUSDT",
		"ALICEUSDT",
		"SXPUSDT",
		"YFIUSDT",
		"KSMUSDT",
		"DASHUSDT",
		"QTUMUSDT",
		"DENTUSDT",
		"XLMUSDT",
		"ZILUSDT",
		"STMXUSDT",
		"THETAUSDT",
		"MATICUSDT",
		"COMPUSDT",
		"ALPHAUSDT",
		"ONTUSDT",
		"HOTUSDT",
		"WAVESUSDT",
		"ZECUSDT",
		"SRMUSDT",
		"XMRUSDT",
		"ONEUSDT",
		"ATOMUSDT",
		"IOTAUSDT",
		"REEFUSDT",
		"1INCHUSDT",
		"BATUSDT",
		"FTMUSDT",
		"BZRXUSDT",
		"AKROUSDT",
		"AXSUSDT",
		"LITUSDT",
		"FLMUSDT",
		"RVNUSDT",
		"GRTUSDT",
		"RUNEUSDT",
		"LINAUSDT",
		"IOSTUSDT",
		"SANDUSDT",
		"HBARUSDT",
		"MANAUSDT",
		"RLCUSDT",
		"ZRXUSDT",
		"ALGOUSDT",
		"TRBUSDT",
		"ICXUSDT",
		"KAVAUSDT",
		"MTLUSDT",
		"OMGUSDT",
		"TOMOUSDT",
		"BELUSDT",
		"STORJUSDT",
		"DGBUSDT",
		"BALUSDT",
		"EGLDUSDT",
		"ZENUSDT",
		"NKNUSDT",
		"CELRUSDT",
		"BTSUSDT",
		"ENJUSDT",
		"OGNUSDT",
		"ANKRUSDT",
		"BANDUSDT",
		"RSRUSDT",
		"SNXUSDT",
		"CVCUSDT",
		"YFIIUSDT",
		"COTIUSDT",
		"RENUSDT",
		"SFPUSDT",
		"SKLUSDT",
		"OCEANUSDT",
		"LRCUSDT",
		"BLZUSDT",
		"HNTUSDT",
		"UNFIUSDT",
		"NEARUSDT",
	}[:100]

	//symbols = symbols[:2]
	mirsMap := make(map[string]map[time.Time][2]float64)
	times := make([]time.Time, 0)
	for _, symbol := range symbols {
		logger.Debugf("%s", symbol)
		mirsMap[symbol] = make(map[time.Time][2]float64)

		file, err := os.Open(
			fmt.Sprintf("/Users/chenjilin/MarketData/mir/4h.mir.%s.csv", symbol),
		)
		if err != nil {
			logger.Debugf("os.Open() error %v", err)
			return
		}
		scanner := bufio.NewScanner(file)
		count := 0
		for scanner.Scan() {
			if count == 0 {
				count++
				continue
			}
			splits := strings.Split(scanner.Text(), ",")
			t, err := common.ParseInt([]byte(splits[1]))
			if err != nil {
				logger.Debugf("%v %s", err, splits[1])
				return
			}
			ts := time.Unix(0, t)
			mir, err := common.ParseFloat([]byte(splits[2]))
			if err != nil {
				logger.Debugf("%v %s", err, splits[2])
				return
			}
			price, err := common.ParseFloat([]byte(splits[3]))
			if err != nil {
				logger.Debugf("%v %s %s", err, splits[3], scanner.Text())
				return
			}
			mirsMap[symbol][ts] = [2]float64{mir, price}
			if symbol == symbols[0] {
				times = append(times, ts)
			}
		}
	}

	sizes := make(map[string]float64)
	costs := make(map[string]float64)
	lastCosts := make(map[string]float64)
	for _, symbol := range symbols {
		sizes[symbol] = 0.0
		costs[symbol] = 0.0
	}
	entryValue := 1.0 / float64(len(symbols))
	netWorth := 1.0
	commission := -0.0004
timeLoop:
	for _, t := range times {
		if t.Sub(t.Truncate(time.Hour)) != time.Minute*15 {
			continue
		}

		for _, symbol := range symbols {
			if v, ok := mirsMap[symbol][t]; ok {
				mir := v[0]
				price := v[1]
				if mir > 0 {
					if sizes[symbol] == 0 {
						netWorth += entryValue * commission
						sizes[symbol] = -entryValue
						costs[symbol] = price
						lastCosts[symbol] = price
					} else if sizes[symbol] > 0 {
						netWorth += sizes[symbol] * (price - costs[symbol]) / costs[symbol]
						netWorth += sizes[symbol] * commission
						netWorth += entryValue * commission
						sizes[symbol] = -entryValue
						costs[symbol] = price
						lastCosts[symbol] = price
					} else if (price-lastCosts[symbol])/lastCosts[symbol] < 0 && sizes[symbol] > -100*entryValue {
						netWorth += entryValue * commission
						costs[symbol] = (price*-entryValue + costs[symbol]*sizes[symbol]) / (-entryValue + sizes[symbol])
						sizes[symbol] -= entryValue
						lastCosts[symbol] = price
					}
				} else {
					if sizes[symbol] == 0 {
						netWorth += entryValue * commission
						sizes[symbol] = entryValue
						costs[symbol] = price
						lastCosts[symbol] = price
					} else if sizes[symbol] < 0 {
						netWorth += sizes[symbol] * (price - costs[symbol]) / costs[symbol]
						netWorth += -sizes[symbol] * commission
						netWorth += entryValue * commission
						sizes[symbol] = entryValue
						costs[symbol] = price
						lastCosts[symbol] = price
					} else if (price-lastCosts[symbol])/lastCosts[symbol] > 0 && sizes[symbol] < 100*entryValue {
						netWorth += entryValue * commission
						costs[symbol] = (price*entryValue + costs[symbol]*sizes[symbol]) / (entryValue + sizes[symbol])
						sizes[symbol] += entryValue
						lastCosts[symbol] = price
					}
				}
			} else {
				logger.Debugf("MISS %s %v", symbol, t)
				continue timeLoop
			}
		}
		logger.Debugf("%v %f %v", t, netWorth, t.Sub(t.Truncate(time.Hour)))
		fields := make(map[string]interface{})
		fields["netWorth"] = netWorth
		pt, err := client.NewPoint(
			"bnswap-trade-mir-alpha",
			map[string]string{},
			fields,
			t,
		)
		if err != nil {
			logger.Debugf("client.NewPoint error %v", err)
		}
		iw.PointCh <- pt
		for _, symbol := range symbols {
			fields := make(map[string]interface{})
			fields["size"] = sizes[symbol]
			pt, _ := client.NewPoint(
				"bnswap-trade-mir-alpha",
				map[string]string{
					"symbol": symbol,
				},
				fields,
				t,
			)
			iw.PointCh <- pt
		}
	}

	defer func() {
		time.Sleep(time.Second * 3)
	}()

}
