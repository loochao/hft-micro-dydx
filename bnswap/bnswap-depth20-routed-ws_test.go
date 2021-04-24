package bnswap

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"testing"
	"time"
)

func TestNewDepth20RoutedWebsocket(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), time.Minute*5)
	symbols := []string{"BTCUSDT", "LINKUSDT", "WAVESUSDT"}
	proxy := "socks5://127.0.0.1:1081"

	channels := make(map[string]chan *common.DepthRawMessage)
	for _, symbol := range symbols {
		channels[symbol] = make(chan *common.DepthRawMessage, 100)
	}

	ws := NewDepth20RoutedWebsocket(ctx, proxy, channels)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ws.Done():
			return
		case msg := <-channels[symbols[0]]:
			logger.Debugf("%s", msg)
		case msg := <-channels[symbols[1]]:
			logger.Debugf("%s", msg)
		case msg := <-channels[symbols[2]]:
			logger.Debugf("%s", msg)
		}
	}
}
