package main

import (
	"context"
	"github.com/geometrybase/hft/bnswap"
	"github.com/geometrybase/hft/common"
	"github.com/geometrybase/hft/logger"
	"github.com/geometrybase/hft/okspot"
	"math"
	"time"
)

func watchBnswapBars(
	ctx context.Context,
	api *bnswap.API,
	symbols []string,
	barsLookback int,
	pullInterval time.Duration,
	pullRetryInterval time.Duration,
	output chan common.OhlcvsMap,
) {
	loopTimer := time.NewTimer(time.Second)
	defer loopTimer.Stop()
	barsMap := make(common.OhlcvsMap)
	for {
		select {
		case <-ctx.Done():
			return
		case <-loopTimer.C:
			allSuccess := true
		symbolLoop:
			for _, symbol := range symbols {
				var history []common.OHLCV
				var err error
				retryCount := 10
				for retryCount > 0 {
					symbolEndTime := time.Now().Truncate(time.Minute * 15).Add(time.Minute * 15)
					symbolStartTime := symbolEndTime.Add(-time.Minute * 3 * time.Duration(barsLookback+30))
					if bars, ok := barsMap[symbol]; ok {
						symbolStartTime = bars[len(bars)-1].Timestamp
						//一分钟内说明是最新数据
						if math.Abs(time.Now().Truncate(time.Minute).Sub(symbolStartTime).Seconds()) < 60 {
							logger.Debugf("SWAP %s HAS NEWEST BAR,  FIRST TIME %v CLOSE %f LAST TIME %v CLOSE %f, CONTINUE",
								symbol,
								barsMap[symbol][0].Timestamp,
								barsMap[symbol][0].Close,
								barsMap[symbol][len(barsMap[symbol])-1].Timestamp,
								barsMap[symbol][len(barsMap[symbol])-1].Close,
							)
							continue symbolLoop
						}
						symbolStartTime = symbolStartTime.Add(-time.Minute * 3 * 30)
					}
					history, err = api.GetHistoryKlines(ctx, symbol, "3m", symbolStartTime)
					if err != nil {
						logger.Debugf("SWAP GetHistoryKlines for %s error %v", symbol, err)
						retryCount--
						time.Sleep(pullRetryInterval)
						continue
					}

					// 假定第一根是最新的不完整BAR
					if len(history) <= 1 {
						logger.Debugf("SWAP %s BAR LEN <= 1", symbol)
						retryCount--
						continue
					}
					//logger.Debugf("SWAP GET %s LEN %d LAST CLOSE %f TIME %v", symbol, len(history), history[len(history)-1].Close, history[len(history)-1].Timestamp)
					if _, ok := barsMap[symbol]; !ok {
						barsMap[symbol] = history
					}
					for _, bar := range history {
						if bar.Timestamp.Sub(symbolStartTime).Seconds() <= 0 {
							continue
						}
						if bar.Timestamp.Sub(symbolEndTime).Seconds() > 0 {
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
					//	"SWAP %s FIRST TIME %v CLOSE %f LAST TIME %v CLOSE %f",
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
				outputMap := make(common.OhlcvsMap)
				length := len(barsMap[symbols[0]])
				for symbol, bars := range barsMap {
					if len(bars) != length {
						logger.Fatalf("SWAP %s LENGTH %d NOT EQUAL TO %d", symbol, len(bars), length)
					}
				}
				for symbol, bars := range barsMap {
					outputMap[symbol] = make([]common.OHLCV, len(bars))
					copy(outputMap[symbol], bars)
				}
				if allSuccess {
					output <- outputMap
				}
			}
			logger.Debugf(
				"PULL SWAP BARS IN %v",
				time.Now().Add(pullInterval/2).Truncate(pullInterval).Add(pullInterval).Sub(time.Now()),
			)

			loopTimer.Reset(
				time.Now().Add(pullInterval / 2).Truncate(pullInterval).Add(pullInterval).Sub(time.Now()),
			)
		}
	}
}

func watchOkspotBars(
	ctx context.Context,
	api *okspot.API,
	symbols []string,
	barsLookback int,
	pullInterval time.Duration,
	pullRetryInterval time.Duration,
	output chan common.OhlcvsMap,
) {
	loopTimer := time.NewTimer(time.Second)
	defer loopTimer.Stop()
	barsMap := make(common.OhlcvsMap)
	for {
		select {
		case <-ctx.Done():
			return
		case <-loopTimer.C:
			allSuccess := true
		symbolLoop:
			for _, symbol := range symbols {
				var history []common.OHLCV
				var err error
				retryCount := 10
				for retryCount > 0 {
					symbolEndTime := time.Now().Truncate(time.Minute * 15).Add(-time.Minute * 15)
					symbolStartTime := symbolEndTime.Add(-time.Minute * 3 * time.Duration(barsLookback+3))
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
						symbolStartTime = symbolStartTime.Add(-time.Minute * 3 * 3)
					}
					params := okspot.MarketDataParams{
						InstrumentId: okspot.SymbolToInstrumentId(symbol),
						End:          &symbolEndTime,
						Start:        &symbolStartTime,
						Granularity:  180,
					}
					history, err = api.GetRecentCandles(ctx, params)
					if err != nil {
						logger.Debugf("SPOT GetRecentCandles for %s error %v", symbol, err)
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
						if bar.Timestamp.Sub(symbolEndTime).Seconds() > 0 {
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
					//	"OK SPOT %s (%d) FIRST TIME %v CLOSE %f LAST TIME %v CLOSE %f",
					//	symbol,
					//	len(barsMap[symbol]),
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
				outputMap := make(common.OhlcvsMap)
				length := len(barsMap[symbols[0]])
				for symbol, bars := range barsMap {
					if len(bars) != length {
						logger.Fatalf("SPOT %s LENGTH %d NOT EQUAL TO %d", symbol, len(bars), length)
					}
				}
				for symbol, bars := range barsMap {
					outputMap[symbol] = make([]common.OHLCV, len(bars))
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
