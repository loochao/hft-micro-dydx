package main

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"time"
)

func watchXYSpread(
	ctx context.Context,
	xSymbol, ySymbol string,
	makerImpact, takerImpact float64,
	xDecay float64,
	xBias time.Duration,
	yDecay float64,
	yBias time.Duration,
	minTimeDelta, maxTimeDelta time.Duration,
	maxAgeDiffBias time.Duration,
	reportCount int,
	spreadLookback time.Duration,
	depthDirLookback time.Duration,
	makerDepthCh, takerDepthCh chan common.Depth,
	reportCh chan SpreadReport,
	outputCh chan *XYSpread,
) {
	var err error
	var xDepth, newXDepth common.Depth
	var yDepth, newYDepth common.Depth
	var xWalkedDepth, yWalkedDepth *common.WalkedMakerTakerDepth
	var spreadTime time.Time
	var adjustedAgeDiff time.Duration
	var xBiasInMs = float64(xBias / time.Millisecond)
	var yBiasInMs = float64(yBias / time.Millisecond)
	var minTimeDeltaInMs = float64(minTimeDelta / time.Millisecond)
	var maxTimeDeltaInMs = float64(maxTimeDelta / time.Millisecond)
	var xDepthFilter = common.NewDepthFilter(xDecay, xBiasInMs, minTimeDeltaInMs, maxTimeDeltaInMs)
	var yDepthFilter = common.NewDepthFilter(yDecay, yBiasInMs, minTimeDeltaInMs, maxTimeDeltaInMs)

	logSilentTime := time.Now()
	walkSpreadTimer := time.NewTimer(time.Hour * 999)
	xWalkDepthTimer := time.NewTimer(time.Hour * 999)
	yWalkDepthTimer := time.NewTimer(time.Hour * 999)

	shortEnterTimedMedian := common.NewTimedMedian(spreadLookback)
	longEnterTimedMedian := common.NewTimedMedian(spreadLookback)

	yDepthDirMedian := common.NewTimedMean(depthDirLookback)
	xDepthDirMedian := common.NewTimedMean(depthDirLookback)

	expectedChanSendingTime := time.Nanosecond * 300
	matchCount := 0
	depthCount := 0
	xExpireCount := 0
	yExpireCount := 0
	var shortLastEnter, longLastEnter float64
	var lastYBidAsk, currentYBidAsk float64
	var lastXBidAsk, currentXBidAsk float64
	for {
		select {
		case <-ctx.Done():
			return
		case <-walkSpreadTimer.C:
			if xWalkedDepth == nil || yWalkedDepth == nil {
				break
			}
			//需要用ema time delta 对age diff进行修正
			adjustedAgeDiff = xWalkedDepth.Time.Sub(yWalkedDepth.Time) + time.Duration(xDepthFilter.TimeDeltaEma-yDepthFilter.TimeDeltaEma)*time.Millisecond
			//取新一点的时间为spread time
			if xWalkedDepth.Time.Sub(yWalkedDepth.Time) < 0 {
				//需要对时间进行补偿
				spreadTime = yWalkedDepth.Time.Add(time.Millisecond * time.Duration(yDepthFilter.TimeDeltaEma))
			} else {
				//需要对时间进行补偿
				spreadTime = xWalkedDepth.Time.Add(time.Millisecond * time.Duration(xDepthFilter.TimeDeltaEma))
			}
			if adjustedAgeDiff > maxAgeDiffBias {
				yExpireCount++
				break
			} else if adjustedAgeDiff < -maxAgeDiffBias {
				xExpireCount++
				break
			}
			matchCount++
			shortLastEnter = (yWalkedDepth.TakerBid - xWalkedDepth.TakerAsk) / xWalkedDepth.TakerAsk
			longLastEnter = (yWalkedDepth.TakerAsk - xWalkedDepth.TakerBid) / xWalkedDepth.TakerBid

			shortEnterTimedMedian.Insert(spreadTime, shortLastEnter)
			longEnterTimedMedian.Insert(spreadTime, longLastEnter)

			if shortEnterTimedMedian.Len() < 2 {
				break
			}
			if shortEnterTimedMedian.Range() < spreadLookback/2 {
				break
			}

			select {
			case <-ctx.Done():
			case outputCh <- &XYSpread{
				YSymbol: ySymbol,
				XSymbol: xSymbol,
				YDepth:  *yWalkedDepth,
				XDepth:  *xWalkedDepth,

				ShortLastEnter:   shortLastEnter,
				ShortLastLeave:   longLastEnter,
				ShortMedianEnter: shortEnterTimedMedian.Median(),
				ShortMedianLeave: longEnterTimedMedian.Median(),

				LongLastEnter:   longLastEnter,
				LongLastLeave:   shortLastEnter,
				LongMedianEnter: longEnterTimedMedian.Median(),
				LongMedianLeave: shortEnterTimedMedian.Median(),

				XDir: xDepthDirMedian.Mean(),
				YDir: yDepthDirMedian.Mean(),

				AgeDiff: adjustedAgeDiff,
				Time:    spreadTime,
			}:
			default:
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("outputCh <- &common.MakerTakerSpread %s-%s len(outputCh) %d", xSymbol, ySymbol, len(outputCh))
					logSilentTime = time.Now().Add(time.Minute)
				}
			}
			break
		case <-xWalkDepthTimer.C:
			if xDepth != nil {
				xWalkedDepth, err = common.WalkMakerTakerDepth20(xDepth, makerImpact, takerImpact)
				if err != nil {
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("maker common.WalkMakerTakerDepth20 error %v %s", err, xSymbol)
						logSilentTime = time.Now().Add(time.Minute)
					}
					break
				}
				walkSpreadTimer.Reset(time.Nanosecond)
			}
			break
		case <-yWalkDepthTimer.C:
			if yDepth != nil {
				yWalkedDepth, err = common.WalkMakerTakerDepth20(yDepth, makerImpact, takerImpact)
				if err != nil {
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("taker common.WalkMakerTakerDepth5 error %v %s", err, ySymbol)
						logSilentTime = time.Now().Add(time.Minute)
					}
					break
				}
				walkSpreadTimer.Reset(time.Nanosecond)
			}
			break

		case newXDepth = <-makerDepthCh:
			if xDepth != nil && xDepth.GetTime().Sub(newXDepth.GetTime()) >= 0 {
				break
			}
			currentXBidAsk = newXDepth.GetBids()[0][0] + newXDepth.GetAsks()[0][0]
			if lastXBidAsk != 0 {
				xDepthDirMedian.Insert(newXDepth.GetTime(), (currentXBidAsk-lastXBidAsk)/lastXBidAsk)
			}
			lastXBidAsk = currentXBidAsk
			xDepth = newXDepth
			if !xDepthFilter.Filter(xDepth) && yDepth != nil {
				adjustedAgeDiff = xDepth.GetTime().Sub(yDepth.GetTime()) + time.Duration(math.Abs(xDepthFilter.TimeDeltaEma-yDepthFilter.TimeDeltaEma))*time.Millisecond
				if adjustedAgeDiff > maxAgeDiffBias {
					//taker已经过期
					yExpireCount++
				}
				if adjustedAgeDiff < -maxAgeDiffBias {
					//maker已经过期
					xExpireCount++
				} else {
					xWalkDepthTimer.Reset(expectedChanSendingTime)
				}
			}
			depthCount++
			if depthCount > reportCount {
				xDepthFilter.GenerateReport()
				yDepthFilter.GenerateReport()
				select {
				case reportCh <- SpreadReport{
					AdjustedAgeDiff:   adjustedAgeDiff,
					MatchRatio:        float64(matchCount) / float64(depthCount),
					XSymbol:           xSymbol,
					YSymbol:           ySymbol,
					XTimeDeltaEma:     xDepthFilter.TimeDeltaEma,
					YTimeDeltaEma:     yDepthFilter.TimeDeltaEma,
					XTimeDelta:        xDepthFilter.TimeDelta,
					YTimeDelta:        yDepthFilter.TimeDelta,
					XDepthFilterRatio: xDepthFilter.Report.FilterRatio,
					YDepthFilterRatio: yDepthFilter.Report.FilterRatio,
					XExpireRatio:      float64(xExpireCount) / float64(depthCount),
					YExpireRatio:      float64(yExpireCount) / float64(depthCount),
				}:
				default:
				}
				matchCount = 0
				depthCount = 0
				yExpireCount = 0
				xExpireCount = 0
			}
			break
		case newYDepth = <-takerDepthCh:
			if yDepth != nil &&
				yDepth.GetTime().Sub(newYDepth.GetTime()) >= 0 {
				break
			}
			currentYBidAsk = newYDepth.GetBids()[0][0] + newYDepth.GetAsks()[0][0]
			if lastYBidAsk != 0 {
				yDepthDirMedian.Insert(newYDepth.GetTime(), (currentYBidAsk-lastYBidAsk)/lastYBidAsk)
			}
			lastYBidAsk = currentYBidAsk
			yDepth = newYDepth
			if !yDepthFilter.Filter(yDepth) && xDepth != nil {
				adjustedAgeDiff = xDepth.GetTime().Sub(yDepth.GetTime()) + time.Duration(math.Abs(xDepthFilter.TimeDeltaEma-yDepthFilter.TimeDeltaEma))*time.Millisecond
				if adjustedAgeDiff > maxAgeDiffBias {
					//taker已经过期
					yExpireCount++
				} else {
					yWalkDepthTimer.Reset(expectedChanSendingTime)
				}
				if adjustedAgeDiff < -maxAgeDiffBias {
					//maker已经过期
					xExpireCount++
				}
			}
			depthCount++
			if depthCount > reportCount {
				xDepthFilter.GenerateReport()
				yDepthFilter.GenerateReport()
				select {
				case reportCh <- SpreadReport{
					AdjustedAgeDiff:       adjustedAgeDiff,
					MatchRatio:            float64(matchCount) / float64(depthCount),
					XSymbol:           xSymbol,
					YSymbol:           ySymbol,
					XTimeDeltaEma:     xDepthFilter.TimeDeltaEma,
					YTimeDeltaEma:     yDepthFilter.TimeDeltaEma,
					XTimeDelta:        xDepthFilter.TimeDelta,
					YTimeDelta:        yDepthFilter.TimeDelta,
					XDepthFilterRatio: xDepthFilter.Report.FilterRatio,
					YDepthFilterRatio: yDepthFilter.Report.FilterRatio,
					XExpireRatio:      float64(xExpireCount) / float64(depthCount),
					YExpireRatio:      float64(yExpireCount) / float64(depthCount),
				}:
				default:
				}
				matchCount = 0
				depthCount = 0
				yExpireCount = 0
				xExpireCount = 0
			}
			break
		}
	}
}
