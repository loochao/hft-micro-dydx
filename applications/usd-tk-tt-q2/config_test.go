package main

import (
	"fmt"
	binance_busdspot "github.com/geometrybase/hft-micro/binance-busdspot"
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
	binance_usdtspot "github.com/geometrybase/hft-micro/binance-usdtspot"
	dydx_usdfuture "github.com/geometrybase/hft-micro/dydx-usdfuture"
	kucoin_usdtfuture "github.com/geometrybase/hft-micro/kucoin-usdtfuture"
	okexv5_usdtspot "github.com/geometrybase/hft-micro/okexv5-usdtspot"
	okexv5_usdtswap "github.com/geometrybase/hft-micro/okexv5-usdtswap"
	"math"
	"sort"
	"strings"
	"testing"
)

func TestShowDydxBnufPairsAndMaxSizes(t *testing.T) {
	symbolMap := make(map[string]string)
	maxPosSizes := make(map[string]float64)
	symbols := make([]string, 0)
	for xSymbol, xMaxPosSize := range dydx_usdfuture.MaxPosSizes {
		ySymbol := strings.Replace(xSymbol, "-USD", "USDT", -1)
		if yMaxPosSize, ok := binance_usdtfuture.MaxPosSizes[ySymbol]; ok {
			symbols = append(symbols, xSymbol)
			symbolMap[xSymbol] = ySymbol
			maxPosSizes[xSymbol] = math.Min(xMaxPosSize, yMaxPosSize) * 0.5
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

func TestShowOkusOkufPairsAndMaxSizes(t *testing.T) {
	symbolMap := make(map[string]string)
	maxPosSizes := make(map[string]float64)
	symbols := make([]string, 0)
	for ySymbol, yMaxPosSize := range okexv5_usdtswap.MaxSizes {
		xSymbol := strings.Replace(ySymbol, "USDT-SWAP", "USDT", -1)
		bnufSymbol := strings.Replace(ySymbol, "-USDT-SWAP", "USDT", -1)
		_, ok1 := okexv5_usdtspot.TickSizes[xSymbol]
		_, ok2 := binance_usdtfuture.TickSizes[bnufSymbol]
		if ok1 && ok2 {
			symbols = append(symbols, xSymbol)
			symbolMap[xSymbol] = ySymbol
			maxPosSizes[xSymbol] = yMaxPosSize * okexv5_usdtswap.Multipliers[ySymbol] * 0.5
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

func TestShowOkufKcufPairsAndMaxSizes(t *testing.T) {
	symbolMap := make(map[string]string)
	maxPosSizes := make(map[string]float64)
	symbols := make([]string, 0)
	for ySymbol, yMaxPosSize := range okexv5_usdtswap.MaxSizes {
		xSymbol := strings.Replace(ySymbol, "-USDT-SWAP", "USDTM", -1)
		if ySymbol == "BTC-USDT-SWAP" {
			xSymbol = "XBTUSDTM"
		}
		bnufSymbol := strings.Replace(ySymbol, "-USDT-SWAP", "USDT", -1)
		_, ok1 := kucoin_usdtfuture.TickSizes[xSymbol]
		_, ok2 := binance_usdtfuture.TickSizes[bnufSymbol]
		if ok1 && ok2 {
			symbols = append(symbols, xSymbol)
			symbolMap[xSymbol] = ySymbol
			maxPosSizes[xSymbol] = yMaxPosSize * okexv5_usdtswap.Multipliers[ySymbol] * 0.25
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

func TestShowBnusBnufPairsAndMaxSizes(t *testing.T) {
	symbolMap := make(map[string]string)
	maxPosSizes := make(map[string]float64)
	symbols := make([]string, 0)
	for xSymbol := range binance_usdtspot.TickSizes {
		ySymbol := strings.Replace(xSymbol, "USDT", "USDT", -1)
		if yMaxPosSize, ok := binance_usdtfuture.MaxPosSizes[ySymbol]; ok {
			symbols = append(symbols, xSymbol)
			symbolMap[xSymbol] = ySymbol
			maxPosSizes[xSymbol] = yMaxPosSize * 0.5
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


func TestShowBnbsBnufPairsAndMaxSizes(t *testing.T) {
	symbolMap := make(map[string]string)
	maxPosSizes := make(map[string]float64)
	symbols := make([]string, 0)
	for xSymbol := range binance_busdspot.TickSizes {
		ySymbol := strings.Replace(xSymbol, "BUSD", "USDT", -1)
		if yMaxPosSize, ok := binance_usdtfuture.MaxPosSizes[ySymbol]; ok {
			symbols = append(symbols, xSymbol)
			symbolMap[xSymbol] = ySymbol
			maxPosSizes[xSymbol] = yMaxPosSize * 0.5
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
