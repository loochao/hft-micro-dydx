package main

import (
	"compress/gzip"
	"github.com/geometrybase/hft-micro/logger"
	"io/ioutil"
	"os"
	"testing"
)

func TestGzipFile(t *testing.T) {

	file, err := os.Open("/Users/chenjilin/Downloads/ftx-usdtfuture-trade/20210509-ETH-PERP.ftx-usdtfuture.trade.jl.gz")
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
