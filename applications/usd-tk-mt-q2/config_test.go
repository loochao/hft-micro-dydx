package main

import (
	"fmt"
	binance_busdspot "github.com/geometrybase/hft-micro/binance-busdspot"
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
	"sort"
	"strings"
	"testing"
)

func TestShowBnusBnufPairsAndMaxSizes(t *testing.T) {
	symbolMap := make(map[string]string)
	maxPosSizes := make(map[string]float64)
	symbols := make([]string, 0)
	for xSymbol := range binance_busdspot.TickSizes {
		ySymbol := strings.Replace(xSymbol, "BUSD", "USDT", -1)
		if yMaxPosSize, ok := binance_usdtfuture.MaxPosSizes[ySymbol]; ok {
			symbols = append(symbols, xSymbol)
			symbolMap[xSymbol] = ySymbol
			maxPosSizes[xSymbol] = yMaxPosSize
		}
	}
	sort.Strings(symbols)
	fmt.Printf("\n\nxyPairs:\n")
	for _, xSymbol := range symbols {
		fmt.Printf("  %s: %s\n", xSymbol, symbolMap[xSymbol])
	}
	fmt.Printf("\n\nmaxPosSizes:\n")
	for _, xSymbol := range symbols {
		fmt.Printf("  %s: %.0f\n", xSymbol, maxPosSizes[xSymbol])
	}
}
