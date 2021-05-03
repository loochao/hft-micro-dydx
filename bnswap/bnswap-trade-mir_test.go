package bnswap

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"testing"
	"time"
)

func TestStreamTradeMIR(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	symbols := []string{"BTCUSDT"}
	channels := make(map[string]chan common.MIR)
	for _, symbol := range symbols {
		channels[symbol] = make(chan common.MIR, 1000)
	}
	go StreamTradeMIR(
		ctx,
		cancel,
		"socks5://127.0.0.1:1080",
		time.Minute,
		10000,
		time.Minute,
		channels,
	)
	for {
		select {
		case d := <-channels[symbols[0]]:
			logger.Debugf("%v %s %f", d.Time, d.Symbol, d.Value)
		}
	}
}
