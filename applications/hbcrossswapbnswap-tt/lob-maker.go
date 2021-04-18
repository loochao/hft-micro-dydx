package main

import (
	"context"
	"github.com/geometrybase/hft-micro/hbcrossswap"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func watchMakerWalkedOrderBooks(
	ctx context.Context,
	cancel context.CancelFunc,
	proxyAddress string,
	contractSizes map[string]float64,
	impact float64, symbols []string,
	outputWLob chan WalkedOrderBook,
) {
	lastEventTimes := make(map[string]time.Time)
	for _, symbol := range symbols {
		lastEventTimes[symbol] = time.Unix(0, 0)
	}
	ws := hbcrossswap.NewDepth20Websocket(
		ctx,
		symbols,
		proxyAddress,
	)
	defer ws.Stop()
	for {
		select {
		case <-ws.Done():
			logger.Debugf("DEPTH20 WS CONTEXT DONE %s", symbols)
			cancel()
			return
		case <-ctx.Done():
			return
		case lob := <-ws.DataCh:
			if lastEventTimes[lob.Symbol].Sub(lob.EventTime) < 0 {
				lastEventTimes[lob.Symbol] = lob.EventTime
				if m, ok := contractSizes[lob.Symbol]; ok {
					if len(outputWLob) > 0 {
						logger.Debugf("MAKER DEPTH OUTPUT LEN %d", len(outputWLob))
					}
					outputWLob <- walkPerpOrderBook(lob, impact, m)
				}
			}
			break
		}
	}
}

func walkPerpOrderBook(orderBook *hbcrossswap.Depth20, impact, contractSize float64) WalkedOrderBook {
	wLob := WalkedOrderBook{
		Symbol:    orderBook.Symbol,
		Type:      WalkedOrderBookTypeMaker,
		ParseTime: orderBook.ParseTime,
		EventTime: orderBook.EventTime,
	}
	totalValue := 0.0
	totalQty := 0.0
	for _, bid := range orderBook.Bids {
		value := bid[0] * bid[1] * contractSize
		wLob.BidFarPrice = bid[0]
		if totalValue+value >= impact {
			totalQty += (impact - totalValue) / bid[0]
			totalValue = impact
			break
		} else {
			totalQty += bid[1] * contractSize
			totalValue += value
		}
	}
	wLob.BidVWAP = totalValue / totalQty

	totalValue = 0.0
	totalQty = 0.0
	for _, ask := range orderBook.Asks {
		value := ask[0] * ask[1] * contractSize
		wLob.TakerAskFarPrice = ask[0]
		if totalValue+value >= impact {
			totalQty += (impact - totalValue) / ask[0]
			totalValue = impact
			break
		} else {
			totalQty += ask[1] * contractSize
			totalValue += value
		}
	}
	wLob.AskVWAP = totalValue / totalQty
	wLob.ImpactValue = impact
	wLob.BidPrice = orderBook.Bids[0][0]
	wLob.BidSize = orderBook.Bids[0][1] * contractSize
	wLob.AskPrice = orderBook.Asks[0][0]
	wLob.AskSize = orderBook.Asks[0][1] * contractSize
	return wLob
}
