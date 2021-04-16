package main

import (
	"context"
	"github.com/geometrybase/hft-micro/kcperp"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func watchPerpWalkedOrderBooks(
	ctx context.Context, api *kcperp.API, proxyAddress string,
	multipliers map[string]float64,
	takerImpact, makerImpact float64, symbols []string,
	outputWLob chan WalkedOrderBook,
) {
	lastEventTimes := make(map[string]time.Time)
	for _, symbol := range symbols {
		lastEventTimes[symbol] = time.Unix(0, 0)
	}
	ws := kcperp.NewDepth5Websocket(
		ctx,
		api,
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
				if m, ok := multipliers[lob.Symbol]; ok {
					outputWLob <- walkPerpOrderBook(lob, takerImpact, makerImpact, m)
				}
			}
			break
		case <-ws.RestartCh:
			break
		}
	}
}

func walkPerpOrderBook(orderBook *kcperp.Depth5, takerImpact, makerImpact, multiplier float64) WalkedOrderBook {
	wLob := WalkedOrderBook{
		Symbol:    orderBook.Symbol,
		Type:      WalkedOrderBookTypePerp,
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
		value := bid[0] * bid[1] * multiplier
		if !hasMakerData {
			wLob.MakerBidFarPrice = bid[0]
			if totalMakerValue+value >= makerImpact {
				totalMakerQty += (makerImpact - totalMakerValue) / bid[0]
				totalMakerValue = makerImpact
				hasMakerData = true
			} else {
				totalMakerQty += bid[1] * multiplier
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
				totalTakerQty += bid[1] * multiplier
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
		value := ask[0] * ask[1] * multiplier
		if !hasMakerData {
			wLob.MakerAskFarPrice = ask[0]
			if totalMakerValue+value >= makerImpact {
				totalMakerQty += (makerImpact - totalMakerValue) / ask[0]
				totalMakerValue = makerImpact
				hasMakerData = true
			} else {
				totalMakerQty += ask[1] * multiplier
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
				totalTakerQty += ask[1] * multiplier
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
	wLob.BidSize = orderBook.Bids[0][1] * multiplier
	wLob.AskPrice = orderBook.Asks[0][0]
	wLob.AskSize = orderBook.Asks[0][1] * multiplier
	return wLob
}

func watchInstrument(
	ctx context.Context, api *kcperp.API, proxyAddress string,
	symbols []string,
	mpCh chan *kcperp.MarkPrice,
) {
	ws := kcperp.NewInstrumentWebsocket(
		ctx,
		api,
		symbols,
		proxyAddress,
		mpCh,
	)
	defer ws.Stop()
	for {
		select {
		case <-ws.Done():
			logger.Fatal("INSTRUMENT WS CONTEXT DONE %s", symbols)
			return
		case <-ctx.Done():
			logger.Debugf("CTX DONE")
			return
		}
	}
}
