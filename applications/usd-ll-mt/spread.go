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
	//取旧一点的时间为spread time
	if strat.xWalkedDepth.Time.Sub(strat.yWalkedDepth.Time) < 0 {
		//需要对时间进行补偿
		strat.spreadTime = strat.xWalkedDepth.Time.Add(time.Millisecond * time.Duration(strat.xDepthFilter.TimeDeltaEma))
	} else {
		//需要对时间进行补偿
		strat.spreadTime = strat.yWalkedDepth.Time.Add(time.Millisecond * time.Duration(strat.yDepthFilter.TimeDeltaEma))
	}
	if strat.adjustedAgeDiff > strat.config.DepthMaxAgeDiffBias {
		strat.yDepthExpireCount++
		strat.xyDepthMatchSum.Insert(0.0)
		strat.xyDepthMatchRatio = strat.xyDepthMatchSum.Sum() / strat.xyDepthMatchWindow
		//logger.Debugf("%s x expire y %v %v %v", xSymbol, xDepthTime.Sub(yDepthTime), adjustedAgeDiff, -time.Duration(xDepthFilter.TimeDeltaEma-yDepthFilter.TimeDeltaEma)*time.Millisecond)
		return
	} else if strat.adjustedAgeDiff < -strat.config.DepthMaxAgeDiffBias {
		strat.xDepthExpireCount++
		strat.xyDepthMatchSum.Insert(0.0)
		strat.xyDepthMatchRatio = strat.xyDepthMatchSum.Sum() / strat.xyDepthMatchWindow
		//logger.Debugf("%s y expire x %v %v %v", xSymbol, xDepthTime.Sub(yDepthTime), adjustedAgeDiff, -time.Duration(xDepthFilter.TimeDeltaEma-yDepthFilter.TimeDeltaEma)*time.Millisecond)
		return
	}
	strat.xyDepthMatchSum.Insert(1.0)
	strat.xyDepthMatchRatio = strat.xyDepthMatchSum.Sum() / strat.xyDepthMatchWindow
	strat.depthMatchCount++

	//假定挂单基于MiroPrice, 考虑挂单的下界偏移进Spread
	//如果想挂得远，成交少，吃大Spread, 可以orderOffsets参数，推NearBot NearTop, 反之亦然
	strat.shortLastEnter = (strat.yWalkedDepth.BidPrice-strat.xWalkedDepth.MircoPrice)/strat.xWalkedDepth.MircoPrice + strat.orderOffset.NearTop
	strat.longLastEnter = (strat.yWalkedDepth.AskPrice-strat.xWalkedDepth.MircoPrice)/strat.xWalkedDepth.MircoPrice + strat.orderOffset.NearBot

	strat.shortEnterTimedMedian.Insert(strat.spreadTime, strat.shortLastEnter)
	strat.longEnterTimedMedian.Insert(strat.spreadTime, strat.longLastEnter)

	if strat.shortEnterTimedMedian.Len() < strat.config.SpreadMinDepthCount {
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
}

func (strat *XYStrategy) walkXDepth() {
	strat.error = common.WalkDepthBBMAA(strat.xDepth, strat.xMultiplier, strat.config.DepthTakerImpact, &strat.xWalkedDepth)
	if strat.error != nil {
		if time.Now().Sub(strat.logSilentTime) > 0 {
			logger.Debugf("x common.WalkDepthBMA error %v %s", strat.error, strat.xSymbol)
			strat.logSilentTime = time.Now().Add(time.Minute)
		}
	} else {
		strat.spreadWalkTimer.Reset(strat.config.SpreadWalkDelay)
	}
}

