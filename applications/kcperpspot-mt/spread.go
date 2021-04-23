package main

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/kcperp"
	"github.com/geometrybase/hft-micro/kcspot"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func watchMakerTakerSpread(
	ctx context.Context,
	makerSymbol, takerSymbol string,
	multiplier,
	makerImpact, takerImpact float64,
	maxAgeDiff,
	maxAge,
	lookbackDuration time.Duration,
	lookbackMinimalWindow int,
	makerDepthCh, takerDepthCh chan *common.DepthRawMessage,
	reportCh chan common.SpreadReport,
	outputCh chan *common.MakerTakerSpread,
) {
	var err error
	var makerRawDepth, takerRawDepth *common.DepthRawMessage
	var makerDepth, newMakerDepth *kcspot.Depth5
	var takerDepth, newTakerDepth *kcperp.Depth5
	var makerWalkedDepth, takerWalkedDepth *common.WalkedMakerTakerDepth
	var spreadTime time.Time
	var ageDiff, age time.Duration
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
	spread := 0.0
	i := 0
	matchCount := 0
	depthCount := 0
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
			age = (time.Now().Sub(makerWalkedDepth.Time) + time.Now().Sub(takerWalkedDepth.Time)) / 2
			if age > maxAge || ageDiff > maxAgeDiff {
				break
			}
			matchCount++
			shortLastEnter = (takerWalkedDepth.TakerBid - makerWalkedDepth.MakerBid) / makerWalkedDepth.MakerBid
			shortLastLeave = (takerWalkedDepth.TakerAsk - makerWalkedDepth.MakerAsk) / makerWalkedDepth.MakerAsk

			longLastEnter = (takerWalkedDepth.TakerAsk - makerWalkedDepth.MakerAsk) / makerWalkedDepth.MakerAsk
			longLastLeave = (takerWalkedDepth.TakerBid - makerWalkedDepth.MakerBid) / makerWalkedDepth.MakerBid

			times = append(times, takerWalkedDepth.Time)
			shortEnterWindow = append(shortEnterWindow, shortLastEnter)
			shortLeaveWindow = append(shortLeaveWindow, shortLastLeave)
			shortEnterSortedSlice = shortEnterSortedSlice.Insert(shortLastEnter)
			shortLeaveSortedSlice = shortLeaveSortedSlice.Insert(shortLastLeave)

			longEnterWindow = append(longEnterWindow, longLastEnter)
			longLeaveWindow = append(longLeaveWindow, longLastLeave)
			longEnterSortedSlice = longEnterSortedSlice.Insert(longLastEnter)
			longLeaveSortedSlice = longLeaveSortedSlice.Insert(longLastLeave)
			cutIndex = 0
			for i, eventTime := range times {
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

				for _, spread = range longEnterWindow[:cutIndex] {
					longEnterSortedSlice = longEnterSortedSlice.Delete(spread)
				}
				for _, spread = range longLeaveWindow[:cutIndex] {
					longLeaveSortedSlice = longLeaveSortedSlice.Delete(spread)
				}
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
			case <-ctx.Done():
			case outputCh <- &common.MakerTakerSpread{
				TakerSymbol: takerSymbol,
				MakerSymbol: makerSymbol,
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

				Age:     age,
				AgeDiff: ageDiff,
				Time:    spreadTime,
			}:
			default:
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("SEND SPREAD FAILED %s-%s", makerSymbol, takerSymbol)
					logSilentTime = time.Now().Add(time.Minute)
				}
			}
			break
		case <-makerWalkDepthTimer.C:
			if makerDepth != nil {
				makerWalkedDepth, err = common.WalkMakerTakerDepth5(makerDepth, makerImpact, takerImpact)
				if err != nil {
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("makerWalkedDepth common.WalkMakerTakerDepth5 error %v", err)
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
						logger.Debugf("takerWalkedDepth common.WalkMakerTakerDepth5 error %v", err)
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
					logger.Debugf("kcspot.ParseDepth5 error %v", err)
					logSilentTime = time.Now().Add(time.Minute)
				}
			} else if makerDepth == nil || newMakerDepth.EventTime.Sub(makerDepth.EventTime) > 0 {
				//需要乘以multiplier
				for i = range newMakerDepth.Bids {
					newMakerDepth.Bids[i][1] *= multiplier
					newMakerDepth.Asks[i][1] *= multiplier
				}
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
					logger.Debugf("kcperp.ParseDepth5 error %v", err)
					logSilentTime = time.Now().Add(time.Minute)
				}
			} else if takerDepth == nil || newTakerDepth.Sequence > takerDepth.Sequence {
				takerDepth = newTakerDepth
				takerWalkDepthTimer.Reset(time.Nanosecond)
			}
			break
		case makerRawDepth = <-makerDepthCh:
			if takerRawDepth != nil {
				ageDiff = makerRawDepth.Time.Sub(takerRawDepth.Time)
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
			if depthCount > 1000 {
				select {
				case reportCh <- common.SpreadReport{
					MaxAge:      maxAge,
					MaxAgeDiff:  maxAgeDiff,
					MatchRatio:  float64(matchCount) / float64(depthCount),
					TakerSymbol: takerSymbol,
					MakerSymbol: makerSymbol,
				}:
				default:
				}
				matchCount = 0
				depthCount = 0
			}
			break
		case takerRawDepth = <-takerDepthCh:
			if makerRawDepth != nil {
				ageDiff = takerRawDepth.Time.Sub(makerRawDepth.Time)
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
			if depthCount > 1000 {
				select {
				case reportCh <- common.SpreadReport{
					MaxAge:      maxAge,
					MaxAgeDiff:  maxAgeDiff,
					MatchRatio:  float64(matchCount) / float64(depthCount),
					TakerSymbol: takerSymbol,
					MakerSymbol: makerSymbol,
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
