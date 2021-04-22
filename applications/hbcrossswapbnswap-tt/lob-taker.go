package main

import (
	"context"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
)

func watchTakerDepthWebsocket(
	ctx context.Context,
	cancel context.CancelFunc,
	takerDecay, takerBias float64,
	proxyAddress string,
	depthReportCh chan common.DepthReport,
	channels map[string]chan *common.DepthRawMessage,
) {
	symbols := make([]string, 0)
	for symbol := range channels {
		symbols = append(symbols, symbol)
	}
	logger.Debugf("START watchMakerDepthWebsocket %s", symbols)
	defer func() {
		logger.Debugf("EXIT watchMakerDepthWebsocket %s", symbols)
	}()
	ws := bnswap.NewDepth20RoutedWebsocket(
		ctx,
		takerDecay,
		takerBias,
		proxyAddress,
		channels,
		depthReportCh,
	)
	defer ws.Stop()
	for {
		select {
		case <-ws.Done():
			cancel()
			logger.Debugf("DEPTH20 WS CONTEXT DONE %v", channels)
			return
		case <-ctx.Done():
			return
		}
	}
}
