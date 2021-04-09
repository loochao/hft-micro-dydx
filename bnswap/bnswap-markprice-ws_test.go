package bnswap

import (
	"context"
	"github.com/geometrybase/hft-micro/logger"
	"testing"
	"time"
)

func TestNewMarkPriceWebsocket(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), time.Minute*3)
	symbols := []string{"BNBUSDT", "ETHUSDT", "FLMUSDT", "BLZUSDT", "TRXUSDT", "EOSUSDT"}
	symbols = symbols[:1]
	proxy := "socks5://127.0.0.1:1080"
	ws := NewMarkPriceWebsocket(ctx, symbols,  proxy)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ws.Done():
			return
		case markPrice := <-ws.DataCh:
			logger.Debugf("%v", markPrice.ToString())
		}
	}
}
