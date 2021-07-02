package main

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"time"
)

func startXYStrategy(
	ctx context.Context,
	xSymbol, ySymbol string,
	config Config,
	orderOffset Offset,
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
	xDepthCh chan common.Depth,
	yDepthCh chan common.Depth,
	saveCh chan *XYStrategy,
) (err error) {

	xBiasInMs := float64(config.DepthXBias / time.Millisecond)
	yBiasInMs := float64(config.DepthYBias / time.Millisecond)
	minTimeDeltaInMs := float64(config.DepthMinTimeDelta / time.Millisecond)
	maxTimeDeltaInMs := float64(config.DepthMaxTimeDelta / time.Millisecond)

	strat := XYStrategy{
		xExchange:           xExchange,
		yExchange:           yExchange,
		xSymbol:             xSymbol,
		ySymbol:             ySymbol,
		config:              config,
		xAccountCh:          xAccountCh,
		yAccountCh:          yAccountCh,
		xPositionCh:         xPositionCh,
		yPositionCh:         yPositionCh,
		xFundingRateCh:      xFundingRateCh,
		yFundingRateCh:      yFundingRateCh,
		xOrderCh:            xOrderCh,
		yOrderCh:            yOrderCh,
		xOrderErrorCh:       xOrderErrorCh,
		yOrderErrorCh:       yOrderErrorCh,
		xOrderRequestCh:     xOrderRequestCh,
		yOrderRequestCh:     yOrderRequestCh,
		xSystemStatusCh:     xSystemStatusCh,
		ySystemStatusCh:     ySystemStatusCh,
		xDepthCh:            xDepthCh,
		yDepthCh:            yDepthCh,
		saveCh:              saveCh,
		xPositionUpdateTime: time.Time{},
		yPositionUpdateTime: time.Time{},
		orderOffset:         orderOffset,
		xDepth:              nil,
		yDepth:              nil,
		xNextDepth:          nil,
		yNextDepth:          nil,
		xDepthTime:          time.Time{},
		yDepthTime:          time.Time{},
		xDepthFilter:        common.NewTimeFilter(config.DepthXDecay, xBiasInMs, minTimeDeltaInMs, maxTimeDeltaInMs),
		yDepthFilter:        common.NewTimeFilter(config.DepthYDecay, yBiasInMs, minTimeDeltaInMs, maxTimeDeltaInMs),
		xWalkedDepth:        common.WalkedDepthBMA{},
		yWalkedDepth:        common.WalkedDepthBMA{},

		xyDepthMatchSum:    common.NewRollingSum(config.DepthReportCount),
		xyDepthMatchWindow: float64(config.DepthReportCount),
		xyDepthMatchRatio:  0.0,

		fundingRateSettleTimer: time.NewTimer(time.Now().Truncate(config.FundingInterval).Add(config.FundingInterval - config.FundingRateSilentTime).Sub(time.Now())),

		xAccount:                nil,
		yAccount:                nil,
		xPosition:               nil,
		yPosition:               nil,
		xOrderSilentTime:        time.Time{},
		yOrderSilentTime:        time.Time{},
		xFundingRate:            nil,
		yFundingRate:            nil,
		xyFundingRate:           nil,
		xLastFilledBuyPrice:     nil,
		xLastFilledSellPrice:    nil,
		yLastFilledBuyPrice:     nil,
		yLastFilledSellPrice:    nil,
		xOrder:                  nil,
		yOrder:                  nil,
		xOrderError:             common.OrderError{},
		yOrderError:             common.OrderError{},
		xyEnterSilentTime:       time.Now().Add(config.RestartSilent),
		enterStep:               0,
		enterTarget:             0,
		targetWeight:            config.TargetWeights[xSymbol],
		usdtAvailable:           0,
		logSilentTime:           time.Time{},
		xWalkDepthTimer:         time.NewTimer(time.Hour * 9999),
		yWalkDepthTimer:         time.NewTimer(time.Hour * 9999),
		realisedSpreadTimer:     time.NewTimer(time.Hour * 9999),
		hedgeYTimer:             time.NewTimer(time.Hour * 9999),
		spreadWalkTimer:         time.NewTimer(time.Hour * 9999),
		saveTimer:               time.NewTimer(config.RestartSilent),
		spreadTime:              time.Time{},
		spread:                  nil,
		shortEnterTimedMedian:   common.NewTimedMedian(config.SpreadLookback),
		longEnterTimedMedian:    common.NewTimedMedian(config.SpreadLookback),
		xTimedPositionChange:    common.NewTimedSum(config.TurnoverLookback),
		yTimedPositionChange:    common.NewTimedSum(config.TurnoverLookback),
		expectedChanSendingTime: time.Nanosecond * 300,
		depthMatchCount:         0,
		depthCount:              0,
		xDepthExpireCount:       0,
		yDepthExpireCount:       0,
		shortLastEnter:          0,
		longLastEnter:           0,
		adjustedAgeDiff:         0,
		spreadReport:            nil,
		stateOutputCh:           nil,
		error:                   nil,
		xSizeDiff:               0,
		ySizeDiff:               0,
		offsetFactor:            0,
		shortTop:                0,
		shortBot:                0,
		longBot:                 0,
		longTop:                 0,
		xSize:                   0,
		ySize:                   0,
		xValue:                  0,
		yValue:                  0,
		xAbsValue:               0,
		yAbsValue:               0,
		midPrice:                0,
		enterValue:              0,
		targetValue:             0,
		size:                    0,
		orderSide:               common.OrderSideUnknown,
		isXSpot:                 xExchange.IsSpot(),
		isYSpot:                 yExchange.IsSpot(),
	}

	strat.xTickSize, err = xExchange.GetTickSize(xSymbol)
	if err != nil {
		return
	}
	strat.xStepSize, err = xExchange.GetStepSize(xSymbol)
	if err != nil {
		return
	}
	strat.xMultiplier, err = xExchange.GetMultiplier(xSymbol)
	if err != nil {
		return
	}
	strat.xMinNotional, err = xExchange.GetMinNotional(xSymbol)
	if err != nil {
		return
	}

	strat.yTickSize, err = yExchange.GetTickSize(ySymbol)
	if err != nil {
		return
	}
	strat.yStepSize, err = yExchange.GetStepSize(ySymbol)
	if err != nil {
		return
	}
	strat.yMultiplier, err = yExchange.GetMultiplier(ySymbol)
	if err != nil {
		return
	}
	strat.yMinNotional, err = yExchange.GetMinNotional(ySymbol)
	if err != nil {
		return
	}

	strat.xyMergedSpotStepSize = common.MergedStepSize(strat.xStepSize*strat.xMultiplier, strat.yStepSize*strat.yMultiplier)

	go strat.startLoop(ctx)
	return
}

