package kcspot

import (
	"context"
	"github.com/geometrybase/hft-micro/logger"
	"log"
	"os"
	"testing"
	"time"
)

func init() {
}

func TestNewDepth5Websocket(t *testing.T) {
	var api *API
	var ctx = context.Background()
	var err error
	api, err = NewAPI(
		os.Getenv("KCPERP_KEY"),
		os.Getenv("KCPERP_SECRET"),
		os.Getenv("KCPERP_PASSPHRASE"),
		"socks5://127.0.0.1:1081")
	if err != nil {
		log.Fatal(err)
	}
	ws := NewDepth5Websocket(ctx, api, []string{"ENJ-USDT"},  "socks5://127.0.0.1:1081" )
	for {
		select {
		case d := <- ws.DataCh:
			logger.Debugf("%v %v", d.EventTime, time.Now().Sub(d.EventTime))
		}
	}
}

