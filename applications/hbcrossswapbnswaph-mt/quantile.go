package main

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/geometrybase/hft-micro/tdigest"
)

func watchDeltaQuantile(
	ctx context.Context,
	hSymbols []string,
	hbSymbolMap map[string]string,
	botQuantile float64,
	topQuantile float64,
	topScale float64,
	botScale float64,
	minimalEnterDelta,
	maximalExitDelta,
	minimalBandOffset float64,
	inputCh chan [2]common.KLinesMap,
	outputCh chan map[string]HBDeltaQuantile,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case data := <-inputCh:
			logger.Debugf("QUANTILES UPDATING...")
			hBarsMap := data[0]
			bBarsMap := data[1]
			quantiles := make(map[string]HBDeltaQuantile)
			for _, hSymbol := range hSymbols {
				hBars, okH := hBarsMap[hSymbol]
				bBars, okB := bBarsMap[hbSymbolMap[hSymbol]]
				if !okH || !okB {
					continue
				}
				bIndex := 0
				quantile, _ := tdigest.New()
				counter := 0
				sumClose := 0.0
				for _, hBar := range hBars {
					for bIndex < len(bBars)-1 && hBar.Timestamp.Sub(bBars[bIndex].Timestamp).Seconds() > 0 {
						bIndex++
					}
					if hBar.Timestamp.Sub(bBars[bIndex].Timestamp).Seconds() == 0 {
						delta := bBars[bIndex].Close - hBar.Close
						_ = quantile.Add(delta)
						counter++
						sumClose += hBar.Close
					}
				}
				if counter > len(bBars)/2 {
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

					q := HBDeltaQuantile{
						HSymbol:  hSymbol,
						ShortTop: top / maClose,
						ShortBot: bot / maClose,
						LongTop:  top / maClose,
						LongBot:  bot / maClose,
						Mid:      mid / maClose,
						MaClose:  maClose,
					}
					if q.ShortTop < minimalEnterDelta {
						q.ShortTop = minimalEnterDelta
					}
					if q.ShortBot > maximalExitDelta {
						q.ShortBot = maximalExitDelta
					}
					if q.LongTop < -maximalExitDelta {
						q.LongTop = -maximalExitDelta
					}
					if q.LongBot > -minimalEnterDelta {
						q.LongBot = -minimalEnterDelta
					}
					quantiles[hSymbol] = q
				}
			}
			if len(quantiles) > 0 {
				logger.Debugf("QUANTILES UPDATED.")
				outputCh <- quantiles
			}
		}
	}

}
