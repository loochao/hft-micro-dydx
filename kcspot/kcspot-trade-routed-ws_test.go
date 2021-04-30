package kcspot

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"testing"
)

func TestNewTradeRoutedWS(t *testing.T) {
	var ctx = context.Background()
	symbols := []string{"FIL-USDT"}
	channels := make(map[string]chan common.Trade)
	for _, symbol := range symbols {
		channels[symbol] = make(chan common.Trade, 1000)
	}
	_ = NewTradeRoutedWS(ctx,  "socks5://127.0.0.1:1080", channels)
	for {
		select {
		case d := <-channels[symbols[0]]:
			logger.Debugf("%v", d)
		}
	}
}
