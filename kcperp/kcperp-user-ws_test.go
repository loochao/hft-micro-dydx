package kcperp

import (
	"context"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"testing"
)

func TestNewUserWebsocket(t *testing.T) {
	var api *API
	var ctx = context.Background()
	var err error
	api, err = NewAPI(
		os.Getenv("KCPERP_KEY"),
		os.Getenv("KCPERP_SECRET"),
		os.Getenv("KCPERP_PASSPHRASE"),
		"socks5://127.0.0.1:1081")
	if err != nil {
		t.Fatal(err)
	}
	ws := NewUserWebsocket(ctx, api, []string{"XBTUSDM"}, "socks5://127.0.0.1:1081")
	for {
		select {
		case d := <-ws.OrderCh:
			logger.Debugf("%v", d)
		case d := <-ws.BalanceCh:
			logger.Debugf("%v", d)
		}
	}
}
