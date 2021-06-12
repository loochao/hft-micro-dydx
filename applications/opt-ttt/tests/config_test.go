package tests

import (
	"fmt"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/kucoin-usdtfuture"
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
