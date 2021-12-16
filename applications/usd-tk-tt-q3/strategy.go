package main

import (
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	stream_stats "github.com/geometrybase/hft-micro/stream-stats"
	"math"
	"sync/atomic"
	"time"
)

func startXYStrategy(
	ctx context.Context,
	xSymbol, ySymbol string,
	config Config,
	xExchange common.UsdExchange,
	yExchange common.UsdExchange,
	xAccountCh chan common.Balance,
	yAccountCh chan common.Balance,
	xPositionCh chan common.Position,
	yPositionCh chan common.Position,
	xFundingRateCh chan common.FundingRate,
	yFundingRateCh chan common.FundingRate,
	xOrderRequestCh chan common.OrderRequest,
	yOrderRequestCh chan common.OrderRequest,
	xOrderCh chan common.Order,
	yOrderCh chan common.Order,
	xOrderErrorCh chan common.OrderError,
	yOrderErrorCh chan common.OrderError,
	xSystemStatusCh chan common.SystemStatus,
	ySystemStatusCh chan common.SystemStatus,
	xyTickerCh chan common.Ticker,
) (strat *XYStrategy, err error) {

	stats, err := stream_stats.NewXYSimplifiedTickerStats(stream_stats.NewXYSimplifiedTickerStatsParams{
		XSymbol:        xSymbol,
		YSymbol:        ySymbol,
		RootPath:       config.StatsRootPath,
		SampleInterval: config.StatsSampleInterval,
		SaveInterval:   config.StatsSaveInterval,

		TimeDeltaLookback: config.TimeDeltaLookback,

		SpreadTDLookback:    config.SpreadTDLookback,
		SpreadTDSubInterval: config.SpreadTDSubInterval,
		SpreadTDCompression: config.SpreadTDCompression,

		XTimeDeltaOffsetBot:  config.XTimeDeltaOffsetBot,
		XTimeDeltaOffsetTop:  config.XTimeDeltaOffsetTop,
		YTimeDeltaOffsetBot:  config.YTimeDeltaOffsetBot,
		YTimeDeltaOffsetTop:  config.YTimeDeltaOffsetTop,
		XYTimeDeltaOffsetBot: config.XYTimeDeltaOffsetBot,
		XYTimeDeltaOffsetTop: config.XYTimeDeltaOffsetTop,

		SpreadLongEnterQuantileBot:  config.SpreadLongEnterQuantileBot,
		SpreadLongLeaveQuantileTop:  config.SpreadLongLeaveQuantileTop,
		SpreadShortEnterQuantileTop: config.SpreadShortEnterQuantileTop,
		SpreadShortLeaveQuantileBot: config.SpreadShortLeaveQuantileBot,
		BaseEnterOffset:             config.SpreadEnterOffset,
		BaseLeaveOffset:             config.SpreadLeaveOffset,
	})
	if err != nil {
		return nil, err
	}

	strat = &XYStrategy{
		xExchange:          xExchange,
		yExchange:          yExchange,
		isXSpot:            xExchange.IsSpot(),
		isYSpot:            yExchange.IsSpot(),
		reduceOnly:         config.ReduceOnlyBySymbol[xSymbol],
		stats:              stats,
		xLeverage:          config.XExchange.Leverage,
		yLeverage:          config.YExchange.Leverage,
		xSymbol:            xSymbol,
		ySymbol:            ySymbol,
		config:             config,
		hedgeCheckTimer:    time.NewTimer(time.Hour * 9999),
		hedgeCheckStopTime: time.Time{},
		xAccountCh:         xAccountCh,
		yAccountCh:         yAccountCh,
		xPositionCh:        xPositionCh,
		yPositionCh:        yPositionCh,
		xFundingRateCh:     xFundingRateCh,
		yFundingRateCh:     yFundingRateCh,
		xOrderCh:           xOrderCh,
		yOrderCh:           yOrderCh,
		xOrderErrorCh:      xOrderErrorCh,
		yOrderErrorCh:      yOrderErrorCh,
		xOrderRequestCh:    xOrderRequestCh,
		yOrderRequestCh:    yOrderRequestCh,
		xSystemStatusCh:    xSystemStatusCh,
		ySystemStatusCh:    ySystemStatusCh,

		xyTickerCh: xyTickerCh,

		xPositionUpdateTime:  time.Time{},
		yPositionUpdateTime:  time.Time{},
		xTicker:              nil,
		yTicker:              nil,
		xTickerTime:          time.Time{},
		yTickerTime:          time.Time{},
		xAccount:             nil,
		yAccount:             nil,
		xPosition:            nil,
		yPosition:            nil,
		xOrderSilentTime:     time.Now().Add(config.RestartSilent),
		yOrderSilentTime:     time.Now().Add(config.RestartSilent),
		xFundingRate:         nil,
		yFundingRate:         nil,
		xyFundingRate:        nil,
		xLastFilledBuyPrice:  nil,
		xLastFilledSellPrice: nil,
		yLastFilledBuyPrice:  nil,
		yLastFilledSellPrice: nil,
		xOrder:               nil,
		yOrder:               nil,
		xOrderError:          common.OrderError{},
		yOrderError:          common.OrderError{},
		enterStep:            0,
		enterTarget:          0,
		usdAvailable:         0,
		logSilentTime:        time.Time{},
		realisedSpreadTimer:  time.NewTimer(time.Hour * 9999),

		xFundingRateCheckTimer: time.NewTimer(time.Second),
		yFundingRateCheckTimer: time.NewTimer(time.Second),

		spreadTickerTime:     time.Time{},
		spreadReady:          false,
		spreadWalkTimer:      time.NewTimer(time.Hour * 9999),
		spreadShortTimedMean: common.NewTimedMean(config.SpreadLookback),
		spreadLongTimedMean:  common.NewTimedMean(config.SpreadLookback),
		spreadLastShort:      0,
		spreadLastLong:       0,
		spreadMedianShort:    0,
		spreadMedianLong:     0,

		xTimedPositionChange: common.NewTimedSum(config.TurnoverLookback),
		yTimedPositionChange: common.NewTimedSum(config.TurnoverLookback),
		tickerMatchCount:     0,
		tickerCount:          0,
		offsetFactor:         0,
		thresholdShortTop:    0,
		thresholdShortBot:    0,
		thresholdLongBot:     0,
		thresholdLongTop:     0,

		targetWeight: config.PosWeights[xSymbol],

		maxPosSize:   config.MaxPosSizes[xSymbol],
		maxPosValue:  config.MaxPosValues[xSymbol],
		maxOrderSize: config.MaxPosSizes[xSymbol] / 4,
		//xSize:                   0,
		//ySize:                   0,
		//xValue:                  0,
		//yValue:                  0,
		//xAbsValue:               0,
		//yAbsValue:               0,
		//xyMidPrice:              0,
		enterValue:              0,
		targetValue:             0,
		stopped:                 0,
		fundingRateSettleSilent: false,
		xExchangeID:             xExchange.GetExchange(),
		yExchangeID:             yExchange.GetExchange(),
		tdSpreadMiddle:          0,
		lastEnterTime:           time.Time{},

		xySuccessRatioTMPath: fmt.Sprintf("%s/%s-%s.XYSRTM.json", config.StatsRootPath, common.SymbolSanitize(xSymbol), common.SymbolSanitize(ySymbol)),
		xySpreadSlippageTMPath: fmt.Sprintf("%s/%s-%s.XYSSTWM.json", config.StatsRootPath, common.SymbolSanitize(xSymbol), common.SymbolSanitize(ySymbol)),
		xSlippageTMPath:      fmt.Sprintf("%s/%s-%s.XSTWM.json", config.StatsRootPath, common.SymbolSanitize(xSymbol), common.SymbolSanitize(ySymbol)),
		ySlippageTMPath:      fmt.Sprintf("%s/%s-%s.YSTWM.json", config.StatsRootPath, common.SymbolSanitize(xSymbol), common.SymbolSanitize(ySymbol)),
	}
	strat.xySuccessRatioTM = stream_stats.LoadOrCreateTimeMean(strat.xySuccessRatioTMPath, config.EnterSlippageLookback)
	strat.xySpreadSlippageTM = stream_stats.LoadOrCreateTimedWeightedMean(strat.xySpreadSlippageTMPath, config.EnterSlippageLookback)
	strat.xSlippageTM = stream_stats.LoadOrCreateTimedWeightedMean(strat.xSlippageTMPath, config.EnterSlippageLookback)
	strat.ySlippageTM = stream_stats.LoadOrCreateTimedWeightedMean(strat.ySlippageTMPath, config.EnterSlippageLookback)

	strat.yTickSize, err = yExchange.GetTickSize(ySymbol)
	if err != nil {
		logger.Debugf("%v", err)
		return
	}
	strat.yStepSize, err = yExchange.GetStepSize(ySymbol)
	if err != nil {
		logger.Debugf("%v", err)
		return
	}
	strat.yMultiplier, err = yExchange.GetMultiplier(ySymbol)
	if err != nil {
		logger.Debugf("%v", err)
		return
	}
	strat.yMinNotional, err = yExchange.GetMinNotional(ySymbol)
	if err != nil {
		logger.Debugf("%v", err)
		return
	}
	strat.yMinSize, err = yExchange.GetMinSize(ySymbol)
	if err != nil {
		logger.Debugf("%v", err)
		return
	}

	strat.xTickSize, err = xExchange.GetTickSize(xSymbol)
	if err != nil {
		logger.Debugf("%v", err)
		return
	}
	strat.xStepSize, err = xExchange.GetStepSize(xSymbol)
	if err != nil {
		logger.Debugf("%v", err)
		return
	}
	strat.xMultiplier, err = xExchange.GetMultiplier(xSymbol)
	if err != nil {
		logger.Debugf("%v", err)
		return
	}
	strat.xMinNotional, err = xExchange.GetMinNotional(xSymbol)
	if err != nil {
		logger.Debugf("%v", err)
		return
	}
	strat.xMinSize, err = xExchange.GetMinSize(xSymbol)
	if err != nil {
		logger.Debugf("%v", err)
		return
	}
	strat.xyMergedStepSize = common.MergedStepSize(strat.xStepSize*strat.xMultiplier, strat.yStepSize*strat.yMultiplier)

	go strat.stats.Start(ctx)
	go strat.Start(ctx)
	return
}

