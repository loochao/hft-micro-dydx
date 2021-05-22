package tests

import (
	"fmt"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/kcperp"
	"strings"
	"testing"
)

func TestConfig_SetDefaultIfNotSet(t *testing.T) {
	for xSymbol := range kcperp.TickSizes {
		ySymbol := strings.Replace(xSymbol, "USDTM", "USDT", -1)
		if _, ok := bnswap.TickSizes[ySymbol]; ok {
			fmt.Printf("  %s: %s\n", xSymbol, ySymbol)
		}
	}
}
