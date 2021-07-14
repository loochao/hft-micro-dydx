package main

import (
	"context"
	"encoding/json"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	stream_stats "github.com/geometrybase/hft-micro/stream-stats"
	"math"
	"os"
	"path"
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
	saveCh chan *XYStrategy,
) (err error) {

	xBiasInMs := float64(config.TickerXBias / time.Millisecond)
	yBiasInMs := float64(config.TickerYBias / time.Millisecond)
	minTimeDeltaInMs := float64(config.TickerMinTimeDelta / time.Millisecond)
	maxTimeDeltaInMs := float64(config.TickerMaxTimeDelta / time.Millisecond)
	var quantileMiddle *float64

	timedTDigest := stream_stats.NewTimedTDigest(config.QuantileLookback, config.QuantileSubInterval)
	if config.QuantilePath != "" {

		longBytes, err := os.ReadFile(path.Join(config.QuantilePath, xSymbol+"-"+ySymbol+".json"))
		if err != nil {
			logger.Debugf("os.ReadFile error %v", err)
		} else {
			err = json.Unmarshal(longBytes, &timedTDigest)
			if err != nil {
				logger.Debugf("json.Unmarshal error %v", err)
				timedTDigest = stream_stats.NewTimedTDigest(config.QuantileLookback, config.QuantileSubInterval)
			} else {
				timedTDigest.Lookback = config.QuantileLookback
				timedTDigest.SubInterval = config.QuantileSubInterval
				quantileMiddle = new(float64)
				*quantileMiddle = timedTDigest.Quantile(0.5)
				logger.Debugf("%s - %s QUANTILE MIDDLE %f", xSymbol, ySymbol, *quantileMiddle)
			}
		}
	}

	strat := XYStrategy{
		xExchange:               xExchange,
		yExchange:               yExchange,
		isXSpot:                 xExchange.IsSpot(),
		isYSpot:                 yExchange.IsSpot(),
		xLeverage:               config.XExchange.Leverage,
		yLeverage:               config.YExchange.Leverage,
		xSymbol:                 xSymbol,
		ySymbol:                 ySymbol,
		targetWeight:            config.TargetWeights[xSymbol],
		maxOrderValue:           config.MaxOrderValues[xSymbol],
		config:                  config,
		hedgeCheckTimer:         time.NewTimer(time.Hour * 9999),
		hedgeCheckStopTime:      time.Time{},
		xAccountCh:              xAccountCh,
		yAccountCh:              yAccountCh,
		xPositionCh:             xPositionCh,
		yPositionCh:             yPositionCh,
		xFundingRateCh:          xFundingRateCh,
		yFundingRateCh:          yFundingRateCh,
		xOrderCh:                xOrderCh,
		yOrderCh:                yOrderCh,
		xOrderErrorCh:           xOrderErrorCh,
		yOrderErrorCh:           yOrderErrorCh,
		xOrderRequestCh:         xOrderRequestCh,
		yOrderRequestCh:         yOrderRequestCh,
		xSystemStatusCh:         xSystemStatusCh,
		ySystemStatusCh:         ySystemStatusCh,
		xyTickerCh:              xyTickerCh,
		saveCh:                  saveCh,
		xPositionUpdateTime:     time.Time{},
		yPositionUpdateTime:     time.Time{},
		xTicker:                 nil,
		yTicker:                 nil,
		xTickerTime:             time.Time{},
		yTickerTime:             time.Time{},
		xTickerFilter:           common.NewTimeFilter(config.TickerXDecay, xBiasInMs, minTimeDeltaInMs, maxTimeDeltaInMs),
		yTickerFilter:           common.NewTimeFilter(config.TickerYDecay, yBiasInMs, minTimeDeltaInMs, maxTimeDeltaInMs),
		xAccount:                nil,
		yAccount:                nil,
		xPosition:               nil,
		yPosition:               nil,
		xOrderSilentTime:        time.Now().Add(config.RestartSilent),
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
		enterStep:               0,
		enterTarget:             0,
		usdAvailable:            0,
		logSilentTime:           time.Time{},
		spreadWalkTimer:         time.NewTimer(time.Hour * 9999),
		realisedSpreadTimer:     time.NewTimer(time.Hour * 9999),
		saveTimer:               time.NewTimer(config.RestartSilent),
		fundingRateSettleTimer:  time.NewTimer(time.Now().Truncate(config.FundingInterval).Add(config.FundingInterval - time.Second).Sub(time.Now())),
		spreadTime:              time.Time{},
		spread:                  nil,
		shortEnterTimedMedian:   common.NewTimedMedian(config.SpreadLookback),
		longEnterTimedMedian:    common.NewTimedMedian(config.SpreadLookback),
		xTimedPositionChange:    common.NewTimedSum(config.TurnoverLookback),
		yTimedPositionChange:    common.NewTimedSum(config.TurnoverLookback),
		expectedChanSendingTime: time.Nanosecond * 300,
		tickerMatchCount:        0,
		tickerCount:             0,
		xTickerExpireCount:      0,
		yTickerExpireCount:      0,
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
		stopped:                 0,
		fundingRateSettleSilent: false,
		xExchangeID:             xExchange.GetExchange(),
		yExchangeID:             yExchange.GetExchange(),
		timedTDigest:            timedTDigest,
		quantileSaveTimer:       time.NewTimer(config.QuantileSaveInterval),
		quantileLastSampleTime:  time.Time{},
		quantileMiddle:          quantileMiddle,
		lastSpreadEnterTime: time.Time{},
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
	strat.xyMergedSpotStepSize = common.MergedStepSize(strat.xStepSize*strat.xMultiplier, strat.yStepSize*strat.yMultiplier)

	go strat.startLoop(ctx)
	return
}

func (strat *XYStrategy) Stop() {
	if atomic.CompareAndSwapInt32(&strat.stopped, 0, 1) {
		strat.handleQuantileSave()
		logger.Debugf("stopped %s %s strategy",strat.xSymbol, strat.ySymbol)
	}
}

func (strat *XYStrategy) startLoop(ctx context.Context) {
	defer strat.spreadWalkTimer.Stop()
	defer strat.realisedSpreadTimer.Stop()
	defer strat.saveTimer.Stop()
	defer strat.Stop()
	var nextXPos, nextYPos common.Position
	strat.xOrderSilentTime = time.Now().Add(strat.config.RestartSilent)
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
		case <-strat.fundingRateSettleTimer.C:
			if time.Now().Truncate(strat.config.FundingInterval).Add(strat.config.FundingInterval).Sub(time.Now()) <= strat.config.FundingRateSilentTime {
				logger.Debugf("%s fundingRate Silent true %v", strat.xSymbol, time.Now().Truncate(strat.config.FundingInterval).Add(strat.config.FundingInterval).Sub(time.Now()))
				strat.fundingRateSettleSilent = true
				strat.fundingRateSettleTimer.Reset(strat.config.FundingRateSilentTime + time.Second)
			} else {
				strat.fundingRateSettleSilent = false
				strat.fundingRateSettleTimer.Reset(time.Now().Truncate(strat.config.FundingInterval).Add(strat.config.FundingInterval - time.Second).Sub(time.Now()))
			}
			break
		case <-strat.saveTimer.C:
			strat.handleSave()
			break
		case <-strat.quantileSaveTimer.C:
			strat.handleQuantileSave()
			strat.quantileSaveTimer.Reset(strat.config.QuantileSaveInterval)
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
func (strat *XYStrategy) handleSave() {
	strat.saveTimer.Reset(strat.config.InternalInflux.SaveInterval)
	if strat.xSystemStatus != common.SystemStatusReady ||
		strat.ySystemStatus != common.SystemStatusReady ||
		strat.xPosition == nil ||
		strat.yPosition == nil ||
		strat.xAccount == nil ||
		strat.yAccount == nil {
		return
	}
	select {
	case strat.saveCh <- strat:
	default:
		logger.Debugf("strat.saveCh <- strat failed %s %s ch len %d", strat.xSymbol, strat.ySymbol, len(strat.saveCh))
	}
}

func (strat *XYStrategy) hedgeYPosition() {
	if strat.xSystemStatus != common.SystemStatusReady ||
		strat.ySystemStatus != common.SystemStatusReady {
		if time.Now().Sub(strat.logSilentTime) > 0 {
			strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
			logger.Debugf("hedgeYPosition xSystemStatus %v ySystemStatus %v", strat.xSystemStatus, strat.ySystemStatus)
		}
		return
	}
	if strat.yPosition == nil ||
		strat.xPosition == nil ||
		strat.spread == nil ||
		time.Now().Sub(strat.yPositionUpdateTime) > strat.config.BalancePositionMaxAge ||
		time.Now().Sub(strat.yOrderSilentTime) < 0 {
		//if time.Now().Sub(strat.logSilentTime) > 0 {
		//	strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
		//	logger.Debugf("hedgeYPosition skipped order silent time %v positionUpdateTime %v", time.Now().Sub(strat.yOrderSilentTime), time.Now().Sub(strat.yPositionUpdateTime))
		//}
		return
	}
	strat.ySizeDiff = -strat.xPosition.GetSize()*strat.xMultiplier/strat.yMultiplier - strat.yPosition.GetSize()
	if math.Abs(strat.ySizeDiff) < strat.yStepSize {
		return
	}
	//如y下单也加上控制，以限下单太大，造成市场冲击
	if strat.ySizeDiff*strat.yMultiplier < -strat.maxOrderValue/strat.yTicker.GetBidPrice() {
		strat.ySizeDiff = -strat.maxOrderValue / strat.yTicker.GetBidPrice() / strat.yMultiplier
	} else if strat.ySizeDiff*strat.yMultiplier > strat.maxOrderValue/strat.yTicker.GetAskPrice() {
		strat.ySizeDiff = strat.maxOrderValue / strat.yTicker.GetAskPrice() / strat.yMultiplier
	}
	strat.ySizeDiff = math.Floor(strat.ySizeDiff/strat.yStepSize) * strat.yStepSize

	if strat.isYSpot {
		if math.Abs(strat.ySizeDiff) < strat.yStepSize {
			return
		} else if strat.ySizeDiff < 0 && -strat.ySizeDiff*strat.yMultiplier*strat.yTicker.GetBidPrice() < strat.yMinNotional {
			return
		} else if strat.ySizeDiff > 0 && strat.ySizeDiff*strat.yMultiplier*strat.yTicker.GetAskPrice() < strat.yMinNotional {
			return
		}
	} else {
		//期货以close仓位，没有minNotional限制
		if math.Abs(strat.ySizeDiff) < strat.yStepSize {
			return
		} else if strat.ySizeDiff < 0 && strat.yPosition.GetSize() <= 0 && -strat.ySizeDiff*strat.yMultiplier*strat.yTicker.GetBidPrice() < strat.yMinNotional {
			return
		} else if strat.ySizeDiff > 0 && strat.yPosition.GetSize() >= 0 && strat.ySizeDiff*strat.yMultiplier*strat.yTicker.GetAskPrice() < strat.yMinNotional {
			return
		}
	}

	strat.reduceOnly = false
	if strat.ySizeDiff*strat.yPosition.GetSize() < 0 && math.Abs(strat.ySizeDiff)*0.995 <= math.Abs(strat.yPosition.GetSize()) {
		strat.reduceOnly = true
	}
	strat.orderSide = common.OrderSideBuy
	if strat.ySizeDiff < 0 {
		strat.orderSide = common.OrderSideSell
		strat.ySizeDiff = -strat.ySizeDiff
	}
	strat.yNewOrderParam = common.NewOrderParam{
		Symbol:     strat.ySymbol,
		Side:       strat.orderSide,
		Type:       common.OrderTypeMarket,
		Size:       strat.ySizeDiff,
		ReduceOnly: strat.reduceOnly,
		ClientID:   strat.yExchange.GenerateClientID(),
	}
	if !strat.config.DryRun {
		select {
		case strat.yOrderRequestCh <- common.OrderRequest{
			New: &strat.yNewOrderParam,
		}:
			strat.yOrderSilentTime = time.Now().Add(strat.config.YOrderSilent)
			strat.yPositionUpdateTime = time.Unix(0, 0)
		}
	} else {
		strat.yOrderSilentTime = time.Now().Add(strat.config.YOrderSilent)
		strat.yPositionUpdateTime = time.Unix(0, 0)
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
	strat.usdAvailable = math.Min(strat.xAccount.GetFree()*strat.xLeverage, strat.yAccount.GetFree()*strat.yLeverage)
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
			//logger.Debugf("%s %v %v %f %f", strat.xSymbol, nextPos.GetEventTime(), strat.xPosition.GetEventTime(), math.Abs(strat.xPosition.GetSize()-nextPos.GetSize()), strat.xStepSize)
			if math.Abs(strat.xPosition.GetSize()-nextPos.GetSize()) >= strat.xStepSize {
				strat.xOrderSilentTime = time.Now().Add(strat.config.XOrderSilent)
				strat.yOrderSilentTime = time.Now()
				if strat.xTicker != nil {
					strat.xTimedPositionChange.Insert(time.Now(), math.Abs(strat.xPosition.GetSize()-nextPos.GetSize())*strat.xMidPrice*strat.xMultiplier)
				}
				logger.Debugf("%s x position change %f -> %f %v", nextPos.GetSymbol(), strat.xPosition.GetSize(), nextPos.GetSize(), nextPos.GetEventTime())
				strat.xPosition = nextPos
				if time.Now().Sub(strat.hedgeCheckStopTime) > 0 {
					strat.hedgeYPosition()
				}else{
					strat.hedgeCheckTimer.Reset(strat.config.HedgeDelay)
				}
			} else {
				strat.xPosition = nextPos
				if time.Now().Sub(strat.hedgeCheckStopTime) > 0 {
					strat.hedgeYPosition()
				}else{
					strat.hedgeCheckTimer.Reset(strat.config.HedgeDelay)
				}
			}
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
		if nextPos.GetEventTime().Sub(strat.yPosition.GetEventTime()) >= -time.Second {
			if math.Abs(strat.yPosition.GetSize()-nextPos.GetSize()) >= strat.yStepSize {
				if strat.yTicker != nil {
					strat.yTimedPositionChange.Insert(time.Now(), math.Abs(strat.yPosition.GetSize()-nextPos.GetSize())*strat.yMidPrice*strat.yMultiplier)
				}
				logger.Debugf("%s y position change %f -> %f %v", nextPos.GetSymbol(), strat.yPosition.GetSize(), nextPos.GetSize(), nextPos.GetEventTime())
			}
			strat.yPosition = nextPos
		}
		strat.yPositionUpdateTime = nextPos.GetParseTime()
	} else {
		strat.yPosition = nextPos
		strat.yPositionUpdateTime = nextPos.GetParseTime()
		logger.Debugf("%s y position change nil -> %f", nextPos.GetSymbol(), nextPos.GetSize())
	}
}

func (strat *XYStrategy) handleQuantileSave() {
	if strat.config.QuantilePath != "" {
		strat.quantileBytes, strat.error = json.Marshal(strat.timedTDigest)
		if strat.error != nil {
			logger.Debugf("json.Marshal(strat.timedTDigest) error %v", strat.error)
		} else {
			strat.quantileFile, strat.error = os.OpenFile(path.Join(strat.config.QuantilePath, strat.xSymbol+"-"+strat.ySymbol+".json"), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
			if strat.error != nil {
				logger.Debugf("os.OpenFile error %v", strat.error)
			} else {
				_, strat.error = strat.quantileFile.Write(strat.quantileBytes)
				if strat.error != nil {
					logger.Debugf("strat.file.Write error %v", strat.error)
				} else {
					strat.error = strat.quantileFile.Close()
					if strat.error != nil {
						logger.Debugf("strat.file.Close() error %v", strat.error)
					}
				}
			}
		}
	}
}
