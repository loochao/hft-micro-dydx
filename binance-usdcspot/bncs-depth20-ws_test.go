package binance_usdcspot

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"testing"
	"time"
)

func TestNewDepth20Ws(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), time.Minute*5)
	symbols := []string{"BTCUSDC", "ETHUSDC", "FLMUSDC", "BLZUSDC", "TRXUSDC", "EOSUSDC"}
	proxy := "socks5://127.0.0.1:1081"
	channels := make(map[string]chan common.Depth)
	ch := make(chan common.Depth)
	for _, symbol := range symbols {
		channels[symbol] = ch
	}
	ws := NewDepth20WS(ctx, proxy, channels)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ws.Done():
			return
		case depth20 := <-ch:
			logger.Debugf("%v", depth20)
		}
	}
}
