package main

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/kcperp"
	"github.com/geometrybase/hft-micro/logger"
)

func takerDepthWebsocketLoop(
	ctx context.Context,
	cancel context.CancelFunc,
	api *kcperp.API,
	proxyAddress string,
	channels map[string]chan *common.DepthRawMessage,
) {
	symbols := make([]string, 0)
	for symbol := range channels {
		symbols = append(symbols, symbol)
	}
	logger.Debugf("START takerDepthWebsocketLoop %s", symbols)
	defer logger.Debugf("EXIT takerDepthWebsocketLoop %s", symbols)
	ws := kcperp.NewDepth5RoutedWebsocket(
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