func (strat *XYStrategy) Stop() {
	if atomic.CompareAndSwapInt32(&strat.stopped, 0, 1) {
		strat.stats.Stop()
		logger.Debugf("%10s %10s stopped", strat.xSymbol, strat.ySymbol)
		strat.saveSlippageTMs()
	}
}

func (strat *XYStrategy) saveSlippageTMs() {
	err := strat.xySuccessRatioTM.Save(strat.xySuccessRatioTMPath)
	if err != nil {
		logger.Debugf("strat.xySuccessRatioTM.Save %s error %v", strat.xySuccessRatioTMPath, err)
	} else {
		logger.Debugf("%10s xySuccessRatioTM %s saved", strat.xSymbol, strat.xySuccessRatioTMPath)
	}
	err = strat.xSlippageTM.Save(strat.xSlippageTMPath)
	if err != nil {
		logger.Debugf("strat.xSlippageTM.Save %s error %v", strat.xSlippageTMPath, err)
	} else {
		logger.Debugf("%10s xSlippageTM %s saved", strat.xSymbol, strat.xSlippageTMPath)
	}
	err = strat.ySlippageTM.Save(strat.ySlippageTMPath)
	if err != nil {
		logger.Debugf("strat.xySuccessRatioTM.Save %s error %v", strat.ySlippageTMPath, err)
	} else {
		logger.Debugf("%10s ySlippageTM %s saved", strat.xSymbol, strat.ySlippageTMPath)
	}
}

