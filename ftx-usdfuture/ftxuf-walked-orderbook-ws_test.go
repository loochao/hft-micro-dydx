package ftx_usdfuture

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"testing"
)

func TestNewWalkedOrderBookWS(t *testing.T) {
	var ctx = context.Background()
	symbols := []string{ "BTC-PERP"}
	channels := make(map[string]chan common.Ticker)
	for _, symbol := range symbols {
		channels[symbol] = make(chan common.Ticker, 100)
	}
	_ = NewWalkedOrderBookWS(ctx, os.Getenv("FTX_TEST_PROXY"), 1000000, channels)
	for {
		select {
		case d := <-channels[symbols[0]]:
			logger.Debugf("%.6f", d.GetBidOffset())
		}
	}
}
