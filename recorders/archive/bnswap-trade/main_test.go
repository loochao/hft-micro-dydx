package main

import (
	"compress/gzip"
	"fmt"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/logger"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"testing"
)

func TestGzipFile(t *testing.T) {
	file, err := os.Open("/Users/chenjilin/Downloads/20210503-BTCUSDT.bnspot.trade.jl.gz")
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


func TestGetSymbols(t *testing.T) {
	symbols := make([]string, 0)
	for symbol := range bnswap.TickSizes {
		symbols = append(symbols, symbol)
	}
	sort.Strings(symbols)
	fmt.Printf("%s", strings.Join(symbols,","))
}