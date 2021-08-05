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

	if strat.adjustedAgeDiff > strat.config.DepthMaxAgeDiffBias {
		//taker已经过期
		strat.yDepthExpireCount++
		//logger.Debugf("%s x expire y %v %v %v", xSymbol, xDepthTime.Sub(yDepthTime), adjustedAgeDiff, -time.Duration(xDepthFilter.TimeDeltaEma-yDepthFilter.TimeDeltaEma)*time.Millisecond)
		return
	} else if strat.adjustedAgeDiff < -strat.config.DepthMaxAgeDiffBias {
		//maker已经过期
		strat.xDepthExpireCount++
		//logger.Debugf("%s y expire x %v %v %v", xSymbol, xDepthTime.Sub(yDepthTime), adjustedAgeDiff, -time.Duration(xDepthFilter.TimeDeltaEma-yDepthFilter.TimeDeltaEma)*time.Millisecond)
		return
	}

	//取旧一点的时间为spread time
	if strat.xWalkedDepth.Time.Sub(strat.yWalkedDepth.Time) < 0 {
		//需要对时间进行补偿
		strat.spreadTime = strat.xWalkedDepth.Time.Add(time.Millisecond * time.Duration(strat.xDepthFilter.TimeDeltaEma))
	} else {
		strat.spreadTime = strat.yWalkedDepth.Time.Add(time.Millisecond * time.Duration(strat.yDepthFilter.TimeDeltaEma))
		//需要对时间进行补偿
	}

	//假定挂单基于MiroPrice, 考虑挂单的下界偏移进Spread
	//如果想挂得远，成交少，吃大Spread, 可以orderOffsets参数，推NearBot NearTop, 反之亦然
	//完全去掉AskPrice - BidPrice
	strat.shortLastEnter = (strat.yWalkedDepth.BidPrice - strat.xWalkedDepth.MidPrice) / strat.xWalkedDepth.MidPrice
	strat.longLastEnter = (strat.yWalkedDepth.AskPrice - strat.xWalkedDepth.MidPrice) / strat.xWalkedDepth.MidPrice
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

	if strat.spreadTime.Sub(strat.quantileLastSampleTime) > strat.config.QuantileSampleInterval {
		strat.quantileLastSampleTime = strat.spreadTime
		_ = strat.timedTDigest.Insert(strat.spreadTime, (strat.shortLastEnter+strat.longLastEnter)*0.5)
		if strat.quantileMiddle == nil {
			strat.quantileMiddle = new(float64)
		}
		*strat.quantileMiddle = strat.timedTDigest.Quantile(0.5)
	}
}

func (strat *XYStrategy) walkXDepth() {

	//x做为挂单边不用walk
	strat.xWalkedDepth.Symbol = strat.xDepth.GetSymbol()
	strat.xWalkedDepth.Time = strat.xDepth.GetTime()
	strat.spreadWalkTimer.Reset(strat.config.SpreadWalkDelay)
	strat.xWalkedDepth.BestBidPrice = strat.xDepth.GetBids()[0][0]
	strat.xWalkedDepth.BestAskPrice = strat.xDepth.GetAsks()[0][0]
	strat.xWalkedDepth.BidPrice = strat.xDepth.GetBids()[0][0]
	strat.xWalkedDepth.AskPrice = strat.xDepth.GetAsks()[0][0]
	strat.xWalkedDepth.MidPrice = (strat.xDepth.GetBids()[0][0] + strat.xDepth.GetAsks()[0][0]) * 0.5
	strat.xWalkedDepth.MircoPrice =
		(strat.xDepth.GetBids()[0][0]*strat.xDepth.GetAsks()[0][1] + strat.xDepth.GetAsks()[0][0]*strat.xDepth.GetBids()[0][0]) / (strat.xDepth.GetBids()[0][1] + strat.xDepth.GetAsks()[0][1])
	strat.spreadWalkTimer.Reset(strat.config.SpreadWalkDelay)
}

func (strat *XYStrategy) walkYDepth() {
	strat.error = common.WalkDepthBBMAA(strat.yDepth, strat.yMultiplier, strat.yImpactValue, &strat.yWalkedDepth)
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
	if !strat.xDepthFilter.Filter(strat.xDepth) {
		strat.xWalkDepthTimer.Reset(strat.config.DepthWalkDelay)
	}
	strat.depthCount++
	if strat.depthCount > strat.config.DepthReportCount {
		strat.xDepthFilter.GenerateReport()
		strat.yDepthFilter.GenerateReport()
		strat.spreadReport = &common.XYSpreadReport{
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
	if !strat.yDepthFilter.Filter(strat.yDepth) {
		strat.yWalkDepthTimer.Reset(strat.config.DepthWalkDelay)
	}

	strat.depthCount++
	if strat.depthCount > strat.config.DepthReportCount {
		strat.xDepthFilter.GenerateReport()
		strat.yDepthFilter.GenerateReport()
		strat.spreadReport = &common.XYSpreadReport{
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
		strat.depthCount = 0
		strat.yDepthExpireCount = 0
		strat.xDepthExpireCount = 0
	}
}
