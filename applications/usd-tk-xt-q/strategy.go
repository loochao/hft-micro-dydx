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
	xPositionCh chan common.Position,
	xFundingRateCh chan common.FundingRate,
	yFundingRateCh chan common.FundingRate,
	xOrderRequestCh chan common.OrderRequest,
	xOrderCh chan common.Order,
	xOrderErrorCh chan common.OrderError,
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
		xSymbol:                 xSymbol,
		ySymbol:                 ySymbol,
		config:                  config,
		xAccountCh:              xAccountCh,
		xPositionCh:             xPositionCh,
		xFundingRateCh:          xFundingRateCh,
		yFundingRateCh:          yFundingRateCh,
		xOrderCh:                xOrderCh,
		xOrderErrorCh:           xOrderErrorCh,
		xOrderRequestCh:         xOrderRequestCh,
		xSystemStatusCh:         xSystemStatusCh,
		ySystemStatusCh:         ySystemStatusCh,
		xyTickerCh:              xyTickerCh,
		saveCh:                  saveCh,
		xPositionUpdateTime:     time.Time{},
		xTicker:                 nil,
		yTicker:                 nil,
		xTickerTime:             time.Time{},
		yTickerTime:             time.Time{},
		xTickerFilter:           common.NewTimeFilter(config.TickerXDecay, xBiasInMs, minTimeDeltaInMs, maxTimeDeltaInMs),
		yTickerFilter:           common.NewTimeFilter(config.TickerYDecay, yBiasInMs, minTimeDeltaInMs, maxTimeDeltaInMs),
		xAccount:                nil,
		xPosition:               nil,
		xOrderSilentTime:        time.Now().Add(config.RestartSilent),
		xFundingRate:            nil,
		yFundingRate:            nil,
		xyFundingRate:           nil,
		xOrder:                  nil,
		xOrderError:             common.OrderError{},
		enterStep:               0,
		usdAvailable:            0,
		logSilentTime:           time.Time{},
		spreadWalkTimer:         time.NewTimer(time.Hour * 9999),
		saveTimer:               time.NewTimer(config.RestartSilent),
		fundingRateSettleTimer:  time.NewTimer(time.Now().Truncate(config.FundingInterval).Add(config.FundingInterval - time.Second).Sub(time.Now())),
		spreadTime:              time.Time{},
		spread:                  nil,
		shortEnterTimedMedian:   common.NewTimedMedian(config.SpreadLookback),
		longEnterTimedMedian:    common.NewTimedMedian(config.SpreadLookback),
		xTimedPositionChange:    common.NewTimedSum(config.TurnoverLookback),
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
		shortTop:                0,
		shortBot:                0,
		longBot:                 0,
		longTop:                 0,
		xSize:                   0,
		xValue:                  0,
		xAbsValue:               0,
		midPrice:                0,
		enterValue:              0,
		size:                    0,
		orderSide:               common.OrderSideUnknown,
		stopped:                 0,
		fundingRateSettleSilent: false,
		xExchangeID:             xExchange.GetExchange(),
		yExchangeID:             yExchange.GetExchange(),
		timedTDigest:            timedTDigest,
		quantileSaveTimer:       time.NewTimer(config.QuantileSaveInterval),
		quantileLastSampleTime:  time.Time{},
		quantile50:              quantileMiddle,
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

	go strat.startLoop(ctx)
	return
}

func (strat *XYStrategy) Stop() {
	if atomic.CompareAndSwapInt32(&strat.stopped, 0, 1) {
		strat.handleQuantileSave()
		logger.Debugf("stopped %s %s", strat.xSymbol, strat.ySymbol)
	}
}

func (strat *XYStrategy) startLoop(ctx context.Context) {
	defer strat.spreadWalkTimer.Stop()
	defer strat.saveTimer.Stop()
	defer strat.Stop()
	var nextXPos common.Position
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
		}
	}
}
func (strat *XYStrategy) handleSave() {
	strat.saveTimer.Reset(strat.config.InternalInflux.SaveInterval)
	if strat.xSystemStatus != common.SystemStatusReady ||
		strat.ySystemStatus != common.SystemStatusReady ||
		strat.xPosition == nil ||
		strat.xAccount == nil {
		return
	}
	select {
	case strat.saveCh <- strat:
	default:
		logger.Debugf("strat.saveCh <- strat failed %s %s ch len %d", strat.xSymbol, strat.ySymbol, len(strat.saveCh))
	}
}


func (strat *XYStrategy) updateEnterStepAndTarget() {
	if strat.xAccount == nil {
		return
	}
	strat.enterStep = strat.xAccount.GetFree() * strat.config.EnterFreePct
	if strat.enterStep < strat.config.EnterMinimalStep {
		strat.enterStep = strat.config.EnterMinimalStep
	}
	strat.usdAvailable = strat.xAccount.GetFree() * strat.xLeverage
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
				if strat.xTicker != nil {
					strat.xTimedPositionChange.Insert(time.Now(), math.Abs(strat.xPosition.GetSize()-nextPos.GetSize())*strat.xMidPrice*strat.xMultiplier)
				}
				logger.Debugf("%s x position change %f -> %f %v", nextPos.GetSymbol(), strat.xPosition.GetSize(), nextPos.GetSize(), nextPos.GetEventTime())
				strat.xPosition = nextPos
			} else {
				strat.xPosition = nextPos
			}
		}
		strat.xPositionUpdateTime = nextPos.GetParseTime()
	} else {
		strat.xPosition = nextPos
		strat.xPositionUpdateTime = nextPos.GetParseTime()
		logger.Debugf("%s x position change nil -> %f", nextPos.GetSymbol(), nextPos.GetSize())
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
