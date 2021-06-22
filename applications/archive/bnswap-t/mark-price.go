package main

import (
	"context"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/logger"
)

func startMarkPriceRoutine(
	ctx context.Context,
	proxyAddress string,
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
			logger.Fatal("MARK PRICE WS CONTEXT DONE %s", symbols)
		case <-ctx.Done():
			return
		case outputCh <- <-ws.DataCh:
			break
		}
	}
}

func (st *Strategy) handleMarkPrice(markPrice *bnswap.MarkPrice) {
	symbolIndex := GetSymbolIndex(markPrice.Symbol)
	if symbolIndex != -1 {
		st.MarkPrices[symbolIndex] = *markPrice
	}
}
