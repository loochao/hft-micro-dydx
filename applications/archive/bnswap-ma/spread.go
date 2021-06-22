package main

import (
	"context"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func watchDepth(
	ctx context.Context,
	takerSymbol string,
	takerImpact float64,
	takerDecay, takerBias float64,
	reportCount int,
	takerDepthCh chan *common.DepthRawMessage,
	reportCh chan common.SpreadReport,
	outputCh chan *common.WalkedTakerDepth,
) {
	var err error
	var  takerRawDepth *common.DepthRawMessage
	var takerDepth, newTakerDepth *bnswap.Depth5
	var  takerWalkedDepth *common.WalkedTakerDepth
	var takerDepthFilter = common.NewDepthFilter(takerDecay, takerBias)

	logSilentTime := time.Now()
	takerWalkDepthTimer := time.NewTimer(time.Hour * 999)
	depthCount := 0
	takerParseTimer := time.NewTimer(time.Hour * 999)

	expectedChanSendingTime := time.Nanosecond * 300
	for {
		select {
		case <-ctx.Done():
			return

		case <-takerWalkDepthTimer.C:
			if takerDepth != nil {
				takerWalkedDepth, err = common.WalkTakerDepth5(takerDepth, takerImpact)
				if err != nil {
					if time.Now().Sub(logSilentTime) > 0 {
						if takerRawDepth == nil {
							logger.Debugf("taker common.WalkMakerTakerDepth5 error %v %s", err, takerSymbol)
						} else {
							logger.Debugf("taker common.WalkMakerTakerDepth5 error %v %s %s", err, takerSymbol, takerRawDepth.Depth)
						}
						logSilentTime = time.Now().Add(time.Minute)
					}
					break
				}
				select {
				case outputCh <- takerWalkedDepth:
				default:
					logger.Debugf("outputCh <- takerWalkedDepth failed ch len %d", len(outputCh))
				}
			}
			break

		case <-takerParseTimer.C:
			if takerRawDepth == nil {
				break
			}
			newTakerDepth, err = bnswap.ParseDepth5(takerRawDepth.Depth)
			if err != nil {
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("bnswap.ParseDepth5 error %v %s %s", err, takerSymbol, takerRawDepth.Depth)
					logSilentTime = time.Now().Add(time.Minute)
				}
			} else if takerDepth == nil || newTakerDepth.LastUpdateId > takerDepth.LastUpdateId {
				takerDepth = newTakerDepth
				takerWalkDepthTimer.Reset(time.Nanosecond)
			}
			break

		case takerRawDepth = <-takerDepthCh:
			if !takerDepthFilter.Filter(takerRawDepth){
				takerParseTimer.Reset(expectedChanSendingTime)
				depthCount++
				if depthCount > reportCount {
					takerDepthFilter.GenerateReport()
					select {
					case reportCh <- common.SpreadReport{
						TakerSymbol:           takerSymbol,
						TakerMsgAvgLen:        takerDepthFilter.Report.MsgAvgLen,
						TakerTimeDeltaEma:     takerDepthFilter.TimeDeltaEma,
						TakerTimeDelta:        takerDepthFilter.TimeDelta,
						TakerDepthFilterRatio: takerDepthFilter.Report.FilterRatio,
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
