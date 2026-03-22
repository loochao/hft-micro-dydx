package okexv5_usdtswap

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"testing"
	"time"
)

func TestNewTradeWs(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), time.Minute*5)
	symbols := []string{"BTC-USDT-SWAP", "DOGE-USDT-SWAP", "WAVES-USDT-SWAP"}
	channels := make(map[string]chan common.Trade)
	ch := make(chan common.Trade, 64)
	for _, symbol := range symbols {
		channels[symbol] = ch
	}
	ws := NewTradeWS(ctx, os.Getenv("OK_PROXY"), channels)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ws.Done():
			return
		case trade := <-ch:
			logger.Debugf("%v", trade)
		}
	}
}
