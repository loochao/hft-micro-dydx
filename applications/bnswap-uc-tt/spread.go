package main

import (
	"context"
	"github.com/geometrybase/hft-micro/bnspot"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"time"
)

func watchMakerTakerSpread(
	ctx context.Context,
	symbol string,
	makerImpact, takerImpact float64,
	makerDecay, makerBias,
	takerDecay, takerBias float64,
	maxAgeDiffBias time.Duration,
	reportCount int,
	lookbackDuration time.Duration,
	lookbackMinimalWindow int,
	makerDepthCh, takerDepthCh chan *common.DepthRawMessage,
	reportCh chan common.SpreadReport,
	outputCh chan *common.MakerTakerSpread,
) {
	logger.Debugf("START watchMakerTakerSpread %s", symbol)
	defer logger.Debugf("EXIT watchMakerTakerSpread %s", symbol)
	var err error
	var makerRawDepth, takerRawDepth *common.DepthRawMessage
	var makerDepth, newMakerDepth *bnspot.Depth5
	var takerDepth, newTakerDepth *bnswap.Depth5
	var makerWalkedDepth, takerWalkedDepth *common.WalkedMakerTakerDepth
	var spreadTime time.Time
	var ageDiff time.Duration
	var maxAgeDiff = time.Duration(takerBias + makerBias)
	var makerDepthFilter = common.NewDepthFilter(makerDecay, makerBias, -5000, 5000)
	var takerDepthFilter = common.NewDepthFilter(takerDecay, takerBias, -5000, 5000)
	shortEnterWindow := make([]float64, 0)
	shortLeaveWindow := make([]float64, 0)
	shortEnterSortedSlice := common.SortedFloatSlice{}
	shortLeaveSortedSlice := common.SortedFloatSlice{}
	longEnterWindow := make([]float64, 0)
	longLeaveWindow := make([]float64, 0)
	longEnterSortedSlice := common.SortedFloatSlice{}
	longLeaveSortedSlice := common.SortedFloatSlice{}
	times := make([]time.Time, 0)

	logSilentTime := time.Now()
	walkSpreadTimer := time.NewTimer(time.Hour * 999)
	makerWalkDepthTimer := time.NewTimer(time.Hour * 999)
	takerWalkDepthTimer := time.NewTimer(time.Hour * 999)
	makerParseTimer := time.NewTimer(time.Hour * 999)
	takerParseTimer := time.NewTimer(time.Hour * 999)

	expectedChanSendingTime := time.Nanosecond * 300
	cutIndex := 0
	i := 0
	matchCount := 0
	depthCount := 0
	var eventTime time.Time
	var shortLastEnter, shortLastLeave, longLastEnter, longLastLeave float64
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

			longLastEnter = (takerWalkedDepth.TakerAsk - makerWalkedDepth.MidPrice) / makerWalkedDepth.MidPrice
			longLastLeave = (takerWalkedDepth.TakerBid - makerWalkedDepth.MidPrice) / makerWalkedDepth.MidPrice

			times = append(times, takerWalkedDepth.Time)
			shortEnterWindow = append(shortEnterWindow, shortLastEnter)
			shortLeaveWindow = append(shortLeaveWindow, shortLastLeave)
			shortEnterSortedSlice = shortEnterSortedSlice.Insert(shortLastEnter)
			shortLeaveSortedSlice = shortLeaveSortedSlice.Insert(shortLastLeave)

			longEnterWindow = append(longEnterWindow, longLastEnter)
			longLeaveWindow = append(longLeaveWindow, longLastLeave)
			longEnterSortedSlice = longEnterSortedSlice.Insert(longLastEnter)
			longLeaveSortedSlice = longLeaveSortedSlice.Insert(longLastLeave)
			cutIndex = -1
			for i, eventTime = range times {
				if spreadTime.Sub(eventTime) > lookbackDuration {
					cutIndex = i
					shortEnterSortedSlice = shortEnterSortedSlice.Delete(shortEnterWindow[i])
					shortLeaveSortedSlice = shortLeaveSortedSlice.Delete(shortLeaveWindow[i])
					longEnterSortedSlice = longEnterSortedSlice.Delete(longEnterWindow[i])
					longLeaveSortedSlice = longLeaveSortedSlice.Delete(longLeaveWindow[i])
				} else {
					break
				}
			}
			//需要offset 1
			cutIndex += 1
			if cutIndex > 0 {
				shortEnterWindow = shortEnterWindow[cutIndex:]
				shortLeaveWindow = shortLeaveWindow[cutIndex:]
				longEnterWindow = longEnterWindow[cutIndex:]
				longLeaveWindow = longLeaveWindow[cutIndex:]
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
			case outputCh <- &common.MakerTakerSpread{
				TakerSymbol: symbol,
				MakerSymbol: symbol,
				TakerDepth:  *takerWalkedDepth,
				MakerDepth:  *makerWalkedDepth,

				ShortLastEnter:   shortLastEnter,
				ShortLastLeave:   shortLastLeave,
				ShortMedianEnter: shortEnterSortedSlice.Median(),
				ShortMedianLeave: shortLeaveSortedSlice.Median(),

				LongLastEnter:   longLastEnter,
				LongLastLeave:   longLastLeave,
				LongMedianEnter: longEnterSortedSlice.Median(),
				LongMedianLeave: longLeaveSortedSlice.Median(),

				AgeDiff: ageDiff,
				Time:    spreadTime,
			}:
			default:
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("outputCh <- &common.MakerTakerSpread %s-%s len(outputCh) %d", symbol, symbol, len(outputCh))
					logSilentTime = time.Now().Add(time.Minute)
				}
			}
			break
		case <-makerWalkDepthTimer.C:
			if makerDepth != nil {
				makerWalkedDepth, err = common.WalkMakerTakerDepth5(makerDepth, makerImpact, takerImpact)
				if err != nil {
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("common.WalkMakerTakerDepth5 error %v %s", err, symbol)
						logSilentTime = time.Now().Add(time.Minute)
					}
					break
				}
				walkSpreadTimer.Reset(time.Nanosecond)
			}
			break
		case <-takerWalkDepthTimer.C:
			if takerDepth != nil {
				takerWalkedDepth, err = common.WalkMakerTakerDepth5(takerDepth, makerImpact, takerImpact)
				if err != nil {
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("common.WalkMakerTakerDepth5 error %v %s", err, symbol)
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
			newMakerDepth, err = bnspot.ParseDepth5(makerRawDepth.Depth)
			if err != nil {
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("bnspot.ParseDepth5 error %v %s %s", err, symbol, makerRawDepth.Depth)
					logSilentTime = time.Now().Add(time.Minute)
				}
			} else if makerDepth == nil || newMakerDepth.LastUpdateId > makerDepth.LastUpdateId {
				makerDepth = newMakerDepth
				makerWalkDepthTimer.Reset(time.Nanosecond)
			}
			break
		case <-takerParseTimer.C:
			if takerRawDepth == nil {
				break
			}
			newTakerDepth, err = bnswap.ParseDepth5(takerRawDepth.Depth)
			if err != nil {
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("bnswap.ParseDepth5 error %v %s %s", err, symbol, takerRawDepth.Depth)
					logSilentTime = time.Now().Add(time.Minute)
				}
			} else if takerDepth == nil || newTakerDepth.LastUpdateId > takerDepth.LastUpdateId {
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
					AdjustedAgeDiff:       maxAgeDiff,
					MatchRatio:            float64(matchCount) / float64(depthCount),
					MakerSymbol:           symbol,
					TakerSymbol:           symbol,
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
					AdjustedAgeDiff:       maxAgeDiff,
					MatchRatio:            float64(matchCount) / float64(depthCount),
					MakerSymbol:           symbol,
					TakerSymbol:           symbol,
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
