package main

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
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

