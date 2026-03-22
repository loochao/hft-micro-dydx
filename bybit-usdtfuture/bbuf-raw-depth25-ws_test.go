package bybit_usdtfuture

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"testing"
)

func TestNewNawOrderBookTickerWS(t *testing.T) {
	var ctx = context.Background()
	symbols := []string{"ETHUSDT"}
	channels := make(map[string]chan *common.RawMessage)
	for _, symbol := range symbols {
		channels[symbol] = make(chan *common.RawMessage, 100)
	}
	_ = NewRawDepth25WS(
		ctx,
		os.Getenv("BYBIT_TEST_PROXY"),
		[]byte{'X','D'},
		channels,
	)
	for {
		select {
		case d := <-channels[symbols[0]]:
			logger.Debugf("%s", d.Data)
		}
	}
}
