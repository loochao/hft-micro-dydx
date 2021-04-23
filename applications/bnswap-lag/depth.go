package main

import (
	"context"
	"github.com/geometrybase/hft-micro/logger"
)

func watchDepthWebsocket(
	ctx context.Context,
	cancel context.CancelFunc,
	takerDecay, takerBias float64,
	proxyAddress string,
	symbols []string, tradeSymbols []string,
	bidPriceCh chan BidPrice,
	timeDeltaCh chan float64,
) {
	logger.Debugf("START watchMakerDepthWebsocket %s", symbols)
	defer func() {
		logger.Debugf("EXIT watchMakerDepthWebsocket %s", symbols)
	}()
	ws := NewDepth20RoutedWebsocket(
		ctx,
		takerDecay,
		takerBias,
		proxyAddress,
		symbols,
		tradeSymbols,
		bidPriceCh,
		timeDeltaCh,
	)
	defer ws.Stop()
	for {
		select {
		case <-ws.Done():
			cancel()
			logger.Debugf("DEPTH20 WS CONTEXT DONE %s", symbols)
			return
		case <-ctx.Done():
			return
		}
	}
}
