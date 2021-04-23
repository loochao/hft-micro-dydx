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

func perpBarsPullingLoop(
	ctx context.Context,
	api *kcperp.API,
	symbols []string,
	barsLookback int,
	pullInterval time.Duration,
	pullRetryInterval time.Duration,
	output chan common.KLinesMap,
) {
	loopTimer := time.NewTimer(time.Second)
	defer loopTimer.Stop()
	barsMap := make(common.KLinesMap)
	klineDuration := kcperp.GranularityDurations[kcperp.Granularity30Min]
	klineGranularity := kcperp.Granularity30Min
	for {
		select {
		case <-ctx.Done():
			return
		case <-loopTimer.C:
			allSuccess := true
		symbolLoop:
			for _, symbol := range symbols {
				var history []common.KLine
				var err error
				retryCount := 10
				for retryCount > 0 {
					symbolEndTime := time.Now().Truncate(klineDuration).Add(-klineDuration)
					symbolStartTime := symbolEndTime.Add(-klineDuration * time.Duration(barsLookback+3))
					if bars, ok := barsMap[symbol]; ok {
						symbolStartTime = bars[len(bars)-1].Timestamp
						//一分钟内说明是最新数据
						if math.Abs(time.Now().Truncate(klineDuration).Sub(symbolStartTime).Seconds()) < 0 {
							logger.Debugf("PERP %s HAS NEWEST BAR,  FIRST TIME %v CLOSE %f LAST TIME %v CLOSE %f, CONTINUE",
								symbol,
								barsMap[symbol][0].Timestamp,
								barsMap[symbol][0].Close,
								barsMap[symbol][len(barsMap[symbol])-1].Timestamp,
								barsMap[symbol][len(barsMap[symbol])-1].Close,
							)
							continue symbolLoop
						}
						symbolStartTime = symbolStartTime.Add(-klineDuration * 3)
					}
					history, err = api.GetKlines(
						ctx, kcperp.KlinesParam{
							Symbol:      symbol,
							From:        symbolStartTime.Unix() * 1000,
							To:          symbolEndTime.Unix() * 1000,
							Granularity: klineGranularity,
						})
					if err != nil {
						logger.Debugf("PERP GetKlines for %s error %v", symbol, err)
						retryCount--
						time.Sleep(pullRetryInterval)
						continue
					}

					// 假定第一根是最新的不完整BAR
					if len(history) <= 1 {
						logger.Debugf("PERP %s BAR LEN <= 1 %d %v %v", symbol, barsLookback, symbolStartTime, symbolEndTime)
						retryCount--
						continue
					}
					//logger.Debugf("PERP GET %s LEN %d LAST CLOSE %f TIME %v", symbol, len(history), history[len(history)-1].Close, history[len(history)-1].Timestamp)
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
					//logger.Debugf(
					//	"PERP %s FIRST TIME %v CLOSE %f LAST TIME %v CLOSE %f",
					//	symbol,
					//	barsMap[symbol][0].Timestamp,
					//	barsMap[symbol][0].Close,
					//	barsMap[symbol][len(barsMap[symbol])-1].Timestamp,
					//	barsMap[symbol][len(barsMap[symbol])-1].Close,
					//)
					break
				}
				if retryCount == 0 {
					allSuccess = false
					break
				}
			}
			if allSuccess {
				outputMap := make(common.KLinesMap)
				length := len(barsMap[symbols[0]])
				for symbol, bars := range barsMap {
					if len(bars) != length {
						logger.Fatalf("PERP %s LENGTH %d NOT EQUAL TO %d", symbol, len(bars), length)
					}
				}
				for symbol, bars := range barsMap {
					outputMap[symbol] = make([]common.KLine, len(bars))
					copy(outputMap[symbol], bars)
				}
				if allSuccess {
					output <- outputMap
				}
			}
			//logger.Debugf(
			//	"PULL PERP BARS IN %v",
			//	time.Now().Add(pullInterval/2).Truncate(pullInterval).Add(pullInterval).Sub(time.Now()),
			//)

			loopTimer.Reset(
				time.Now().Add(pullInterval / 2).Truncate(pullInterval).Add(pullInterval).Sub(time.Now()),
			)
		}
	}
}

