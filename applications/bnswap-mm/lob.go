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
	quantilesCh chan map[string]Quantile,
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
	var quantiles map[string]Quantile
	for {
		select {
		case quantiles = <- quantilesCh:
			logger.Debugf("QUANTILES %v", quantiles)
			break
		case <-ws.Done():
			logger.Fatal("DEPTH20 WS CONTEXT DONE %s", symbols)
		case <-ctx.Done():
			return
		case lob := <-ws.DataCh:
			if lastUpdatedIds[lob.Symbol] < lob.LastUpdateId {
				lastUpdatedIds[lob.Symbol] = lob.LastUpdateId
				if quantiles != nil {
					if q, ok := quantiles[lob.Symbol]; ok {
						outputWLob <- walkSwapOrderBook(lob, q.Open, q.Close)
					}
				}
			}
			break
		}
	}
}

func walkSwapOrderBook(orderBook *bnswap.Depth, openImpact, closeImpact float64) WalkedOrderBook {
	wLob := WalkedOrderBook{
		Symbol:      orderBook.Symbol,
		ArrivalTime: orderBook.ArrivalTime,
		EventTime:   orderBook.EventTime,
	}
	totalTakerValue := 0.0
	totalTakerQty := 0.0
	totalMakerValue := 0.0
	totalMakerQty := 0.0
	hasMakerData := false
	hasTakerData := false
	for _, bid := range orderBook.Bids {
		value := bid[0] * bid[1]
		if !hasMakerData {
			wLob.OpenBidFarPrice = bid[0]
			if totalMakerValue+value >= closeImpact {
				totalMakerQty += (closeImpact - totalMakerValue) / bid[0]
				totalMakerValue = closeImpact
				hasMakerData = true
			} else {
				totalMakerQty += bid[1]
				totalMakerValue += value
			}
		}
		if !hasTakerData {
			wLob.CloseBidFarPrice = bid[0]
			if totalTakerValue+value >= openImpact {
				totalTakerQty += (openImpact - totalTakerValue) / bid[0]
				totalTakerValue = openImpact
				hasTakerData = true
			} else {
				totalTakerQty += bid[1]
				totalTakerValue += value
			}
		}
		if hasMakerData && hasTakerData {
			break
		}
	}
	wLob.CloseBidVWAP = totalTakerValue / totalTakerQty
	wLob.OpenBidVWAP = totalMakerValue / totalMakerQty

	totalTakerValue = 0.0
	totalTakerQty = 0.0
	totalMakerValue = 0.0
	totalMakerQty = 0.0
	hasMakerData = false
	hasTakerData = false
	for _, ask := range orderBook.Asks {
		value := ask[0] * ask[1]
		if !hasMakerData {
			wLob.OpenAskFarPrice = ask[0]
			if totalMakerValue+value >= closeImpact {
				totalMakerQty += (closeImpact - totalMakerValue) / ask[0]
				totalMakerValue = closeImpact
				hasMakerData = true
			} else {
				totalMakerQty += ask[1]
				totalMakerValue += value
			}
		}
		if !hasTakerData {
			wLob.CloseAskFarPrice = ask[0]
			if totalTakerValue+value >= openImpact {
				totalTakerQty += (openImpact - totalTakerValue) / ask[0]
				totalTakerValue = openImpact
				hasTakerData = true
			} else {
				totalTakerQty += ask[1]
				totalTakerValue += value
			}
		}
		if hasMakerData && hasTakerData {
			break
		}
	}
	wLob.CloseAskVWAP = totalTakerValue / totalTakerQty
	wLob.OpenAskVWAP = totalMakerValue / totalMakerQty
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
