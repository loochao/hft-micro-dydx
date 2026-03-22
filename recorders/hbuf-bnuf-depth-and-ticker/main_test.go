package main

import (
	"compress/gzip"
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
	huobi_usdtfuture "github.com/geometrybase/hft-micro/huobi-usdtfuture"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"strings"
	"testing"
)

func TestGetSymbols(t *testing.T) {
	symbols := make([]string, 0)
	for key := range huobi_usdtfuture.PriceTicks{
		if _, ok := binance_usdtfuture.TickSizes[strings.Replace(key, "-USDT", "USDT", -1)]; ok {
			symbols = append(symbols, key)
		}
	}
	logger.Debugf("%s", strings.Join(symbols, ","))
}

func TestCreateGzipFile(t *testing.T) {
	file, err := os.OpenFile(
		"/home/clu/Downloads/TEST.depth5.jl.gz",
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
