package okexv5_usdtswap

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"testing"
	"time"
)

func TestNewWalkedDepth5WS(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), time.Minute*5)
	//symbols := []string{"BTC-USDT-SWAP", "DOGE-USDT-SWAP", "WAVES-USDT-SWAP"}
	symbols := []string{"BABYDOGE-USDT-SWAP"}
	channels := make(map[string]chan common.Ticker)
	ch := make(chan common.Ticker, 64)
	for _, symbol := range symbols {
		channels[symbol] = ch
	}
	ws := NewWalkedDepth5WS(ctx, os.Getenv("OK_PROXY"), 10000, channels)
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
