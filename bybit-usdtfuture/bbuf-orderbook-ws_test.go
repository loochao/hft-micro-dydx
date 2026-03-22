package bybit_usdtfuture

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"testing"
)

func TestNewOrderBookWS(t *testing.T) {
	var ctx = context.Background()
	symbols := []string{"ETHUSDT"}
	channels := make(map[string]chan common.Depth)
	for _, symbol := range symbols {
		channels[symbol] = make(chan common.Depth, 100)
	}
	_ = NewOrderBookWS(ctx, os.Getenv("TEST_PROXY"), channels)
	for {
		select {
		case d := <-channels[symbols[0]]:
			logger.Debugf("%v", d)
		}
	}
}
