package ftx_usdspot

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"os"
	"testing"
)

func TestNewDepthWS(t *testing.T) {
	var ctx = context.Background()
	symbols := []string{ "DOGE/USD"}
	channels := make(map[string]chan common.Depth)
	for _, symbol := range symbols {
		channels[symbol] = make(chan common.Depth, 100)
	}
	_ = NewDepthWS(ctx, os.Getenv("FTX_TEST_PROXY"), channels)
	for {
		select {
		case _ = <-channels[symbols[0]]:
			//logger.Debugf("%v", d)
		}
	}
}
