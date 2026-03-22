package binance_usdtspot

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"testing"
	"time"
)

func TestNewDepth5Ws(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), time.Minute*5)
	symbols := []string{"SCUSDT","BTCUSDT", "ETHUSDT", "FLMUSDT", "BLZUSDT", "TRXUSDT", "EOSUSDT"}
	proxy := "socks5://127.0.0.1:1081"
	channels := make(map[string]chan common.Depth)
	ch := make(chan common.Depth)
	for _, symbol := range symbols[:1] {
		channels[symbol] = ch
	}
	ws := NewDepth5WS(ctx, proxy, channels)
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
