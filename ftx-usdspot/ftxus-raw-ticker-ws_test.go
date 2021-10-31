package ftx_usdspot

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"testing"
)

func TestNewRawTickerWS(t *testing.T) {
	var ctx = context.Background()
	symbols := []string{ "WAVES/USD"}
	channels := make(map[string]chan *common.RawMessage)
	for _, symbol := range symbols {
		channels[symbol] = make(chan *common.RawMessage, 100)
	}
	_ = NewRawTickerWS(ctx, os.Getenv("FTX_TEST_PROXY"), []byte{'B'}, channels)
	for {
		select {
		case d := <-channels[symbols[0]]:
			logger.Debugf("%s", d.Data)
		}
	}
}
