package bnspot

import (
	"context"
	"testing"
	"time"
)

func TestNewDepth20Ws(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), time.Minute*5)
	symbols := []string{"BTCUSDT", "ETHUSDT", "FLMUSDT", "BLZUSDT", "TRXUSDT", "EOSUSDT"}
	readTimeout := time.Second * 10
	proxy := "socks5://127.0.0.1:1080"
	ws := NewDepth20Websocket(ctx, symbols, readTimeout, proxy)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ws.Done():
			return
		case depth20 := <-ws.DataCh:
			_ = depth20
			//Debugf("%v", *depth20)
		}
	}
}
