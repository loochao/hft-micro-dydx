package main

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func watchXYSpread(
	ctx context.Context,
	xSymbol, ySymbol string,
	xMultiplier, yMultiplier, impact float64,
	xDecay float64,
	xBias time.Duration,
	yDecay float64,
	yBias time.Duration,
	minTimeDelta, maxTimeDelta time.Duration,
	maxAgeDiffBias time.Duration,
	reportCount int,
	spreadLookback time.Duration,
	xDepthCh, yDepthCh chan common.Depth,
	reportCh chan SpreadReport,
	outputCh chan *XYSpread,
) {

	var err error
	var xDepth, yDepth common.Depth
	var xWalkedDepth, yWalkedDepth = &common.WalkedDepthBAM{}, &common.WalkedDepthBAM{}
	var spreadTime time.Time
	var adjustedAgeDiff time.Duration
	var xBiasInMs = float64(xBias / time.Millisecond)
	var yBiasInMs = float64(yBias / time.Millisecond)
	var minTimeDeltaInMs = float64(minTimeDelta / time.Millisecond)
	var maxTimeDeltaInMs = float64(maxTimeDelta / time.Millisecond)
	var xDepthFilter = common.NewDepthFilter(xDecay, xBiasInMs, minTimeDeltaInMs, maxTimeDeltaInMs)
	var yDepthFilter = common.NewDepthFilter(yDecay, yBiasInMs, minTimeDeltaInMs, maxTimeDeltaInMs)
	var xDepthTime = time.Unix(0, 0)
	var yDepthTime = time.Unix(0, 0)

	logSilentTime := time.Now()
	walkSpreadTimer := time.NewTimer(time.Hour * 999)
	xWalkDepthTimer := time.NewTimer(time.Hour * 999)
	yWalkDepthTimer := time.NewTimer(time.Hour * 999)

	shortEnterTimedMedian := common.NewTimedMedian(spreadLookback)
	longEnterTimedMedian := common.NewTimedMedian(spreadLookback)

	expectedChanSendingTime := time.Nanosecond * 300
	matchCount := 0
	depthCount := 0
	xExpireCount := 0
	yExpireCount := 0
	var shortLastEnter, longLastEnter float64
	for {
		select {
		case <-ctx.Done():
			return
		case <-walkSpreadTimer.C:
			if xWalkedDepth.Symbol == "" || yWalkedDepth.Symbol == "" {
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
				//logger.Debugf("%s x expire y %v %v %v", xSymbol, xDepthTime.Sub(yDepthTime), adjustedAgeDiff, -time.Duration(xDepthFilter.TimeDeltaEma-yDepthFilter.TimeDeltaEma)*time.Millisecond)
				break
			} else if adjustedAgeDiff < -maxAgeDiffBias {
				//logger.Debugf("%s y expire x %v %v %v", xSymbol, xDepthTime.Sub(yDepthTime), adjustedAgeDiff, -time.Duration(xDepthFilter.TimeDeltaEma-yDepthFilter.TimeDeltaEma)*time.Millisecond)
				xExpireCount++
				break
			}
			matchCount++
			shortLastEnter = (yWalkedDepth.BidPrice - xWalkedDepth.AskPrice) / xWalkedDepth.AskPrice
			longLastEnter = (yWalkedDepth.AskPrice - xWalkedDepth.BidPrice) / xWalkedDepth.BidPrice

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
				Time:            spreadTime,
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
				err = common.WalkCoinDepthWithMultiplier(xDepth, xMultiplier, impact, xWalkedDepth)
				if err != nil {
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("maker common.WalkMakerTakerDepth error %v %s", err, xSymbol)
						logSilentTime = time.Now().Add(time.Minute)
					}
					break
				}
				walkSpreadTimer.Reset(time.Nanosecond)
			}
			break
		case <-yWalkDepthTimer.C:
			if yDepth != nil {
				err = common.WalkCoinDepthWithMultiplier(yDepth, yMultiplier, impact, yWalkedDepth)
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

		case xDepth = <-xDepthCh:
			if xDepth.GetTime().Sub(xDepthTime) < 0 {
				break
			}
			xDepthTime = xDepth.GetTime()
			if !xDepthFilter.Filter(xDepth) && yDepth != nil {
				adjustedAgeDiff = xDepthTime.Sub(yDepthTime) + time.Duration(xDepthFilter.TimeDeltaEma-yDepthFilter.TimeDeltaEma)*time.Millisecond
				if adjustedAgeDiff > maxAgeDiffBias {
					//taker已经过期
					yExpireCount++
					//logger.Debugf("%s x expire y %v %v %v", xSymbol, xDepthTime.Sub(yDepthTime), adjustedAgeDiff, -time.Duration(xDepthFilter.TimeDeltaEma-yDepthFilter.TimeDeltaEma)*time.Millisecond)
				} else if adjustedAgeDiff < -maxAgeDiffBias {
					//maker已经过期
					xExpireCount++
					//logger.Debugf("%s y expire x %v %v %v", xSymbol, xDepthTime.Sub(yDepthTime), adjustedAgeDiff, -time.Duration(xDepthFilter.TimeDeltaEma-yDepthFilter.TimeDeltaEma)*time.Millisecond)
				} else {
					xWalkDepthTimer.Reset(expectedChanSendingTime)
				}
			}
			depthCount++
			if depthCount > reportCount {
				xDepthFilter.GenerateReport()
				yDepthFilter.GenerateReport()
				report := SpreadReport{
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
				}
				if xDepth != nil && yDepth != nil {
					report.AgeDiff = xDepthTime.Sub(yDepthTime)
					report.AgeDiff = xDepthTime.Sub(yDepthTime) + time.Duration(xDepthFilter.TimeDeltaEma-yDepthFilter.TimeDeltaEma)*time.Millisecond
				}
				select {
				case reportCh <- report:
				default:
				}
				matchCount = 0
				depthCount = 0
				yExpireCount = 0
				xExpireCount = 0
			}
			break
		case yDepth = <-yDepthCh:
			if yDepth.GetTime().Sub(yDepthTime) < 0 {
				break
			}
			yDepthTime = yDepth.GetTime()
			if !yDepthFilter.Filter(yDepth) && xDepth != nil {
				adjustedAgeDiff = xDepthTime.Sub(yDepthTime) + time.Duration(xDepthFilter.TimeDeltaEma-yDepthFilter.TimeDeltaEma)*time.Millisecond
				if adjustedAgeDiff < -maxAgeDiffBias {
					//maker已经过期
					//logger.Debugf("%s y expire x %v %v %v", xSymbol, xDepthTime.Sub(yDepthTime), adjustedAgeDiff, -time.Duration(xDepthFilter.TimeDeltaEma-yDepthFilter.TimeDeltaEma)*time.Millisecond)
					xExpireCount++
				} else if adjustedAgeDiff > maxAgeDiffBias {
					//logger.Debugf("%s x expire y %v %v %v", xSymbol, xDepthTime.Sub(yDepthTime), adjustedAgeDiff, -time.Duration(xDepthFilter.TimeDeltaEma-yDepthFilter.TimeDeltaEma)*time.Millisecond)
					//taker已经过期
					yExpireCount++
				} else {
					yWalkDepthTimer.Reset(expectedChanSendingTime)
				}
			}
			depthCount++
			if depthCount > reportCount {
				xDepthFilter.GenerateReport()
				yDepthFilter.GenerateReport()
				report := SpreadReport{
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
				}
				if xDepth != nil && yDepth != nil {
					report.AgeDiff = xDepthTime.Sub(yDepthTime)
					report.AgeDiff = xDepthTime.Sub(yDepthTime) + time.Duration(xDepthFilter.TimeDeltaEma-yDepthFilter.TimeDeltaEma)*time.Millisecond
				}
				select {
				case reportCh <- report:
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
