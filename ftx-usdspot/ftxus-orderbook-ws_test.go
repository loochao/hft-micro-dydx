package ftx_usdspot

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"testing"
)

func TestNewOrderBookWS(t *testing.T) {
	var ctx = context.Background()
	symbols := []string{ "DOGE/USD"}
	channels := make(map[string]chan common.Depth)
	for _, symbol := range symbols {
		channels[symbol] = make(chan common.Depth, 100)
	}
	_ = NewOrderBookWS(ctx, os.Getenv("FTX_TEST_PROXY"), channels)
	for {
		select {
		case d := <-channels[symbols[0]]:
			logger.Debugf("%v", d)
		}
	}
}