func spotBarsPullingLoop(
	ctx context.Context,
	api *kcspot.API,
	symbols []string,
	barsLookback int,
	pullInterval time.Duration,
	pullRetryInterval time.Duration,
	output chan common.KLinesMap,
) {
	loopTimer := time.NewTimer(time.Second)
	defer loopTimer.Stop()
	barsMap := make(common.KLinesMap)
	candleDuration := kcspot.CandleTypeDurations[kcspot.CandleType30Min]
	candleType := kcspot.CandleType30Min
	for {
		select {
		case <-ctx.Done():
			return
		case <-loopTimer.C:
			allSuccess := true
		symbolLoop:
			for _, symbol := range symbols {
				var history []common.KLine
				var err error
				retryCount := 10
				for retryCount > 0 {
					symbolEndTime := time.Now().Truncate(candleDuration).Add(-candleDuration)
					symbolStartTime := symbolEndTime.Add(-candleDuration * time.Duration(barsLookback+3))
					if bars, ok := barsMap[symbol]; ok {
						symbolStartTime = bars[len(bars)-1].Timestamp
						//一分钟内说明是最新数据
						if math.Abs(time.Now().Truncate(time.Minute).Sub(symbolStartTime).Seconds()) < 60 {
							logger.Debugf("SPOT %s HAS NEWEST BAR,  FIRST TIME %v CLOSE %f LAST TIME %v CLOSE %f, CONTINUE",
								symbol,
								barsMap[symbol][0].Timestamp,
								barsMap[symbol][0].Close,
								barsMap[symbol][len(barsMap[symbol])-1].Timestamp,
								barsMap[symbol][len(barsMap[symbol])-1].Close,
							)
							continue symbolLoop
						}
						symbolStartTime = symbolStartTime.Add(-candleDuration * 3)
					}
					history, err = api.GetCandles(ctx, kcspot.CandlesParam{
						Symbol:  symbol,
						StartAt: symbolStartTime.Unix(),
						EndAt:   symbolEndTime.Unix(),
						Type:    candleType,
					})
					if err != nil {
						logger.Debugf("SPOT GetHistoryKlines for %s error %v", symbol, err)
						retryCount--
						time.Sleep(pullRetryInterval)
						continue
					}
					// 假定第一根是最新的不完整BAR
					if len(history) <= 1 {
						logger.Debugf("SPOT %s BAR LEN <= 1", symbol)
						retryCount--
						continue
					}
					//logger.Debugf("SPOT GET %s LEN %d LAST CLOSE %f TIME %v", symbol, len(history), history[len(history)-1].Close, history[len(history)-1].Timestamp)
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
					//logger.Debugf(
					//	"SPOT %s FIRST TIME %v CLOSE %f LAST TIME %v CLOSE %f",
					//	symbol,
					//	barsMap[symbol][0].Timestamp,
					//	barsMap[symbol][0].Close,
					//	barsMap[symbol][len(barsMap[symbol])-1].Timestamp,
					//	barsMap[symbol][len(barsMap[symbol])-1].Close,
					//)
					break
				}
				if retryCount == 0 {
					allSuccess = false
					break
				}
			}
			if allSuccess {
				outputMap := make(common.KLinesMap)
				length := len(barsMap[symbols[0]])
				for symbol, bars := range barsMap {
					if len(bars) != length {
						logger.Fatalf("SPOT %s LENGTH %d NOT EQUAL TO %d", symbol, len(bars), length)
					}
				}
				for symbol, bars := range barsMap {
					outputMap[symbol] = make([]common.KLine, len(bars))
					copy(outputMap[symbol], bars)
				}
				if allSuccess {
					output <- outputMap
				}
			}
			logger.Debugf(
				"PULL SPOT BARS IN %v",
				time.Now().Add(pullInterval/2).Truncate(pullInterval).Add(pullInterval).Sub(time.Now()),
			)
			loopTimer.Reset(
				time.Now().Add(pullInterval / 2).Truncate(pullInterval).Add(pullInterval).Sub(time.Now()),
			)
		}
	}
}
