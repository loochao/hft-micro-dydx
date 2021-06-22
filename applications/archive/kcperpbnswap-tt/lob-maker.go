package main

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/kucoin-usdtfuture"
	"github.com/geometrybase/hft-micro/logger"
)

func makerRoutedDepthLoop(
	ctx context.Context,
	cancel context.CancelFunc,
	api *kucoin_usdtfuture.API,
	proxyAddress string,
	channels map[string]chan *common.DepthRawMessage,
) {
	symbols := make([]string, 0)
	for symbol := range channels {
		symbols = append(symbols, symbol)
	}
	logger.Debugf("START makerRoutedDepthLoop %s", symbols)
	defer logger.Debugf("EXIT makerRoutedDepthLoop %s", symbols)
	ws := kucoin_usdtfuture.NewDepth5RoutedWebsocket(
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
			return
		case <-ctx.Done():
			return
		}
	}
}
