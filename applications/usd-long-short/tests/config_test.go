package tests

import (
	"fmt"
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
	binance_usdtspot "github.com/geometrybase/hft-micro/binance-usdtspot"
	"github.com/geometrybase/hft-micro/kucoin-usdtfuture"
	"sort"
	"strings"
	"testing"
)

func TestConfig_SetDefaultIfNotSet(t *testing.T) {
	for xSymbol := range kucoin_usdtfuture.TickSizes {
		ySymbol := strings.Replace(xSymbol, "USDTM", "USDT", -1)
		if _, ok := binance_usdtfuture.TickSizes[ySymbol]; ok {
			fmt.Printf("  %s: %s\n", xSymbol, ySymbol)
		}
	}
}

func TestBnspotBnswapSymbols(t *testing.T) {
	symbols := make([]string, 0)
	for symbol := range binance_usdtfuture.TickSizes {
		if _, ok := binance_usdtspot.TickSizes[symbol]; ok {
			symbols = append(symbols, symbol)
		}
	}
	sort.Strings(symbols)
	for _, symbol := range symbols {
		fmt.Printf("  %s: %s\n", symbol, symbol)
	}
	fmt.Printf("\n")
	fmt.Printf("%s", strings.Join(symbols, ","))
}
