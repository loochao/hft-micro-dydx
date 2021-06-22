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
	timedTradeVolume := common.NewTimedSum(tradeLookback)
	timedTickDirection := common.NewTimedSum(tradeLookback)
	updateTimer := time.NewTimer(updateInterval)
	var depth common.Depth
	var trade common.Trade
	var loopTime = time.Now()
	for {

		select {
		case <-ctx.Done():
			return
		case <-updateTimer.C:
			updateTimer.Reset(updateInterval)
			if time.Now().Sub(loopTime) < updateInterval {
				continue
			}
			if depth == nil || trade == nil {
				continue
			}
			loopTime = time.Now()
			bookVolume := 0.0
			bids := depth.GetBids()
			asks := depth.GetAsks()
			for i := 0; i < depthLevel && i < len(bids); i++ {
				bookVolume += bids[i][0] * bids[i][1]
			}
			for i := 0; i < depthLevel && i < len(asks); i++ {
				bookVolume += asks[i][0] * asks[i][1]
			}
			direction := 0.0
			tradeBookRatio := 0.0
			if bookVolume != 0 {
				tradeBookRatio = timedTradeVolume.Sum() / bookVolume
			}
			if timedTradeVolume.Sum() != 0 {
				direction = timedTickDirection.Sum() / timedTradeVolume.Sum()
			}
			signal := Signal{
				Symbol:         symbol,
				TradeBookRatio: tradeBookRatio,
				TradeVolume:    timedTradeVolume.Sum(),
				BookVolume:     bookVolume,
				Direction:      direction,
				BestBidPrice:   bids[0][0],
				BestAskPrice:   bids[0][0],
				LastTradePrice: trade.GetPrice(),
			}
			select {
			case outputCh <- signal:
			default:
				logger.Debugf("outputCh <- signal failed ch len %d", len(outputCh))
			}
			break
		case trade = <-tradeCh:
			value := trade.GetSize() * trade.GetPrice()
			if trade.IsUpTick() {
				timedTickDirection.Insert(trade.GetTime(), value)
			} else {
				timedTickDirection.Insert(trade.GetTime(), -value)
			}
			timedTradeVolume.Insert(trade.GetTime(), value)
			break
		case depth = <-depthCh:
			break
		}

	}
}
