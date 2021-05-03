package bnswap

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"time"
)

func StreamTradeMIR(
	ctx context.Context,
	cancel context.CancelFunc,
	proxyAddress string,
	lookback time.Duration,
	minTradeValue float64,
	updateInterval time.Duration,
	channels map[string]chan common.MIR,
) {
	tradesCh := make(map[string]chan common.Trade)
	for symbol, output := range channels {
		tradesCh[symbol] = make(chan common.Trade, 10000)
		go common.StreamMIR(
			ctx,
			symbol,
			lookback,
			minTradeValue,
			updateInterval,
			tradesCh[symbol],
			output,
		)
	}
	ws := NewTradeRoutedWS(
		ctx,
		proxyAddress,
		tradesCh,
	)
	defer ws.Stop()
	for {
		select {
		case <-ws.Done():
			cancel()
			return
		case <-ctx.Done():
			return
		}
	}

}

