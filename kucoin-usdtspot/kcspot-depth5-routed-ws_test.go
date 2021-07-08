package kucoin_usdtspot

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
		os.Getenv("KCSPOT_KEY"),
		os.Getenv("KCSPOT_SECRET"),
		os.Getenv("KCSPOT_PASSPHRASE"),
		"socks5://127.0.0.1:1081")
	if err != nil {
		log.Fatal(err)
	}
	//symbols := []string{"CRV-USDT", "ATOM-USDT", "WAVES-USDT"}
	symbols := []string{"CRV-USDT"}
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
		case <-channels[symbols[0]]:
			//logger.Debugf("%v", d)
		case <-channels[symbols[1]]:
			//logger.Debugf("%v", d)
		case <-channels[symbols[2]]:
			//logger.Debugf("%v", d)
		}
	}
}
