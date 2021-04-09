package main

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
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

			outputCh <- Spread{
				Symbol: symbol,
				SwapOrderBook: swapLob,
				SpotOrderBook: spotLob,
				LastUpdateTime: lob.ArrivalTime,
				LastEnter: lastEnterSpread,
				LastExit: lastExitSpread,
				MedianEnter: medianEnterSpread,
				MedianExit: medianExitSpread,
				Age: age,
				AgeDiff: ageDiff,
			}
		}
	}
}
