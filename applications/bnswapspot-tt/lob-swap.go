package main

import (
	"context"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func watchSwapWalkedOrderBooks(
	ctx context.Context, proxyAddress string,
	takerImpact, makerImpact float64, symbols []string,
	outputWLob chan WalkedOrderBook,
) {
	lastUpdatedIds := make(map[string]int64)
	ws := bnswap.NewDepth20Ws(
		ctx,
		symbols,
		time.Minute,
		proxyAddress,
	)
	defer ws.Stop()

	for {
		select {
		case <-ws.Done():
			logger.Fatal("DEPTH20 WS CONTEXT DONE %s", symbols)
		case <-ctx.Done():
			return
		case lob := <-ws.DataCh:
			if lastUpdatedIds[lob.Symbol] < lob.LastUpdateId {
				lastUpdatedIds[lob.Symbol] = lob.LastUpdateId
				outputWLob <- walkSwapOrderBook(lob, takerImpact, makerImpact)
			}
			break
		}
	}
}

func walkSwapOrderBook(orderBook *bnswap.Depth, takerImpact, makerImpact float64) WalkedOrderBook {
	wLob := WalkedOrderBook{
		Symbol:      orderBook.Symbol,
		Type:        WalkedOrderBookTypeSwap,
		ArrivalTime: orderBook.ArrivalTime,
		EventTime:   orderBook.EventTime,
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
