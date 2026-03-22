package bnmargin

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNewDepth5Ws(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), time.Minute*5)
	symbols := []string{"BTCUSDT", "ETHUSDT", "FLMUSDT", "BLZUSDT", "TRXUSDT", "EOSUSDT"}
	proxy := "socks5://127.0.0.1:1081"
	channels := make(map[string]chan common.Depth)
	ch := make(chan common.Depth)
	for _, symbol := range symbols {
		channels[symbol] = ch
	}
	ws := NewDepth5WS(ctx, proxy, channels)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ws.Done():
			return
		case depth5 := <-ch:
			wd, _ := common.WalkMakerTakerDepth5(depth5, 1000.0, 10000.0)
			//_ = depth5
			logger.Debugf("%v", wd)
			assert.GreaterOrEqual(t, wd.TakerFarAsk, wd.MakerFarAsk)
			assert.GreaterOrEqual(t, wd.TakerFarAsk, wd.TakerAsk)
			assert.GreaterOrEqual(t, wd.TakerAsk, wd.MakerAsk)
			assert.GreaterOrEqual(t, wd.MakerFarAsk, wd.MakerAsk)
			assert.GreaterOrEqual(t, wd.MakerAsk, wd.MakerBid)
			assert.GreaterOrEqual(t, wd.MakerBid, wd.TakerBid)
			assert.GreaterOrEqual(t, wd.MakerBid, wd.MakerFarBid)
			assert.GreaterOrEqual(t, wd.TakerBid, wd.TakerFarBid)
			assert.GreaterOrEqual(t, wd.MakerFarBid, wd.TakerFarBid)
		}
	}
}
