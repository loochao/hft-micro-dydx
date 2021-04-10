package main

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"time"
)

func watchSpread(
	ctx context.Context,
	symbols []string,
	lookbackDuration time.Duration,
	lookbackWindow int,
	walkedOrderBookCh chan WalkedOrderBook,
	outputCh chan Spread,
) {
	longSpreadWindows := make(map[string][]float64)
	shortSpreadWindows := make(map[string][]float64)
	longSpreadSortedSlices := make(map[string]common.SortedFloatSlice)
	shortSpreadSortedSlices := make(map[string]common.SortedFloatSlice)
	arrivalTimes := make(map[string][]time.Time)
	for _, symbol := range symbols {
		longSpreadWindows[symbol] = make([]float64, 0)
		shortSpreadWindows[symbol] = make([]float64, 0)
		arrivalTimes[symbol] = make([]time.Time, 0)
		longSpreadSortedSlices[symbol] = common.SortedFloatSlice{}
		shortSpreadSortedSlices[symbol] = common.SortedFloatSlice{}
	}
	for {
		select {
		case <-ctx.Done():
			return
		case lob := <-walkedOrderBookCh:

			symbol := lob.Symbol

			lastLongSpread := (lob.CloseAskVWAP - lob.OpenBidVWAP) / lob.OpenBidVWAP
			lastShortSpread := (lob.OpenAskVWAP - lob.CloseBidVWAP) / lob.OpenAskVWAP

			arrivalTimes[symbol] = append(arrivalTimes[symbol], lob.ArrivalTime)
			longSpreadWindows[symbol] = append(longSpreadWindows[symbol], lastLongSpread)
			shortSpreadWindows[symbol] = append(shortSpreadWindows[symbol], lastShortSpread)
			longSpreadSortedSlices[symbol] = longSpreadSortedSlices[symbol].Insert(lastLongSpread)
			shortSpreadSortedSlices[symbol] = shortSpreadSortedSlices[symbol].Insert(lastShortSpread)
			cutIndex := 0
			for i, arrivalTime := range arrivalTimes[symbol] {
				if lob.ArrivalTime.Sub(arrivalTime) > lookbackDuration {
					cutIndex = i
				} else {
					break
				}
			}
			if cutIndex > 0 {
				for _, d := range longSpreadWindows[symbol][:cutIndex] {
					longSpreadSortedSlices[symbol] = longSpreadSortedSlices[symbol].Delete(d)
				}
				for _, d := range shortSpreadWindows[symbol][:cutIndex] {
					shortSpreadSortedSlices[symbol] = shortSpreadSortedSlices[symbol].Delete(d)
				}
				arrivalTimes[symbol] = arrivalTimes[symbol][cutIndex:]
				longSpreadWindows[symbol] = longSpreadWindows[symbol][cutIndex:]
				shortSpreadWindows[symbol] = shortSpreadWindows[symbol][cutIndex:]
			}

			if len(longSpreadWindows[symbol]) < lookbackWindow ||
				len(shortSpreadWindows[symbol]) < lookbackWindow {
				break
			}

			arrivalTimeDiff := lob.ArrivalTime.Sub(arrivalTimes[symbol][0])
			if arrivalTimeDiff < lookbackDuration/2 {
				break
			}

			medianLongSpread := longSpreadSortedSlices[symbol].Median()
			medianShortSpread := shortSpreadSortedSlices[symbol].Median()

			outputCh <- Spread{
				Symbol:      symbol,
				OrderBook:   lob,
				EventTime:   lob.EventTime,
				LastLong:    lastLongSpread,
				LastShort:   lastShortSpread,
				MedianLong:  medianLongSpread,
				MedianShort: medianShortSpread,
			}
		}
	}
}
