package dydx_usdfuture

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"testing"
)

func TestNewNawOrderBookTickerWS(t *testing.T) {
	var ctx = context.Background()
	markets := []string{"ETH-USD"}
	channels := make(map[string]chan *common.RawMessage)
	for _, symbol := range markets {
		channels[symbol] = make(chan *common.RawMessage, 100)
	}
	_ = NewRawDepthWS(
		ctx,
		os.Getenv("BYBIT_TEST_PROXY"),
		[]byte{'X','D'},
		channels,
	)
	for {
		select {
		case d := <-channels[markets[0]]:
			logger.Debugf("%s", d.Data)
		}
	}
}
