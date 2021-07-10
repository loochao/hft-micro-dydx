package main

import (
	"github.com/geometrybase/hft-micro/common"
	"time"
)

func (strat *XYStrategy) updateSpread() {
	//ticker 盘口无变动可能更新得很慢

	//需要用ema time delta 对age diff进行修正
	strat.adjustedAgeDiff = strat.xTicker.GetTime().Sub(strat.yTicker.GetTime()) + time.Duration(strat.xTickerFilter.TimeDeltaEma-strat.yTickerFilter.TimeDeltaEma)*time.Millisecond

	//取新一点的时间为spread time
	if strat.xTicker.GetTime().Sub(strat.yTicker.GetTime()) < 0 {
		//需要对时间进行补偿
		strat.spreadTime = strat.yTicker.GetTime().Add(time.Millisecond * time.Duration(strat.yTickerFilter.TimeDeltaEma))
	} else {
		//需要对时间进行补偿
		strat.spreadTime = strat.xTicker.GetTime().Add(time.Millisecond * time.Duration(strat.xTickerFilter.TimeDeltaEma))
	}

	if strat.adjustedAgeDiff > strat.config.TickerMaxAgeDiffBias {
		strat.yTickerExpireCount++
		//logger.Debugf("%s x expire y %v %v %v", xSymbol, xTickerTime.Sub(yTickerTime), adjustedAgeDiff, -time.Duration(xTickerFilter.TimeDeltaEma-yTickerFilter.TimeDeltaEma)*time.Millisecond)
		return
	} else if strat.adjustedAgeDiff < -strat.config.TickerMaxAgeDiffBias {
		//logger.Debugf("%s y expire x %v %v %v", xSymbol, xTickerTime.Sub(yTickerTime), adjustedAgeDiff, -time.Duration(xTickerFilter.TimeDeltaEma-yTickerFilter.TimeDeltaEma)*time.Millisecond)
		strat.xTickerExpireCount++
		return
	}

	strat.tickerMatchCount++

	//假定挂单基于MidPrice, 考虑挂单的下界偏移进Spread
	//如果想挂得远，成交少，吃大Spread, 可以orderOffsets参数，推NearBot NearTop, 反之亦然
	strat.shortLastEnter = (strat.yTicker.GetBidPrice() - strat.xTicker.GetAskPrice()) / strat.xTicker.GetAskPrice()
	strat.longLastEnter = (strat.yTicker.GetAskPrice() - strat.xTicker.GetBidPrice()) / strat.xTicker.GetBidPrice()

	strat.shortEnterTimedMedian.Insert(strat.spreadTime, strat.shortLastEnter)
	strat.longEnterTimedMedian.Insert(strat.spreadTime, strat.longLastEnter)

	strat.spread = &common.XYSpread{
		ShortLastEnter:   strat.shortLastEnter,
		ShortLastLeave:   strat.longLastEnter,
		ShortMedianEnter: strat.shortEnterTimedMedian.Median(),
		ShortMedianLeave: strat.longEnterTimedMedian.Median(),

		LongLastEnter:   strat.longLastEnter,
		LongLastLeave:   strat.shortLastEnter,
		LongMedianEnter: strat.longEnterTimedMedian.Median(),
		LongMedianLeave: strat.shortEnterTimedMedian.Median(),
		EventTime:       strat.spreadTime,
		ParseTime:       time.Now(),
	}
	strat.updateXPosition()
	if time.Now().Sub(strat.hedgeCheckStopTime) > 0 {
		strat.hedgeYPosition()
	}
}

func (strat *XYStrategy) handleTicker() {
	if strat.nextTicker.GetExchange() == strat.xExchangeID {
		strat.xNextTicker = strat.nextTicker
		strat.handleXTicker()
	} else if strat.nextTicker.GetExchange() == strat.yExchangeID {
		strat.yNextTicker = strat.nextTicker
		strat.handleYTicker()
	}
}

