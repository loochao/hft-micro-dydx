package bitfinex_usdtfuture

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"testing"
)

func TestNewTickerWS(t *testing.T) {
	var api *API
	var ctx = context.Background()
	symbols := []string{"BTCF0:USTF0"}
	channels := make(map[string]chan common.Ticker)
	outputCh := make(chan common.Ticker, 4)
	for _, symbol := range symbols {
		channels[symbol] = outputCh
	}
	_ = NewTickerWS(
		ctx, api,
		"socks5://127.0.0.1:1081",
		channels,
	)
	for {
		select {
		case d := <-outputCh:
			logger.Debugf("%v", d)
		}
	}
}
