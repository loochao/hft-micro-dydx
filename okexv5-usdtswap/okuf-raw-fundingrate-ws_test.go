package okexv5_usdtswap

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"testing"
	"time"
)

func TestNewRawFundingRateWS(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), time.Minute*5)
	//symbols := []string{"BTC-USDT-SWAP", "DOGE-USDT-SWAP", "WAVES-USDT-SWAP"}
	symbols := []string{"BABYDOGE-USDT-SWAP"}
	channels := make(map[string]chan *common.RawMessage)
	ch := make(chan *common.RawMessage, 64)
	for _, symbol := range symbols {
		channels[symbol] = ch
	}
	ws := NewRawFundingRateWS(ctx, os.Getenv("OK_PROXY"), []byte{'F'}, channels)
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
