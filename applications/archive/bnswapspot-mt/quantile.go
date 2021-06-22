package main

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/tdigest"
)

func watchDeltaQuantile(
	ctx context.Context,
	symbols []string,
	botQuantile float64,
	topQuantile float64,
	topScale float64,
	botScale float64,
	minimalEnterDelta,
	maximalExitDelta,
	minimalBandOffset float64,
	inputCh chan [2]common.KLinesMap,
	outputCh chan map[string]Quantile,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case data := <-inputCh:
			//logger.Debugf("QUANTILES UPDATING...")
			spotBarsMap := data[0]
			swapBarsMap := data[1]
			quantiles := make(map[string]Quantile)
			for _, symbol := range symbols {
				swapBars, okSwap := swapBarsMap[symbol]
				spotBars, okSpot := spotBarsMap[symbol]
				if !okSpot || !okSwap {
					continue
				}
				swapIndex := 0
				quantile, _ := tdigest.New()
				counter := 0
				sumClose := 0.0
				for _, spotBar := range spotBars {
					for swapIndex < len(swapBars)-1 && spotBar.Timestamp.Sub(swapBars[swapIndex].Timestamp).Seconds() > 0 {
						swapIndex++
					}
					if spotBar.Timestamp.Sub(swapBars[swapIndex].Timestamp).Seconds() == 0 {
						delta := swapBars[swapIndex].Close - spotBar.Close
						_ = quantile.Add(delta)
						counter++
						sumClose += spotBar.Close
					}
				}
				if counter > len(swapBars)/2 {
					maClose := sumClose / float64(counter)
					top := quantile.Quantile(topQuantile)
					bot := quantile.Quantile(botQuantile)
					mid := quantile.Quantile(0.5)

					botBand := mid - bot
					if botBand/maClose < minimalBandOffset {
						botBand = maClose * minimalBandOffset
					}
					bot = mid - botScale*botBand

					topBand := top - mid
					if topBand/maClose < minimalBandOffset {
						topBand = maClose * minimalBandOffset
					}
					top = mid + topScale*topBand

					q := Quantile{
						Symbol:  symbol,
						Top:     top / maClose,
						Bot:     bot / maClose,
						Mid:     mid / maClose,
						MaClose: maClose,
					}
					if q.Top < minimalEnterDelta {
						q.Top = minimalEnterDelta
					}
					if q.Bot > maximalExitDelta {
						q.Bot = maximalExitDelta
					}
					q.FarBot = (q.Bot - q.Mid)*2 + q.Mid
					q.FarTop = (q.Top - q.Mid)*2 + q.Mid
					quantiles[symbol] = q
					//logger.Debugf("%s QUANTILE DONE", symbol)
				}
			}
			if len(quantiles) > 0 {
				//logger.Debugf("QUANTILES UPDATED.")
				outputCh <- quantiles
			}
		}
	}

}
