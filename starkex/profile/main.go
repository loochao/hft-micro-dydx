package main

import (
	"fmt"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/geometrybase/hft-micro/starkex"
	"math/big"
	"os"
	"runtime"
	"runtime/pprof"
	"time"
)


const (
	MOCK_SIGNATURE         = "00cecbe513ecdbf782cd02b2a5efb03e58d5f63d15f2b840e9bc0029af04e8dd0090b822b16f50b2120e4ea9852b340f7936ff6069d02acca02f2ed03029ace5"
	MOCK_PUBLIC_KEY_EVEN_Y = "5c749cd4c44bdc730bc90af9bfbdede9deb2c1c96c05806ce1bc1cb4fed64f7"
	MOCK_SIGNATURE_EVEN_Y  = "00fc0756522d78bef51f70e3981dc4d1e82273f59cdac6bc31c5776baabae6ec0158963bfd45d88a99fb2d6d72c9bbcf90b24c3c0ef2394ad8d05f9d3983443a"
)

var MOCK_PUBLIC_KEY, _ = new(big.Int).SetString("3b865a18323b8d147a12c556bfb1d502516c325b1477a23ba6c77af31f020fd", 16)
var MOCK_PRIVATE_KEY, _ = new(big.Int).SetString("58c7d5a90b1776bde86ebac077e053ed85b0f7164f53b080304a531947f46e3", 16)

func main() {
	var cpuProfFile *os.File
	var err error
	cpuProfFile, err = os.Create("/Users/chenjilin/Downloads/starkex"+time.Now().Format("-060102.cpu.prof"))
	if err != nil {
		logger.Warnf("os.Create error %v", err)
		return
	}
	err = pprof.StartCPUProfile(cpuProfFile)
	if err != nil {
		logger.Warnf("pprof.StartCPUProfile error %v", err)
		return
	}

	tt, err := time.Parse("2006-01-02T15:04:05.999Z", "2020-09-17T04:15:55.028Z")
	if err != nil {
		panic(err)
	}
	var so *starkex.StarkwareOrder
	for i := 0; i < 10000; i++ {
		fmt.Printf("%d\n", i)
		so, err = starkex.NewStarkwareOrder(
			starkex.NETWORK_ID_ROPSTEN,
			starkex.MARKET_ETH_USD,
			starkex.ORDER_SIDE_BUY,
			12345,
			145.0005,
			350.00067,
			0.125,
			"This is an ID that the client came up with to describe this order",
			tt.Unix(),
		)
		if err != nil {
			logger.Debugf("%v", err)
			continue
		}
		_, err = so.Sign(MOCK_PRIVATE_KEY)
		if err != nil {
			logger.Debugf("%v", err)
		}
	}


	pprof.StopCPUProfile()
	var heapProfFile *os.File
	runtime.GC() // profile all outstanding allocations
	if heapProfFile, err = os.Create("/Users/chenjilin/Downloads/starkex"+time.Now().Format("-060102.heap.prof")); err != nil {
		logger.Warnf("os.Create error %v",  err)
	} else if err = pprof.WriteHeapProfile(heapProfFile); err != nil {
		logger.Warnf("pprof.WriteHeapProfile error %v", err)
	}
}
