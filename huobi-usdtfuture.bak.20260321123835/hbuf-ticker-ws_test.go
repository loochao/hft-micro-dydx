package huobi_usdtfuture

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"testing"
)

func TestNewTickerWS(t *testing.T) {
	var ctx = context.Background()
	var symbols = []string{"FIL-USDT", "WAVES-USDT", "LINK-USDT"}
	var channels = make(map[string]chan common.Ticker)
	for _, symbol := range symbols {
		channels[symbol] = make(chan common.Ticker, 100)
	}
	ws := NewTickerWS(
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
