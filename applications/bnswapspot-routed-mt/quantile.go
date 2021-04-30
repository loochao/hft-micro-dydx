package main

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/geometrybase/hft-micro/tdigest"
)

func watchDeltaQuantile(
	ctx context.Context,
	symbols []string,
	botQuantile float64,
	topQuantile float64,
	minimalEnterDelta,
	maximalExitDelta,
	minimalBandOffset float64,
	frAvgCh chan float64,
	inputCh chan [2]common.KLinesMap,
	outputCh chan map[string]Quantile,
) {
	var bandScale *float64
	var bearScale = 0.618
	var normalScale = 1.0
	var bullScale = 1.0 - 0.618
	var crazyScale = 2.0
	var topOffset = 0.0
	var botOffset = 0.0
	var meanFr = 0.0
	for {
		select {
		case <-ctx.Done():
			return
		case meanFr = <-frAvgCh:
			if meanFr < 0.0000 {
				bandScale = &bearScale
				topOffset = -0.00222
				botOffset = -0.00222
				//logger.Debugf("FR SUM %f BEAR BAND SCALE %f", meanFr, *bandScale)
			} else if meanFr < 0.0003 {
				topOffset = -0.00111
				botOffset = -0.00055
				bandScale = &normalScale
				//logger.Debugf("FR SUM %f NORM BAND SCALE %f", meanFr, *bandScale)
			} else if meanFr < 0.0005 {
				topOffset = 0.0
				botOffset = 0.0
				bandScale = &bullScale
				//logger.Debugf("FR SUM %f BULL BAND SCALE %f", meanFr, *bandScale)
			} else {
				bandScale = &crazyScale
				topOffset = 0.00222
				botOffset = 0.00055
				//logger.Debugf("FR SUM %f CRAZY BAND SCALE %f", meanFr, *bandScale)
			}
			break
		case data := <-inputCh:
			if bandScale == nil {
				continue
			}
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
					bot = mid - botBand

					topBand := top - mid
					if topBand/maClose < minimalBandOffset {
						topBand = maClose * minimalBandOffset
					}
					top = mid + *bandScale*topBand

					q := Quantile{
						Symbol:      symbol,
						Top:         top / maClose,
						Bot:         bot / maClose,
						Mid:         mid / maClose,
						MeanFr:      meanFr,
						OriginalTop: quantile.Quantile(topQuantile) / maClose,
						OriginalBot: quantile.Quantile(botQuantile) / maClose,
						MaClose:     maClose,
					}
					if q.Top < minimalEnterDelta+topOffset {
						q.Top = minimalEnterDelta + topOffset
					}
					if q.Bot > maximalExitDelta+botOffset {
						q.Bot = maximalExitDelta + topOffset
					}
					quantiles[symbol] = q
				}
			}
			if len(quantiles) > 0 {
				select {
				case outputCh <- quantiles:
				default:
					logger.Debugf("outputCh <- quantiles failed ch len %d", len(outputCh))
				}
			}
		}
	}

}
