package main

import (
	"compress/gzip"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/influx/client"
	kucoin_usdtfuture "github.com/geometrybase/hft-micro/kucoin-usdtfuture"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/montanaflynn/stats"
	"io"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"time"
)

func optBySymbol(xSymbol, ySymbol string, writer *common.InfluxWriter, measurement string) (map[string]Result, error) {
	fileName := fmt.Sprintf("/Users/chenjilin/Downloads/20210820-20210916-%s-%s-24h0m0s-3s-1ms.gz", xSymbol, ySymbol)
	f, err := os.OpenFile(fileName, os.O_RDONLY, 0600)
	if err != nil {
		return nil, err
	}
	gr, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}

	//startTime := time.Now()
	data := make([]*common.MatchedSpread, 40000000)
	counter := 0
	//weeks2 := time.Hour * 24 * 18
	for err != io.EOF {
		ms := &common.MatchedSpread{}
		err = binary.Read(gr, binary.BigEndian, ms)
		if err != nil && err != io.EOF {
			return nil, err
		}
		//if time.Now().Sub(time.Unix(0, ms.EventTime)) < weeks2 {
		if err != io.EOF {
			data[counter] = ms
			counter++
			if counter == len(data) {
				dataNew := make([]*common.MatchedSpread, len(data)+10000000)
				copy(dataNew[:len(data)], data)
				data = dataNew
			}
		}
		//}
	}
	err = gr.Close()
	if err != nil {
		return nil, err
	}
	err = f.Close()
	if err != nil {
		return nil, err
	}

	//logger.Debugf("READ ALL DATA, TAKE %v", time.Now().Sub(startTime))

	outputMap := make(map[string]Result)

	for i := 1.0; i <= 4.0; i += 1.0 {
		for j := 1.0; j <= 2.0; j += 1.0 {
			params := Params{
				XSymbol:        xSymbol,
				YSymbol:        ySymbol,
				EnterOffset:    0.0005 * i,
				LeaveOffset:    0.001,
				FrFactor:       0.8,
				StartValue:     10000,
				EnterStep:      0.1*j,
				enterInterval:  time.Second * 5,
				OutputInterval: time.Minute,
				BestSizeFactor: 8.0,
				Leverage:       1.0,
				TradeCost:      -0.0006,
				MaxFundingRate: 0.003,
			}
			result := strategyA(params, data)
			std, err := stats.StandardDeviation(result.NetWorth)
			if err != nil {
				logger.Debugf("error %v", err)
			}
			paramsContent, err := json.Marshal(params)
			if err != nil {
				logger.Debugf("error %v", err)
			} else {
				outputMap[string(paramsContent)] = *result
			}
			fmt.Printf("%s ENTER OFFSET %.4f STEP %.4f MFR %.4f NW %.4f SR %.4f TV %.2f\n",
				result.Params.XSymbol,
				result.Params.EnterOffset,
				result.Params.EnterStep,
				result.Params.MaxFundingRate,
				result.NetWorth[len(result.NetWorth)-1],
				(result.NetWorth[len(result.NetWorth)-1]-1.0)/std,
				result.Turnover,
			)

			if writer != nil {
				for t, eventTime := range result.EventTimes {
					fields := make(map[string]interface{})
					fields["netWorth"] = result.NetWorth[t]
					fields["position"] = result.Positions[t]
					fields["cost"] = result.Costs[t]
					fields["midPrice"] = result.MidPrices[t]
					fields["fundingRate"] = result.FundingRates[t]
					pt, err := client.NewPoint(
						measurement,
						map[string]string{
							"xSymbol":     xSymbol,
							"enterOffset": fmt.Sprintf("%.4f", result.Params.EnterOffset),
							"enterStep": fmt.Sprintf("%.4f", result.Params.EnterStep),
						},
						fields,
						eventTime,
					)
					if err == nil {
						writer.PointCh <- pt
					}
				}
			}
		}
	}
	return outputMap, nil
}

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

	dataPath := "/Users/chenjilin/Projects/hft-micro/applications/usd-tk-tt-q-t/configs/kcuf-bnuf-opt/"
	symbolsMap := map[string]string{
		"XBTUSDTM": "BTCUSDT",
	}
	symbols := []string{"XBTUSDTM"}
	for symbol := range kucoin_usdtfuture.TickSizes {
		if _, ok := binance_usdtfuture.TickSizes[strings.Replace(symbol, "USDTM", "USDT", -1)]; ok {
			symbols = append(symbols, symbol)
			symbolsMap[symbol] = strings.Replace(symbol, "USDTM", "USDT", -1)
		}
	}
	sort.Strings(symbols)
	symbols = []string{"VETUSDTM"}
	for _, xSymbol := range symbols {
		ySymbol := symbolsMap[xSymbol]
		outputPath := fmt.Sprintf("%s%s-%s.json", dataPath, xSymbol, ySymbol)

		_, err := os.Stat(outputPath)
		if err != nil && !os.IsNotExist(err) {
			logger.Debugf("%v", err)
			continue
		} else if err == nil {
			continue
		}

		output, err := optBySymbol(xSymbol, ySymbol, nil, "kcuf-bnuf-opt-q-t")
		if err != nil {
			logger.Debugf("optBySymbol error %v", err)
		} else {
			contents, err := json.Marshal(output)
			if err != nil {
				logger.Debugf("%v", err)
			} else {
				err := ioutil.WriteFile(outputPath, contents, 0775)
				if err != nil {
					logger.Debugf("%v", err)
				}
			}
		}
	}
}
