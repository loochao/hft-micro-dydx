package main

import (
	"github.com/geometrybase/hft-micro/common"
	"time"
)

func (strat *XYStrategy) walkSpread() {

	//需要用ema time delta 对age diff进行修正
	strat.adjustedAgeDiff = strat.xTicker.GetTime().Sub(strat.yTicker.GetTime()) + time.Duration(strat.xTickerFilter.TimeDeltaEma-strat.yTickerFilter.TimeDeltaEma)*time.Millisecond
	if strat.adjustedAgeDiff > strat.config.TickerMaxAgeDiffBias {
		strat.yTickerExpireCount++
		//logger.Debugf("%s x expire y %v %v %v", xSymbol, xTickerTime.Sub(yTickerTime), adjustedAgeDiff, -time.Duration(xTickerFilter.TimeDeltaEma-yTickerFilter.TimeDeltaEma)*time.Millisecond)
		return
	} else if strat.adjustedAgeDiff < -strat.config.TickerMaxAgeDiffBias {
		//logger.Debugf("%s y expire x %v %v %v", xSymbol, xTickerTime.Sub(yTickerTime), adjustedAgeDiff, -time.Duration(xTickerFilter.TimeDeltaEma-yTickerFilter.TimeDeltaEma)*time.Millisecond)
		strat.xTickerExpireCount++
		return
	}

	//取新一点的时间为spread time
	if strat.xTicker.GetTime().Sub(strat.yTicker.GetTime()) < 0 {
		//需要对时间进行补偿
		strat.spreadTime = strat.yTicker.GetTime().Add(time.Millisecond * time.Duration(strat.yTickerFilter.TimeDeltaEma))
	} else {
		//需要对时间进行补偿
		strat.spreadTime = strat.xTicker.GetTime().Add(time.Millisecond * time.Duration(strat.xTickerFilter.TimeDeltaEma))
	}
	strat.tickerMatchCount++

	//假定挂单基于MidPrice
	strat.shortLastEnter = (strat.yTicker.GetBidPrice()-strat.xMidPrice)/strat.xMidPrice
	strat.longLastEnter = (strat.yTicker.GetAskPrice()-strat.xMidPrice)/strat.xMidPrice

	strat.shortEnterTimedMedian.Insert(strat.spreadTime, strat.shortLastEnter)
	strat.longEnterTimedMedian.Insert(strat.spreadTime, strat.longLastEnter)

	if strat.shortEnterTimedMedian.Len() < strat.config.SpreadMinTickerCount {
		return
	}
	if strat.shortEnterTimedMedian.Range() < strat.config.SpreadLookback/2 {
		return
	}
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
	strat.hedgeYPosition()
	strat.updateXOrder()

	if strat.spreadTime.Sub(strat.quantileLastSampleTime) > strat.config.QuantileSampleInterval {
		strat.quantileLastSampleTime = strat.spreadTime
		_ = strat.timedTDigest.Insert(strat.spreadTime, (strat.shortLastEnter+strat.longLastEnter)*0.5)
		if strat.quantileMiddle == nil {
			strat.quantileMiddle = new(float64)
		}
		*strat.quantileMiddle = strat.timedTDigest.Quantile(0.5)
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
	strat.xMidPrice = 0.5*(strat.xTicker.GetAskPrice()+strat.xTicker.GetBidPrice())
	strat.xTickerTime = strat.xTicker.GetTime()
	if !strat.xTickerFilter.Filter(strat.xTicker) && strat.yTicker != nil {
		strat.spreadWalkTimer.Reset(strat.config.SpreadWalkDelay)
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
	strat.yMidPrice = 0.5*(strat.yTicker.GetAskPrice()+strat.yTicker.GetBidPrice())
	strat.yTickerTime = strat.yTicker.GetTime()
	if !strat.yTickerFilter.Filter(strat.yTicker) && strat.xTicker != nil {
		strat.spreadWalkTimer.Reset(strat.config.SpreadWalkDelay)
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
