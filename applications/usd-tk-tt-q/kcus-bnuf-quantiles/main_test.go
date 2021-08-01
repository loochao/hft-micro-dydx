package main

import (
	"fmt"
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
	kucoin_usdtspot "github.com/geometrybase/hft-micro/kucoin-usdtspot"
	"github.com/geometrybase/hft-micro/logger"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"math"
	"sort"
	"strings"
	"testing"
)

func TestXYPairs(t *testing.T) {
	symbols := make([]string, 0)
	for symbol := range kucoin_usdtspot.TickSizes {
		if _, ok := binance_usdtfuture.TickSizes[strings.Replace(symbol, "-USDT", "USDT", -1)]; ok {
			symbols = append(symbols, symbol)
		}
	}
	sort.Strings(symbols)
	fmt.Printf("\n\nxyPairs:\n")
	for _, symbol := range symbols {
		fmt.Printf("  %s: %s\n", symbol, strings.Replace(symbol,"-USDT", "USDT", -1))
	}
	fmt.Printf("\n")
}

func TestWeights(t *testing.T) {
	type Data struct {
		MaxOrderValues map[string]float64 `yaml:"maxOrderValues"`
	}
	data := Data{}
	contents, err := ioutil.ReadFile("/Users/chenjilin/Projects/hft-micro/applications/usd-tk-tt-q/kcus-bnuf-quantiles/config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	err = yaml.Unmarshal(contents, &data)
	if err != nil {
		t.Fatal(err)
	}
	symbols := make([]string, 0)
	sum := 0.0
	for symbol, value := range data.MaxOrderValues {
		sum += value
		symbols = append(symbols, symbol)
	}
	sort.Strings(symbols)
	fmt.Printf("\ntargetWeights:\n")
	qMean := sum/float64(len(symbols))
	for _, xSymbol := range symbols {
		weight := data.MaxOrderValues[xSymbol] / qMean
		weight = math.Sqrt(weight)
		if weight > 1.0 {
			weight = 1.0
		}
		fmt.Printf("  %s: %.5f\n", xSymbol, weight)
	}
	logger.Debugf("%v", data)
}
