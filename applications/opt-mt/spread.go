package main

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"time"
)

func watchMakerTakerSpread(
	ctx context.Context,
	makerSymbol, takerSymbol string,
	makerImpact, takerImpact float64,
	makerDecay float64,
	makerBias time.Duration,
	takerDecay float64,
	takerBias time.Duration,
	minTimeDelta, maxTimeDelta time.Duration,
	maxAgeDiffBias time.Duration,
	reportCount int,
	lookbackDuration time.Duration,
	makerDepthCh, takerDepthCh chan common.Depth,
	reportCh chan common.SpreadReport,
	outputCh chan *common.MakerTakerSpread,
) {
	var err error
	var makerDepth, newMakerDepth common.Depth
	var takerDepth, newTakerDepth common.Depth
	var makerWalkedDepth, takerWalkedDepth *common.WalkedMakerTakerDepth
	var spreadTime time.Time
	var adjustedAgeDiff time.Duration
	var makerBiasInMs = float64(makerBias / time.Millisecond)
	var takerBiasInMs = float64(takerBias / time.Millisecond)
	var minTimeDeltaInMs = float64(minTimeDelta / time.Millisecond)
	var maxTimeDeltaInMs = float64(maxTimeDelta / time.Millisecond)
	//var maxAgeDiff = makerTakerBias
	var makerDepthFilter = common.NewDepthFilter(makerDecay, makerBiasInMs, minTimeDeltaInMs, maxTimeDeltaInMs)
	var takerDepthFilter = common.NewDepthFilter(takerDecay, takerBiasInMs, minTimeDeltaInMs, maxTimeDeltaInMs)

	logSilentTime := time.Now()
	walkSpreadTimer := time.NewTimer(time.Hour * 999)
	makerWalkDepthTimer := time.NewTimer(time.Hour * 999)
	takerWalkDepthTimer := time.NewTimer(time.Hour * 999)

	shortEnterTimedMedian := common.NewTimedMedian(lookbackDuration)
	longEnterTimedMedian := common.NewTimedMedian(lookbackDuration)

	expectedChanSendingTime := time.Nanosecond * 300
	matchCount := 0
	depthCount := 0
	makerExpireCount := 0
	takerExpireCount := 0
	var shortLastEnter, longLastEnter float64
	for {
		select {
		case <-ctx.Done():
			return
		case <-walkSpreadTimer.C:
			if makerWalkedDepth == nil || takerWalkedDepth == nil {
				break
			}
			//需要用ema time delta 对age diff进行修正
			adjustedAgeDiff = makerWalkedDepth.Time.Sub(takerWalkedDepth.Time) + time.Duration(makerDepthFilter.TimeDeltaEma-takerDepthFilter.TimeDeltaEma)*time.Millisecond
			//取新一点的时间为spread time
			if makerWalkedDepth.Time.Sub(takerWalkedDepth.Time) < 0 {
				//需要对时间进行补偿
				spreadTime = takerWalkedDepth.Time.Add(time.Millisecond * time.Duration(takerDepthFilter.TimeDeltaEma))
			} else {
				//需要对时间进行补偿
				spreadTime = makerWalkedDepth.Time.Add(time.Millisecond * time.Duration(makerDepthFilter.TimeDeltaEma))
			}
			if adjustedAgeDiff > maxAgeDiffBias {
				//logger.Debugf("adjustedAgeDiff %v maxAgeDiffBias %v failed, taker expire", adjustedAgeDiff, maxAgeDiffBias)
				takerExpireCount++
				break
			} else if adjustedAgeDiff < -maxAgeDiffBias {
				//logger.Debugf("adjustedAgeDiff %v maxAgeDiffBias %v failed, maker expire mema %f %f tema %f",
				//	adjustedAgeDiff, maxAgeDiffBias,
				//	makerDepthFilter.TimeDelta,
				//	makerDepthFilter.TimeDeltaEma,
				//	takerDepthFilter.TimeDeltaEma,
				//)
				makerExpireCount++
				break
			}
			matchCount++
			shortLastEnter = (takerWalkedDepth.TakerBid - makerWalkedDepth.MakerBid) / makerWalkedDepth.MakerBid
			longLastEnter = (takerWalkedDepth.TakerAsk - makerWalkedDepth.MakerAsk) / makerWalkedDepth.MakerAsk

			shortEnterTimedMedian.Insert(spreadTime, shortLastEnter)
			longEnterTimedMedian.Insert(spreadTime, longLastEnter)

			if shortEnterTimedMedian.Len() < 2 {
				break
			}
			if shortEnterTimedMedian.Range() < lookbackDuration/2 {
				break
			}

			select {
			case <-ctx.Done():
			case outputCh <- &common.MakerTakerSpread{
				TakerSymbol: takerSymbol,
				MakerSymbol: makerSymbol,
				TakerDepth:  *takerWalkedDepth,
				MakerDepth:  *makerWalkedDepth,

				ShortLastEnter:   shortLastEnter,
				ShortLastLeave:   longLastEnter,
				ShortMedianEnter: shortEnterTimedMedian.Median(),
				ShortMedianLeave: longEnterTimedMedian.Median(),

				LongLastEnter:   longLastEnter,
				LongLastLeave:   shortLastEnter,
				LongMedianEnter: longEnterTimedMedian.Median(),
				LongMedianLeave: shortEnterTimedMedian.Median(),

				AgeDiff: adjustedAgeDiff,
				Time:    spreadTime,
			}:
			default:
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("outputCh <- &common.MakerTakerSpread %s-%s len(outputCh) %d", makerSymbol, takerSymbol, len(outputCh))
					logSilentTime = time.Now().Add(time.Minute)
				}
			}
			break
		case <-makerWalkDepthTimer.C:
			if makerDepth != nil {
				makerWalkedDepth, err = common.WalkMakerTakerDepth20(makerDepth, makerImpact, takerImpact)
				if err != nil {
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("maker common.WalkMakerTakerDepth20 error %v %s", err, makerSymbol)
						logSilentTime = time.Now().Add(time.Minute)
					}
					break
				}
				walkSpreadTimer.Reset(time.Nanosecond)
			}
			break
		case <-takerWalkDepthTimer.C:
			if takerDepth != nil {
				takerWalkedDepth, err = common.WalkMakerTakerDepth20(takerDepth, makerImpact, takerImpact)
				if err != nil {
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("taker common.WalkMakerTakerDepth5 error %v %s", err, takerSymbol)
						logSilentTime = time.Now().Add(time.Minute)
					}
					break
				}
				walkSpreadTimer.Reset(time.Nanosecond)
			}
			break

		case newMakerDepth = <-makerDepthCh:
			if makerDepth != nil && makerDepth.GetTime().Sub(newMakerDepth.GetTime()) >= 0 {
				break
			}
			makerDepth = newMakerDepth
			if !makerDepthFilter.Filter(makerDepth) && takerDepth != nil {
				adjustedAgeDiff = makerDepth.GetTime().Sub(takerDepth.GetTime()) + time.Duration(math.Abs(makerDepthFilter.TimeDeltaEma-takerDepthFilter.TimeDeltaEma))*time.Millisecond
				if adjustedAgeDiff > maxAgeDiffBias {
					//taker已经过期
					takerExpireCount++
				}
				if adjustedAgeDiff < -maxAgeDiffBias {
					//maker已经过期
					makerExpireCount++
				} else {
					makerWalkDepthTimer.Reset(expectedChanSendingTime)
				}
			}
			depthCount++
			if depthCount > reportCount {
				makerDepthFilter.GenerateReport()
				takerDepthFilter.GenerateReport()
				select {
				case reportCh <- common.SpreadReport{
					AdjustedAgeDiff:       adjustedAgeDiff,
					MatchRatio:            float64(matchCount) / float64(depthCount),
					MakerSymbol:           makerSymbol,
					TakerSymbol:           takerSymbol,
					MakerTimeDeltaEma:     makerDepthFilter.TimeDeltaEma,
					TakerTimeDeltaEma:     takerDepthFilter.TimeDeltaEma,
					MakerTimeDelta:        makerDepthFilter.TimeDelta,
					TakerTimeDelta:        takerDepthFilter.TimeDelta,
					MakerDepthFilterRatio: makerDepthFilter.Report.FilterRatio,
					TakerDepthFilterRatio: takerDepthFilter.Report.FilterRatio,
					MakerExpireRatio:      float64(makerExpireCount) / float64(depthCount),
					TakerExpireRatio:      float64(takerExpireCount) / float64(depthCount),
				}:
				default:
				}
				matchCount = 0
				depthCount = 0
				takerExpireCount = 0
				makerExpireCount = 0
			}
			break
		case newTakerDepth = <-takerDepthCh:
			if takerDepth != nil &&
				takerDepth.GetTime().Sub(newTakerDepth.GetTime()) >= 0 {
				break
			}
			takerDepth = newTakerDepth
			if !takerDepthFilter.Filter(takerDepth) && makerDepth != nil {
				adjustedAgeDiff = makerDepth.GetTime().Sub(takerDepth.GetTime()) + time.Duration(math.Abs(makerDepthFilter.TimeDeltaEma-takerDepthFilter.TimeDeltaEma))*time.Millisecond
				if adjustedAgeDiff > maxAgeDiffBias {
					//taker已经过期
					takerExpireCount++
				} else {
					takerWalkDepthTimer.Reset(expectedChanSendingTime)
				}
				if adjustedAgeDiff < -maxAgeDiffBias {
					//maker已经过期
					makerExpireCount++
				}
			}
			depthCount++
			if depthCount > reportCount {
				makerDepthFilter.GenerateReport()
				takerDepthFilter.GenerateReport()
				select {
				case reportCh <- common.SpreadReport{
					AdjustedAgeDiff:       adjustedAgeDiff,
					MatchRatio:            float64(matchCount) / float64(depthCount),
					MakerSymbol:           makerSymbol,
					TakerSymbol:           takerSymbol,
					MakerTimeDeltaEma:     makerDepthFilter.TimeDeltaEma,
					TakerTimeDeltaEma:     takerDepthFilter.TimeDeltaEma,
					MakerTimeDelta:        makerDepthFilter.TimeDelta,
					TakerTimeDelta:        takerDepthFilter.TimeDelta,
					MakerDepthFilterRatio: makerDepthFilter.Report.FilterRatio,
					TakerDepthFilterRatio: takerDepthFilter.Report.FilterRatio,
					MakerExpireRatio:      float64(makerExpireCount) / float64(depthCount),
					TakerExpireRatio:      float64(takerExpireCount) / float64(depthCount),
				}:
				default:
				}
				matchCount = 0
				depthCount = 0
				takerExpireCount = 0
				makerExpireCount = 0
			}
			break
		}
	}
}