func (strat *XYStrategy) walkYDepth() {
	strat.error = common.WalkDepthBBMAA(strat.yDepth, strat.yMultiplier, strat.config.DepthTakerImpact, &strat.yWalkedDepth)
	if strat.error != nil {
		if time.Now().Sub(strat.logSilentTime) > 0 {
			logger.Debugf("y common.WalkDepthBMA error %v %s", strat.error, strat.ySymbol)
			strat.logSilentTime = time.Now().Add(time.Minute)
		}
	} else {
		strat.spreadWalkTimer.Reset(strat.config.SpreadWalkDelay)
	}
}
func (strat *XYStrategy) handleDepth() {
	switch strat.nextDepth.GetExchange() {
	case strat.xExchangeID:
		strat.xNextDepth = strat.nextDepth
		strat.handleXDepth()
		break
	case strat.yExchangeID:
		strat.yNextDepth = strat.nextDepth
		strat.handleYDepth()
		break
	default:
		if time.Now().Sub(strat.logSilentTime) > 0 {
			logger.Debugf("unknown exchanged id %d", strat.nextDepth.GetExchange())
			strat.logSilentTime = time.Now().Add(time.Minute)
		}
	}
}

func (strat *XYStrategy) handleXDepth() {
	switch strat.nextDepth.GetExchange() {
	case strat.xExchangeID:
	case strat.yExchangeID:
	default:
		if time.Now().Sub(strat.logSilentTime) > 0 {
			logger.Debugf("unknown exchanged id %d", strat.nextDepth.GetExchange())
			strat.logSilentTime = time.Now().Add(time.Minute)
		}
	}
	if strat.xDepth == strat.xNextDepth {
		return
	}
	if strat.xNextDepth.GetTime().Sub(strat.xDepthTime) < 0 {
		return
	}
	strat.xDepth = strat.xNextDepth
	strat.xDepthTime = strat.xDepth.GetTime()
	if !strat.xDepthFilter.Filter(strat.xDepth) && strat.yDepth != nil {
		strat.adjustedAgeDiff = strat.xDepthTime.Sub(strat.yDepthTime) + time.Duration(strat.xDepthFilter.TimeDeltaEma-strat.yDepthFilter.TimeDeltaEma)*time.Millisecond
		if strat.adjustedAgeDiff > strat.config.DepthMaxAgeDiffBias {
			//taker已经过期
			strat.yDepthExpireCount++
			strat.xyDepthMatchSum.Insert(0.0)
			//logger.Debugf("%s x expire y %v %v %v", xSymbol, xDepthTime.Sub(yDepthTime), adjustedAgeDiff, -time.Duration(xDepthFilter.TimeDeltaEma-yDepthFilter.TimeDeltaEma)*time.Millisecond)
		} else if strat.adjustedAgeDiff < -strat.config.DepthMaxAgeDiffBias {
			//maker已经过期
			strat.xDepthExpireCount++
			strat.xyDepthMatchSum.Insert(0.0)
			//logger.Debugf("%s y expire x %v %v %v", xSymbol, xDepthTime.Sub(yDepthTime), adjustedAgeDiff, -time.Duration(xDepthFilter.TimeDeltaEma-yDepthFilter.TimeDeltaEma)*time.Millisecond)
		} else {
			strat.xWalkDepthTimer.Reset(strat.config.DepthWalkDelay)
		}
	} else {
		strat.xyDepthMatchSum.Insert(0.0)
	}
	strat.xyDepthMatchRatio = strat.xyDepthMatchSum.Sum() / strat.xyDepthMatchWindow
	strat.depthCount++
	if strat.depthCount > strat.config.DepthReportCount {
		strat.xDepthFilter.GenerateReport()
		strat.yDepthFilter.GenerateReport()
		strat.spreadReport = &common.XYSpreadReport{
			MatchRatio:         float64(strat.depthMatchCount) / float64(strat.depthCount),
			XSymbol:            strat.xSymbol,
			YSymbol:            strat.ySymbol,
			XTimeDeltaEma:      strat.xDepthFilter.TimeDeltaEma,
			YTimeDeltaEma:      strat.yDepthFilter.TimeDeltaEma,
			XTimeDelta:         strat.xDepthFilter.TimeDelta,
			YTimeDelta:         strat.yDepthFilter.TimeDelta,
			XTickerFilterRatio: strat.xDepthFilter.Report.FilterRatio,
			YTickerFilterRatio: strat.yDepthFilter.Report.FilterRatio,
			XExpireRatio:       float64(strat.xDepthExpireCount) / float64(strat.depthCount),
			YExpireRatio:       float64(strat.yDepthExpireCount) / float64(strat.depthCount),
		}
		strat.depthMatchCount = 0
		strat.depthCount = 0
		strat.yDepthExpireCount = 0
		strat.xDepthExpireCount = 0
	}
}

