package main

import (
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
	kucoin_usdtfuture "github.com/geometrybase/hft-micro/kucoin-usdtfuture"
	"github.com/geometrybase/hft-micro/logger"
	okexv5_usdtswap "github.com/geometrybase/hft-micro/okexv5-usdtswap"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestGenerateTDs(t *testing.T) {
	symbolMap := make(map[string]string)
	maxPosSizes := make(map[string]float64)
	symbols := make([]string, 0)
	for xSymbol, xMaxPosSize := range okexv5_usdtswap.MaxSizes {
		ySymbol := strings.Replace(xSymbol, "-USDT-SWAP", "USDTM", -1)
		if xSymbol == "BTC-USDT-SWAP" {
			ySymbol = "XBTUSDTM"
		}
		bnufSymbol := strings.Replace(xSymbol, "-USDT-SWAP", "USDT", -1)
		_, ok1 := kucoin_usdtfuture.TickSizes[ySymbol]
		_, ok2 := binance_usdtfuture.TickSizes[bnufSymbol]
		if ok1 && ok2 {
			symbols = append(symbols, xSymbol)
			symbolMap[xSymbol] = ySymbol
			maxPosSizes[xSymbol] = xMaxPosSize * okexv5_usdtswap.Multipliers[xSymbol] * 0.25
		}
	}
	sort.Strings(symbols)
	//fmt.Printf("\n\nxyPairs:\n")
	//for _, xSymbol := range symbols {
	//	fmt.Printf("  %s: %s\n", xSymbol, symbolMap[xSymbol])
	//}
	//fmt.Printf("\n\nmaxPosSizes:\n")
	//for _, xSymbol := range symbols {
	//	fmt.Printf("  %s: %.0f\n", xSymbol, maxPosSizes[xSymbol])
	//}
	var startTime, endTime time.Time
	var err error

	//xPath := ""
	if startTime, err = time.Parse("20060102", "20210718"); err != nil {
		t.Fatal(err)
	}
	if endTime, err = time.Parse("20060102", "20210721"); err != nil {
		t.Fatal(err)
	}
	dateStrs := ""
	for i := startTime; i.Sub(endTime) <= 0; i = i.Add(time.Hour * 24) {
		dateStrs += i.Format("20060102,")
	}
	dateStrs = dateStrs[:len(dateStrs)-1]
	logger.Debugf("%s", dateStrs)

	for _, xSymbol := range symbols {
		ySymbol := symbolMap[xSymbol]
		logger.Debugf("%s %s", xSymbol, ySymbol)
	}
}
