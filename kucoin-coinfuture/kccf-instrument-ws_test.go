package kucoin_coinfuture

import (
	"context"
	"github.com/geometrybase/hft-micro/logger"
	"log"
	"os"
	"testing"
)

func TestNewInstrumentWebsocket(t *testing.T) {
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
	ws := NewInstrumentWebsocket(ctx, api, []string{"BNBUSDTM"}, "socks5://127.0.0.1:1081",nil  )
	for {
		select {
		case d := <-ws.MarkPriceCh:
			logger.Debugf("%v", d)
			break
		}
	}
}
