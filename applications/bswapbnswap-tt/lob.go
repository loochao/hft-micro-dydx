package main

import (
	"context"
	"github.com/geometrybase/hft-micro/bnspot"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
)

func makerDepthWebsocketLoop(
	ctx context.Context,
	cancel context.CancelFunc,
	proxyAddress string,
	channels map[string]chan *common.DepthRawMessage,
) {
	symbols := make([]string, 0)
	for symbol := range channels {
		symbols = append(symbols, symbol)
	}
	logger.Debugf("START makerDepthWebsocketLoop %s", symbols)
	defer logger.Debugf("EXIT makerDepthWebsocketLoop %s", symbols)
	ws := bnspot.NewDepth5RoutedWebsocket(ctx, proxyAddress, channels)
	defer ws.Stop()
	for {
		select {
		case <-ws.Done():
			cancel()
			logger.Debugf("<-ws.Done() %s", symbols)
			return
		case <-ctx.Done():
			logger.Debugf("<-ctx.Done() %s", symbols)
			return
		}
	}
}


func takerDepthWebsocketLoop(
	ctx context.Context,
	cancel context.CancelFunc,
	proxyAddress string,
	channels map[string]chan *common.DepthRawMessage,
) {
	symbols := make([]string, 0)
	for symbol := range channels {
		symbols = append(symbols, symbol)
	}
	logger.Debugf("START takerDepthWebsocketLoop", symbols)
	defer logger.Debugf("EXIT takerDepthWebsocketLoop %s", symbols)
	ws := bnswap.NewDepth5RoutedWebsocket(ctx, proxyAddress, channels)
	defer ws.Stop()
	for {
		select {
		case <-ws.Done():
			cancel()
			logger.Debugf("<-ws.Done() %s", symbols)
			return
		case <-ctx.Done():
			logger.Debugf("<-ctx.Done() %s", symbols)
			return
		}
	}
}

