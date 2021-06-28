package main

import (
	"fmt"
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
	kucoin_usdtfuture "github.com/geometrybase/hft-micro/kucoin-usdtfuture"
	"math"
	"sort"
	"testing"
)

var maxOrderValues = map[string]float64{"1INCHUSDTM": 3006, "AAVEUSDTM": 2125, "ADAUSDTM": 10660, "ALGOUSDTM": 9706, "ATOMUSDTM": 1241, "AVAXUSDTM": 3578, "BANDUSDTM": 918, "BATUSDTM": 1424, "BCHUSDTM": 11720, "BTTUSDTM": 3000, "CHZUSDTM": 4669, "COMPUSDTM": 1041, "CRVUSDTM": 6164, "DASHUSDTM": 1205, "DENTUSDTM": 1800, "DGBUSDTM": 410, "DOGEUSDTM": 8750, "DOTUSDTM": 5951, "ENJUSDTM": 4484, "EOSUSDTM": 22451, "ETCUSDTM": 3655, "ETHUSDTM": 31843, "FILUSDTM": 7084, "FTMUSDTM": 3530, "GRTUSDTM": 747, "ICPUSDTM": 11384, "IOSTUSDTM": 1400, "KSMUSDTM": 1614, "LINKUSDTM": 6555, "LTCUSDTM": 9725, "LUNAUSDTM": 2576, "MANAUSDTM": 4390, "MATICUSDTM": 10415, "MKRUSDTM": 942, "NEOUSDTM": 3172, "OCEANUSDTM": 5432, "ONTUSDTM": 1432, "QTUMUSDTM": 1815, "RVNUSDTM": 670, "SNXUSDTM": 882, "SOLUSDTM": 4198, "SUSHIUSDTM": 4754, "SXPUSDTM": 3567, "THETAUSDTM": 4688, "TRXUSDTM": 12250, "UNIUSDTM": 2742, "VETUSDTM": 5950, "WAVESUSDTM": 1803, "XBTUSDTM": 27013, "XEMUSDTM": 2811, "XLMUSDTM": 2465, "XMRUSDTM": 1717, "XRPUSDTM": 41450, "XTZUSDTM": 5068, "YFIUSDTM": 1922, "ZECUSDTM": 1500}

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
