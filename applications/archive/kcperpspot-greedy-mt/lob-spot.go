package main

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/kucoin-usdtspot"
	"github.com/geometrybase/hft-micro/logger"
)

func makerDepthWSLoop(
	ctx context.Context,
	cancel context.CancelFunc,
	api *kucoin_usdtspot.API,
	proxyAddress string,
	channels map[string]chan *common.DepthRawMessage,
) {
	symbols := make([]string, 0)
	for symbol := range channels {
		symbols = append(symbols, symbol)
	}
	logger.Debugf("START makerDepthWSLoop %s", symbols)
	defer logger.Debugf("EXIT makerDepthWSLoop %s", symbols)
	ws := kucoin_usdtspot.NewDepth5RoutedWebsocket(
		ctx,
		api,
		proxyAddress,
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

