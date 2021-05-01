package main

import (
	"context"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func StreamWalkedDepth(
	ctx context.Context,
	symbol string,
	timeDecay, timeBias,
	makerImpact, takerImpact float64,
	reportCount int,
	rawDepthCh chan *common.DepthRawMessage,
	reportCh chan DepthReport,
	outputCh chan common.WalkedMakerTakerDepth,
) {
	var err error
	var takerRawDepth *common.DepthRawMessage
	var takerDepth, newTakerDepth *bnswap.Depth20
	var takerWalkedDepth *common.WalkedMakerTakerDepth
	var takerDepthFilter = common.NewDepthFilter(timeDecay, timeBias)

	logSilentTime := time.Now()
	takerWalkDepthTimer := time.NewTimer(time.Hour * 999)
	takerParseTimer := time.NewTimer(time.Hour * 999)
	defer takerWalkDepthTimer.Stop()
	defer takerParseTimer.Stop()
	depthCount := 0

	expectedChanSendingTime := time.Nanosecond * 300
	for {
		select {
		case <-ctx.Done():
			return
		case <-takerWalkDepthTimer.C:
			if takerDepth != nil {
				takerWalkedDepth, err = common.WalkMakerTakerDepth20(takerDepth, makerImpact, takerImpact)
				if err != nil {
					if time.Now().Sub(logSilentTime) > 0 {
						if takerRawDepth == nil {
							logger.Debugf("common.WalkMakerTakerDepth20 error %v %s", err, symbol)
						} else {
							logger.Debugf("common.WalkMakerTakerDepth20 error %v %s %s", err, symbol, takerRawDepth.Depth)
						}
						logSilentTime = time.Now().Add(time.Minute)
					}
					break
				}
				select {
				case <-ctx.Done():
				case outputCh <- *takerWalkedDepth:
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("outputCh <- *takerWalkedDepth %s len(outputCh) %d", symbol, len(outputCh))
						logSilentTime = time.Now().Add(time.Minute)
					}
				}
			}
			break

		case <-takerParseTimer.C:
			if takerRawDepth == nil {
				break
			}
			newTakerDepth, err = bnswap.ParseDepth20(takerRawDepth.Depth)
			if err != nil {
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("bnswap.ParseDepth20 error %v %s %s", err, symbol, takerRawDepth.Depth)
					logSilentTime = time.Now().Add(time.Minute)
				}
			} else if takerDepth == nil || newTakerDepth.LastUpdateId > takerDepth.LastUpdateId {
				takerDepth = newTakerDepth
				takerWalkDepthTimer.Reset(time.Nanosecond)
			}
			break

		case takerRawDepth = <-rawDepthCh:
			if !takerDepthFilter.Filter(takerRawDepth) {
				takerParseTimer.Reset(expectedChanSendingTime)
				depthCount++
				if depthCount > reportCount {
					takerDepthFilter.GenerateReport()
					select {
					case reportCh <- DepthReport{
						Symbol:       symbol,
						MsgAvgLen:    takerDepthFilter.Report.MsgAvgLen,
						TimeDeltaEma: takerDepthFilter.TimeDeltaEma,
						TimeDelta:    takerDepthFilter.TimeDelta,
						FilterRatio:  takerDepthFilter.Report.FilterRatio,
					}:
					default:
					}
					depthCount = 0
				}
			}
			break
		}
	}
}

