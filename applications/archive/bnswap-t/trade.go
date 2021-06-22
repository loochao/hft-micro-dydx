package main

import (
	"context"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func startTradesRoutine(
	ctx context.Context,
	proxyAddress string,
	symbols []string,
	outputChs [SYMBOLS_LEN]chan *bnswap.Trade,
) {
	lastEventTimes := [SYMBOLS_LEN]int64{}
	for i := 0; i < SYMBOLS_LEN; i++ {
		lastEventTimes[i] = 0
	}
	ws := bnswap.NewTradeWebsocket(
		ctx,
		symbols,
		time.Minute,
		proxyAddress,
	)
	defer ws.Stop()

	index := -1
	for {
		select {
		case <-ws.Done():
			logger.Fatal("TRADE WS CONTEXT DONE %s", symbols)
		case <-ctx.Done():
			return
		case trade := <-ws.DataCh:
			index = GetSymbolIndex(trade.Symbol)
			if index != -1 {
				if lastEventTimes[index] < trade.EventTime {
					lastEventTimes[index] = trade.EventTime
					outputChs[index] <- trade
				}
			}
			break
		}
	}
}
