package ftx_usdfuture

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"testing"
)

func TestNewUserWS(t *testing.T) {

	var ctx = context.Background()
	symbols := []string{"LTC-PERP", "ETH-PERP", "DOGE-PERP", "WAVES-PERP"}
	channels := make(map[string]chan common.Depth)
	for _, symbol := range symbols {
		channels[symbol] = make(chan common.Depth, 100)
	}
	ws := NewUserWS(
		os.Getenv("FTX_TEST_KEY"),
		os.Getenv("FTX_TEST_SECRET"),
		"",
		os.Getenv("FTX_TEST_PROXY"),
	)
	go ws.Start(ctx)
	for {
		select {
		case <-ws.Done():
			return
		case f := <-ws.FillCh:
			logger.Debugf("%v", f)
		case o := <-ws.OrderCh:
			logger.Debugf("%v", o)
		}
	}
}
