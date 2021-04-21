package bnswap

import (
	"context"
	"github.com/geometrybase/hft-micro/logger"
	"testing"
	"time"
)

func TestNewDepth20FilteredWebsocket(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), time.Minute*5)
	symbols := []string{"BTCUSDT", "LINKUSDT", "WAVESUSDT"}
	proxy := "socks5://127.0.0.1:1081"


	ws := NewDepth20FilteredWebsocket(ctx, 0.995, 50, symbols,  proxy)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ws.Done():
			return
		case depth20 := <-ws.DataCh:
			logger.Debugf("%v", *depth20)
		}
	}
}