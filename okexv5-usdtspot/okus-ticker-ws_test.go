package okexv5_usdtspot

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"testing"
	"time"
)

func TestNewTickerWs(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), time.Minute*5)
	symbols := []string{"BTC-USDT", "DOGE-USDT", "WAVES-USDT"}
	channels := make(map[string]chan common.Ticker)
	ch := make(chan common.Ticker, 0)
	for _, symbol := range symbols {
		channels[symbol] = ch
	}
	ws := NewTickerWS(ctx, os.Getenv("OK_PROXY"), channels)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ws.Done():
			return
		case ticker := <-ch:
			logger.Debugf("%v", ticker)
		}
	}
}
