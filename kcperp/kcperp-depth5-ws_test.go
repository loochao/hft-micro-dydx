package kcperp

import (
	"context"
	"github.com/geometrybase/hft-micro/logger"
	"log"
	"os"
	"testing"
)

func TestNewDepth5Websocket(t *testing.T) {
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
	ws := NewDepth5Websocket(ctx, api, []string{"BNBUSDTM"}, "socks5://127.0.0.1:1081")
	for {
		select {
		case d := <-ws.DataCh:
			logger.Debugf("%v", d)
		}
	}
}
