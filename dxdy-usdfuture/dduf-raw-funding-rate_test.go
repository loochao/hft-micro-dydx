package dxdy_usdtfuture

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"testing"
)

func TestStreamRawFundingRate(t *testing.T) {
	var ctx = context.Background()
	markets := []string{"ETH-USD"}
	channels := make(map[string]chan *common.RawMessage)
	for _, symbol := range markets {
		channels[symbol] = make(chan *common.RawMessage, 100)
	}
	go StreamRawFundingRate(
		ctx,
		os.Getenv("BYBIT_TEST_PROXY"),
		[]byte{'X','F'},
		channels,
	)
	for {
		select {
		case d := <-channels[markets[0]]:
			logger.Debugf("%s", d.Data)
		}
	}
}
