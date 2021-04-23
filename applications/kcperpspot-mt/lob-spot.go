package main

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/kcspot"
	"github.com/geometrybase/hft-micro/logger"
)

func makerDepthWSLoop(
	ctx context.Context,
	cancel context.CancelFunc,
	api *kcspot.API,
	proxyAddress string,
	makerDecay, makerBias float64,
	reportCount int,
	depthReportCh chan common.DepthReport,
	channels map[string]chan *common.DepthRawMessage,
) {
	symbols := make([]string, 0)
	for symbol := range channels {
		symbols = append(symbols, symbol)
	}
	logger.Debugf("START makerDepthWSLoop %s", symbols)
	defer logger.Debugf("EXIT makerDepthWSLoop %s", symbols)
	ws := kcspot.NewDepth5RoutedWebsocket(
		ctx,
		api,
		proxyAddress,
		makerDecay,
		makerBias,
		reportCount,
		depthReportCh,
		channels,
	)
	defer ws.Stop()
	for {
		select {
		case <-ws.Done():
			cancel()
			logger.Debugf("<-ws.Done() %v", channels)
			return
		case <-ctx.Done():
			return
		case <-ws.RestartCh:
			logger.Debugf("<-ws.RestartCh %v", channels)
		}
	}
}

