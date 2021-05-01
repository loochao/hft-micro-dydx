package main

import (
	"context"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
)

func StreamDepth5(
	ctx context.Context,
	cancel context.CancelFunc,
	proxyAddress string,
	channels map[string]chan *common.DepthRawMessage,
) {
	symbols := make([]string, 0)
	for symbol := range channels {
		symbols = append(symbols, symbol)
	}
	logger.Debugf("START StreamDepth5 %s", symbols)
	defer logger.Debugf("EXIT StreamDepth5 %s", symbols)
	ws := bnswap.NewDepth5RoutedWebsocket(
		ctx,
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
