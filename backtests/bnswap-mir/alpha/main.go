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
		5000,
	)
	if err != nil {
		panic(err)
	}
	defer iw.Stop()

	//symbolsStr := "BTCUSDT,ETHUSDT,FILUSDT,DOGEUSDT,XRPUSDT,MATICUSDT,LTCUSDT,EOSUSDT,LINKUSDT,MKRUSDT"
	symbolsStr := "LINKUSDT,FILUSDT"

	symbols := strings.Split(symbolsStr, ",")
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
				logger.Debugf("%v %s", err, splits[3])
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
	for _, symbol := range symbols {
		sizes[symbol] = 0.0
		costs[symbol] = 0.0
	}
	entryValue := 1.0 / float64(len(symbols))
	netWorth := 1.0
	commission := -0.000
timeLoop:
	for _, t := range times {
		if t.Truncate(time.Minute*5).Sub(t) != 0 {
			continue
		}
		alphas := make([]float64, len(symbols))
		prices := make(map[string]float64)
		mirs := make(map[string]float64)
		for i, symbol := range symbols {
			if v, ok := mirsMap[symbol][t]; ok {
				alphas[i] = -v[0]
				mirs[symbol] = v[0]
				prices[symbol] = v[1]
			} else {
				logger.Debugf("MISS %s %v", symbol, t)
				continue timeLoop
			}
		}
		sm, err := common.RankSymbols(symbols, alphas)
		if err != nil {
			logger.Debugf("common.RankSymbols() error %v", err)
			return
		}
		//logger.Debugf("%v", sm)
		for rank, symbol := range sm {
			if rank < len(symbols)/2 {
				if sizes[symbol] == 0 {
					netWorth += entryValue * commission
					sizes[symbol] = -entryValue
					costs[symbol] = prices[symbol]
				} else if sizes[symbol] > 0 {
					netWorth += sizes[symbol] * (prices[symbol] - costs[symbol]) / costs[symbol]
					netWorth += sizes[symbol] * commission
					netWorth += entryValue * commission
					sizes[symbol] = -entryValue
					costs[symbol] = prices[symbol]
				}
			} else {
				if sizes[symbol] == 0 {
					netWorth += entryValue * commission
					sizes[symbol] = entryValue
					costs[symbol] = prices[symbol]
				} else if sizes[symbol] < 0 {
					netWorth += sizes[symbol] * (prices[symbol] - costs[symbol]) / costs[symbol]
					netWorth += -sizes[symbol] * commission
					netWorth += entryValue * commission
					sizes[symbol] = entryValue
					costs[symbol] = prices[symbol]
				}
			}
		}
		logger.Debugf("%v %f", t, netWorth)
		fields := make(map[string]interface{})
		fields["netWorth"] = netWorth
		pt, err := client.NewPoint(
			"bnswap-trade-mir-alpha",
			map[string]string{},
			fields,
			t,
		)
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

}
