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
	xPositionCh chan common.Position,
	xFundingRateCh chan common.FundingRate,
	yFundingRateCh chan common.FundingRate,
	xOrderRequestCh chan common.OrderRequest,
	xOrderCh chan common.Order,
	xOrderErrorCh chan common.OrderError,
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
		xExchange:       xExchange,
		reduceOnly:      config.ReduceOnlyBySymbol[xSymbol],
		stats:           stats,
		xLeverage:       config.XExchange.Leverage,
		xSymbol:         xSymbol,
		ySymbol:         ySymbol,
		config:          config,
		xAccountCh:      xAccountCh,
		xPositionCh:     xPositionCh,
		xFundingRateCh:  xFundingRateCh,
		yFundingRateCh:  yFundingRateCh,
		xOrderCh:        xOrderCh,
		xOrderErrorCh:   xOrderErrorCh,
		xOrderRequestCh: xOrderRequestCh,
		xSystemStatusCh: xSystemStatusCh,
		ySystemStatusCh: ySystemStatusCh,

		xyTickerCh: xyTickerCh,

		xPositionUpdateTime:  time.Time{},
		yPositionUpdateTime:  time.Time{},
		xTicker:              nil,
		yTicker:              nil,
		xTickerTime:          time.Time{},
		yTickerTime:          time.Time{},
		xAccount:             nil,
		xPosition:            nil,
		xOrderSilentTime:     time.Now().Add(config.RestartSilent),
		xFundingRate:         nil,
		yFundingRate:         nil,
		xyFundingRate:        nil,
		xLastFilledBuyPrice:  nil,
		xLastFilledSellPrice: nil,
		xOrder:               nil,
		xOrderError:          common.OrderError{},
		enterStep:            0,
		enterTarget:          0,
		usdAvailable:         0,
		logSilentTime:        time.Time{},
		realisedSpreadTimer:  time.NewTimer(time.Hour * 9999),

		xFundingRateCheckTimer: time.NewTimer(time.Second),

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

		enterValue:              0,
		targetValue:             0,
		stopped:                 0,
		fundingRateSettleSilent: false,
		xExchangeID:             xExchange.GetExchange(),
		yExchangeID:             yExchange.GetExchange(),
		tdSpreadMiddle:          0,
		lastEnterTime:           time.Time{},

		xSlippageTMPath: fmt.Sprintf("%s/%s-%s.XSTM.json", config.StatsRootPath, common.SymbolSanitize(xSymbol), common.SymbolSanitize(ySymbol)),
	}
	strat.xSlippageTM = stream_stats.LoadOrCreateTimeMean(strat.xSlippageTMPath, config.EnterSlippageLookback)

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
	strat.yMultiplier, err = yExchange.GetMultiplier(ySymbol)
	if err != nil {
		logger.Debugf("%v", err)
		return
	}

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
	err := strat.xSlippageTM.Save(strat.xSlippageTMPath)
	if err != nil {
		logger.Debugf("strat.xSlippageTM.Save %s error %v", strat.xSlippageTMPath, err)
	} else {
		logger.Debugf("%10s xSlippageTM %s saved", strat.xSymbol, strat.xSlippageTMPath)
	}
}

func (strat *XYStrategy) Start(ctx context.Context) {
	defer strat.spreadWalkTimer.Stop()
	defer strat.realisedSpreadTimer.Stop()
	defer strat.Stop()
	var nextXPos common.Position
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

		case strat.xAccount = <-strat.xAccountCh:
			strat.updateEnterStepAndTarget()
			break
		case nextXPos = <-strat.xPositionCh:
			strat.handleXPosition(nextXPos)
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
		case strat.xOrderError = <-strat.xOrderErrorCh:
			strat.handleXOrderError()
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
	if strat.xAccount == nil {
		return
	}
	strat.enterStep = strat.xAccount.GetFree() * strat.config.EnterFreePct * strat.targetWeight
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
	strat.usdAvailable = strat.xAccount.GetFree()*strat.xLeverage
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
			if math.Abs(strat.xPosition.GetSize()-nextPos.GetSize()) != 0 {
				logger.Debugf("X POS %f %s %v %v %f %f", nextPos.GetSize()-strat.xPosition.GetSize(), strat.xSymbol, nextPos.GetEventTime(), strat.xPosition.GetEventTime(), math.Abs(strat.xPosition.GetSize()-nextPos.GetSize()), strat.xStepSize)
			}
			//logger.Debugf("%s %v %v %f %f", strat.xSymbol, nextPos.GetEventTime(), strat.xPosition.GetEventTime(), math.Abs(strat.xPosition.GetSize()-nextPos.GetSize()), strat.xStepSize)
			if math.Abs(strat.xPosition.GetSize()-nextPos.GetSize()) >= strat.xStepSize {
				strat.xOrderSilentTime = time.Now().Add(strat.config.XOrderSilent)
				if strat.xTicker != nil {
					strat.xTimedPositionChange.Insert(time.Now(), math.Abs(strat.xPosition.GetSize()-nextPos.GetSize())*strat.xMidPrice*strat.xMultiplier)
				}
				logger.Debugf("%10s x position change %f -> %f %f %v", nextPos.GetSymbol(), strat.xPosition.GetSize(), nextPos.GetSize(), nextPos.GetPrice(), nextPos.GetEventTime())
				strat.xPosition = nextPos
			} else {
				strat.xPosition = nextPos
			}
		}
		strat.xPositionUpdateTime = nextPos.GetParseTime()
	} else {
		strat.xPosition = nextPos
		strat.xPositionUpdateTime = nextPos.GetParseTime()
		logger.Debugf("%10s x position change nil -> %f %f", nextPos.GetSymbol(), nextPos.GetSize(), nextPos.GetPrice())
	}
}

