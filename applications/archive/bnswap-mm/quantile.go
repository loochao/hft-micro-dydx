package main

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/tdigest"
)

func watchDeltaQuantile(
	ctx context.Context,
	openQuantile float64,
	closeQuantile float64,
	inputCh chan common.KLinesMap,
	outputCh chan map[string]Quantile,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case swapBarsMap := <-inputCh:
			quantiles := make(map[string]Quantile)
			for symbol, bars := range swapBarsMap {
				quantile, _ := tdigest.New()
				for _, bar := range bars {
					_ = quantile.Add(bar.Volume*bar.Close)
				}
				quantiles[symbol] = Quantile{
					Open:  quantile.Quantile(openQuantile),
					Close: quantile.Quantile(closeQuantile),
				}
			}
			outputCh <- quantiles
		}
	}

}
