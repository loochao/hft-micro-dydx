package binance_coinfuture

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"testing"
	"time"
)

func TestNewDepth5WS(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), time.Minute*5)
	symbols := []string{"DOTUSD_PERP"}
	proxy := "socks5://127.0.0.1:1080"

	channels := make(map[string]chan common.Depth)
	for _, symbol := range symbols {
		channels[symbol] = make(chan common.Depth, 100)
	}

	ws := NewDepth5WS(ctx, proxy, channels)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ws.Done():
			return
		case msg := <-channels[symbols[0]]:
			logger.Debugf("%v", msg)
		//case msg := <-channels[symbols[1]]:
		//	logger.Debugf("%v", msg)
		//case msg := <-channels[symbols[2]]:
		//	logger.Debugf("%v", msg)
		}
	}
}