func (strat *XYStrategy) startLoop(ctx context.Context) {
	defer strat.xWalkDepthTimer.Stop()
	defer strat.yWalkDepthTimer.Stop()
	defer strat.realisedSpreadTimer.Stop()
	defer strat.saveTimer.Stop()
	var nextXPos, nextYPos common.Position
	for {
		select {
		case <-ctx.Done():
			return
		case strat.xSystemStatus = <-strat.xSystemStatusCh:
			break
		case strat.ySystemStatus = <-strat.ySystemStatusCh:
			break
		case <-strat.saveTimer.C:
			select {
			case strat.saveCh <- strat:
			default:
				logger.Debugf("strat.saveCh <- strat failed %s %s ch len %d", strat.xSymbol, strat.ySymbol, len(strat.saveCh))
			}
			strat.saveTimer.Reset(strat.config.InternalInflux.SaveInterval)
			break
		case <-strat.hedgeYTimer.C:
			strat.changeYPosition()
			strat.hedgeCounter--
			if strat.hedgeCounter > 0 {
				strat.hedgeYTimer.Reset(strat.config.HedgeCheckInterval)
			}
			break
		case <-strat.fundingRateSettleTimer.C:
			if time.Now().Truncate(strat.config.FundingInterval).Add(strat.config.FundingInterval).Sub(time.Now()) <= strat.config.FundingRateSilentTime {
				logger.Debugf("%s fundingRate Silent true %v", strat.xSymbol, time.Now().Truncate(strat.config.FundingInterval).Add(strat.config.FundingInterval).Sub(time.Now()))
				strat.fundingRateSettleSilent = true
				strat.fundingRateSettleTimer.Reset(strat.config.FundingRateSilentTime * 2)
			} else {
				strat.fundingRateSettleSilent = false
				strat.fundingRateSettleTimer.Reset(time.Minute)
			}
			break
		case <-strat.spreadWalkTimer.C:
			strat.walkSpread()
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
		case <-strat.xWalkDepthTimer.C:
			strat.walkXDepth()
			break
		case <-strat.yWalkDepthTimer.C:
			strat.walkYDepth()
			break
		case strat.xNextDepth = <-strat.xDepthCh:
			strat.handleXDepth()
			break
		case strat.yNextDepth = <-strat.yDepthCh:
			strat.handleYDepth()
			break
		case <-strat.realisedSpreadTimer.C:
			strat.handleRealisedSpread()
			break
		}
	}
}

