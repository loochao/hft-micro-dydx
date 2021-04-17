package main

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"time"
)

func watchSpread(
	ctx context.Context,
	hSymbols []string,
	bhSymbolsMap map[string]string,
	maxAgeDiff,
	maxAge,
	lookbackDuration time.Duration,
	lookbackWindow int,
	walkedOrderBookCh chan WalkedOrderBook,
	outputCh chan Spread,
) {
	hOrderBooks := make(map[string]WalkedOrderBook)
	bOrderBooks := make(map[string]WalkedOrderBook)
	shortEnterSpreadWindows := make(map[string][]float64)
	shortExitSpreadWindows := make(map[string][]float64)
	shortEnterSpreadSortedSlices := make(map[string]common.SortedFloatSlice)
	shortExitSpreadSortedSlices := make(map[string]common.SortedFloatSlice)

	longEnterSpreadWindows := make(map[string][]float64)
	longExitSpreadWindows := make(map[string][]float64)
	longEnterSpreadSortedSlices := make(map[string]common.SortedFloatSlice)
	longExitSpreadSortedSlices := make(map[string]common.SortedFloatSlice)

	parseTimes := make(map[string][]time.Time)
	for _, hSymbol := range hSymbols {

		shortEnterSpreadWindows[hSymbol] = make([]float64, 0)
		shortExitSpreadWindows[hSymbol] = make([]float64, 0)
		shortEnterSpreadSortedSlices[hSymbol] = common.SortedFloatSlice{}
		shortExitSpreadSortedSlices[hSymbol] = common.SortedFloatSlice{}

		longEnterSpreadWindows[hSymbol] = make([]float64, 0)
		longExitSpreadWindows[hSymbol] = make([]float64, 0)
		longEnterSpreadSortedSlices[hSymbol] = common.SortedFloatSlice{}
		longExitSpreadSortedSlices[hSymbol] = common.SortedFloatSlice{}

		parseTimes[hSymbol] = make([]time.Time, 0)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case lob := <-walkedOrderBookCh:
			hSymbol := lob.Symbol
			var hLob, bLob WalkedOrderBook
			var ok bool
			if lob.Type == WalkedOrderBookTypeTaker {
				hSymbol = bhSymbolsMap[hSymbol]
				bOrderBooks[hSymbol] = lob
				bLob = lob
				if hLob, ok = hOrderBooks[hSymbol]; !ok {
					break
				}
			} else if lob.Type == WalkedOrderBookTypeMaker {
				hOrderBooks[hSymbol] = lob
				hLob = lob
				if bLob, ok = bOrderBooks[hSymbol]; !ok {
					break
				}
			} else {
				break
			}

			ageDiff := hLob.ParseTime.Sub(bLob.ParseTime)
			if ageDiff < 0 {
				ageDiff = -ageDiff
			}
			age := (time.Now().Sub(hLob.ParseTime) + time.Now().Sub(bLob.ParseTime)) / 2
			if age > maxAge ||
				ageDiff > maxAgeDiff {
				break
			}

			shortLastEnterSpread := (bLob.TakerBidVWAP - hLob.MakerBidVWAP) / hLob.MakerBidVWAP
			shortLastExitSpread := (bLob.TakerAskVWAP - hLob.MakerAskVWAP) / hLob.MakerAskVWAP

			longLastEnterSpread := (hLob.MakerAskVWAP - bLob.TakerAskVWAP ) / hLob.MakerAskVWAP
			longLastExitSpread := (hLob.MakerBidVWAP - bLob.TakerBidVWAP) / hLob.MakerBidVWAP

			parseTimes[hSymbol] = append(parseTimes[hSymbol], bLob.ParseTime)
			shortEnterSpreadWindows[hSymbol] = append(shortEnterSpreadWindows[hSymbol], shortLastEnterSpread)
			shortExitSpreadWindows[hSymbol] = append(shortExitSpreadWindows[hSymbol], shortLastExitSpread)
			shortEnterSpreadSortedSlices[hSymbol] = shortEnterSpreadSortedSlices[hSymbol].Insert(shortLastEnterSpread)
			shortExitSpreadSortedSlices[hSymbol] = shortExitSpreadSortedSlices[hSymbol].Insert(shortLastExitSpread)

			longEnterSpreadWindows[hSymbol] = append(longEnterSpreadWindows[hSymbol], longLastEnterSpread)
			longExitSpreadWindows[hSymbol] = append(longExitSpreadWindows[hSymbol], longLastExitSpread)
			longEnterSpreadSortedSlices[hSymbol] = longEnterSpreadSortedSlices[hSymbol].Insert(longLastEnterSpread)
			longExitSpreadSortedSlices[hSymbol] = longExitSpreadSortedSlices[hSymbol].Insert(longLastExitSpread)
			cutIndex := 0
			for i, arrivalTime := range parseTimes[hSymbol] {
				if lob.ParseTime.Sub(arrivalTime) > lookbackDuration {
					cutIndex = i
				} else {
					break
				}
			}
			if cutIndex > 0 {
				for _, d := range shortEnterSpreadWindows[hSymbol][:cutIndex] {
					shortEnterSpreadSortedSlices[hSymbol] = shortEnterSpreadSortedSlices[hSymbol].Delete(d)
				}
				for _, d := range shortExitSpreadWindows[hSymbol][:cutIndex] {
					shortExitSpreadSortedSlices[hSymbol] = shortExitSpreadSortedSlices[hSymbol].Delete(d)
				}
				shortEnterSpreadWindows[hSymbol] = shortEnterSpreadWindows[hSymbol][cutIndex:]
				shortExitSpreadWindows[hSymbol] = shortExitSpreadWindows[hSymbol][cutIndex:]

				for _, d := range longEnterSpreadWindows[hSymbol][:cutIndex] {
					longEnterSpreadSortedSlices[hSymbol] = longEnterSpreadSortedSlices[hSymbol].Delete(d)
				}
				for _, d := range longExitSpreadWindows[hSymbol][:cutIndex] {
					longExitSpreadSortedSlices[hSymbol] = longExitSpreadSortedSlices[hSymbol].Delete(d)
				}
				longEnterSpreadWindows[hSymbol] = longEnterSpreadWindows[hSymbol][cutIndex:]
				longExitSpreadWindows[hSymbol] = longExitSpreadWindows[hSymbol][cutIndex:]

				parseTimes[hSymbol] = parseTimes[hSymbol][cutIndex:]
			}

			if len(shortEnterSpreadWindows[hSymbol]) < lookbackWindow ||
				len(shortExitSpreadWindows[hSymbol]) < lookbackWindow {
				break
			}

			arrivalTimeDiff := lob.ParseTime.Sub(parseTimes[hSymbol][0])
			if arrivalTimeDiff < lookbackDuration/2 {
				break
			}

			shortMedianEnterSpread := shortEnterSpreadSortedSlices[hSymbol].Median()
			shortMedianExitSpread := shortExitSpreadSortedSlices[hSymbol].Median()

			longMedianEnterSpread := longEnterSpreadSortedSlices[hSymbol].Median()
			longMedianExitSpread := longExitSpreadSortedSlices[hSymbol].Median()

			outputCh <- Spread{
				HSymbol:        hSymbol,
				MakerOrderBook: bLob,
				TakerOrderBook: hLob,
				LastUpdateTime: lob.ParseTime,

				ShortLastEnter:   shortLastEnterSpread,
				ShortLastExit:    shortLastExitSpread,
				ShortMedianEnter: shortMedianEnterSpread,
				ShortMedianExit:  shortMedianExitSpread,

				LongLastEnter:   longLastEnterSpread,
				LongLastExit:    longLastExitSpread,
				LongMedianEnter: longMedianEnterSpread,
				LongMedianExit:  longMedianExitSpread,

				Age:              age,
				AgeDiff:          ageDiff,
			}
		}
	}
}
