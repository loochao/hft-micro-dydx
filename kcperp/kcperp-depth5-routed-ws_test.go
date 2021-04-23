package kcperp

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"log"
	"os"
	"testing"
)

func TestNewDepth5RoutedWebsocket(t *testing.T) {
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
	symbols := []string{"XBTUSDTM", "ATOMUSDTM", "WAVESUSDTM"}
	channels := make(map[string]chan *common.DepthRawMessage)
	for _, symbol := range symbols {
		channels[symbol] = make(chan *common.DepthRawMessage, 1000)
	}
	reportCh := make(chan common.DepthReport, 100)
	_ = NewDepth5RoutedWebsocket(
		ctx, api,
		"socks5://127.0.0.1:1081",
		0.9999, 1000,
		50,
		reportCh,
		channels,
	)
	for {
		select {
		case r := <-reportCh:
			logger.Debugf("%v", r)
		case <-channels[symbols[0]]:
			//logger.Debugf("%v", d)
		case <-channels[symbols[1]]:
			//logger.Debugf("%v", d)
		case <-channels[symbols[2]]:
			//logger.Debugf("%v", d)
		}
	}
}
