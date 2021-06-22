package main

import (
	"context"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/geometrybase/hft-micro/tdigest"
	"math"
	"time"
)

func watchHighLowQuantile(
	ctx context.Context,
	api *bnswap.API,
	symbols []string,
	barsLookback int,
	pullInterval time.Duration,
	pullRetryInterval time.Duration,
	requestInterval time.Duration,
	quantileOffset float64,
	dirWindow int,
	output chan HighLowQuantile,
) {
	loopTimer := time.NewTimer(time.Second)
	defer loopTimer.Stop()
	outputTimer := time.NewTimer(time.Second)
	defer outputTimer.Stop()
	barsMap := make(common.KLinesMap)
	nextPullTimes := make(map[string]time.Time)
	for i, takerSymbol := range symbols {
		nextPullTimes[takerSymbol] = time.Now().Add(requestInterval * time.Duration(i))
	}
	klineInterval := "1m"
	klineDuration := time.Minute
	globalNextPullTime := time.Now()
	globalNextRetryTime := time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		case <-outputTimer.C:
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
				td, _ := tdigest.New()
				for _, bar := range barsMap[symbol] {
					_ = td.Add(bar.Open - bar.Low)
				}
				dir := 0.0
				if len(barsMap[symbol]) > dirWindow {
					if barsMap[symbol][len(barsMap[symbol])-1].Close > barsMap[symbol][len(barsMap[symbol])-dirWindow].Close {
						dir = 1
					} else {
						dir = -1
					}
				}
				select {
				case <-ctx.Done():
					return
				case output <- HighLowQuantile{
					Symbol: symbol,
					Top:    td.Quantile(0.5 + quantileOffset),
					Mid:    td.Quantile(0.5),
					Bot:    td.Quantile(0.5 - quantileOffset),
					Dir:    dir,
				}:
				}
				continue
			}
			loopTimer.Reset(time.Second)
			break
		}
	}
}
