package hbspot

import (
	"context"
	"github.com/geometrybase/hft-micro/logger"
	"testing"
)

func TestNewDepth20FilteredWebsocket(t *testing.T) {
	var ctx = context.Background()
	ws := NewDepth20FilteredWebsocket(ctx, 0.9999, 20, []string{"btcusdt", "linkusdt", "wavesusdt"}, "socks5://127.0.0.1:1081")
	for {
		select {
		case d := <-ws.DataCh:
			logger.Debugf("%v", d)
		}
	}
}
