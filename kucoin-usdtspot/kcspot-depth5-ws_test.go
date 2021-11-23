package kucoin_usdtspot

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"log"
	"os"
	"testing"
)



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
	symbols := []string{"ENJ-USDT", "WAVES-USDT"}
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
			logger.Debugf("%s %v", d.GetSymbol(), d.GetEventTime())
		}
	}
}


