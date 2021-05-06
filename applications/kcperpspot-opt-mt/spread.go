package main

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/kcperp"
	"github.com/geometrybase/hft-micro/kcspot"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"time"
)

func watchMakerTakerSpread(
	ctx context.Context,
	makerSymbol, takerSymbol string,
	multiplier,
	takerImpact float64,
	makerDecay, makerBias,
	takerDecay, takerBias float64,
	maxAgeDiffBias time.Duration,
	reportCount int,
	lookbackDuration time.Duration,
	lookbackMinimalWindow int,
	makerDepthCh, takerDepthCh chan *common.DepthRawMessage,
	reportCh chan common.SpreadReport,
	outputCh chan *common.ShortSpread,
) {
	var err error
	var makerRawDepth, takerRawDepth *common.DepthRawMessage
	var makerDepth, newMakerDepth *kcspot.Depth5
	var takerDepth, newTakerDepth *kcperp.Depth5
	var makerWalkedDepth, takerWalkedDepth *common.WalkedTakerDepth
	var spreadTime time.Time
	var ageDiff time.Duration
	var maxAgeDiff = time.Duration(takerBias + makerBias)
	var makerDepthFilter = common.NewDepthFilter(makerDecay, makerBias)
	var takerDepthFilter = common.NewDepthFilter(takerDecay, takerBias)

	shortEnterWindow := make([]float64, 0)
	shortLeaveWindow := make([]float64, 0)
	shortEnterSortedSlice := common.SortedFloatSlice{}
	shortLeaveSortedSlice := common.SortedFloatSlice{}
	times := make([]time.Time, 0)

	logSilentTime := time.Now()
	walkSpreadTimer := time.NewTimer(time.Hour * 999)
	makerWalkDepthTimer := time.NewTimer(time.Hour * 999)
	takerWalkDepthTimer := time.NewTimer(time.Hour * 999)
	makerParseTimer := time.NewTimer(time.Hour * 999)
	takerParseTimer := time.NewTimer(time.Hour * 999)

	expectedChanSendingTime := time.Nanosecond * 300
	cutIndex := 0
	spread := 0.0
	i := 0
	matchCount := 0
	depthCount := 0
	var eventTime time.Time
	var shortLastEnter, shortLastLeave float64
	for {
		select {
		case <-ctx.Done():
			return
		case <-walkSpreadTimer.C:
			if makerWalkedDepth == nil || takerWalkedDepth == nil {
				break
			}
			ageDiff = makerWalkedDepth.Time.Sub(takerWalkedDepth.Time)
			//取新一点的时间为spread time
			if ageDiff < 0 {
				spreadTime = takerWalkedDepth.Time
				ageDiff = -ageDiff
			} else {
				spreadTime = makerWalkedDepth.Time
			}
			if ageDiff > maxAgeDiff {
				break
			}
			matchCount++
			shortLastEnter = (takerWalkedDepth.TakerBid - makerWalkedDepth.MidPrice) / makerWalkedDepth.MidPrice
			shortLastLeave = (takerWalkedDepth.TakerAsk - makerWalkedDepth.MidPrice) / makerWalkedDepth.MidPrice

			times = append(times, takerWalkedDepth.Time)
			shortEnterWindow = append(shortEnterWindow, shortLastEnter)
			shortLeaveWindow = append(shortLeaveWindow, shortLastLeave)
			shortEnterSortedSlice = shortEnterSortedSlice.Insert(shortLastEnter)
			shortLeaveSortedSlice = shortLeaveSortedSlice.Insert(shortLastLeave)

			cutIndex = 0
			for i, eventTime = range times {
				if spreadTime.Sub(eventTime) > lookbackDuration {
					cutIndex = i
				} else {
					break
				}
			}
			if cutIndex > 0 {
				for _, spread = range shortEnterWindow[:cutIndex] {
					shortEnterSortedSlice = shortEnterSortedSlice.Delete(spread)
				}
				for _, spread = range shortLeaveWindow[:cutIndex] {
					shortLeaveSortedSlice = shortLeaveSortedSlice.Delete(spread)
				}
				shortEnterWindow = shortEnterWindow[cutIndex:]
				shortLeaveWindow = shortLeaveWindow[cutIndex:]

				times = times[cutIndex:]
			}

			if len(shortEnterWindow) < lookbackMinimalWindow ||
				len(shortLeaveWindow) < lookbackMinimalWindow {
				break
			}

			if spreadTime.Sub(times[0]) < lookbackDuration/2 {
				break
			}

			select {
			case <-ctx.Done():
			case outputCh <- &common.ShortSpread{
				TakerSymbol: takerSymbol,
				MakerSymbol: makerSymbol,
				TakerDepth:  *takerWalkedDepth,
				MakerDepth:  *makerWalkedDepth,

				LastEnter:   shortLastEnter,
				LastLeave:   shortLastLeave,
				MedianEnter: shortEnterSortedSlice.Median(),
				MedianLeave: shortLeaveSortedSlice.Median(),
				AgeDiff:     ageDiff,
				Time:        spreadTime,
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
				makerWalkedDepth, err = common.WalkTakerDepth5(makerDepth, takerImpact)
				if err != nil {
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("common.WalkTakerDepth5 error %v %s", err, makerSymbol)
						logSilentTime = time.Now().Add(time.Minute)
					}
					break
				}
				walkSpreadTimer.Reset(time.Nanosecond)
			}
			break
		case <-takerWalkDepthTimer.C:
			if takerDepth != nil {
				takerWalkedDepth, err = common.WalkTakerDepth5(takerDepth, takerImpact)
				if err != nil {
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("common.WalkTakerDepth5 error %v %s", err, takerSymbol)
						logSilentTime = time.Now().Add(time.Minute)
					}
					break
				}
				walkSpreadTimer.Reset(time.Nanosecond)
			}
			break
		case <-makerParseTimer.C:
			if makerRawDepth == nil {
				break
			}
			newMakerDepth, err = kcspot.ParseDepth5(makerRawDepth.Depth)
			if err != nil {
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("kcspot.ParseDepth5 error %v %s %s", err, makerSymbol, makerRawDepth.Depth)
					logSilentTime = time.Now().Add(time.Minute)
				}
			} else if makerDepth == nil || newMakerDepth.EventTime.Sub(makerDepth.EventTime) > 0 {
				makerDepth = newMakerDepth
				makerWalkDepthTimer.Reset(time.Nanosecond)
			}
			break
		case <-takerParseTimer.C:
			if takerRawDepth == nil {
				break
			}
			newTakerDepth, err = kcperp.ParseDepth5(takerRawDepth.Depth)
			if err != nil {
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("kcperp.ParseDepth5 error %v %s %s", err, takerSymbol, takerRawDepth.Depth)
					logSilentTime = time.Now().Add(time.Minute)
				}
			} else if takerDepth == nil || newTakerDepth.Sequence > takerDepth.Sequence {
				//需要乘以multiplier
				for i = range newTakerDepth.Bids {
					newTakerDepth.Bids[i][1] *= multiplier
					newTakerDepth.Asks[i][1] *= multiplier
				}
				takerDepth = newTakerDepth
				takerWalkDepthTimer.Reset(time.Nanosecond)
			}
			break
		case makerRawDepth = <-makerDepthCh:
			if !makerDepthFilter.Filter(makerRawDepth) && takerRawDepth != nil {
				maxAgeDiff = time.Duration(math.Abs(makerDepthFilter.TimeDeltaEma-takerDepthFilter.TimeDeltaEma))*time.Millisecond + maxAgeDiffBias
				ageDiff = makerRawDepth.Time.Sub(takerRawDepth.Time)
				//logger.Debugf("%v\t%v\t%.2f\t%.2f", maxAgeDiff, ageDiff, makerDepthFilter.TimeDeltaEma, takerDepthFilter.TimeDeltaEma)
				if ageDiff > maxAgeDiff {
					//taker已经过期
					takerRawDepth = nil
					takerDepth = nil
					takerWalkedDepth = nil
				} else if ageDiff < -maxAgeDiff {
					//maker已经过期
					makerRawDepth = nil
					makerDepth = nil
					makerWalkedDepth = nil
				}
			}
			makerParseTimer.Reset(expectedChanSendingTime)
			depthCount++
			if depthCount > reportCount {
				makerDepthFilter.GenerateReport()
				takerDepthFilter.GenerateReport()
				select {
				case reportCh <- common.SpreadReport{
					MaxAgeDiff:            maxAgeDiff,
					MatchRatio:            float64(matchCount) / float64(depthCount),
					MakerSymbol:           makerSymbol,
					TakerSymbol:           takerSymbol,
					MakerMsgAvgLen:        makerDepthFilter.Report.MsgAvgLen,
					TakerMsgAvgLen:        takerDepthFilter.Report.MsgAvgLen,
					MakerTimeDeltaEma:     makerDepthFilter.TimeDeltaEma,
					TakerTimeDeltaEma:     takerDepthFilter.TimeDeltaEma,
					MakerTimeDelta:        makerDepthFilter.TimeDelta,
					TakerTimeDelta:        takerDepthFilter.TimeDelta,
					MakerDepthFilterRatio: makerDepthFilter.Report.FilterRatio,
					TakerDepthFilterRatio: takerDepthFilter.Report.FilterRatio,
				}:
				default:
				}
				matchCount = 0
				depthCount = 0
			}
			break
		case takerRawDepth = <-takerDepthCh:
			if !takerDepthFilter.Filter(takerRawDepth) && makerRawDepth != nil {
				maxAgeDiff = time.Duration(math.Abs(makerDepthFilter.TimeDeltaEma-takerDepthFilter.TimeDeltaEma))*time.Millisecond + maxAgeDiffBias
				ageDiff = takerRawDepth.Time.Sub(makerRawDepth.Time)
				//logger.Debugf("%v\t%v\t%.2f\t%.2f", maxAgeDiff, ageDiff, makerDepthFilter.TimeDeltaEma, takerDepthFilter.TimeDeltaEma)
				if ageDiff > maxAgeDiff {
					//maker已经过期
					makerRawDepth = nil
					makerDepth = nil
					makerWalkedDepth = nil
				} else if ageDiff < -maxAgeDiff {
					//taker已经过期
					takerRawDepth = nil
					takerDepth = nil
					takerWalkedDepth = nil
				}
			}
			takerParseTimer.Reset(expectedChanSendingTime)
			depthCount++
			if depthCount > reportCount {
				makerDepthFilter.GenerateReport()
				takerDepthFilter.GenerateReport()
				select {
				case reportCh <- common.SpreadReport{
					MaxAgeDiff:            maxAgeDiff,
					MatchRatio:            float64(matchCount) / float64(depthCount),
					MakerSymbol:           makerSymbol,
					TakerSymbol:           takerSymbol,
					MakerMsgAvgLen:        makerDepthFilter.Report.MsgAvgLen,
					TakerMsgAvgLen:        takerDepthFilter.Report.MsgAvgLen,
					MakerTimeDeltaEma:     makerDepthFilter.TimeDeltaEma,
					TakerTimeDeltaEma:     takerDepthFilter.TimeDeltaEma,
					MakerTimeDelta:        makerDepthFilter.TimeDelta,
					TakerTimeDelta:        takerDepthFilter.TimeDelta,
					MakerDepthFilterRatio: makerDepthFilter.Report.FilterRatio,
					TakerDepthFilterRatio: takerDepthFilter.Report.FilterRatio,
				}:
				default:
				}
				matchCount = 0
				depthCount = 0
			}
			break
		}
	}
}
