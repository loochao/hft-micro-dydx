package main

import (
	"compress/gzip"
	"fmt"
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
	kucoin_usdtfuture "github.com/geometrybase/hft-micro/kucoin-usdtfuture"
	"github.com/geometrybase/hft-micro/logger"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

//func TestArchive(t *testing.T) {
//	archiveFiles(context.Background(), "/Users/chenjilin/Desktop/leadlag-btcusdt-btcbusd/")
//}
//

func TestListSymbolPairs(t *testing.T) {
	for symbol := range binance_usdtfuture.TickSizes	 {
		if _, ok := kucoin_usdtfuture.TickSizes[symbol+"M"]; ok {
			 fmt.Printf("  \"%s\":\"%sM\",\n", symbol, symbol)
		}
	}
}

func TestGzipFile(t *testing.T) {
	symbols := make([]string, 0)
	for key := range binance_usdtfuture.TickSizes {
		symbols = append(symbols, key)
	}
	logger.Debugf("%s", strings.Join(symbols, ","))
	file, err := os.Open("/Users/chenjilin/Downloads/20210621-BTCUSDT.depth5.jl.gz")
	if err != nil {
		t.Fatal(err)
	}
	gr, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	contents, err := ioutil.ReadAll(gr)
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%s", contents)
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
