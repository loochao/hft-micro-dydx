package main

import (
	"context"
	"github.com/geometrybase/hft-micro/bnspot"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
)

func bnspotDepthLoop(
	ctx context.Context,
	cancel context.CancelFunc,
	proxyAddress string,
	channels map[string]chan *common.DepthRawMessage,
) {
	symbols := make([]string, 0)
	for symbol := range channels {
		symbols = append(symbols, symbol)
	}
	logger.Debugf("START bnswapDepthLoop %s", symbols)
	defer logger.Debugf("EXIT bnswapDepthLoop %s", symbols)
	ws := bnspot.NewDepth20RoutedWebsocket(
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
