package main

import (
	"context"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
)

func takerRoutedKlineLoop(
	ctx context.Context,
	cancel context.CancelFunc,
	proxyAddress string,
	channels map[string]chan common.KLine,
) {
	symbols := make([]string, 0)
	for symbol := range channels {
		symbols = append(symbols, symbol)
	}
	logger.Debugf("START takerRoutedKlineLoop %s", symbols)
	defer logger.Debugf("EXIT takerRoutedKlineLoop %s", symbols)
	ws := bnswap.NewKline1MRoutedWebsocket(
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
