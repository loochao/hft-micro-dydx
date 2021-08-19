package main

import (
	"compress/gzip"
	binance_usdtspot "github.com/geometrybase/hft-micro/binance-usdtspot"
	kucoin_usdtfuture "github.com/geometrybase/hft-micro/kucoin-usdtfuture"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"strings"
	"testing"
)

func TestGetSymbols(t *testing.T) {
	symbols := []string{"XBTUSDTM"}
	for key := range kucoin_usdtfuture.TickSizes{
		if _, ok := binance_usdtspot.TickSizes[strings.Replace(key, "USDTM", "USDT", -1)]; ok {
			symbols = append(symbols, key)
		}
	}
	logger.Debugf("%d %s", len(symbols),strings.Join(symbols, ","))
}

func TestCreateGzipFile(t *testing.T) {
	file, err := os.OpenFile(
		"/Users/chenjilin/Downloads/TEST.depth5.jl.gz",
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0755,
	)
	if err != nil {
		t.Fatal(err)
	}
	gw, err := gzip.NewWriterLevel(file, gzip.BestCompression)
	if err != nil {
		logger.Debugf("gzip.NewWriterLevel error %v, stop ws", err)
		return
	}
	gw.Write([]byte(`123123`))
	gw.Write([]byte(`\n`))
	gw.Write([]byte(`123`))
	gw.Close()
	file.Close()
}
