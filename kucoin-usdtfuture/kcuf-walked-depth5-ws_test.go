package kucoin_usdtfuture

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"log"
	"os"
	"testing"
)

func TestNewWalkedDepth5WS(t *testing.T) {
	var api *API
	var ctx = context.Background()
	var err error
	api, err = NewAPI(
		os.Getenv("KCPERP_KEY"),
		os.Getenv("KCPERP_SECRET"),
		os.Getenv("KCPERP_PASSPHRASE"),
		"socks5://127.0.0.1:1084")
	if err != nil {
		log.Fatal(err)
	}
	symbols := []string{"XBTUSDTM", "ATOMUSDTM", "WAVESUSDTM"}
	channels := make(map[string]chan common.Ticker)
	outputCh := make(chan common.Ticker, 128)
	for _, symbol := range symbols {
		channels[symbol] = outputCh
	}
	ws := NewWalkedDepth5WS(
		ctx, api,
		"socks5://127.0.0.1:1081",
		1000,
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