func (strat *XYStrategy) handleYDepth() {
	if strat.yDepth == strat.yNextDepth {
		return
	}
	if strat.yNextDepth.GetTime().Sub(strat.yDepthTime) < 0 {
		return
	}
	strat.yDepth = strat.yNextDepth
	strat.yDepthTime = strat.yDepth.GetTime()
	if !strat.yDepthFilter.Filter(strat.yDepth) && strat.xDepth != nil {
		strat.adjustedAgeDiff = strat.xDepthTime.Sub(strat.yDepthTime) + time.Duration(strat.xDepthFilter.TimeDeltaEma-strat.yDepthFilter.TimeDeltaEma)*time.Millisecond
		if strat.adjustedAgeDiff < -strat.config.DepthMaxAgeDiffBias {
			//maker已经过期
			//logger.Debugf("%s y expire x %v %v %v", xSymbol, xDepthTime.Sub(yDepthTime), adjustedAgeDiff, -time.Duration(xDepthFilter.TimeDeltaEma-yDepthFilter.TimeDeltaEma)*time.Millisecond)
			strat.xyDepthMatchSum.Insert(0.0)
			strat.xDepthExpireCount++
		} else if strat.adjustedAgeDiff > strat.config.DepthMaxAgeDiffBias {
			//logger.Debugf("%s x expire y %v %v %v", xSymbol, xDepthTime.Sub(yDepthTime), adjustedAgeDiff, -time.Duration(xDepthFilter.TimeDeltaEma-yDepthFilter.TimeDeltaEma)*time.Millisecond)
			//taker已经过期
			strat.xyDepthMatchSum.Insert(0.0)
			strat.yDepthExpireCount++
		} else {
			strat.yWalkDepthTimer.Reset(strat.config.DepthWalkDelay)
		}
	} else {
		strat.xyDepthMatchSum.Insert(0.0)
	}
	strat.xyDepthMatchRatio = strat.xyDepthMatchSum.Sum() / strat.xyDepthMatchWindow
	strat.depthCount++
	if strat.depthCount > strat.config.DepthReportCount {
		strat.xDepthFilter.GenerateReport()
		strat.yDepthFilter.GenerateReport()
		strat.spreadReport = &common.XYSpreadReport{
			MatchRatio:         float64(strat.depthMatchCount) / float64(strat.depthCount),
			XSymbol:            strat.xSymbol,
			YSymbol:            strat.ySymbol,
			XTimeDeltaEma:      strat.xDepthFilter.TimeDeltaEma,
			YTimeDeltaEma:      strat.yDepthFilter.TimeDeltaEma,
			XTimeDelta:         strat.xDepthFilter.TimeDelta,
			YTimeDelta:         strat.yDepthFilter.TimeDelta,
			XTickerFilterRatio: strat.xDepthFilter.Report.FilterRatio,
			YTickerFilterRatio: strat.yDepthFilter.Report.FilterRatio,
			XExpireRatio:       float64(strat.xDepthExpireCount) / float64(strat.depthCount),
			YExpireRatio:       float64(strat.yDepthExpireCount) / float64(strat.depthCount),
		}
		strat.depthMatchCount = 0
		strat.depthCount = 0
		strat.yDepthExpireCount = 0
		strat.xDepthExpireCount = 0
	}
}
