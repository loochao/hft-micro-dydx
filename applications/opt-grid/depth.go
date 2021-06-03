package main

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func walkMakerDepth(
	ctx context.Context,
	makerSymbol string,
	makerImpact, takerImpact float64,
	makerDecay float64,
	makerBias time.Duration,
	minTimeDelta, maxTimeDelta time.Duration,
	makerDepthCh chan common.Depth,
	outputCh chan *common.WalkedMakerTakerDepth,
) {
	var err error
	var makerDepth, newMakerDepth common.Depth
	var makerWalkedDepth *common.WalkedMakerTakerDepth
	var makerBiasInMs = float64(makerBias / time.Millisecond)
	var minTimeDeltaInMs = float64(minTimeDelta / time.Millisecond)
	var maxTimeDeltaInMs = float64(maxTimeDelta / time.Millisecond)
	var makerDepthFilter = common.NewDepthFilter(makerDecay, makerBiasInMs, minTimeDeltaInMs, maxTimeDeltaInMs)
	logSilentTime := time.Now()
	makerWalkDepthTimer := time.NewTimer(time.Hour * 999)
	expectedChanSendingTime := time.Nanosecond * 300
	for {
		select {
		case <-ctx.Done():
			return
		case <-makerWalkDepthTimer.C:
			if makerDepth != nil {
				makerWalkedDepth, err = common.WalkMakerTakerDepth20(makerDepth, makerImpact, takerImpact)
				if err != nil {
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("maker common.WalkMakerTakerDepth20 error %v %s", err, makerSymbol)
						logSilentTime = time.Now().Add(time.Minute)
					}
					break
				}
				select {
				case outputCh <- makerWalkedDepth:
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("outputCh <- makerWalkedDepth failed, ch len %s %d", makerSymbol, len(outputCh))
						logSilentTime = time.Now().Add(time.Minute)
					}
				}
			}
			break
		case newMakerDepth = <-makerDepthCh:
			if makerDepth != nil && makerDepth.GetTime().Sub(newMakerDepth.GetTime()) >= 0 {
				break
			}
			makerDepth = newMakerDepth
			if !makerDepthFilter.Filter(makerDepth) {
				makerWalkDepthTimer.Reset(expectedChanSendingTime)
			}
			break
		}
	}
}
