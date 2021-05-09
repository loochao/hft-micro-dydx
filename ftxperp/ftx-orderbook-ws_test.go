package ftxperp

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"testing"
)

func TestNewDepth20RoutedWebsocket(t *testing.T) {
	var ctx = context.Background()
	symbols := []string{"ETH-PERP"}
	channels := make(map[string]chan *common.DepthRawMessage)
	for _, symbol := range symbols {
		channels[symbol] = make(chan *common.DepthRawMessage, 100)
	}
	_ = NewOrderBookWS(ctx, os.Getenv("FTX_TEST_PROXY"), channels)
	for {
		select {
		case d := <-channels[symbols[0]]:
			logger.Debugf("%s %v %s", d.Symbol, d.Time, d.Depth)
		}
	}
}
