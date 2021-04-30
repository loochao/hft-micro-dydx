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
	for {
		select {
		case <-ctx.Done():
			return
		case frSum := <-frAvgCh:
			if frSum < 0.0000 {
				bandScale = &bearScale
				logger.Debugf("FR SUM %f BEAR BAND SCALE %f", frSum, *bandScale)
			} else if frSum < 0.0003 {
				bandScale = &normalScale
				logger.Debugf("FR SUM %f NORM BAND SCALE %f", frSum, *bandScale)
			} else if frSum < 0.000618 {
				bandScale = &bullScale
				logger.Debugf("FR SUM %f BULL BAND SCALE %f", frSum, *bandScale)
			} else {
				bandScale = &crazyScale
				logger.Debugf("FR SUM %f CRAZY BAND SCALE %f", frSum, *bandScale)
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
						OriginalTop: quantile.Quantile(topQuantile) / maClose,
						OriginalBot: quantile.Quantile(botQuantile) / maClose,
						MaClose:     maClose,
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
				select {
				case outputCh <- quantiles:
				default:
					logger.Debugf("outputCh <- quantiles failed ch len %d", len(outputCh))
				}
			}
		}
	}

}
