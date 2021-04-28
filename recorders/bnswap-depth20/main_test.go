package main

import (
	"compress/gzip"
	"github.com/geometrybase/hft-micro/logger"
	"io/ioutil"
	"os"
	"testing"
)

func TestGzipFile(t *testing.T) {
	file, err := os.Open("/Users/chenjilin/Downloads/depth20/20210428-BTCUSDT.depth20.jl.gzip")
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
