package hbcrossswap

import (
	"context"
	"github.com/geometrybase/hft-micro/logger"
	"testing"
)

func TestNewDepth20RoutedWebsocket(t *testing.T) {
	var ctx = context.Background()
	var symbols = []string{"FIL-USDT", "WAVES-USDT", "LINK-USDT"}
	var channels = make(map[string]chan []byte)
	for _, symbol := range symbols {
		channels[symbol] = make(chan []byte, 100)
	}
	ws := NewDepth20RoutedWebsocket(
		ctx, 0.9999, 300, "socks5://127.0.0.1:1081", channels, nil,
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