func (strat *XYStrategy) Start(ctx context.Context) {
	defer strat.spreadWalkTimer.Stop()
	defer strat.realisedSpreadTimer.Stop()
	defer strat.Stop()
	var nextXPos, nextYPos common.Position
	strat.xOrderSilentTime = time.Now().Add(strat.config.RestartSilent)
	strat.lastEnterTime = strat.xOrderSilentTime
	for {
		select {
		case <-ctx.Done():
			return
		case strat.xSystemStatus = <-strat.xSystemStatusCh:
			if strat.xSystemStatus != common.SystemStatusReady {
				strat.xOrderSilentTime = time.Now().Add(strat.config.RestartSilent)
			}
			break
		case strat.ySystemStatus = <-strat.ySystemStatusCh:
			if strat.ySystemStatus != common.SystemStatusReady {
				strat.xOrderSilentTime = time.Now().Add(strat.config.RestartSilent)
			}
			break
		case <-strat.xFundingRateCheckTimer.C:
			if strat.config.XFundingRateTimeOffset == 0 {
				if time.Now().Add(strat.config.XFundingRateInterval).Truncate(strat.config.XFundingRateInterval).Sub(time.Now()) <= strat.config.FundingRateSilentTime {
					logger.Debugf("%10s x fundingRate silent true %v", strat.xSymbol, time.Now().Add(strat.config.XFundingRateInterval).Truncate(strat.config.XFundingRateInterval).Sub(time.Now()))
					strat.fundingRateSettleSilent = true
					strat.xFundingRateCheckTimer.Reset(strat.config.FundingRateSilentTime + time.Second)
				} else {
					strat.fundingRateSettleSilent = false
					strat.xFundingRateCheckTimer.Reset(time.Second)
					if strat.xFundingRateFactor == nil {
						strat.xFundingRateFactor = new(float64)
					}
					t := 1.0 - time.Now().Add(strat.config.XFundingRateInterval).Truncate(strat.config.XFundingRateInterval).Sub(time.Now()).Seconds()/strat.config.XFundingRateInterval.Seconds()
					*strat.xFundingRateFactor = strat.config.FundingRateOffsetMin + (strat.config.FundingRateOffsetMax-strat.config.FundingRateOffsetMin)*strat.config.XFundingRateEaseFn(t)
				}
			} else {
				if time.Now().Add(strat.config.XFundingRateTimeOffset).Truncate(strat.config.XFundingRateInterval).Add(strat.config.XFundingRateTimeOffset).Sub(time.Now()) <= strat.config.FundingRateSilentTime {
					logger.Debugf("%10s x fundingRate silent true %v", strat.xSymbol, time.Now().Add(strat.config.XFundingRateTimeOffset).Truncate(strat.config.XFundingRateInterval).Add(strat.config.XFundingRateTimeOffset).Sub(time.Now()))
					strat.fundingRateSettleSilent = true
					strat.xFundingRateCheckTimer.Reset(strat.config.FundingRateSilentTime + time.Second)
				} else {
					strat.fundingRateSettleSilent = false
					strat.xFundingRateCheckTimer.Reset(time.Second)
					if strat.xFundingRateFactor == nil {
						strat.xFundingRateFactor = new(float64)
					}
					t := 1.0 - time.Now().Add(strat.config.XFundingRateTimeOffset).Truncate(strat.config.XFundingRateInterval).Add(strat.config.XFundingRateTimeOffset).Sub(time.Now()).Seconds()/strat.config.XFundingRateInterval.Seconds()
					*strat.xFundingRateFactor = strat.config.FundingRateOffsetMin + (strat.config.FundingRateOffsetMax-strat.config.FundingRateOffsetMin)*strat.config.XFundingRateEaseFn(t)
				}
			}
			break
		case <-strat.yFundingRateCheckTimer.C:
			if strat.config.YFundingRateTimeOffset == 0 {
				if time.Now().Add(strat.config.YFundingRateInterval).Truncate(strat.config.YFundingRateInterval).Sub(time.Now()) <= strat.config.FundingRateSilentTime {
					logger.Debugf("%10s y fundingRate silent true %v", strat.xSymbol, time.Now().Add(strat.config.YFundingRateInterval).Truncate(strat.config.YFundingRateInterval).Sub(time.Now()))
					strat.fundingRateSettleSilent = true
					strat.yFundingRateCheckTimer.Reset(strat.config.FundingRateSilentTime + time.Second)
				} else {
					strat.fundingRateSettleSilent = false
					strat.yFundingRateCheckTimer.Reset(time.Second)
					if strat.yFundingRateFactor == nil {
						strat.yFundingRateFactor = new(float64)
					}
					t := 1.0 - time.Now().Add(strat.config.YFundingRateInterval).Truncate(strat.config.YFundingRateInterval).Sub(time.Now()).Seconds()/strat.config.YFundingRateInterval.Seconds()
					*strat.yFundingRateFactor = strat.config.FundingRateOffsetMin + (strat.config.FundingRateOffsetMax-strat.config.FundingRateOffsetMin)*strat.config.YFundingRateEaseFn(t)
				}
			} else {
				if time.Now().Add(strat.config.YFundingRateTimeOffset).Truncate(strat.config.YFundingRateInterval).Add(strat.config.YFundingRateTimeOffset).Sub(time.Now()) <= strat.config.FundingRateSilentTime {
					logger.Debugf("%10s y fundingRate silent true %v", strat.xSymbol, time.Now().Add(strat.config.YFundingRateTimeOffset).Truncate(strat.config.YFundingRateInterval).Add(strat.config.YFundingRateTimeOffset).Sub(time.Now()))
					strat.fundingRateSettleSilent = true
					strat.yFundingRateCheckTimer.Reset(strat.config.FundingRateSilentTime + time.Second)
				} else {
					strat.fundingRateSettleSilent = false
					strat.yFundingRateCheckTimer.Reset(time.Second)
					if strat.yFundingRateFactor == nil {
						strat.yFundingRateFactor = new(float64)
					}
					t := 1.0 - time.Now().Add(strat.config.YFundingRateTimeOffset).Truncate(strat.config.YFundingRateInterval).Add(strat.config.YFundingRateTimeOffset).Sub(time.Now()).Seconds()/strat.config.YFundingRateInterval.Seconds()
					*strat.yFundingRateFactor = strat.config.FundingRateOffsetMin + (strat.config.FundingRateOffsetMax-strat.config.FundingRateOffsetMin)*strat.config.YFundingRateEaseFn(t)
				}
			}
			break
		case <-strat.hedgeCheckTimer.C:
			strat.hedgeYPosition()
			if time.Now().Sub(strat.hedgeCheckStopTime) > 0 {
				strat.hedgeCheckTimer.Reset(time.Hour * 9999)
			} else {
				strat.hedgeCheckTimer.Reset(strat.config.HedgeCheckInterval)
			}
			break
		case strat.xAccount = <-strat.xAccountCh:
			strat.updateEnterStepAndTarget()
			break
		case strat.yAccount = <-strat.yAccountCh:
			strat.updateEnterStepAndTarget()
			break
		case nextXPos = <-strat.xPositionCh:
			strat.handleXPosition(nextXPos)
			break
		case nextYPos = <-strat.yPositionCh:
			strat.handleYPosition(nextYPos)
			break
		case strat.xFundingRate = <-strat.xFundingRateCh:
			strat.handleFundingRate()
			break
		case strat.yFundingRate = <-strat.yFundingRateCh:
			strat.handleFundingRate()
			break
		case strat.xOrder = <-strat.xOrderCh:
			strat.handleXOrder()
			break
		case strat.yOrder = <-strat.yOrderCh:
			strat.handleYOrder()
			break
		case strat.xOrderError = <-strat.xOrderErrorCh:
			strat.handleXOrderError()
			break
		case strat.yOrderError = <-strat.yOrderErrorCh:
			strat.handleYOrderError()
			break
		case <-strat.spreadWalkTimer.C:
			strat.updateSpread()
			break
		case strat.nextTicker = <-strat.xyTickerCh:
			strat.handleTicker()
			break
		case <-strat.realisedSpreadTimer.C:
			strat.handleRealisedSpread()
			break
		}
	}
}

