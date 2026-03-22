package binance_usdtfuture

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"testing"
	"time"
)

func TestNewWalkedDepth5WS(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), time.Minute*5)
	symbols := []string{"SCUSDT", "BTCUSDT", "LINKUSDT", "WAVESUSDT"}
	proxy := "socks5://127.0.0.1:1084"

	ch := make(chan common.Ticker, 100)
	channels := make(map[string]chan common.Ticker)
	for _, symbol := range symbols[:] {
		channels[symbol] = ch
	}

	ws := NewWalkedDepth5WS(ctx, proxy, 100000, channels)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ws.Done():
			return
		case msg := <-ch:
			logger.Debugf("%v", msg)
		}
	}
}
