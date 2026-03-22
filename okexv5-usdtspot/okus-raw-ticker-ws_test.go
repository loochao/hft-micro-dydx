package okexv5_usdtspot

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"testing"
	"time"
)

func TestNewRawTickerWs(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), time.Minute*5)
	symbols := []string{"BTC-USDT", "DOGE-USDT", "WAVES-USDT"}
	channels := make(map[string]chan *common.RawMessage)
	ch := make(chan *common.RawMessage)
	for _, symbol := range symbols {
		channels[symbol] = ch
	}
	ws := NewRawTickerWS(ctx, os.Getenv("OK_PROXY"), []byte{'B'}, channels)
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
