package main

import (
	"fmt"
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
	kucoin_usdtspot "github.com/geometrybase/hft-micro/kucoin-usdtspot"
	"sort"
	"strings"
	"testing"
)

func TestGetSymbols(t *testing.T) {
	symbols := make([]string, 0)
	for key := range kucoin_usdtspot.TickSizes {
		if _, ok := binance_usdtfuture.TickSizes[strings.Replace(key, "-USDT", "USDT", -1)]; ok {
			symbols = append(symbols, key)
		}
	}
	sort.Strings(symbols)
	fmt.Printf("%s", strings.Join(symbols, ","))
}

