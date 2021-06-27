package main

import (
	"fmt"
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
	kucoin_usdtfuture "github.com/geometrybase/hft-micro/kucoin-usdtfuture"
	"math"
	"sort"
	"testing"
)

var maxOrderValues = map[string]float64{
	"1INCHUSDTM": 6012,
	"AAVEUSDTM":  425071,
	"ADAUSDTM":   2132,
	"ALGOUSDTM":  19412,
	"ATOMUSDTM":  24818,
	"AVAXUSDTM":  71560,
	"BANDUSDTM":  18368,
	"BATUSDTM":   2849,
	"BCHUSDTM":   2343938,
	"BNBUSDTM":   1224789,
	"BTTUSDTM":   6,
	"CHZUSDTM":   9338,
	"COMPUSDTM":  208141,
	"CRVUSDTM":   12327,
	"DASHUSDTM":  241010,
	"DENTUSDTM":  36,
	"DGBUSDTM":   82,
	"DOGEUSDTM":  175,
	"DOTUSDTM":   11902,
	"ENJUSDTM":   8967,
	"EOSUSDTM":   44902,
	"ETCUSDTM":   73102,
	"ETHUSDTM":   6368636,
	"FILUSDTM":   141682,
	"FTMUSDTM":   7059,
	"GRTUSDTM":   1494,
	"ICPUSDTM":   2276734,
	"IOSTUSDTM":  28,
	"KSMUSDTM":   322798,
	"LINKUSDTM":  131095,
	"LTCUSDTM":   194498,
	"LUNAUSDTM":  5153,
	"MANAUSDTM":  8779,
	"MATICUSDTM": 2083,
	"MKRUSDTM":   1884423,
	"NEOUSDTM":   63434,
	"OCEANUSDTM": 10865,
	"ONTUSDTM":   2863,
	"QTUMUSDTM":  36295,
	"RVNUSDTM":   134,
	"SNXUSDTM":   17634,
	"SOLUSDTM":   83966,
	"SUSHIUSDTM": 9507,
	"SXPUSDTM":   7134,
	"THETAUSDTM": 93764,
	"TRXUSDTM":   245,
	"UNIUSDTM":   5483,
	"VETUSDTM":   119,
	"WAVESUSDTM": 36058,
	"XBTUSDTM":   54026363,
	"XEMUSDTM":   5622,
	"XLMUSDTM":   493,
	"XMRUSDTM":   343485,
	"XRPUSDTM":   8290,
	"XTZUSDTM":   10137,
	"YFIUSDTM":   38446533,
	"ZECUSDTM":   299967,
}

func TestPrintPairs(t *testing.T) {
	symbols := make([]string, 0)
	for kSymbol := range maxOrderValues {
		symbols = append(symbols, kSymbol)
	}
	sort.Strings(symbols)
	for _, kSymbol := range symbols {
		bSymbol := kSymbol[:len(kSymbol)-1]
		if _, ok := binance_usdtfuture.TickSizes[bSymbol]; ok {
			fmt.Printf("\"%s\": \"%s\",\n", kSymbol, bSymbol)
		}
	}
}

func TestMaxOrderSize(t *testing.T) {
	weights := make(map[string]float64)
	totalSize := 0.0
	symbols := make([]string, 0)
	for kSymbol := range maxOrderValues {
		symbols = append(symbols, kSymbol)
	}
	sort.Strings(symbols)
	fmt.Printf("\n\n\n")
	for _, symbol := range symbols {
		size := maxOrderValues[symbol]
		fmt.Printf("%s: %.0f\n", symbol, size*kucoin_usdtfuture.Multipliers[symbol]*0.5)
		totalSize += size
	}
	fmt.Printf("\n\n\n")
	meanSize := totalSize / float64(len(maxOrderValues))
	for symbol, size := range maxOrderValues {
		weights[symbol] = math.Sqrt(size / meanSize)
		if weights[symbol] > 1 {
			weights[symbol] = 1.0
		}
	}
	for _, symbol := range symbols {
		fmt.Printf("%s: %.2f\n", symbol, weights[symbol])
	}
	sort.Strings(symbols)
	for _, symbol := range symbols {
		fmt.Printf("  %s: %s\n", symbol, symbol)
	}
}
