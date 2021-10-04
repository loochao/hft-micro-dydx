package dxdy_usdtfuture

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"testing"
	"time"
)

func TestNewTradeRoutedWS(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), time.Minute*5)
	symbols := []string{"BTCUSDT", "LINKUSDT", "WAVESUSDT"}
	proxy := "socks5://127.0.0.1:1080"

	channels := make(map[string]chan common.Trade)
	for _, symbol := range symbols {
		channels[symbol] = make(chan common.Trade, 100)
	}

	ws := NewTradeRoutedWS(ctx, proxy, channels)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ws.Done():
			return
		case msg := <-channels[symbols[0]]:
			logger.Debugf("%s %v %f", msg.GetSymbol(), msg.GetPrice(), msg.GetSize())
		case msg := <-channels[symbols[1]]:
			logger.Debugf("%s %v %f", msg.GetSymbol(), msg.GetPrice(), msg.GetSize())
		case msg := <-channels[symbols[2]]:
			logger.Debugf("%s %v %f", msg.GetSymbol(), msg.GetPrice(), msg.GetSize())
		}
	}
}
