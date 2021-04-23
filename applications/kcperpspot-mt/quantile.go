package main

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/tdigest"
)

func deltaQuantileLoop(
	ctx context.Context,
	spotSymbols []string,
	spSymbolMap map[string]string,
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
			perpBarsMap := data[1]
			quantiles := make(map[string]Quantile)
			for _, symbol := range spotSymbols {
				perpBars, okPerp := perpBarsMap[spSymbolMap[symbol]]
				spotBars, okSpot := spotBarsMap[symbol]
				if !okSpot || !okPerp {
					continue
				}
				perpIndex := 0
				quantile, _ := tdigest.New()
				counter := 0
				sumClose := 0.0
				for _, spotBar := range spotBars {
					for perpIndex < len(perpBars)-1 && spotBar.Timestamp.Sub(perpBars[perpIndex].Timestamp).Seconds() > 0 {
						perpIndex++
					}
					if spotBar.Timestamp.Sub(perpBars[perpIndex].Timestamp).Seconds() == 0 {
						delta := perpBars[perpIndex].Close - spotBar.Close
						_ = quantile.Add(delta)
						counter++
						sumClose += spotBar.Close
					}
				}
				if counter > len(perpBars)/2 {
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
					quantiles[symbol] = q
				}
			}
			if len(quantiles) > 0 {
				//logger.Debugf("QUANTILES UPDATED.")
				outputCh <- quantiles
			}
		}
	}

}
