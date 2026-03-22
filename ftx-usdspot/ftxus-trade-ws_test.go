package ftx_usdspot

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"testing"
)

func TestNewTradeWS(t *testing.T) {
	var ctx = context.Background()
	symbols := []string{"ETH-PERP"}
	channels := make(map[string]chan common.Trade)
	for _, symbol := range symbols {
		channels[symbol] = make(chan common.Trade, 100)
	}
	_ = NewTradeWS(ctx, os.Getenv("FTX_TEST_PROXY"), channels)
	for {
		select {
		case d := <-channels[symbols[0]]:
			logger.Debugf("%s %f %f %v", d.GetSymbol(), d.GetSize(), d.GetPrice(), d.IsUpTick())
		}
	}
}
