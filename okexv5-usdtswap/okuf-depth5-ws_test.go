package okexv5_usdtswap

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"testing"
	"time"
)

func TestNewDepth5Ws(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), time.Minute*5)
	//symbols := []string{"BTC-USDT-SWAP", "DOGE-USDT-SWAP", "WAVES-USDT-SWAP"}
	symbols := []string{"BABYDOGE-USDT-SWAP"}
	channels := make(map[string]chan common.Depth)
	ch := make(chan common.Depth, 64)
	for _, symbol := range symbols {
		channels[symbol] = ch
	}
	ws := NewDepth5WS(ctx, os.Getenv("OK_PROXY"), channels)
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
