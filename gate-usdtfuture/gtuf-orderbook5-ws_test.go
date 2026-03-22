package gate_usdtfuture

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"testing"
)

func TestNewOrderBook5WS(t *testing.T) {
	channels := make(map[string]chan common.Depth)
	channels["GITCOIN_USDT"] = make(chan common.Depth, 100)
	_ = NewOrderBook5WS(context.Background(), "socks5://127.0.0.1:1083", channels)
	for {
		select {
		case d := <-channels["GITCOIN_USDT"]:
			logger.Debugf("%v", d)
		}
	}
}
