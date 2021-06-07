package bncoinfuture

import (
	"context"
	"github.com/geometrybase/hft-micro/logger"
	"testing"
	"time"
)

func TestNewTradeWebsocket(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), time.Minute*5)
	symbols := []string{"BTCUSDT", "ETHUSDT", "FLMUSDT", "BLZUSDT", "TRXUSDT", "EOSUSDT"}
	symbols = symbols[:1]
	readTimeout := time.Second * 10
	proxy := "socks5://127.0.0.1:1080"
	ws := NewTradeWebsocket(ctx, symbols, readTimeout, proxy)
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
