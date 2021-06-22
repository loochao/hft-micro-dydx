package main

import (
	"context"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/kucoin-usdtfuture"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"time"
)

func watchMakerBars(
	ctx context.Context,
	api *kucoin_usdtfuture.API,
	takerSymbols []string,
	barsLookback int,
	pullInterval time.Duration,
	pullRetryInterval time.Duration,
	requestInterval time.Duration,
	output chan common.KLinesMap,
) {
	loopTimer := time.NewTimer(time.Second)
	defer loopTimer.Stop()
	outputTimer := time.NewTimer(time.Second)
	defer outputTimer.Stop()
	barsMap := make(common.KLinesMap)
	nextPullTimes := make(map[string]time.Time)
	for i, takerSymbol := range takerSymbols {
		nextPullTimes[takerSymbol] = time.Now().Add(requestInterval * time.Duration(i))
	}
	klineDuration := kucoin_usdtfuture.GranularityDurations[kucoin_usdtfuture.Granularity5Min]
	klineGranularity := kucoin_usdtfuture.Granularity5Min
	globalNextPullTime := time.Now()
	globalNextRetryTime := time.Now()
	outputResults := true
	for {
		select {
		case <-ctx.Done():
			return
		case <-outputTimer.C:
			outputResults = true
			outputTimer.Reset(pullInterval)
			break
		case <-loopTimer.C:
			globalNextPullTime = time.Now().Add(pullInterval)
			globalNextRetryTime = time.Now()
			for symbol, nextPullTime := range nextPullTimes {
				select {
				case <-ctx.Done():
					return
				default:
				}
				if time.Now().Sub(nextPullTime) < 0 {
					continue
				}
				symbolEndTime := time.Now().Truncate(klineDuration)
				symbolStartTime := symbolEndTime.Add(-klineDuration * time.Duration(barsLookback+3))
				if bars, ok := barsMap[symbol]; ok {
					symbolStartTime = bars[len(bars)-1].Timestamp
					//一分钟内说明是最新数据
					if math.Abs(time.Now().Truncate(time.Minute).Sub(symbolStartTime).Seconds()) < 60 {
						logger.Debugf("TAKER %s HAS NEWEST BAR,  FIRST TIME %v CLOSE %f LAST TIME %v CLOSE %f, CONTINUE",
							symbol,
							barsMap[symbol][0].Timestamp,
							barsMap[symbol][0].Close,
							barsMap[symbol][len(barsMap[symbol])-1].Timestamp,
							barsMap[symbol][len(barsMap[symbol])-1].Close,
						)
						globalNextPullTime = globalNextPullTime.Add(requestInterval)
						nextPullTimes[symbol] = globalNextPullTime
						continue
					}
					symbolStartTime = symbolStartTime.Add(-klineDuration * 3)
				}
				history, err := api.GetKlines(
					ctx, kucoin_usdtfuture.KlinesParam{
						Symbol:      symbol,
						From:        symbolStartTime.Unix() * 1000,
						To:          symbolEndTime.Unix() * 1000,
						Granularity: klineGranularity,
					})
				//history, err := api.GetHistoryKLines(ctx, symbol, klineInterval, symbolStartTime)
				if err != nil {
					logger.Debugf("api.GetKlines for %s error %v", symbol, err)
					select {
					case <-ctx.Done():
						return
					default:
					}
					globalNextRetryTime = globalNextRetryTime.Add(pullRetryInterval)
					nextPullTimes[symbol] = globalNextRetryTime
					continue
				}

				if len(history) <= 1 {
					logger.Debugf("TAKER %s BAR LEN <= 1", symbol)
					globalNextRetryTime = globalNextRetryTime.Add(pullRetryInterval)
					nextPullTimes[symbol] = globalNextRetryTime
					continue
				}
				//logger.Debugf("TAKER GET %s LEN %d LAST CLOSE %f TIME %v", symbol, len(history), history[len(history)-1].Close, history[len(history)-1].Timestamp)
				if _, ok := barsMap[symbol]; !ok {
					barsMap[symbol] = history
				}
				for _, bar := range history {
					if bar.Timestamp.Sub(symbolStartTime).Seconds() <= 0 {
						continue
					}
					if bar.Timestamp.Sub(time.Now().Truncate(time.Minute)).Seconds() > 0 {
						continue
					}
					lastBar := barsMap[symbol][len(barsMap[symbol])-1]
					if bar.Timestamp.Sub(lastBar.Timestamp).Seconds() <= 0 {
						continue
					}
					bar := bar
					barsMap[symbol] = append(barsMap[symbol], bar)
				}
				if len(barsMap[symbol]) > barsLookback {
					barsMap[symbol] = barsMap[symbol][len(barsMap[symbol])-barsLookback:]
				}
				globalNextPullTime = globalNextPullTime.Add(requestInterval)
				nextPullTimes[symbol] = globalNextPullTime
				continue
			}

			if outputResults {
				allSuccess := true
				for _, symbol := range takerSymbols {
					if len(barsMap[symbol]) == 0 {
						allSuccess = false
						break
					}
				}
				if allSuccess {
					outputMap := make(common.KLinesMap)
					for symbol, bars := range barsMap {
						outputMap[symbol] = make([]common.KLine, len(bars))
						copy(outputMap[symbol], bars)
					}
					select {
					case <-ctx.Done():
						return
					case output <- outputMap:
					}
					outputResults = false
					logger.Debugf("OUTPUT MAKER BARS")
				}
			}
			loopTimer.Reset(time.Second)
			break
		}
	}
}
func watchTakerBars(
	ctx context.Context,
	api *bnswap.API,
	takerSymbols []string,
	barsLookback int,
	pullInterval time.Duration,
	pullRetryInterval time.Duration,
	requestInterval time.Duration,
	output chan common.KLinesMap,
) {
	loopTimer := time.NewTimer(time.Second)
	defer loopTimer.Stop()
	outputTimer := time.NewTimer(time.Second)
	defer outputTimer.Stop()
	barsMap := make(common.KLinesMap)
	nextPullTimes := make(map[string]time.Time)
	for i, takerSymbol := range takerSymbols {
		nextPullTimes[takerSymbol] = time.Now().Add(requestInterval * time.Duration(i))
	}
	klineInterval := "5m"
	klineDuration := time.Minute * 5
	globalNextPullTime := time.Now()
	globalNextRetryTime := time.Now()
	outputResults := true
	for {
		select {
		case <-ctx.Done():
			return
		case <-outputTimer.C:
			outputResults = true
			outputTimer.Reset(pullInterval)
			break
		case <-loopTimer.C:
			globalNextPullTime = time.Now().Add(pullInterval)
			globalNextRetryTime = time.Now()
			for symbol, nextPullTime := range nextPullTimes {
				select {
				case <-ctx.Done():
					return
				default:
				}
				if time.Now().Sub(nextPullTime) < 0 {
					continue
				}
				symbolEndTime := time.Now().Truncate(klineDuration)
				symbolStartTime := symbolEndTime.Add(-klineDuration * time.Duration(barsLookback+3))
				if bars, ok := barsMap[symbol]; ok {
					symbolStartTime = bars[len(bars)-1].Timestamp
					//一分钟内说明是最新数据
					if math.Abs(time.Now().Truncate(time.Minute).Sub(symbolStartTime).Seconds()) < 60 {
						logger.Debugf("TAKER %s HAS NEWEST BAR,  FIRST TIME %v CLOSE %f LAST TIME %v CLOSE %f, CONTINUE",
							symbol,
							barsMap[symbol][0].Timestamp,
							barsMap[symbol][0].Close,
							barsMap[symbol][len(barsMap[symbol])-1].Timestamp,
							barsMap[symbol][len(barsMap[symbol])-1].Close,
						)
						globalNextPullTime = globalNextPullTime.Add(requestInterval)
						nextPullTimes[symbol] = globalNextPullTime
						continue
					}
					symbolStartTime = symbolStartTime.Add(-klineDuration * 3)
				}
				history, err := api.GetHistoryKLines(ctx, symbol, klineInterval, symbolStartTime)
				if err != nil {
					logger.Debugf("TAKER GetHistoryKlines for %s error %v", symbol, err)
					select {
					case <-ctx.Done():
						return
					default:
					}
					globalNextRetryTime = globalNextRetryTime.Add(pullRetryInterval)
					nextPullTimes[symbol] = globalNextRetryTime
					continue
				}

				if len(history) <= 1 {
					logger.Debugf("TAKER %s BAR LEN <= 1", symbol)
					globalNextRetryTime = globalNextRetryTime.Add(pullRetryInterval)
					nextPullTimes[symbol] = globalNextRetryTime
					continue
				}
				//logger.Debugf("TAKER GET %s LEN %d LAST CLOSE %f TIME %v", symbol, len(history), history[len(history)-1].Close, history[len(history)-1].Timestamp)
				if _, ok := barsMap[symbol]; !ok {
					barsMap[symbol] = history
				}
				for _, bar := range history {
					if bar.Timestamp.Sub(symbolStartTime).Seconds() <= 0 {
						continue
					}
					if bar.Timestamp.Sub(time.Now().Truncate(time.Minute)).Seconds() > 0 {
						continue
					}
					lastBar := barsMap[symbol][len(barsMap[symbol])-1]
					if bar.Timestamp.Sub(lastBar.Timestamp).Seconds() <= 0 {
						continue
					}
					bar := bar
					barsMap[symbol] = append(barsMap[symbol], bar)
				}
				if len(barsMap[symbol]) > barsLookback {
					barsMap[symbol] = barsMap[symbol][len(barsMap[symbol])-barsLookback:]
				}
				globalNextPullTime = globalNextPullTime.Add(requestInterval)
				nextPullTimes[symbol] = globalNextPullTime
				continue
			}

			if outputResults {
				allSuccess := true
				for _, symbol := range takerSymbols {
					if len(barsMap[symbol]) == 0 {
						allSuccess = false
						break
					}
				}
				if allSuccess {
					outputMap := make(common.KLinesMap)
					for symbol, bars := range barsMap {
						outputMap[symbol] = make([]common.KLine, len(bars))
						copy(outputMap[symbol], bars)
					}
					select {
					case <-ctx.Done():
						return
					case output <- outputMap:
					}
					outputResults = false
					logger.Debugf("OUTPUT TAKER BARS")
				}
			}
			loopTimer.Reset(time.Second)
			break
		}
	}
}
