package hbspot

import (
	"github.com/geometrybase/hft-micro/logger"
	"testing"
)

func TestNewDepth20Websocket(t *testing.T) {
	//var ctx = context.Background()
	//ws := NewDepth20Websocket(ctx, []string{"btcusdt"}, "socks5://127.0.0.1:1081")
	//for {
	//	select {
	//	case d := <-ws.DataCh:
	//		logger.Debugf("%v", d)
	//	}
	//}
	logger.Debugf("%d", len(`{"ch":"market.btcusdt.depth.step1","ts":`))
}
