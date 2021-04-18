package main

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func watchSpread(
	ctx context.Context,
	makerSymbols []string,
	takerMakerSymbolsMap map[string]string,
	maxAgeDiff,
	maxAge,
	lookbackDuration time.Duration,
	lookbackWindow int,
	walkedOrderBookCh chan WalkedOrderBook,
	outputCh chan Spread,
) {
	defer func(){
		logger.Debugf("LOOP END watchSpread %s")
	}()
	makerOrderBooks := make(map[string]WalkedOrderBook)
	takerOrderBooks := make(map[string]WalkedOrderBook)
	shortEnterSpreadWindows := make(map[string][]float64)
	shortExitSpreadWindows := make(map[string][]float64)
	shortEnterSpreadSortedSlices := make(map[string]common.SortedFloatSlice)
	shortExitSpreadSortedSlices := make(map[string]common.SortedFloatSlice)

	longEnterSpreadWindows := make(map[string][]float64)
	longExitSpreadWindows := make(map[string][]float64)
	longEnterSpreadSortedSlices := make(map[string]common.SortedFloatSlice)
	longExitSpreadSortedSlices := make(map[string]common.SortedFloatSlice)

	parseTimes := make(map[string][]time.Time)
	for _, hSymbol := range makerSymbols {

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
			makerSymbol := lob.Symbol
			var makerLob, takerLob WalkedOrderBook
			var ok bool
			if lob.Type == WalkedOrderBookTypeTaker {
				makerSymbol = takerMakerSymbolsMap[lob.Symbol]
				takerOrderBooks[makerSymbol] = lob
				takerLob = lob
				if makerLob, ok = makerOrderBooks[makerSymbol]; !ok {
					break
				}
			} else if lob.Type == WalkedOrderBookTypeMaker {
				makerOrderBooks[makerSymbol] = lob
				makerLob = lob
				if takerLob, ok = takerOrderBooks[makerSymbol]; !ok {
					break
				}
			} else {
				break
			}

			ageDiff := makerLob.ParseTime.Sub(takerLob.ParseTime)
			if ageDiff < 0 {
				ageDiff = -ageDiff
			}
			age := (time.Now().Sub(makerLob.ParseTime) + time.Now().Sub(takerLob.ParseTime)) / 2
			if age > maxAge ||
				ageDiff > maxAgeDiff {
				break
			}

			shortLastEnterSpread := (takerLob.BidVWAP - makerLob.AskVWAP) / makerLob.AskVWAP
			shortLastExitSpread := (takerLob.AskVWAP - makerLob.BidVWAP) / makerLob.BidVWAP

			longLastEnterSpread := (takerLob.AskVWAP - makerLob.BidVWAP) / makerLob.BidVWAP
			longLastExitSpread := (takerLob.BidVWAP - makerLob.AskVWAP) / makerLob.AskVWAP

			parseTimes[makerSymbol] = append(parseTimes[makerSymbol], takerLob.ParseTime)
			shortEnterSpreadWindows[makerSymbol] = append(shortEnterSpreadWindows[makerSymbol], shortLastEnterSpread)
			shortExitSpreadWindows[makerSymbol] = append(shortExitSpreadWindows[makerSymbol], shortLastExitSpread)
			shortEnterSpreadSortedSlices[makerSymbol] = shortEnterSpreadSortedSlices[makerSymbol].Insert(shortLastEnterSpread)
			shortExitSpreadSortedSlices[makerSymbol] = shortExitSpreadSortedSlices[makerSymbol].Insert(shortLastExitSpread)

			longEnterSpreadWindows[makerSymbol] = append(longEnterSpreadWindows[makerSymbol], longLastEnterSpread)
			longExitSpreadWindows[makerSymbol] = append(longExitSpreadWindows[makerSymbol], longLastExitSpread)
			longEnterSpreadSortedSlices[makerSymbol] = longEnterSpreadSortedSlices[makerSymbol].Insert(longLastEnterSpread)
			longExitSpreadSortedSlices[makerSymbol] = longExitSpreadSortedSlices[makerSymbol].Insert(longLastExitSpread)
			cutIndex := 0
			for i, arrivalTime := range parseTimes[makerSymbol] {
				if lob.ParseTime.Sub(arrivalTime) > lookbackDuration {
					cutIndex = i
				} else {
					break
				}
			}
			if cutIndex > 0 {
				for _, d := range shortEnterSpreadWindows[makerSymbol][:cutIndex] {
					shortEnterSpreadSortedSlices[makerSymbol] = shortEnterSpreadSortedSlices[makerSymbol].Delete(d)
				}
				for _, d := range shortExitSpreadWindows[makerSymbol][:cutIndex] {
					shortExitSpreadSortedSlices[makerSymbol] = shortExitSpreadSortedSlices[makerSymbol].Delete(d)
				}
				shortEnterSpreadWindows[makerSymbol] = shortEnterSpreadWindows[makerSymbol][cutIndex:]
				shortExitSpreadWindows[makerSymbol] = shortExitSpreadWindows[makerSymbol][cutIndex:]

				for _, d := range longEnterSpreadWindows[makerSymbol][:cutIndex] {
					longEnterSpreadSortedSlices[makerSymbol] = longEnterSpreadSortedSlices[makerSymbol].Delete(d)
				}
				for _, d := range longExitSpreadWindows[makerSymbol][:cutIndex] {
					longExitSpreadSortedSlices[makerSymbol] = longExitSpreadSortedSlices[makerSymbol].Delete(d)
				}
				longEnterSpreadWindows[makerSymbol] = longEnterSpreadWindows[makerSymbol][cutIndex:]
				longExitSpreadWindows[makerSymbol] = longExitSpreadWindows[makerSymbol][cutIndex:]

				parseTimes[makerSymbol] = parseTimes[makerSymbol][cutIndex:]
			}

			if len(shortEnterSpreadWindows[makerSymbol]) < lookbackWindow ||
				len(shortExitSpreadWindows[makerSymbol]) < lookbackWindow {
				break
			}

			arrivalTimeDiff := lob.ParseTime.Sub(parseTimes[makerSymbol][0])
			if arrivalTimeDiff < lookbackDuration/2 {
				break
			}

			shortMedianEnterSpread := shortEnterSpreadSortedSlices[makerSymbol].Median()
			shortMedianExitSpread := shortExitSpreadSortedSlices[makerSymbol].Median()

			longMedianEnterSpread := longEnterSpreadSortedSlices[makerSymbol].Median()
			longMedianExitSpread := longExitSpreadSortedSlices[makerSymbol].Median()
			if len(outputCh) > 0 {
				logger.Debugf("LEN SPREAD CH %d", len(outputCh))
			}

			outputCh <- Spread{
				Symbol:         makerSymbol,
				MakerOrderBook: takerLob,
				TakerOrderBook: makerLob,
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
