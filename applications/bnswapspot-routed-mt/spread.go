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

func watchSpread(
	ctx context.Context,
	symbols []string,
	maxAgeDiff,
	maxAge,
	lookbackDuration time.Duration,
	lookbackWindow int,
	walkedOrderBookCh chan WalkedOrderBook,
	outputCh chan Spread,
) {
	swapOrderBooks := make(map[string]WalkedOrderBook)
	spotOrderBooks := make(map[string]WalkedOrderBook)
	enterSpreadWindows := make(map[string][]float64)
	exitSpreadWindows := make(map[string][]float64)
	enterSpreadSortedSlices := make(map[string]common.SortedFloatSlice)
	exitSpreadSortedSlices := make(map[string]common.SortedFloatSlice)
	arrivalTimes := make(map[string][]time.Time)
	for _, symbol := range symbols {
		enterSpreadWindows[symbol] = make([]float64, 0)
		exitSpreadWindows[symbol] = make([]float64, 0)
		arrivalTimes[symbol] = make([]time.Time, 0)
		enterSpreadSortedSlices[symbol] = common.SortedFloatSlice{}
		exitSpreadSortedSlices[symbol] = common.SortedFloatSlice{}
	}
	for {
		select {
		case <-ctx.Done():
			return
		case lob := <-walkedOrderBookCh:
			symbol := lob.Symbol
			var spotLob, swapLob WalkedOrderBook
			var ok bool
			if lob.Type == WalkedOrderBookTypeSwap {
				swapOrderBooks[symbol] = lob
				swapLob = lob
				if spotLob, ok = spotOrderBooks[symbol]; !ok {
					break
				}
			} else if lob.Type == WalkedOrderBookTypeSpot {
				spotOrderBooks[symbol] = lob
				spotLob = lob
				if swapLob, ok = swapOrderBooks[symbol]; !ok {
					break
				}
			} else {
				break
			}

			ageDiff := spotLob.ArrivalTime.Sub(swapLob.ArrivalTime)
			if ageDiff < 0 {
				ageDiff = -ageDiff
			}
			age := (time.Now().Sub(spotLob.ArrivalTime) + time.Now().Sub(swapLob.ArrivalTime)) / 2
			if age > maxAge ||
				ageDiff > maxAgeDiff {
				break
			}

			lastEnterSpread := (swapLob.TakerBidVWAP - spotLob.MakerBidVWAP) / spotLob.MakerBidVWAP
			lastExitSpread := (swapLob.TakerAskVWAP - spotLob.MakerAskVWAP) / spotLob.MakerAskVWAP

			arrivalTimes[symbol] = append(arrivalTimes[symbol], swapLob.ArrivalTime)
			enterSpreadWindows[symbol] = append(enterSpreadWindows[symbol], lastEnterSpread)
			exitSpreadWindows[symbol] = append(exitSpreadWindows[symbol], lastExitSpread)
			enterSpreadSortedSlices[symbol] = enterSpreadSortedSlices[symbol].Insert(lastEnterSpread)
			exitSpreadSortedSlices[symbol] = exitSpreadSortedSlices[symbol].Insert(lastExitSpread)
			cutIndex := 0
			for i, arrivalTime := range arrivalTimes[symbol] {
				if lob.ArrivalTime.Sub(arrivalTime) > lookbackDuration {
					cutIndex = i
				} else {
					break
				}
			}
			if cutIndex > 0 {
				for _, d := range enterSpreadWindows[symbol][:cutIndex] {
					enterSpreadSortedSlices[symbol] = enterSpreadSortedSlices[symbol].Delete(d)
				}
				for _, d := range exitSpreadWindows[symbol][:cutIndex] {
					exitSpreadSortedSlices[symbol] = exitSpreadSortedSlices[symbol].Delete(d)
				}
				arrivalTimes[symbol] = arrivalTimes[symbol][cutIndex:]
				enterSpreadWindows[symbol] = enterSpreadWindows[symbol][cutIndex:]
				exitSpreadWindows[symbol] = exitSpreadWindows[symbol][cutIndex:]
			}

			if len(enterSpreadWindows[symbol]) < lookbackWindow ||
				len(exitSpreadWindows[symbol]) < lookbackWindow {
				break
			}

			arrivalTimeDiff := lob.ArrivalTime.Sub(arrivalTimes[symbol][0])
			if arrivalTimeDiff < lookbackDuration/2 {
				break
			}

			medianEnterSpread := enterSpreadSortedSlices[symbol].Median()
			medianExitSpread := exitSpreadSortedSlices[symbol].Median()

			select {
			case <-ctx.Done():
				return
			case outputCh <- Spread{
				Symbol:         symbol,
				SwapOrderBook:  swapLob,
				SpotOrderBook:  spotLob,
				LastUpdateTime: lob.ArrivalTime,
				LastEnter:      lastEnterSpread,
				LastExit:       lastExitSpread,
				MedianEnter:    medianEnterSpread,
				MedianExit:     medianExitSpread,
				Age:            age,
				AgeDiff:        ageDiff,
			}:
			}

		}
	}
}

