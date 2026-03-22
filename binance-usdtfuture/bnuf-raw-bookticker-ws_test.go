package binance_usdtfuture

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"testing"
	"time"
)

func TestRawNewBookTickerWS(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), time.Minute*5)
	symbols := []string{"SCUSDT", "BTCUSDT", "LINKUSDT", "WAVESUSDT"}
	proxy := "socks5://127.0.0.1:1080"

	ch := make(chan *common.RawMessage, 100)
	channels := make(map[string]chan *common.RawMessage)
	for _, symbol := range symbols[:] {
		channels[symbol] = ch
	}

	ws := NewRawBookTickerWS(ctx, proxy, []byte{'Y', 'T'}, channels)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ws.Done():
			return
		case msg := <-ch:
			logger.Debugf("%s", msg.Data)
		}
	}
}
