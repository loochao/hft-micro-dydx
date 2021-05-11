package main

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func streamSignal(
	ctx context.Context,
	symbol string,
	updateInterval time.Duration,
	tradeLookback time.Duration,
	depthLevel int,
	depthCh chan common.Depth,
	tradeCh chan common.Trade,
	outputCh chan Signal,
) {
	timedBuyVolume := common.NewTimedSum(tradeLookback)
	timedSellVolume := common.NewTimedSum(tradeLookback)
	updateTimer := time.NewTimer(updateInterval)
	timedDirection := common.NewTimedMean(tradeLookback)
	var depth common.Depth
	var trade common.Trade
	for {
		select {
		case <-ctx.Done():
			return
		case <-updateTimer.C:
			updateTimer.Reset(updateInterval)
			if depth == nil || trade == nil {
				continue
			}
			if timedBuyVolume.Range() < tradeLookback/2 {
				continue
			}
			if timedSellVolume.Range() < tradeLookback/2 {
				continue
			}
			bidVolume := 0.0
			askVolume := 0.0
			bids := depth.GetBids()
			asks := depth.GetAsks()
			for i := 0; i < depthLevel && i < len(bids); i++ {
				bidVolume += bids[i][0] * bids[i][1]
			}
			for i := 0; i < depthLevel && i < len(asks); i++ {
				askVolume += asks[i][0] * asks[i][1]
			}
			signal := Signal{
				Symbol:         symbol,
				BidVolume:      bidVolume,
				AskVolume:      askVolume,
				BuyVolume:      timedBuyVolume.Sum(),
				SellVolume:     timedSellVolume.Sum(),
				BestBidPrice:   bids[0][0],
				BestAskPrice:   bids[0][0],
				LastTradePrice: trade.GetPrice(),
			}
			select {
			case outputCh <- signal:
			default:
				logger.Debugf("outputCh <- signal failed ch len %d", len(outputCh))
			}
		case trade = <-tradeCh:
			if trade.IsUpTick() {
				timedBuyVolume.Insert(trade.GetTime(), trade.GetSize()*trade.GetPrice())
			} else {
				timedSellVolume.Insert(trade.GetTime(), trade.GetSize()*trade.GetPrice())
			}
			break
		case depth = <-depthCh:
			break
		}
	}
}
