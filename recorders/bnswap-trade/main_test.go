package main

import (
	"compress/gzip"
	"github.com/geometrybase/hft-micro/logger"
	"io/ioutil"
	"os"
	"testing"
)

func TestGzipFile(t *testing.T) {

	//archiveFiles(context.Background(), "/Users/chenjilin/MarketData/bnswap-depth20/")


	file, err := os.Open("/Users/chenjilin/Downloads/bnswap-trade/20210428-BTCUSDT.bnswap.trade.jl.gzip")
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
