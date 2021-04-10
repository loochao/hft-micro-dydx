package main

import (
	"context"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func watchSwapWalkedOrderBooks(
	ctx context.Context, proxyAddress string,
	symbols []string,
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
		case quantiles = <-quantilesCh:
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
	totalCloseValue := 0.0
	totalCloseQty := 0.0
	totalOpenValue := 0.0
	totalOpenQty := 0.0
	hasOpenData := false
	hasCloseData := false
	for _, bid := range orderBook.Bids {
		value := bid[0] * bid[1]
		if !hasOpenData {
			wLob.OpenBidFarPrice = bid[0]
			if totalOpenValue+value >= openImpact {
				totalOpenQty += (openImpact - totalOpenValue) / bid[0]
				totalOpenValue = openImpact
				hasOpenData = true
			} else {
				totalOpenQty += bid[1]
				totalOpenValue += value
			}
		}
		if !hasCloseData {
			wLob.CloseBidFarPrice = bid[0]
			if totalCloseValue+value >= closeImpact {
				totalCloseQty += (closeImpact - totalCloseValue) / bid[0]
				totalCloseValue = closeImpact
				hasCloseData = true
			} else {
				totalCloseQty += bid[1]
				totalCloseValue += value
			}
		}
		if hasOpenData && hasCloseData {
			break
		}
	}
	wLob.CloseBidVWAP = totalCloseValue / totalCloseQty
	wLob.OpenBidVWAP = totalOpenValue / totalOpenQty

	totalCloseValue = 0.0
	totalCloseQty = 0.0
	totalOpenValue = 0.0
	totalOpenQty = 0.0
	hasOpenData = false
	hasCloseData = false
	for _, ask := range orderBook.Asks {
		value := ask[0] * ask[1]
		if !hasOpenData {
			wLob.OpenAskFarPrice = ask[0]
			if totalOpenValue+value >= openImpact {
				totalOpenQty += (openImpact - totalOpenValue) / ask[0]
				totalOpenValue = openImpact
				hasOpenData = true
			} else {
				totalOpenQty += ask[1]
				totalOpenValue += value
			}
		}
		if !hasCloseData {
			wLob.CloseAskFarPrice = ask[0]
			if totalCloseValue+value >= closeImpact {
				totalCloseQty += (closeImpact - totalCloseValue) / ask[0]
				totalCloseValue = closeImpact
				hasCloseData = true
			} else {
				totalCloseQty += ask[1]
				totalCloseValue += value
			}
		}
		if hasOpenData && hasCloseData {
			break
		}
	}

	wLob.CloseAskVWAP = totalCloseValue / totalCloseQty
	wLob.OpenAskVWAP = totalOpenValue / totalOpenQty
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
