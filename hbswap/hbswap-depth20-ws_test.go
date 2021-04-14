package hbswap

import (
	"context"
	"github.com/geometrybase/hft-micro/logger"
	"log"
	"os"
	"testing"
)

func TestNewDepth20Websocket(t *testing.T) {
	var api *API
	var ctx = context.Background()
	var err error
	api, err = NewAPI(
		os.Getenv("HBSWAP_KEY"),
		os.Getenv("HBSWAP_SECRET"),
		"socks5://127.0.0.1:1081")
	if err != nil {
		log.Fatal(err)
	}
	ws := NewDepth20Websocket(ctx, api, []string{"BTC-USDT"}, "socks5://127.0.0.1:1081")
	for {
		select {
		case d := <-ws.DataCh:
			logger.Debugf("%v", d)
		}
	}
}
