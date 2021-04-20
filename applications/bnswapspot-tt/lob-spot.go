package main

import (
	"context"
	"github.com/geometrybase/hft-micro/bnspot"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func watchSpotWalkedOrderBooks(
	ctx context.Context, proxyAddress string,
	takerImpact, makerImpact float64, symbols []string, output chan WalkedOrderBook) {
	ws := bnspot.NewDepth20Websocket(ctx, symbols, time.Minute, proxyAddress)
	defer ws.Stop()

	lastUpdatedIds := make(map[string]int64)
	for {
		select {
		case data := <-ws.DataCh:
			if lastUpdatedIds[data.Symbol] < data.LastUpdateId {
				lastUpdatedIds[data.Symbol] = data.LastUpdateId
				output <- walkSpotOrderBook(data, takerImpact, makerImpact)
			}
			break
		case <-ws.Done():
			logger.Fatal("DEPTH20 WS CONTEXT DONE %s", symbols)
		case <-ctx.Done():
			return
		}
	}
}

func walkSpotOrderBook(orderBook *bnspot.Depth20, takerImpact, makerImpact float64) WalkedOrderBook {
	wLob := WalkedOrderBook{
		Symbol:      orderBook.Symbol,
		Type:        WalkedOrderBookTypeSpot,
		ArrivalTime: orderBook.ParseTime,
	}
	totalTakerValue := 0.0
	totalTakerQty := 0.0
	hasTakerData := false
	for _, bid := range orderBook.Bids {
		value := bid[0] * bid[1]
		if !hasTakerData {
			wLob.TakerBidFarPrice = bid[0]
			if totalTakerValue+value >= takerImpact {
				totalTakerQty += (takerImpact - totalTakerValue) / bid[0]
				totalTakerValue = takerImpact
				hasTakerData = true
			} else {
				totalTakerQty += bid[1]
				totalTakerValue += value
			}
		}
		if hasTakerData {
			break
		}
	}
	wLob.TakerBidVWAP = totalTakerValue / totalTakerQty

	totalTakerValue = 0.0
	totalTakerQty = 0.0
	hasTakerData = false
	for _, ask := range orderBook.Asks {
		value := ask[0] * ask[1]
		if !hasTakerData {
			wLob.TakerAskFarPrice = ask[0]
			if totalTakerValue+value >= takerImpact {
				totalTakerQty += (takerImpact - totalTakerValue) / ask[0]
				totalTakerValue = takerImpact
				hasTakerData = true
			} else {
				totalTakerQty += ask[1]
				totalTakerValue += value
			}
		}
		if hasTakerData {
			break
		}
	}
	wLob.TakerAskVWAP = totalTakerValue / totalTakerQty
	wLob.ImpactValue = takerImpact
	wLob.BidPrice = orderBook.Bids[0][0]
	wLob.BidSize = orderBook.Bids[0][1]
	wLob.AskPrice = orderBook.Asks[0][0]
	wLob.AskSize = orderBook.Asks[0][1]
	return wLob
}
