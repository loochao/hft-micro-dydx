package tests

import (
	"fmt"
	"github.com/geometrybase/hft-micro/bnspot"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/kucoin-usdtfuture"
	"sort"
	"strings"
	"testing"
)

func TestConfig_SetDefaultIfNotSet(t *testing.T) {
	for xSymbol := range kucoin_usdtfuture.TickSizes {
		ySymbol := strings.Replace(xSymbol, "USDTM", "USDT", -1)
		if _, ok := bnswap.TickSizes[ySymbol]; ok {
			fmt.Printf("  %s: %s\n", xSymbol, ySymbol)
		}
	}
}

func TestBnspotBnswapSymbols(t *testing.T) {
	symbols := make([]string, 0)
	for symbol := range bnswap.TickSizes {
		if _, ok := bnspot.TickSizes[symbol]; ok {
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

