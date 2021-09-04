package ftx_usdspot

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"testing"
)

func TestNewTickerWS(t *testing.T) {
	var ctx = context.Background()
	symbols := []string{ "HT/USD"}
	channels := make(map[string]chan common.Ticker)
	for _, symbol := range symbols {
		channels[symbol] = make(chan common.Ticker, 100)
	}
	_ = NewTickerWS(ctx, os.Getenv("FTX_TEST_PROXY"), channels)
	for {
		select {
		case d := <-channels[symbols[0]]:
			logger.Debugf("%v", d)
		}
	}
}
