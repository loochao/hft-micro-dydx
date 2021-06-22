package main

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/geometrybase/hft-micro/tdigest"
)

func watchDeltaQuantile(
	ctx context.Context,
	makerSymbols []string,
	makerTakerSymbolMap map[string]string,
	botQuantile float64,
	topQuantile float64,
	minimalEnterDelta,
	maximalExitDelta,
	minimalBandOffset float64,
	inputCh chan [2]common.KLinesMap,
	outputCh chan map[string]MakerTakerDeltaQuantile,
) {
	defer func() {
		logger.Debugf("EXIT watchDeltaQuantile")
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case data := <-inputCh:
			//logger.Debugf("QUANTILES UPDATING... %v",data)
			makerBarsMap := data[0]
			takerBarsMap := data[1]
			quantiles := make(map[string]MakerTakerDeltaQuantile)
			for _, makerSymbol := range makerSymbols {
				makerBars, okMaker := makerBarsMap[makerSymbol]
				takerBars, okTaker := takerBarsMap[makerTakerSymbolMap[makerSymbol]]
				if !okMaker || !okTaker {
					logger.Debugf("%s %s NOT FOUND BARS", makerSymbol, makerTakerSymbolMap[makerSymbol])
					continue
				}
				bIndex := 0
				quantile, _ := tdigest.New()
				counter := 0
				sumClose := 0.0
				for _, makerBar := range makerBars {
					for bIndex < len(takerBars)-1 && makerBar.Timestamp.Sub(takerBars[bIndex].Timestamp).Seconds() > 0 {
						bIndex++
					}
					if makerBar.Timestamp.Sub(takerBars[bIndex].Timestamp).Seconds() == 0 {
						delta := takerBars[bIndex].Close - makerBar.Close
						_ = quantile.Add(delta)
						counter++
						sumClose += makerBar.Close
					}
				}
				if counter > len(takerBars)/2 {
					maClose := sumClose / float64(counter)
					top := quantile.Quantile(topQuantile)
					bot := quantile.Quantile(botQuantile)
					mid := quantile.Quantile(0.5)

					botBand := mid - bot
					if botBand/maClose < minimalBandOffset {
						botBand = maClose * minimalBandOffset
					}
					bot = mid - botBand

					topBand := top - mid
					if topBand/maClose < minimalBandOffset {
						topBand = maClose * minimalBandOffset
					}
					top = mid + topBand

					q := MakerTakerDeltaQuantile{
						Symbol:   makerSymbol,
						ShortTop: top / maClose,
						ShortBot: (bot + botBand*0.5) / maClose,
						LongTop:  (top - topBand*0.5) / maClose,
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
					quantiles[makerSymbol] = q
				}else {
					logger.Debugf("BAR LEN %d %d MATCHED %d", len(takerBars), len(makerBars), counter)
				}
			}
			//logger.Debugf("QUANTILES UPDATED.")
			if len(quantiles) > 0 {
				//logger.Debugf("QUANTILES UPDATED.")
				outputCh <- quantiles
			}
		}
	}

}
