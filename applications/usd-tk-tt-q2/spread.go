package main

import (
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func (strat *XYStrategy) updateSpread() {
	if !strat.stats.Ready.True() ||
		strat.xTicker == nil ||
		strat.yTicker == nil ||
		strat.xTickerTimeDelta > strat.stats.XTimeDeltaTop.Load() ||
		strat.xTickerTimeDelta < strat.stats.XTimeDeltaBot.Load() ||
		strat.yTickerTimeDelta > strat.stats.YTimeDeltaTop.Load() ||
		strat.yTickerTimeDelta < strat.stats.YTimeDeltaBot.Load() {
		return
	}
	strat.xyTickerTimeDelta = strat.yTickerTime.Sub(strat.xTickerTime)
	if strat.xyTickerTimeDelta > strat.stats.XYTimeDeltaTop.Load() ||
		strat.xyTickerTimeDelta < strat.stats.XYTimeDeltaBot.Load() {
		return
	}

	strat.spreadEventTime = time.Now()
	if strat.xyTickerTimeDelta > 0 {
		strat.spreadTickerTime = strat.yTickerTime
	} else {
		strat.spreadTickerTime = strat.xTickerTime
	}

	strat.tickerMatchCount++

	strat.spreadLastShort = (strat.yTicker.GetBidPrice()*strat.yExchange.GetPriceFactor() - strat.xTicker.GetAskPrice()*strat.xExchange.GetPriceFactor()) / (strat.xTicker.GetAskPrice()*strat.xExchange.GetPriceFactor())
	strat.spreadLastLong = (strat.yTicker.GetAskPrice()*strat.yExchange.GetPriceFactor() - strat.xTicker.GetBidPrice()*strat.xExchange.GetPriceFactor()) / (strat.xTicker.GetBidPrice()*strat.xExchange.GetPriceFactor())

	strat.spreadMedianShort = strat.spreadShortTimedMedian.Insert(strat.spreadTickerTime, strat.spreadLastShort)
	strat.spreadMedianLong = strat.spreadLongTimedMedian.Insert(strat.spreadTickerTime, strat.spreadLastLong)

	strat.spreadReady = true

	strat.updateXPosition()
	if time.Now().Sub(strat.hedgeCheckStopTime) > 0 {
		strat.hedgeYPosition()
	}
}

func (strat *XYStrategy) handleTicker() {
	if strat.xSystemStatus != common.SystemStatusReady ||
		strat.ySystemStatus != common.SystemStatusReady ||
		time.Now().Sub(strat.nextTicker.GetEventTime()) > strat.config.TickerMaxRemoteLocalTimeDiff {
		return
	} else if strat.nextTicker.GetExchange() == strat.xExchangeID {
		strat.xNextTicker = strat.nextTicker
		strat.handleXTicker()
	} else if strat.nextTicker.GetExchange() == strat.yExchangeID {
		strat.yNextTicker = strat.nextTicker
		strat.handleYTicker()
	} else {
		logger.Debugf("other ticker %v", strat.nextTicker)
	}
}

func (strat *XYStrategy) handleXTicker() {
	if strat.xTicker == strat.xNextTicker {
		return
	}
	if strat.xNextTicker.GetEventTime().Sub(strat.xTickerTime) < 0 {
		return
	}
	strat.xTicker = strat.xNextTicker
	strat.xMidPrice = 0.5 * (strat.xTicker.GetAskPrice() + strat.xTicker.GetBidPrice())
	strat.xTickerTime = strat.xTicker.GetEventTime()
	strat.xTickerTimeDelta = strat.xTickerTime.Sub(time.Now())
	strat.spreadWalkTimer.Reset(strat.config.SpreadWalkDelay)
	strat.tickerCount++
	select {
	case strat.stats.XTickerCh <- strat.xTicker:
	default:
	}
}

func (strat *XYStrategy) handleYTicker() {
	if strat.yTicker == strat.yNextTicker {
		return
	}
	if strat.yNextTicker.GetEventTime().Sub(strat.yTickerTime) < 0 {
		return
	}
	strat.yTicker = strat.yNextTicker
	strat.yMidPrice = 0.5 * (strat.yTicker.GetAskPrice() + strat.yTicker.GetBidPrice())
	strat.yTickerTime = strat.yTicker.GetEventTime()
	strat.yTickerTimeDelta = strat.yTickerTime.Sub(time.Now())
	strat.tickerCount++
	strat.spreadWalkTimer.Reset(strat.config.SpreadWalkDelay)
	select {
	case strat.stats.YTickerCh <- strat.yTicker:
	default:
	}
}
