package main

import (
	"compress/gzip"
	"context"
	"github.com/geometrybase/hft-micro/logger"
	"io/ioutil"
	"os"
	"testing"
)


func TestArchive(t *testing.T) {
	archiveFiles(context.Background(), "/Users/chenjilin/Desktop/leadlag-btcusdt-btcbusd/")
}

func TestGzipFile(t *testing.T) {



	file, err := os.Open("/Users/chenjilin/Desktop/leadlag-btcusdt-btcbusd/20210620-BTCUSDT,BTCBUSD.depth5.jl.gz")
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
