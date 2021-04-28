package main

import (
	"context"
	"testing"
)

func TestGzipFile(t *testing.T) {

	archiveFiles(context.Background(), "/Users/chenjilin/MarketData/bnswap-depth20/")


	//file, err := os.Open("/Users/chenjilin/MarketData/bnswap-depth20/2021042805-BTCUSDT.depth20.jl.gzip")
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
