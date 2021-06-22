package main

import (
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/bnspot"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func bnspotDepthWalkingLoop(
	ctx context.Context,
	symbol string,
	levelDecay,
	timeDecay, timeBias float64,
	lookbackDuration time.Duration,
	reportCount int,
	rawDepthCh chan *common.DepthRawMessage,
	reportCh chan DepthReport,
	outputCh chan WalkedDepth20,
) {
	var err error
	var takerRawDepth *common.DepthRawMessage
	var takerDepth, newTakerDepth *bnspot.Depth20
	var takerWalkedDepth *WalkedDepth20
	var takerDepthFilter = common.NewDepthFilter(timeDecay, timeBias)

	logSilentTime := time.Now()
	takerWalkDepthTimer := time.NewTimer(time.Hour * 999)
	takerParseTimer := time.NewTimer(time.Hour * 999)
	takerUpdateSignalTimer := time.NewTimer(time.Hour * 999)
	defer takerWalkDepthTimer.Stop()
	defer takerParseTimer.Stop()
	defer takerUpdateSignalTimer.Stop()

	bidAskRatioWindow := make([]float64, 0)
	bidAskRatioSortedSlice := common.SortedFloatSlice{}
	askBidRatioWindow := make([]float64, 0)
	askBidRatioSortedSlice := common.SortedFloatSlice{}
	times := make([]time.Time, 0)

	depthCount := 0
	cutIndex := 0
	i := 0
	eventTime := time.Now()
	bidAskRatio := 0.0
	askBidRatio := 0.0

	expectedChanSendingTime := time.Nanosecond * 300
	for {
		select {
		case <-ctx.Done():
			return
		case <-takerUpdateSignalTimer.C:
			if takerWalkedDepth == nil {
				break
			}

			times = append(times, takerWalkedDepth.Time)
			bidAskRatioWindow = append(bidAskRatioWindow, takerWalkedDepth.BidAskRatio)
			askBidRatioWindow = append(askBidRatioWindow, takerWalkedDepth.AskBidRatio)
			bidAskRatioSortedSlice = bidAskRatioSortedSlice.Insert(takerWalkedDepth.BidAskRatio)
			askBidRatioSortedSlice = askBidRatioSortedSlice.Insert(takerWalkedDepth.AskBidRatio)

			cutIndex = 0
			for i, eventTime = range times {
				if takerWalkedDepth.Time.Sub(eventTime) > lookbackDuration {
					cutIndex = i
				} else {
					break
				}
			}
			if cutIndex > 0 {
				for _, bidAskRatio = range bidAskRatioWindow[:cutIndex] {
					bidAskRatioSortedSlice = bidAskRatioSortedSlice.Delete(bidAskRatio)
				}
				bidAskRatioWindow = bidAskRatioWindow[cutIndex:]

				for _, askBidRatio = range askBidRatioWindow[:cutIndex] {
					askBidRatioSortedSlice = askBidRatioSortedSlice.Delete(askBidRatio)
				}
				askBidRatioWindow = askBidRatioWindow[cutIndex:]
				times = times[cutIndex:]
			}

			if len(times) == 0 {
				break
			}

			if takerWalkedDepth.Time.Sub(times[0]) < lookbackDuration/2 {
				break
			}

			takerWalkedDepth.EmaAskBidRatio = askBidRatioSortedSlice.Median()
			takerWalkedDepth.EmaBidAskRatio = bidAskRatioSortedSlice.Median()

			select {
			case <-ctx.Done():
			case outputCh <- *takerWalkedDepth:
			default:
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("outputCh <- *takerWalkedDepth %s len(outputCh) %d", symbol, len(outputCh))
					logSilentTime = time.Now().Add(time.Minute)
				}
			}
			break

		case <-takerWalkDepthTimer.C:
			if takerDepth != nil {
				takerWalkedDepth, err = WalkBnspotDepth20(takerDepth, levelDecay)
				if err != nil {
					if time.Now().Sub(logSilentTime) > 0 {
						if takerRawDepth == nil {
							logger.Debugf("WalkBnspotDepth20 error %v %s", err, symbol)
						} else {
							logger.Debugf("WalkBnspotDepth20 error %v %s %s", err, symbol, takerRawDepth.Depth)
						}
						logSilentTime = time.Now().Add(time.Minute)
					}
					break
				}
				takerUpdateSignalTimer.Reset(time.Nanosecond)
			}
			break

		case <-takerParseTimer.C:
			if takerRawDepth == nil {
				break
			}
			newTakerDepth, err = bnspot.ParseDepth20(takerRawDepth.Depth)
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

func WalkBnspotDepth20(depth20 *bnspot.Depth20, decay float64) (*WalkedDepth20, error) {
	bidSize := 0.0
	askSize := 0.0
	bidValue := 0.0
	askValue := 0.0
	factor := 1.0
	for i := 0; i < 20; i++ {
		bidSize += factor * depth20.Bids[i][1]
		askSize += factor * depth20.Asks[i][1]
		bidValue += factor * depth20.Asks[i][1] * depth20.Bids[i][0]
		askValue += factor * depth20.Bids[i][1] * depth20.Asks[i][0]
		factor *= decay
	}
	if bidSize == 0 || askSize == 0 {
		return nil, fmt.Errorf("bad size bid %f ask %f", bidSize, askSize)
	}
	return &WalkedDepth20{
		Symbol:         depth20.Symbol,
		Time:           depth20.ParseTime,
		BidAskRatio:    bidSize / askSize,
		AskBidRatio:    askSize / bidSize,
		EmaBidAskRatio: bidSize / askSize,
		EmaAskBidRatio: askSize / bidSize,
		BidSize:        bidSize,
		AskSize:        askSize,
		BidPrice:       depth20.Bids[0][0],
		AskPrice:       depth20.Asks[0][0],
		MidPrice:       (depth20.Bids[0][0] + depth20.Asks[0][0]) / 2,
		MircoPrice:     (bidValue + askValue) / (bidSize + askSize),
	}, nil
}
