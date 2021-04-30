package hbspot

import (
	"context"
	"github.com/geometrybase/hft-micro/logger"
	"testing"
)

func TestNewTradeRoutedWS(t *testing.T) {
	var ctx = context.Background()
	symbols := []string{"filusdt"}
	channels := make(map[string]chan TradeDetail)
	for _, symbol := range symbols {
		channels[symbol] = make(chan TradeDetail, 1000)
	}
	_ = NewTradeRoutedWS(ctx, "socks5://127.0.0.1:1080", channels)
	for {
		select {
		case d := <-channels[symbols[0]]:
			logger.Debugf("%v", d)
		}
	}
}
