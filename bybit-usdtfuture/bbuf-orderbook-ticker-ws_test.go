package bybit_usdtfuture

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"testing"
)

func TestNewOrderBookTickerWS(t *testing.T) {
	var ctx = context.Background()
	symbols := []string{"ETHUSDT"}
	channels := make(map[string]chan common.Ticker)
	for _, symbol := range symbols {
		channels[symbol] = make(chan common.Ticker, 100)
	}
	_ = NewOrderBookTickerWS(ctx, os.Getenv("BYBIT_TEST_PROXY"), channels)
	for {
		select {
		case d := <-channels[symbols[0]]:
			logger.Debugf("%v", d)
		}
	}
}