func watchSingleSpread(
	ctx context.Context,
	symbol string,
	maxAgeDiff,
	maxAge,
	lookbackDuration time.Duration,
	lookbackWindow int,
	walkedOrderBookCh chan *WalkedOrderBook,
	outputCh chan Spread,
) {
	var swapOrderBook, spotOrderBook *WalkedOrderBook
	enterSpreadWindow := make([]float64, 0)
	exitSpreadWindow := make([]float64, 0)
	enterSpreadSortedSlice := common.SortedFloatSlice{}
	exitSpreadSortedSlice := common.SortedFloatSlice{}
	arrivalTimes := make([]time.Time, 0)
	for {
		select {
		case <-ctx.Done():
			return
		case lob := <-walkedOrderBookCh:
			if lob.Type == WalkedOrderBookTypeSwap {
				swapOrderBook = lob
				if spotOrderBook == nil {
					break
				}
			} else if lob.Type == WalkedOrderBookTypeSpot {
				spotOrderBook = lob
				if swapOrderBook == nil {
					break
				}
			} else {
				break
			}

			ageDiff := spotOrderBook.ArrivalTime.Sub(swapOrderBook.ArrivalTime)
			if ageDiff < 0 {
				ageDiff = -ageDiff
			}
			age := (time.Now().Sub(spotOrderBook.ArrivalTime) + time.Now().Sub(swapOrderBook.ArrivalTime)) / 2
			if age > maxAge ||
				ageDiff > maxAgeDiff {
				break
			}

			lastEnterSpread := (swapOrderBook.TakerBidVWAP - spotOrderBook.MakerBidVWAP) / spotOrderBook.MakerBidVWAP
			lastExitSpread := (swapOrderBook.TakerAskVWAP - spotOrderBook.MakerAskVWAP) / spotOrderBook.MakerAskVWAP

			arrivalTimes = append(arrivalTimes, swapOrderBook.ArrivalTime)
			enterSpreadWindow = append(enterSpreadWindow, lastEnterSpread)
			exitSpreadWindow = append(exitSpreadWindow, lastExitSpread)
			enterSpreadSortedSlice = enterSpreadSortedSlice.Insert(lastEnterSpread)
			exitSpreadSortedSlice = exitSpreadSortedSlice.Insert(lastExitSpread)
			cutIndex := 0
			for i, arrivalTime := range arrivalTimes {
				if lob.ArrivalTime.Sub(arrivalTime) > lookbackDuration {
					cutIndex = i
				} else {
					break
				}
			}
			if cutIndex > 0 {
				for _, d := range enterSpreadWindow[:cutIndex] {
					enterSpreadSortedSlice = enterSpreadSortedSlice.Delete(d)
				}
				for _, d := range exitSpreadWindow[:cutIndex] {
					exitSpreadSortedSlice = exitSpreadSortedSlice.Delete(d)
				}
				arrivalTimes = arrivalTimes[cutIndex:]
				enterSpreadWindow = enterSpreadWindow[cutIndex:]
				exitSpreadWindow = exitSpreadWindow[cutIndex:]
			}

			if len(enterSpreadWindow) < lookbackWindow ||
				len(exitSpreadWindow) < lookbackWindow {
				break
			}

			arrivalTimeDiff := lob.ArrivalTime.Sub(arrivalTimes[0])
			if arrivalTimeDiff < lookbackDuration/2 {
				break
			}

			medianEnterSpread := enterSpreadSortedSlice.Median()
			medianExitSpread := exitSpreadSortedSlice.Median()

			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Millisecond):
				logger.Debugf("SPREAD TO OUTPUT CH TIMEOUT IN 1MS, CH LEN %d", len(outputCh))
			case outputCh <- Spread{
				Symbol:         symbol,
				SwapOrderBook:  *swapOrderBook,
				SpotOrderBook:  *spotOrderBook,
				LastUpdateTime: lob.ArrivalTime,
				LastEnter:      lastEnterSpread,
				LastExit:       lastExitSpread,
				MedianEnter:    medianEnterSpread,
				MedianExit:     medianExitSpread,
				Age:            age,
				AgeDiff:        ageDiff,
			}:
			}

		}
	}
}

func watchMakerTakerSpread(
	ctx context.Context,
	makerSymbol, takerSymbol string,
	multiplier,
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
	var err error
	var makerRawDepth, takerRawDepth *common.DepthRawMessage
	var makerDepth, newMakerDepth *kcspot.Depth5
	var takerDepth, newTakerDepth *kcperp.Depth5
	var makerWalkedDepth, takerWalkedDepth *common.WalkedMakerTakerDepth
	var spreadTime time.Time
	var ageDiff time.Duration
	var maxAgeDiff = time.Duration(takerBias + makerBias)
	var makerDepthFilter = common.NewDepthFilter(makerDecay, makerBias)
	var takerDepthFilter = common.NewDepthFilter(takerDecay, takerBias)
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
			if ageDiff > maxAgeDiff {
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

				AgeDiff: ageDiff,
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
				makerWalkedDepth, err = common.WalkMakerTakerDepth5(makerDepth, makerImpact, takerImpact)
				if err != nil {
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("common.WalkMakerTakerDepth5 error %v %s", err, makerSymbol)
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
						logger.Debugf("common.WalkMakerTakerDepth5 error %v %s", err, takerSymbol)
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