func (strat *XYStrategy) handleXTicker() {
	if strat.xTicker == strat.xNextTicker {
		return
	}
	if strat.xNextTicker.GetTime().Sub(strat.xTickerTime) < 0 {
		return
	}
	strat.xTicker = strat.xNextTicker
	strat.xMidPrice = 0.5 * (strat.xTicker.GetAskPrice() + strat.xTicker.GetBidPrice())
	strat.xTickerTime = strat.xTicker.GetTime()
	if !strat.xTickerFilter.Filter(strat.xTicker) && strat.yTicker != nil {
		strat.adjustedAgeDiff = strat.xTickerTime.Sub(strat.yTickerTime) + time.Duration(strat.xTickerFilter.TimeDeltaEma-strat.yTickerFilter.TimeDeltaEma)*time.Millisecond
		if strat.adjustedAgeDiff > strat.config.TickerMaxAgeDiffBias {
			//taker已经过期
			strat.yTickerExpireCount++
			//logger.Debugf("%s x expire y %v %v %v", xSymbol, xTickerTime.Sub(yTickerTime), adjustedAgeDiff, -time.Duration(xTickerFilter.TimeDeltaEma-yTickerFilter.TimeDeltaEma)*time.Millisecond)
		} else if strat.adjustedAgeDiff < -strat.config.TickerMaxAgeDiffBias {
			//maker已经过期
			strat.xTickerExpireCount++
			//logger.Debugf("%s y expire x %v %v %v", xSymbol, xTickerTime.Sub(yTickerTime), adjustedAgeDiff, -time.Duration(xTickerFilter.TimeDeltaEma-yTickerFilter.TimeDeltaEma)*time.Millisecond)
		} else {
			strat.spreadWalkTimer.Reset(strat.config.SpreadWalkDelay)
		}
	}
	strat.tickerCount++
	if strat.tickerCount > strat.config.TickerReportCount {
		strat.xTickerFilter.GenerateReport()
		strat.yTickerFilter.GenerateReport()
		strat.spreadReport = &common.XYSpreadReport{
			MatchRatio:         float64(strat.tickerMatchCount) / float64(strat.tickerCount),
			XSymbol:            strat.xSymbol,
			YSymbol:            strat.ySymbol,
			XTimeDeltaEma:      strat.xTickerFilter.TimeDeltaEma,
			YTimeDeltaEma:      strat.yTickerFilter.TimeDeltaEma,
			XTimeDelta:         strat.xTickerFilter.TimeDelta,
			YTimeDelta:         strat.yTickerFilter.TimeDelta,
			XTickerFilterRatio: strat.xTickerFilter.Report.FilterRatio,
			YTickerFilterRatio: strat.yTickerFilter.Report.FilterRatio,
			XExpireRatio:       float64(strat.xTickerExpireCount) / float64(strat.tickerCount),
			YExpireRatio:       float64(strat.yTickerExpireCount) / float64(strat.tickerCount),
		}
		strat.tickerMatchCount = 0
		strat.tickerCount = 0
		strat.yTickerExpireCount = 0
		strat.xTickerExpireCount = 0
	}
}

func (strat *XYStrategy) handleYTicker() {
	if strat.yTicker == strat.yNextTicker {
		return
	}
	if strat.yNextTicker.GetTime().Sub(strat.yTickerTime) < 0 {
		return
	}
	strat.yTicker = strat.yNextTicker
	strat.yMidPrice = 0.5 * (strat.yTicker.GetAskPrice() + strat.yTicker.GetBidPrice())
	strat.yTickerTime = strat.yTicker.GetTime()
	if !strat.yTickerFilter.Filter(strat.yTicker) && strat.xTicker != nil {
		strat.adjustedAgeDiff = strat.xTickerTime.Sub(strat.yTickerTime) + time.Duration(strat.xTickerFilter.TimeDeltaEma-strat.yTickerFilter.TimeDeltaEma)*time.Millisecond
		if strat.adjustedAgeDiff < -strat.config.TickerMaxAgeDiffBias {
			//maker已经过期
			//logger.Debugf("%s y expire x %v %v %v", xSymbol, xTickerTime.Sub(yTickerTime), adjustedAgeDiff, -time.Duration(xTickerFilter.TimeDeltaEma-yTickerFilter.TimeDeltaEma)*time.Millisecond)
			strat.xTickerExpireCount++
		} else if strat.adjustedAgeDiff > strat.config.TickerMaxAgeDiffBias {
			//logger.Debugf("%s x expire y %v %v %v", xSymbol, xTickerTime.Sub(yTickerTime), adjustedAgeDiff, -time.Duration(xTickerFilter.TimeDeltaEma-yTickerFilter.TimeDeltaEma)*time.Millisecond)
			//taker已经过期
			strat.yTickerExpireCount++
		} else {
			strat.spreadWalkTimer.Reset(strat.config.SpreadWalkDelay)
		}
	}
	strat.tickerCount++
	if strat.tickerCount > strat.config.TickerReportCount {
		strat.xTickerFilter.GenerateReport()
		strat.yTickerFilter.GenerateReport()
		strat.spreadReport = &common.XYSpreadReport{
			MatchRatio:         float64(strat.tickerMatchCount) / float64(strat.tickerCount),
			XSymbol:            strat.xSymbol,
			YSymbol:            strat.ySymbol,
			XTimeDeltaEma:      strat.xTickerFilter.TimeDeltaEma,
			YTimeDeltaEma:      strat.yTickerFilter.TimeDeltaEma,
			XTimeDelta:         strat.xTickerFilter.TimeDelta,
			YTimeDelta:         strat.yTickerFilter.TimeDelta,
			XTickerFilterRatio: strat.xTickerFilter.Report.FilterRatio,
			YTickerFilterRatio: strat.yTickerFilter.Report.FilterRatio,
			XExpireRatio:       float64(strat.xTickerExpireCount) / float64(strat.tickerCount),
			YExpireRatio:       float64(strat.yTickerExpireCount) / float64(strat.tickerCount),
		}
		strat.tickerMatchCount = 0
		strat.tickerCount = 0
		strat.yTickerExpireCount = 0
		strat.xTickerExpireCount = 0
	}
}