func (strat *XYStrategy) handleXPosition(nextPos common.Position) {
	if nextPos.GetSymbol() != strat.xSymbol {
		logger.Debugf("bad next position, symbol %s %s not match %v", nextPos.GetSymbol(), strat.xSymbol, nextPos)
		return
	}
	if strat.xPosition != nil {
		if strat.xPosition == nextPos {
			logger.Debugf("bad strat.xPosition == nextPos pass same pointer")
			return
		}
		if nextPos.GetEventTime().Sub(strat.xPosition.GetEventTime()) >= 0 {
			if strat.xPosition.GetSize() != nextPos.GetSize() {
				if strat.xWalkedDepth.Symbol != "" {
					strat.xTimedPositionChange.Insert(time.Now(), math.Abs(strat.xPosition.GetSize()-nextPos.GetSize())*strat.xWalkedDepth.MidPrice*strat.xMultiplier)
				}
				logger.Debugf("%s x position change %f -> %f %v", nextPos.GetSymbol(), strat.xPosition.GetSize(), nextPos.GetSize(), nextPos.GetEventTime())
			}
			strat.xPosition = nextPos
			strat.changeYPosition()
		}
		strat.xPositionUpdateTime = nextPos.GetParseTime()
	} else {
		strat.xPosition = nextPos
		strat.xPositionUpdateTime = nextPos.GetParseTime()
		logger.Debugf("%s x position change nil -> %f", nextPos.GetSymbol(), nextPos.GetSize())
	}
}

func (strat *XYStrategy) handleYPosition(nextPos common.Position) {
	if nextPos.GetSymbol() != strat.ySymbol {
		logger.Debugf("bad next position, symbol %s %s not match %v", nextPos.GetSymbol(), strat.ySymbol, nextPos)
		return
	}
	if strat.yPosition != nil {
		if strat.yPosition == nextPos {
			logger.Debugf("bad strat.yPosition == nextPos pass same pointer")
			return
		}
		if nextPos.GetEventTime().Sub(strat.yPosition.GetEventTime()) >= 0 {
			if strat.yPosition.GetSize() != nextPos.GetSize() {
				if strat.yWalkedDepth.Symbol != "" {
					strat.yTimedPositionChange.Insert(time.Now(), math.Abs(strat.yPosition.GetSize()-nextPos.GetSize())*strat.yWalkedDepth.MidPrice*strat.yMultiplier)
				}
				logger.Debugf("%s y position change %f -> %f %v", nextPos.GetSymbol(), strat.yPosition.GetSize(), nextPos.GetSize(), nextPos.GetEventTime())
			}
			strat.yPosition = nextPos
			strat.changeYPosition()
		}
		strat.yPositionUpdateTime = nextPos.GetParseTime()
	} else {
		strat.yPosition = nextPos
		strat.yPositionUpdateTime = nextPos.GetParseTime()
		logger.Debugf("%s y position change nil -> %f", nextPos.GetSymbol(), nextPos.GetSize())
	}
}
