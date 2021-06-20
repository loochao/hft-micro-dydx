package main

import (
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func (strat *XYStrategy) walkSpread() {
	if strat.xWalkedDepth.Symbol == "" || strat.yWalkedDepth.Symbol == "" {
		return
	}
	//需要用ema time delta 对age diff进行修正
	strat.adjustedAgeDiff = strat.xWalkedDepth.Time.Sub(strat.yWalkedDepth.Time) + time.Duration(strat.xDepthFilter.TimeDeltaEma-strat.yDepthFilter.TimeDeltaEma)*time.Millisecond
	//取新一点的时间为spread time
	if strat.xWalkedDepth.Time.Sub(strat.yWalkedDepth.Time) < 0 {
		//需要对时间进行补偿
		strat.spreadTime = strat.yWalkedDepth.Time.Add(time.Millisecond * time.Duration(strat.yDepthFilter.TimeDeltaEma))
	} else {
		//需要对时间进行补偿
		strat.spreadTime = strat.xWalkedDepth.Time.Add(time.Millisecond * time.Duration(strat.xDepthFilter.TimeDeltaEma))
	}
	if strat.adjustedAgeDiff > strat.params.depthMaxAgeDiffBias {
		strat.yDepthExpireCount++
		//logger.Debugf("%s x expire y %v %v %v", xSymbol, xDepthTime.Sub(yDepthTime), adjustedAgeDiff, -time.Duration(xDepthFilter.TimeDeltaEma-yDepthFilter.TimeDeltaEma)*time.Millisecond)
	} else if strat.adjustedAgeDiff < -strat.params.depthMaxAgeDiffBias {
		//logger.Debugf("%s y expire x %v %v %v", xSymbol, xDepthTime.Sub(yDepthTime), adjustedAgeDiff, -time.Duration(xDepthFilter.TimeDeltaEma-yDepthFilter.TimeDeltaEma)*time.Millisecond)
		strat.xDepthExpireCount++
	}
	strat.depthMatchCount++
	strat.shortLastEnter = (strat.yWalkedDepth.BidPrice - strat.xWalkedDepth.AskPrice) / strat.yWalkedDepth.AskPrice
	strat.longLastEnter = (strat.yWalkedDepth.AskPrice - strat.xWalkedDepth.BidPrice) / strat.yWalkedDepth.BidPrice

	strat.shortEnterTimedMedian.Insert(strat.spreadTime, strat.shortLastEnter)
	strat.longEnterTimedMedian.Insert(strat.spreadTime, strat.longLastEnter)

	if strat.shortEnterTimedMedian.Len() < strat.params.spreadMinDepthCount {
		return
	}
	if strat.shortEnterTimedMedian.Range() < strat.params.spreadLookback/2 {
		return
	}
	strat.spread = &XYSpread{
		ShortLastEnter:   strat.shortLastEnter,
		ShortLastLeave:   strat.longLastEnter,
		ShortMedianEnter: strat.shortEnterTimedMedian.Median(),
		ShortMedianLeave: strat.longEnterTimedMedian.Median(),

		LongLastEnter:   strat.longLastEnter,
		LongLastLeave:   strat.shortLastEnter,
		LongMedianEnter: strat.longEnterTimedMedian.Median(),
		LongMedianLeave: strat.shortEnterTimedMedian.Median(),
		Time:            strat.spreadTime,
	}
	strat.changeXPosition()
}

func (strat *XYStrategy) walkXDepth() {
	strat.error = common.WalkDepthWithMultiplier(strat.xDepth, strat.params.xMultiplier, strat.params.depthTakerImpact, &strat.xWalkedDepth)
	if strat.error != nil {
		if time.Now().Sub(strat.logSilentTime) > 0 {
			logger.Debugf("x common.WalkDepthWithMultiplier error %v %s", strat.error, strat.xSymbol)
			strat.logSilentTime = time.Now().Add(time.Minute)
		}
	} else {
		strat.spreadWalkTimer.Reset(strat.params.spreadWalkDelay)
	}
}

func (strat *XYStrategy) walkYDepth() {
	strat.error = common.WalkDepthWithMultiplier(strat.yDepth, strat.params.yMultiplier, strat.params.depthTakerImpact, &strat.yWalkedDepth)
	if strat.error != nil {
		if time.Now().Sub(strat.logSilentTime) > 0 {
			logger.Debugf("y common.WalkDepthWithMultiplier error %v %s", strat.error, strat.ySymbol)
			strat.logSilentTime = time.Now().Add(time.Minute)
		}
	} else {
		if strat.markedYAskPrice != nil {
			if strat.yWalkedDepth.AskPrice < *strat.markedYAskPrice+strat.params.yTickSize {
				*strat.markedYAskPrice = strat.yWalkedDepth.AskPrice
			} else {
				logger.Debugf("Y %s markedYAskPrice %f >= trailed askPrice %f + %f, change y position", strat.ySymbol, *strat.markedYAskPrice, strat.yWalkedDepth.AskPrice, strat.params.yTickSize)
				strat.markedYAskPrice = nil
				strat.changeYPosition()
			}
		} else if strat.markedYBidPrice != nil {
			if strat.yWalkedDepth.BidPrice > *strat.markedYBidPrice-strat.params.yTickSize {
				*strat.markedYBidPrice = strat.yWalkedDepth.BidPrice
			} else {
				logger.Debugf("Y %s markedYBidPrice %f <= trailed bidPrice %f - %f, change y position", strat.ySymbol, *strat.markedYBidPrice, strat.yWalkedDepth.BidPrice, strat.params.yTickSize)
				strat.markedYBidPrice = nil
				strat.changeYPosition()
			}
		}
		strat.spreadWalkTimer.Reset(strat.params.spreadWalkDelay)
	}
}

func (strat *XYStrategy) handleXDepth() {
	if strat.xDepth.GetTime().Sub(strat.xDepthTime) < 0 {
		return
	}
	strat.xDepthTime = strat.xDepth.GetTime()
	if !strat.xDepthFilter.Filter(strat.xDepth) && strat.yDepth != nil {
		strat.adjustedAgeDiff = strat.xDepthTime.Sub(strat.yDepthTime) + time.Duration(strat.xDepthFilter.TimeDeltaEma-strat.yDepthFilter.TimeDeltaEma)*time.Millisecond
		if strat.adjustedAgeDiff > strat.params.depthMaxAgeDiffBias {
			//taker已经过期
			strat.yDepthExpireCount++
			//logger.Debugf("%s x expire y %v %v %v", xSymbol, xDepthTime.Sub(yDepthTime), adjustedAgeDiff, -time.Duration(xDepthFilter.TimeDeltaEma-yDepthFilter.TimeDeltaEma)*time.Millisecond)
		} else if strat.adjustedAgeDiff < -strat.params.depthMaxAgeDiffBias {
			//maker已经过期
			strat.xDepthExpireCount++
			//logger.Debugf("%s y expire x %v %v %v", xSymbol, xDepthTime.Sub(yDepthTime), adjustedAgeDiff, -time.Duration(xDepthFilter.TimeDeltaEma-yDepthFilter.TimeDeltaEma)*time.Millisecond)
		} else {
			strat.xWalkDepthTimer.Reset(strat.params.depthWalkDelay)
		}
	}
	strat.depthCount++
	if strat.depthCount > strat.params.depthReportCount {
		strat.xDepthFilter.GenerateReport()
		strat.yDepthFilter.GenerateReport()
		strat.spreadReport = &SpreadReport{
			MatchRatio:        float64(strat.depthMatchCount) / float64(strat.depthCount),
			XSymbol:           strat.xSymbol,
			YSymbol:           strat.ySymbol,
			XTimeDeltaEma:     strat.xDepthFilter.TimeDeltaEma,
			YTimeDeltaEma:     strat.yDepthFilter.TimeDeltaEma,
			XTimeDelta:        strat.xDepthFilter.TimeDelta,
			YTimeDelta:        strat.yDepthFilter.TimeDelta,
			XDepthFilterRatio: strat.xDepthFilter.Report.FilterRatio,
			YDepthFilterRatio: strat.yDepthFilter.Report.FilterRatio,
			XExpireRatio:      float64(strat.xDepthExpireCount) / float64(strat.depthCount),
			YExpireRatio:      float64(strat.yDepthExpireCount) / float64(strat.depthCount),
		}
		if strat.xDepth != nil && strat.yDepth != nil {
			strat.spreadReport.AgeDiff = strat.xDepthTime.Sub(strat.yDepthTime) + time.Duration(strat.xDepthFilter.TimeDeltaEma-strat.yDepthFilter.TimeDeltaEma)*time.Millisecond
		}
		strat.depthMatchCount = 0
		strat.depthCount = 0
		strat.yDepthExpireCount = 0
		strat.xDepthExpireCount = 0
	}
}

func (strat *XYStrategy) handleYDepth() {
	if strat.yDepth.GetTime().Sub(strat.yDepthTime) < 0 {
		return
	}
	strat.yDepthTime = strat.yDepth.GetTime()
	if !strat.yDepthFilter.Filter(strat.yDepth) && strat.xDepth != nil {
		strat.adjustedAgeDiff = strat.xDepthTime.Sub(strat.yDepthTime) + time.Duration(strat.xDepthFilter.TimeDeltaEma-strat.yDepthFilter.TimeDeltaEma)*time.Millisecond
		if strat.adjustedAgeDiff < -strat.params.depthMaxAgeDiffBias {
			//maker已经过期
			//logger.Debugf("%s y expire x %v %v %v", xSymbol, xDepthTime.Sub(yDepthTime), adjustedAgeDiff, -time.Duration(xDepthFilter.TimeDeltaEma-yDepthFilter.TimeDeltaEma)*time.Millisecond)
			strat.xDepthExpireCount++
		} else if strat.adjustedAgeDiff > strat.params.depthMaxAgeDiffBias {
			//logger.Debugf("%s x expire y %v %v %v", xSymbol, xDepthTime.Sub(yDepthTime), adjustedAgeDiff, -time.Duration(xDepthFilter.TimeDeltaEma-yDepthFilter.TimeDeltaEma)*time.Millisecond)
			//taker已经过期
			strat.yDepthExpireCount++
		} else {
			strat.yWalkDepthTimer.Reset(strat.params.depthWalkDelay)
		}
	}
	strat.depthCount++
	if strat.depthCount > strat.params.depthReportCount {
		strat.xDepthFilter.GenerateReport()
		strat.yDepthFilter.GenerateReport()
		strat.spreadReport = &SpreadReport{
			MatchRatio:        float64(strat.depthMatchCount) / float64(strat.depthCount),
			XSymbol:           strat.xSymbol,
			YSymbol:           strat.ySymbol,
			XTimeDeltaEma:     strat.xDepthFilter.TimeDeltaEma,
			YTimeDeltaEma:     strat.yDepthFilter.TimeDeltaEma,
			XTimeDelta:        strat.xDepthFilter.TimeDelta,
			YTimeDelta:        strat.yDepthFilter.TimeDelta,
			XDepthFilterRatio: strat.xDepthFilter.Report.FilterRatio,
			YDepthFilterRatio: strat.yDepthFilter.Report.FilterRatio,
			XExpireRatio:      float64(strat.xDepthExpireCount) / float64(strat.depthCount),
			YExpireRatio:      float64(strat.yDepthExpireCount) / float64(strat.depthCount),
		}
		if strat.xDepth != nil && strat.yDepth != nil {
			strat.spreadReport.AgeDiff = strat.xDepthTime.Sub(strat.yDepthTime) + time.Duration(strat.xDepthFilter.TimeDeltaEma-strat.yDepthFilter.TimeDeltaEma)*time.Millisecond
		}
		strat.depthMatchCount = 0
		strat.depthCount = 0
		strat.yDepthExpireCount = 0
		strat.xDepthExpireCount = 0
	}
}
