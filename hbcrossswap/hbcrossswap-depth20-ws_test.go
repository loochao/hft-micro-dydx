package hbcrossswap

import (
	"context"
	"github.com/geometrybase/hft-micro/logger"
	"testing"
)

func TestNewDepth20Websocket(t *testing.T) {
	var ctx = context.Background()
	ws := NewDepth20Websocket(ctx, []string{"FIL-USDT"}, "socks5://127.0.0.1:1081")
	for {
		select {
		case d := <-ws.DataCh:
			logger.Debugf("%v", d)
		}
	}
}