func (strat *XYStrategy) updateEnterStepAndTarget() {
	if strat.xAccount == nil || strat.yAccount == nil {
		return
	}
	strat.enterStep = (strat.xAccount.GetFree() + strat.yAccount.GetFree()) * strat.config.EnterFreePct * strat.targetWeight
	if strat.enterStep < strat.config.EnterMinimalStep {
		strat.enterStep = strat.config.EnterMinimalStep
	}
	strat.enterTarget = strat.enterStep * strat.config.EnterTargetFactor * strat.targetWeight
	//logger.Debugf(
	//	"%s ACCOUNT X %f %f Y %f %f W %f T %f",
	//	strat.xSymbol,
	//	strat.xAccount.GetFree(),
	//	strat.xAccount.GetBalance(),
	//	strat.yAccount.GetFree(),
	//	strat.yAccount.GetBalance(),
	//	strat.targetWeight,
	//	strat.enterTarget,
	//)
	strat.usdAvailable = math.Min(strat.xAccount.GetFree()*strat.xLeverage, strat.yAccount.GetFree()/strat.config.HedgeRatio*strat.yLeverage)
}

func (strat *XYStrategy) handleXPosition(nextPos common.Position) {
	if nextPos.GetSymbol() != strat.xSymbol {
		logger.Debugf("%10s bad next position, symbol %s not match %v", nextPos.GetSymbol(), strat.xSymbol, nextPos)
		return
	}
	if strat.xPosition != nil {
		if strat.xPosition == nextPos {
			logger.Debugf("%10s bad strat.xPosition == nextPos pass same pointer", strat.xSymbol)
			return
		}
		if nextPos.GetEventTime().Sub(strat.xPosition.GetEventTime()) >= 0 {
			//if math.Abs(strat.xPosition.GetSize()-nextPos.GetSize()) != 0 {
			//	logger.Debugf("X POS %f %s %v %v %f %f", nextPos.GetSize()-strat.xPosition.GetSize(), strat.xSymbol, nextPos.GetEventTime(), strat.xPosition.GetEventTime(), math.Abs(strat.xPosition.GetSize()-nextPos.GetSize()), strat.xStepSize)
			//}
			//logger.Debugf("%s %v %v %f %f", strat.xSymbol, nextPos.GetEventTime(), strat.xPosition.GetEventTime(), math.Abs(strat.xPosition.GetSize()-nextPos.GetSize()), strat.xStepSize)
			if math.Abs(strat.xPosition.GetSize()-nextPos.GetSize()) >= strat.xStepSize {
				strat.xOrderSilentTime = time.Now().Add(strat.config.XOrderSilent)
				strat.yOrderSilentTime = time.Now()
				if strat.xTicker != nil {
					strat.xTimedPositionChange.Insert(time.Now(), math.Abs(strat.xPosition.GetSize()-nextPos.GetSize())*strat.xMidPrice*strat.xMultiplier)
				}
				logger.Debugf("%10s x position change %f -> %f %f %v", nextPos.GetSymbol(), strat.xPosition.GetSize(), nextPos.GetSize(), nextPos.GetPrice(), nextPos.GetEventTime())
				strat.xPosition = nextPos
				if time.Now().Sub(strat.hedgeCheckStopTime) > 0 ||
					strat.config.HedgeDelay == 0 {
					strat.hedgeYPosition()
				} else {
					strat.hedgeCheckTimer.Reset(strat.config.HedgeDelay)
				}
			} else {
				strat.xPosition = nextPos
				if time.Now().Sub(strat.hedgeCheckStopTime) > 0 ||
					strat.config.HedgeDelay == 0 {
					strat.hedgeYPosition()
				} else {
					strat.hedgeCheckTimer.Reset(strat.config.HedgeDelay)
				}
			}
		}
		strat.xPositionUpdateTime = nextPos.GetParseTime()
	} else {
		strat.xPosition = nextPos
		strat.xPositionUpdateTime = nextPos.GetParseTime()
		logger.Debugf("%10s x position change nil -> %f %f", nextPos.GetSymbol(), nextPos.GetSize(), nextPos.GetPrice())
	}
}

