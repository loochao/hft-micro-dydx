package main

import (
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	kucoin_usdtfuture "github.com/geometrybase/hft-micro/kucoin-usdtfuture"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"runtime"
	"runtime/pprof"
	"sort"
)

var outputCh = make(chan common.Ticker, 128)

func init() {
	var api *kucoin_usdtfuture.API
	var ctx = context.Background()
	var err error
	api, err = kucoin_usdtfuture.NewAPI(
		os.Getenv("KCPERP_KEY"),
		os.Getenv("KCPERP_SECRET"),
		os.Getenv("KCPERP_PASSPHRASE"),
		"socks5://127.0.0.1:1080")
	if err != nil {
		panic(err)
	}
	symbols := []string{}
	for symbol := range kucoin_usdtfuture.TickSizes {
		symbols = append(symbols, symbol)
	}
	sort.Strings(symbols)
	symbols = symbols[:50]
	channels := make(map[string]chan common.Ticker)
	for _, symbol := range symbols {
		channels[symbol] = outputCh
	}
	_ = kucoin_usdtfuture.NewFastDepth5TickerWS(
		ctx, api,
		"socks5://127.0.0.1:1081",
		channels,
	)
	counter := 0
	for counter < 1000 {
		counter++
		select {
		case <-outputCh:
		}
	}
}

func main() {
	var err error
	var cpuProfFile *os.File
	cpuProfFile, err = os.Create("/Users/chenjilin/Projects/hft-micro/benchmarks/outputs/kcuf-fast-depth5-ticker-ws.cpu.prof")
	if err != nil {
		panic(err)
	}
	err = pprof.StartCPUProfile(cpuProfFile)
	if err != nil {
		panic(err)
	}
	defer cpuProfFile.Close()
	defer pprof.StopCPUProfile()

	defer func(){
		logger.Debugf("stop heap")
		var heapProfFile *os.File
		runtime.GC()
		if heapProfFile, err = os.Create("/Users/chenjilin/Projects/hft-micro/benchmarks/outputs/kcuf-fast-depth5-ticker-ws.heap.prof"); err != nil {
			panic(err)
		} else if err = pprof.WriteHeapProfile(heapProfFile); err != nil {
			panic(err)
		}
		heapProfFile.Close()
	}()
	for i := 0; i < 100000; i++ {
		if i % 1000 == 0 {
			fmt.Printf("%d\n", i)
		}
		select {
		case <-outputCh:
		}
	}
}
