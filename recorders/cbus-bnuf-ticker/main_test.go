package main

import (
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
	coinbase_usdspot "github.com/geometrybase/hft-micro/coinbase-usdspot"
	"github.com/geometrybase/hft-micro/logger"
	"strings"
	"testing"
)

func TestGetSymbols(t *testing.T) {
	symbols := make([]string, 0)
	for key := range coinbase_usdspot.TickSizes{
		if _, ok := binance_usdtfuture.TickSizes[strings.Replace(key, "-USD", "USDT", -1)]; ok {
			symbols = append(symbols, key)
		}
	}
	logger.Debugf("%s", strings.Join(symbols, ","))
}

