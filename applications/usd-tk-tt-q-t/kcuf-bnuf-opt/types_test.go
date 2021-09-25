package main

import (
	"encoding/json"
	"fmt"
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
	kucoin_usdtfuture "github.com/geometrybase/hft-micro/kucoin-usdtfuture"
	"github.com/geometrybase/hft-micro/logger"
	"io/ioutil"
	"sort"
	"strings"
	"testing"
)

func TestParamsAsMapKey(t *testing.T) {
	paramA := Params{
		XSymbol:     "XBTUSDTM",
		YSymbol:     "XBTUSDTM",
		EnterOffset: 0.1,
		LeaveOffset: 0.1,
	}
	paramB := paramA
	paramC := paramA
	paramC.EnterOffset = 0.2

	data := map[Params]float64{}
	data[paramA] = 1.0
	data[paramB] = 2.0
	data[paramC] = 3.0
	logger.Debugf("%v", data)
}

func TestSelectParams(t *testing.T) {
	dataPath := "/Users/chenjilin/Projects/hft-micro/applications/usd-tk-tt-q-t/configs/kcuf-bnuf-opt/"
	files, err := ioutil.ReadDir(dataPath)
	if err != nil {
		t.Fatal(err)
	}
	pnls := make(map[string]float64)
	pnlsBySymbol := make(map[string]map[string]float64)
	for _, file := range files {
		xSymbol := strings.Split(file.Name(), "-")[0]
		logger.Debugf("%s", xSymbol)
		contents, err := ioutil.ReadFile(fmt.Sprintf("%s%s", dataPath, file.Name()))
		if err != nil {
			t.Fatal(err)
		}
		dm := make(map[string]Result)
		err = json.Unmarshal(contents, &dm)
		if err != nil {
			t.Fatal(err)
		}
		pnlsBySymbol[xSymbol] = make(map[string]float64)
		for paramStr, result := range dm {
			param := &Params{}
			err = json.Unmarshal([]byte(paramStr), param)
			if err != nil {
				t.Fatal(err)
			}
			pnlKey := fmt.Sprintf("%.4f-%.4f", param.EnterOffset, param.EnterStep)
			pnlsBySymbol[xSymbol][pnlKey] = result.NetWorth[len(result.NetWorth)-1] - 1.0
			if _, ok := pnls[pnlKey]; ok {
				pnls[pnlKey] += result.Turnover * (result.NetWorth[len(result.NetWorth)-1] - 1.0)
			} else {
				pnls[pnlKey] = result.Turnover * (result.NetWorth[len(result.NetWorth)-1] - 1.0)
			}
		}
	}

	keys := make([]string, 0)
	for key := range pnls {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	fmt.Printf("\n\n")
	for _, key := range keys {
		pnl := pnls[key]
		fmt.Printf("%s: %.4f\n", key, pnl)
	}
	fmt.Printf("\n\n")


	symbolsMap := map[string]string{
		"XBTUSDTM": "BTCUSDT",
	}
	for symbol := range kucoin_usdtfuture.TickSizes {
		if _, ok := binance_usdtfuture.TickSizes[strings.Replace(symbol, "USDTM", "USDT", -1)]; ok {
			symbolsMap[symbol] = strings.Replace(symbol, "USDTM", "USDT", -1)
		}
	}

	pnlKey := "0.0020-0.2000"
	for _, file := range files {
		xSymbol := strings.Split(file.Name(), "-")[0]
		if pnlsBySymbol[xSymbol][pnlKey] > -0.01 {
			fmt.Printf("  %s: %s\n", xSymbol, symbolsMap[xSymbol])
			//fmt.Printf("  %s: %.4f\n", xSymbol, pnlsBySymbol[xSymbol][pnlKey])
		}
	}

	fmt.Printf("\n\n")
	for _, file := range files {
		xSymbol := strings.Split(file.Name(), "-")[0]
		if pnlsBySymbol[xSymbol][pnlKey] <= 0 {
			fmt.Printf("%s: %.4f\n", xSymbol, pnlsBySymbol[xSymbol][pnlKey])
		}
	}
}
