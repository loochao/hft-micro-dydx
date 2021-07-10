package binance_busdspot

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"testing"
	"time"
)

func TestNewBookTickerWS(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), time.Minute*5)
	symbols := []string{"SCUSDT","BTCUSDT", "ETHUSDT", "FLMUSDT", "BLZUSDT", "TRXUSDT", "EOSUSDT"}
	proxy := "socks5://127.0.0.1:1081"
	channels := make(map[string]chan common.Ticker)
	ch := make(chan common.Ticker)
	for _, symbol := range symbols[:1] {
		channels[symbol] = ch
	}
	ws := NewBookTickerWS(ctx, proxy, channels)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ws.Done():
			return
		case _ = <-ch:
			//logger.Debugf("%v", depth5)
			break
		}
	}
}
