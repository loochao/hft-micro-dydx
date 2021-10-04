package dxdy_usdtfuture

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
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
		case _ = <-channels[markets[0]]:
			//logger.Debugf("%s", d.Prefix)
		}
	}
}
