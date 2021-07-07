package okex_usdtspot

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"testing"
	"time"
)

func TestNewTickerWs(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), time.Minute*5)
	symbols := []string{"BTC-USDT", "DOGE-USDT", "WAVES-USDT"}
	proxy := "socks5://127.0.0.1:1083"
	channels := make(map[string]chan common.Ticker)
	ch := make(chan common.Ticker)
	for _, symbol := range symbols {
		channels[symbol] = ch
	}
	ws := NewTickerWS(ctx, proxy, channels)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ws.Done():
			return
		case depth5 := <-ch:
			logger.Debugf("%v", depth5)
		}
	}
}
