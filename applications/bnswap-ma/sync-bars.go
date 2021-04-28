package main

import (
	"context"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/geometrybase/hft-micro/talib"
	"time"
)

func watchSignal(
	ctx context.Context,
	api *bnswap.API,
	symbol string,
	periodFast, periodSlow int,
	wsBarCh chan common.KLine,
	output chan Signal,
) {
	klineInterval := "1m"
	klineDuration := time.Minute
	pullTimer := time.NewTimer(time.Second)
	var bars []common.KLine
	var err error
	var closes []float64
	var i int
	var fast, slow []float64
	for {
		select {
		case <-ctx.Done():
			return
		case bar := <-wsBarCh:
			if bars != nil && len(bars) > 0 && bar.Timestamp.Sub(bars[len(bars)-1].Timestamp) > 0 {
				bars = append(bars, bar)
				if len(bars) >= periodFast+periodSlow {
					bars = bars[len(bars)-periodFast-periodSlow:]
					closes = make([]float64, len(bars))
					for i, bar = range bars {
						closes[i] = bar.Close
					}
					fast = talib.Ma(closes, periodFast, talib.SMA)
					slow = talib.Ma(closes, periodSlow, talib.SMA)
					if fast[len(fast)-1] > slow[len(slow)-1] && fast[len(fast)-2] <= slow[len(slow)-2] {
						select {
						case output <- Signal{
							Symbol:    symbol,
							Direction: 1,
							Fast:      fast[len(fast)-1],
							Slow:      slow[len(slow)-1],
							Close:     closes[len(closes)-1],
						}:
						default:
						}
					} else if fast[len(fast)-1] < slow[len(slow)-1] && fast[len(fast)-2] >= slow[len(slow)-2] {
						select {
						case output <- Signal{
							Symbol:    symbol,
							Direction: -1,
							Fast:      fast[len(fast)-1],
							Slow:      slow[len(slow)-1],
							Close:     closes[len(closes)-1],
						}:
						default:
						}
					} else {
						select {
						case output <- Signal{
							Symbol:    symbol,
							Direction: 0,
							Fast:      fast[len(fast)-1],
							Slow:      slow[len(slow)-1],
							Close:     closes[len(closes)-1],
						}:
						default:
						}
					}
				}
			}
		case <-pullTimer.C:
			symbolEndTime := time.Now().Truncate(klineDuration)
			symbolStartTime := symbolEndTime.Add(-klineDuration * time.Duration(periodSlow+periodFast+3))
			bars, err = api.GetHistoryKLines(ctx, symbol, klineInterval, symbolStartTime)
			if err != nil {
				logger.Debugf("TAKER GetHistoryKlines for %s error %v", symbol, err)
				pullTimer.Reset(time.Minute)
				continue
			}
			pullTimer.Reset(time.Hour)
		}
	}
}
