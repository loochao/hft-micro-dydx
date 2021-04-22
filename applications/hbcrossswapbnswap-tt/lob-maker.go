package main

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/hbcrossswap"
	"github.com/geometrybase/hft-micro/logger"
)

func watchMakerDepthWebsocket(
	ctx context.Context,
	cancel context.CancelFunc,
	makerDecay, makerBias float64,
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
	ws := hbcrossswap.NewDepth20RoutedWebsocket(
		ctx,
		makerDecay,
		makerBias,
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
		case <-ws.RestartCh:
			logger.Debugf("DEPTH20 WS RESTART %v", channels)
		}
	}
}
