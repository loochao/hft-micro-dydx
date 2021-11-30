package kucoin_usdtfuture

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"log"
	"os"
	"testing"
)

func TestNewDepth5TickerWS(t *testing.T) {
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
		case <-ws.Done():
			return
		case d := <-outputCh:
			logger.Debugf("%s %v", d.GetSymbol(), d)
		}
	}
}

//var outputCh = make(chan common.Ticker, 128)
//
//func init() {
//	var api *API
//	var ctx = context.Background()
//	var err error
//	api, err = NewAPI(
//		os.Getenv("KCPERP_KEY"),
//		os.Getenv("KCPERP_SECRET"),
//		os.Getenv("KCPERP_PASSPHRASE"),
//		"socks5://127.0.0.1:1080")
//	if err != nil {
//		panic(err)
//	}
//	symbols := []string{"XBTUSDTM", "ATOMUSDTM", "WAVESUSDTM"}
//	channels := make(map[string]chan common.Ticker)
//	for _, symbol := range symbols {
//		channels[symbol] = outputCh
//	}
//	_ = NewDepth5TickerWS(
//		ctx, api,
//		"socks5://127.0.0.1:1081",
//		channels,
//	)
//	counter := 0
//	for counter < 100 {
//		logger.Debugf("pre counter %d", counter)
//		counter++
//		select {
//		case <-outputCh:
//		}
//	}
//}
//
//func BenchmarkNewDepth5TickerWS(b *testing.B) {
//	logger.Debugf("start benchmark \n\n")
//	b.ResetTimer()
//	b.ReportAllocs()
//	for i := 0; i < 1000; i++ {
//		select {
//		case <-outputCh:
//		}
//	}
//}
