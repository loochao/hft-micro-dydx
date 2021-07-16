package main

import (
	"compress/gzip"
	"github.com/geometrybase/hft-micro/logger"
	"io/ioutil"
	"os"
	"testing"
)

func TestGzipFile(t *testing.T) {

	//archiveFiles(context.Background(), "/Users/chenjilin/MarketData/bnspot-depth20/")


	file, err := os.Open("/Users/chenjilin/Downloads/bnspot-trade/20210428-BTCUSDT.bnspot.trade.jl.gzip")
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
