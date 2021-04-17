package main

import (
	"context"
	"github.com/geometrybase/hft-micro/hbcrossswap"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func watchHWalkedOrderBooks(
	ctx context.Context, proxyAddress string,
	contractSizes map[string]float64,
	takerImpact, makerImpact float64, symbols []string,
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
			logger.Fatal("DEPTH50 WS CONTEXT DONE %s", symbols)
		case <-ctx.Done():
			return
		case lob := <-ws.DataCh:
			if lastEventTimes[lob.Symbol].Sub(lob.EventTime) < 0 {
				lastEventTimes[lob.Symbol] = lob.EventTime
				if m, ok := contractSizes[lob.Symbol]; ok {
					outputWLob <- walkPerpOrderBook(lob, takerImpact, makerImpact, m)
				}
			}
			break
		}
	}
}

func walkPerpOrderBook(orderBook *hbcrossswap.Depth20, takerImpact, makerImpact, contractSize float64) WalkedOrderBook {
	wLob := WalkedOrderBook{
		Symbol:    orderBook.Symbol,
		Type:      WalkedOrderBookTypeMaker,
		ParseTime: orderBook.ParseTime,
		EventTime: orderBook.EventTime,
	}
	totalTakerValue := 0.0
	totalTakerQty := 0.0
	totalMakerValue := 0.0
	totalMakerQty := 0.0
	hasMakerData := false
	hasTakerData := false
	for _, bid := range orderBook.Bids {
		value := bid[0] * bid[1] * contractSize
		if !hasMakerData {
			wLob.MakerBidFarPrice = bid[0]
			if totalMakerValue+value >= makerImpact {
				totalMakerQty += (makerImpact - totalMakerValue) / bid[0]
				totalMakerValue = makerImpact
				hasMakerData = true
			} else {
				totalMakerQty += bid[1] * contractSize
				totalMakerValue += value
			}
		}
		if !hasTakerData {
			wLob.TakerBidFarPrice = bid[0]
			if totalTakerValue+value >= takerImpact {
				totalTakerQty += (takerImpact - totalTakerValue) / bid[0]
				totalTakerValue = takerImpact
				hasTakerData = true
			} else {
				totalTakerQty += bid[1] * contractSize
				totalTakerValue += value
			}
		}
		if hasMakerData && hasTakerData {
			break
		}
	}
	wLob.TakerBidVWAP = totalTakerValue / totalTakerQty
	wLob.MakerBidVWAP = totalMakerValue / totalMakerQty

	totalTakerValue = 0.0
	totalTakerQty = 0.0
	totalMakerValue = 0.0
	totalMakerQty = 0.0
	hasMakerData = false
	hasTakerData = false
	for _, ask := range orderBook.Asks {
		value := ask[0] * ask[1] * contractSize
		if !hasMakerData {
			wLob.MakerAskFarPrice = ask[0]
			if totalMakerValue+value >= makerImpact {
				totalMakerQty += (makerImpact - totalMakerValue) / ask[0]
				totalMakerValue = makerImpact
				hasMakerData = true
			} else {
				totalMakerQty += ask[1] * contractSize
				totalMakerValue += value
			}
		}
		if !hasTakerData {
			wLob.TakerAskFarPrice = ask[0]
			if totalTakerValue+value >= takerImpact {
				totalTakerQty += (takerImpact - totalTakerValue) / ask[0]
				totalTakerValue = takerImpact
				hasTakerData = true
			} else {
				totalTakerQty += ask[1] * contractSize
				totalTakerValue += value
			}
		}
		if hasMakerData && hasTakerData {
			break
		}
	}
	wLob.TakerAskVWAP = totalTakerValue / totalTakerQty
	wLob.MakerAskVWAP = totalMakerValue / totalMakerQty
	wLob.ImpactValue = takerImpact
	wLob.BidPrice = orderBook.Bids[0][0]
	wLob.BidSize = orderBook.Bids[0][1] * contractSize
	wLob.AskPrice = orderBook.Asks[0][0]
	wLob.AskSize = orderBook.Asks[0][1] * contractSize
	return wLob
}

