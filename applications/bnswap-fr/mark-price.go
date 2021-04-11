package main

import (
	"context"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/logger"
)

func watchMarkPrice(
	ctx context.Context, proxyAddress string,
	symbols []string,
	outputCh chan *bnswap.MarkPrice,
) {
	ws := bnswap.NewMarkPriceWebsocket(
		ctx,
		symbols,
		proxyAddress,
	)
	defer ws.Stop()

	for {
		select {
		case <-ws.Done():
			logger.Fatal("DEPTH20 WS CONTEXT DONE %s", symbols)
		case <-ctx.Done():
			return
		case outputCh <- <-ws.DataCh:
			break
		}
	}
}
