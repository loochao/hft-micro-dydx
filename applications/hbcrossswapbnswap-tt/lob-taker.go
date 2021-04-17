package main

import (
	"context"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func watchTakerWalkedOrderBooks(
	ctx context.Context, proxyAddress string,
	impact float64, symbols []string, output chan WalkedOrderBook) {
	lastEventTimes := make(map[string]time.Time)
	for _, s := range symbols {
		lastEventTimes[s] = time.Unix(0, 0)
	}

	ws := bnswap.NewDepth20Websocket(ctx, symbols, proxyAddress)
	defer ws.Stop()

	for {
		select {
		case <-ws.Done():
			logger.Fatal("B DEPTH50 WS CONTEXT DONE %s", symbols)
		case <-ctx.Done():
			return
		case data := <-ws.DataCh:
			if lastEventTimes[data.Symbol].Sub(data.EventTime) < 0 {
				lastEventTimes[data.Symbol] = data.EventTime
				output <- walkTakerOrderBook(data, impact)
			}
			break
		}
	}
}

func walkTakerOrderBook(orderBook *bnswap.Depth20, impact float64) WalkedOrderBook {
	wLob := WalkedOrderBook{
		Symbol:    orderBook.Symbol,
		Type:      WalkedOrderBookTypeTaker,
		ParseTime: orderBook.ParseTime,
	}
	totalValue := 0.0
	totalQty := 0.0
	for _, bid := range orderBook.Bids {
		value := bid[0] * bid[1]
		wLob.BidFarPrice = bid[0]
		if totalValue+value >= impact {
			totalQty += (impact - totalValue) / bid[0]
			totalValue = impact
			break
		} else {
			totalQty += bid[1]
			totalValue += value
		}
	}
	wLob.BidVWAP = totalValue / totalQty

	totalValue = 0.0
	totalQty = 0.0
	for _, ask := range orderBook.Asks {
		value := ask[0] * ask[1]
		wLob.TakerAskFarPrice = ask[0]
		if totalValue+value >= impact {
			totalQty += (impact - totalValue) / ask[0]
			totalValue = impact
			break
		} else {
			totalQty += ask[1]
			totalValue += value
		}
	}
	wLob.AskVWAP = totalValue / totalQty
	wLob.ImpactValue = impact
	wLob.BidPrice = orderBook.Bids[0][0]
	wLob.BidSize = orderBook.Bids[0][1]
	wLob.AskPrice = orderBook.Asks[0][0]
	wLob.AskSize = orderBook.Asks[0][1]
	return wLob
}
