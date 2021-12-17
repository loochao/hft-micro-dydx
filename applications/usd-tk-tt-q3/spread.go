package main

import (
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func (strat *XYStrategy) updateSpread() {

	if !strat.Stats.Ready ||
		strat.xTicker == nil ||
		strat.yTicker == nil ||
		strat.xTickerTimeDelta > strat.Stats.XEventTimeDeltaTop ||
		strat.xTickerTimeDelta < strat.Stats.XEventTimeDeltaBot ||
		strat.yTickerTimeDelta > strat.Stats.YEventTimeDeltaTop ||
		strat.yTickerTimeDelta < strat.Stats.YEventTimeDeltaBot ||
		strat.xTicker.GetBidPrice() <= 0 ||
		strat.xTicker.GetAskPrice() <= 0 ||
		strat.yTicker.GetBidPrice() <= 0 ||
		strat.yTicker.GetAskPrice() <= 0 {
		return
	}

	strat.xyTickerTimeDelta = strat.yTickerTime.Sub(strat.xTickerTime)
	if strat.xyTickerTimeDelta > strat.Stats.XYEventTimeDeltaTop ||
		strat.xyTickerTimeDelta < strat.Stats.XYEventTimeDeltaBot {
		return
	}

	strat.spreadEventTime = time.Now()
	if strat.xyTickerTimeDelta > 0 {
		strat.spreadTickerTime = strat.xTickerTime
	} else {
		strat.spreadTickerTime = strat.yTickerTime
	}

	strat.tickerMatchCount++

	strat.spreadLastShort = (strat.yTicker.GetBidPrice() - strat.xTicker.GetAskPrice()) / strat.xTicker.GetAskPrice()
	strat.spreadLastLong = (strat.yTicker.GetAskPrice() - strat.xTicker.GetBidPrice()) / strat.xTicker.GetBidPrice()

	strat.spreadMedianShort = strat.spreadShortTimedMean.Insert(strat.spreadTickerTime, strat.spreadLastShort)
	strat.spreadMedianLong = strat.spreadLongTimedMean.Insert(strat.spreadTickerTime, strat.spreadLastLong)

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
	//有可能是同一个指针地址
	//if strat.xTicker == strat.xNextTicker {
	//	return
	//}
	if strat.xNextTicker.GetEventTime().Sub(strat.xTickerTime) < 0 {
		return
	}
	strat.xTicker = strat.xNextTicker
	strat.xMidPrice = 0.5 * (strat.xTicker.GetAskPrice() + strat.xTicker.GetBidPrice())
	strat.xTickerTime = strat.xTicker.GetEventTime()
	strat.xTickerTimeDelta = strat.xTickerTime.Sub(time.Now())
	if strat.config.SpreadWalkByXTicker {
		strat.spreadWalkTimer.Reset(strat.config.SpreadWalkDelay)
	}
	strat.tickerCount++
	select {
	case strat.Stats.XTickerCh <- strat.xTicker:
	default:
	}
}

func (strat *XYStrategy) handleYTicker() {
	//有可能是同一个指针地址
	//if strat.yTicker == strat.yNextTicker {
	//	return
	//}
	if strat.yNextTicker.GetEventTime().Sub(strat.yTickerTime) < 0 {
		return
	}
	strat.yTicker = strat.yNextTicker
	strat.yMidPrice = 0.5 * (strat.yTicker.GetAskPrice() + strat.yTicker.GetBidPrice())
	strat.yTickerTime = strat.yTicker.GetEventTime()
	strat.yTickerTimeDelta = strat.yTickerTime.Sub(time.Now())
	strat.tickerCount++
	if strat.config.SpreadWalkByYTicker {
		strat.spreadWalkTimer.Reset(strat.config.SpreadWalkDelay)
	}
	select {
	case strat.Stats.YTickerCh <- strat.yTicker:
	default:
	}
}
