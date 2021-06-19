package okex_usdtspot

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"testing"
)

func TestNewDepth20RoutedWebsocket(t *testing.T) {
	var ctx = context.Background()
	symbols := []string{"BTC-USDT", "ETH-USDT", "LINK-USDT", "THETA-USDT"}
	channels := make(map[string]chan *common.DepthRawMessage)
	for _, symbol := range symbols {
		channels[symbol] = make(chan *common.DepthRawMessage, 100)
	}
	_ = NewDepth5RoutedWebsocket(ctx, "socks5://127.0.0.1:1081", channels)
	for {
		select {
		case d := <-channels[symbols[0]]:
			logger.Debugf("%s %v %s", d.Symbol, d.Time, d.Depth)
		case d := <-channels[symbols[1]]:
			logger.Debugf("%s %v %s", d.Symbol, d.Time, d.Depth)
		case d := <-channels[symbols[2]]:
			logger.Debugf("%s %v %s", d.Symbol, d.Time, d.Depth)
		case d := <-channels[symbols[3]]:
			logger.Debugf("%s %v %s", d.Symbol, d.Time, d.Depth)
		}
	}
}
