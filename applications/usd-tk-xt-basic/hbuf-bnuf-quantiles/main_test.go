package main

import (
	"fmt"
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
	huobi_usdtfuture "github.com/geometrybase/hft-micro/huobi-usdtfuture"
	"sort"
	"strings"
	"testing"
)

func TestABC(t *testing.T) {
	symbols := make([]string, 0)
	for symbol := range huobi_usdtfuture.PriceTicks {
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
