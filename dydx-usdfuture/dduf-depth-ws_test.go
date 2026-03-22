package dydx_usdfuture

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"testing"
)

func TestNewDepthWS(t *testing.T) {
	var ctx = context.Background()
	markets := []string{"FIL-USD"}
	channels := make(map[string]chan common.Depth)
	for _, symbol := range markets {
		channels[symbol] = make(chan common.Depth, 100)
	}
	_ = NewDepthWS(
		ctx,
		"socks5://127.0.0.1:1080",
		channels,
	)
	for {
		select {
		case _ = <-channels[markets[0]]:
			//logger.Debugf("BID %v ASK %v", d.GetBids()[0], d.GetAsks()[0])
		}
	}
}
