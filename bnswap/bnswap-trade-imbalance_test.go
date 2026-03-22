package bnswap

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"testing"
	"time"
)

func TestWatchTimedTradeImbalances(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	symbols := []string{"BTCUSDT"}
	channels := make(map[string]chan *common.Signal)
	for _, symbol := range symbols {
		channels[symbol] = make(chan *common.Signal, 1000)
	}
	go StreamTimedTradeImbalances(
		ctx,
		cancel,
		"socks5://127.0.0.1:1080",
		time.Second*30,
		channels,
	)
	for {
		select {
		case d := <-channels[symbols[0]]:
			logger.Debugf("%v %s %f", d.Time, d.Name, d.Value)
		}
	}
}
