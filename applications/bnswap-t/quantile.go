package main

import (
	"context"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/tdigest"
	"math"
	"time"
)

func startQuantileRoutine(
	ctx context.Context,
	quantile float64,
	updateInterval time.Duration,
	tradeCh chan *bnswap.Trade,
	outputCh chan float64,
) {
	td, _ := tdigest.New()
	timer := time.NewTimer(updateInterval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			q := td.Quantile(quantile)
			if !math.IsNaN(q) {
				outputCh <- td.Quantile(quantile)
			}
			timer.Reset(updateInterval)
		case trade := <-tradeCh:
			_ = td.Add(trade.Quantity)
		}
	}
}