func (strat *XYStrategy) handleYPosition(nextPos common.Position) {
	if nextPos.GetSymbol() != strat.ySymbol {
		logger.Debugf("%10s bad next position, %s not match %v", nextPos.GetSymbol(), strat.ySymbol, nextPos)
		return
	}
	if strat.yPosition != nil {
		if strat.yPosition == nextPos {
			logger.Debugf("%10sbad strat.yPosition == nextPos pass same pointer", nextPos.GetSymbol())
			return
		}
		if nextPos.GetEventTime().Sub(strat.yPosition.GetEventTime()) >= -time.Second {
			if math.Abs(strat.yPosition.GetSize()-nextPos.GetSize()) >= strat.yStepSize {
				if strat.yTicker != nil {
					strat.yTimedPositionChange.Insert(time.Now(), math.Abs(strat.yPosition.GetSize()-nextPos.GetSize())*strat.yMidPrice*strat.yMultiplier)
				}
				logger.Debugf("%10s y position change %f -> %f %f %v", nextPos.GetSymbol(), strat.yPosition.GetSize(), nextPos.GetSize(), nextPos.GetPrice(), nextPos.GetEventTime())
			}
			strat.yPosition = nextPos
		}
		strat.yPositionUpdateTime = nextPos.GetParseTime()
	} else {
		strat.yPosition = nextPos
		strat.yPositionUpdateTime = nextPos.GetParseTime()
		logger.Debugf("%10s y position change nil -> %f %f", nextPos.GetSymbol(), nextPos.GetSize(), nextPos.GetPrice())
	}
}
