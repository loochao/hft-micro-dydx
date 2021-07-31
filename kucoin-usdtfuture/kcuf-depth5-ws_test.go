package kucoin_usdtfuture

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"log"
	"os"
	"testing"
)

var Global_D *Depth5

func BenchmarkParseDepthWithAllocate(b *testing.B) {
	msg := []byte(`{"data":{"sequence":1619017805158,"asks":[[299.64,553],[299.71,56],[299.74,489],[299.77,473],[299.80,830]],"bids":[[299.45,20],[299.39,217],[299.29,212],[299.22,196],[299.17,194]],"ts":1623511662489,"timestamp":1623511662489},"subject":"level2","topic":"/contractMarket/level2Depth5:COMPUSDTM","type":"message"}`)
	var depth *Depth5
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		depth = &Depth5{}
		_ = ParseDepth5(msg, depth)
	}
	Global_D = depth
}

func BenchmarkParseDepth5WithNoAllocate(b *testing.B) {
	msg := []byte(`{"data":{"sequence":1619017805158,"asks":[[299.64,553],[299.71,56],[299.74,489],[299.77,473],[299.80,830]],"bids":[[299.45,20],[299.39,217],[299.29,212],[299.22,196],[299.17,194]],"ts":1623511662489,"timestamp":1623511662489},"subject":"level2","topic":"/contractMarket/level2Depth5:COMPUSDTM","type":"message"}`)
	depth := &Depth5{}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = ParseDepth5(msg, depth)
	}
	Global_D = depth
}

func BenchmarkParseDepth5WithPool(b *testing.B) {
	msg := []byte(`{"data":{"sequence":1619017805158,"asks":[[299.64,553],[299.71,56],[299.74,489],[299.77,473],[299.80,830]],"bids":[[299.45,20],[299.39,217],[299.29,212],[299.22,196],[299.17,194]],"ts":1623511662489,"timestamp":1623511662489},"subject":"level2","topic":"/contractMarket/level2Depth5:COMPUSDTM","type":"message"}`)
	pool := [1024]*Depth5{}
	for i := 0; i < 1024; i++ {
		pool[i] = &Depth5{}
	}
	var depth *Depth5
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		depth = pool[i%1024]
		_ = ParseDepth5(msg, depth)
	}
	Global_D = depth
}

func TestNewDepth5WS(t *testing.T) {
	var api *API
	var ctx = context.Background()
	var err error
	api, err = NewAPI(
		os.Getenv("KCPERP_KEY"),
		os.Getenv("KCPERP_SECRET"),
		os.Getenv("KCPERP_PASSPHRASE"),
		"socks5://127.0.0.1:1080")
	if err != nil {
		log.Fatal(err)
	}
	symbols := []string{"XBTUSDTM", "ATOMUSDTM", "WAVESUSDTM"}
	//symbols := make([]string, 0)
	//for symbol := range TickSizes {
	//	symbols = append(symbols, symbol)
	//}
	channels := make(map[string]chan common.Depth)
	outputCh := make(chan common.Depth, 128)
	for _, symbol := range symbols {
		channels[symbol] = outputCh
	}
	ws := NewDepth5WS(
		ctx, api,
		"socks5://127.0.0.1:1081",
		channels,
	)
	for {
		select {
		case <- ws.Done():
			return
		case d := <-outputCh:
			logger.Debugf("%s %v", d.GetSymbol(), d)
		}
	}
}
