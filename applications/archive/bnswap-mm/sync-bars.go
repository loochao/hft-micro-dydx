package main

import (
	"context"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"time"
)

func watchSwapBars(
	ctx context.Context,
	api *bnswap.API,
	symbols []string,
	barsLookback int,
	pullInterval time.Duration,
	pullRetryInterval time.Duration,
	output chan common.KLinesMap,
) {
	loopTimer := time.NewTimer(time.Second)
	defer loopTimer.Stop()
	barsMap := make(common.KLinesMap)
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
					symbolEndTime := time.Now().Truncate(time.Minute * 15).Add(time.Minute * 15)
					symbolStartTime := symbolEndTime.Add(-time.Minute * time.Duration(barsLookback+1))
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
						symbolStartTime = symbolStartTime.Add(-time.Minute *  3)
					}
					history, err = api.GetHistoryKLines(ctx, symbol, "1m", symbolStartTime)
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
				outputMap := make(common.KLinesMap)
				length := len(barsMap[symbols[0]])
				for symbol, bars := range barsMap {
					if len(bars) != length {
						logger.Fatalf("SWAP %s LENGTH %d NOT EQUAL TO %d", symbol, len(bars), length)
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
			//	"PULL SWAP BARS IN %v",
			//	time.Now().Add(pullInterval/2).Truncate(pullInterval).Add(pullInterval).Sub(time.Now()),
			//)

			loopTimer.Reset(
				time.Now().Add(pullInterval / 2).Truncate(pullInterval).Add(pullInterval).Sub(time.Now()),
			)
		}
	}
}

