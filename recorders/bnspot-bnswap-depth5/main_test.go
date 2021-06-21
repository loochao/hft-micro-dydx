package main

import (
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
	"github.com/geometrybase/hft-micro/logger"
	"strings"
	"testing"
)

//func TestArchive(t *testing.T) {
//	archiveFiles(context.Background(), "/Users/chenjilin/Desktop/leadlag-btcusdt-btcbusd/")
//}
//
func TestGzipFile(t *testing.T) {
	symbols := make([]string, 0)
	for key := range binance_usdtfuture.TickSizes {
		symbols = append(symbols, key)
	}
	logger.Debugf("%s", strings.Join(symbols,","))
	//file, err := os.Open("/Users/chenjilin/Downloads/20210621-BTCUSDT.depth5.jl.gz")
	//if err != nil {
	//	t.Fatal(err)
	//}
	//gr, err := gzip.NewReader(file)
	//if err != nil {
	//	t.Fatal(err)
	//}
	//contents, err := ioutil.ReadAll(gr)
	//if err != nil {
	//	t.Fatal(err)
	//}
	//logger.Debugf("%s", contents)
}
