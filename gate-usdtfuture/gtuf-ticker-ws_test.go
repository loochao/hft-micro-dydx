package gate_usdtfuture

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"testing"
)

func TestNewTickerWS(t *testing.T) {
	channels := make(map[string]chan common.Ticker)
	channels["GITCOIN_USDT"] = make(chan common.Ticker, 100)
	_ = NewTickerWS(context.Background(), "socks5://127.0.0.1:1083", channels)
	for {
		select {
		case d := <-channels["GITCOIN_USDT"]:
			logger.Debugf("%v", d)
		}
	}
}
