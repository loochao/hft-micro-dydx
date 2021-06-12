package kucoin_usdtfuture

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"log"
	"os"
	"testing"
	"time"
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
	//symbols := []string{"XBTUSDTM", "ATOMUSDTM", "WAVESUSDTM"}
	symbols := []string{"CRVUSDTM", "XBTUSDTM"}
	channels := make(map[string]chan *common.DepthRawMessage)
	for _, symbol := range symbols {
		channels[symbol] = make(chan *common.DepthRawMessage, 1000)
	}
	reportCh := make(chan common.TimeReport, 100)
	_ = NewDepth5RoutedWebsocket(
		ctx, api,
		"socks5://127.0.0.1:1081",
		channels,
	)
	for {
		select {
		case r := <-reportCh:
			logger.Debugf("%v", r)
		case d := <-channels[symbols[0]]:
			logger.Debugf("CRV %v %v", d.Time, time.Now().Sub(d.Time))
		case d := <-channels[symbols[1]]:
			logger.Debugf("XBT %v %v", d.Time, time.Now().Sub(d.Time))
			//logger.Debugf("%v", d)
		//case <-channels[symbols[2]]:
			//logger.Debugf("%v", d)
		}
	}
}
