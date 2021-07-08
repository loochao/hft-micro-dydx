package kucoin_usdtspot

import (
	"context"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"testing"
)

func init() {
}

func TestNewUserWebsocket(t *testing.T) {
	var api *API
	var ctx = context.Background()
	var err error
	api, err = NewAPI(
		os.Getenv("KCSPOT_KEY"),
		os.Getenv("KCSPOT_SECRET"),
		os.Getenv("KCSPOT_PASSPHRASE"),
	 "socks5://127.0.0.1:1081")
	if err != nil {
		t.Fatal(err)
	}
	ws := NewUserWebsocket(ctx, api, "socks5://127.0.0.1:1081")
	for {
		select {
		case d := <-ws.OrderCh:
			logger.Debugf("%v", d)
		case d := <-ws.BalanceCh:
			logger.Debugf("%v", d)
		}
	}
}
