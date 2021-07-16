package huobi_usdtfuture

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"testing"
)

func TestNewDepth20RoutedWebsocket(t *testing.T) {
	var ctx = context.Background()
	var symbols = []string{"FIL-USDT", "WAVES-USDT", "LINK-USDT"}
	var channels = make(map[string]chan *common.DepthRawMessage)
	for _, symbol := range symbols {
		channels[symbol] = make(chan *common.DepthRawMessage, 100)
	}
	ws := NewDepth20RoutedWebsocket(
		ctx,  "socks5://127.0.0.1:1081", channels,
	)
	for {
		select {
		case <-ws.Done():
			return
		case d := <-channels[symbols[0]]:
			logger.Debugf("%s %s", symbols[0], d)
		case d := <-channels[symbols[1]]:
			logger.Debugf("%s %s", symbols[1], d)
		case d := <-channels[symbols[2]]:
			logger.Debugf("%s %s", symbols[2], d)
		}
	}
}
