package coinbase_usdspot

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"testing"
)

func TestNewMatchRoutedWS(t *testing.T) {
	var ctx = context.Background()
	symbols := []string{"BTC-USD"}
	channels := make(map[string]chan common.Trade)
	for _, symbol := range symbols {
		channels[symbol] = make(chan common.Trade, 1000)
	}
	_ = NewMatchRoutedWS(
		ctx,
		"socks5://127.0.0.1:1080",
		channels,
	)
	for {
		select {
		case d := <-channels[symbols[0]]:
			logger.Debugf("FIL-USD %v %f %f", d.GetTime(), d.GetPrice(), d.GetSize())
		}
	}
}
