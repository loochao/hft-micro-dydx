package main

import (
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
	bybit_usdtfuture "github.com/geometrybase/hft-micro/bybit-usdtfuture"
	"github.com/geometrybase/hft-micro/logger"
	"strings"
	"testing"
)

func TestGetSymbols(t *testing.T) {
	symbols := make([]string, 0)
	for key := range bybit_usdtfuture.TickSizes {
		if _, ok := binance_usdtfuture.TickSizes[key]; ok {
			symbols = append(symbols, key)
		}
	}
	logger.Debugf("%s", strings.Join(symbols, ","))
}

