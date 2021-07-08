package main

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
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

	strat := XYStrategy{
		xExchange:           xExchange,
		yExchange:           yExchange,
		xExchangeID:         xExchange.GetExchange(),
		yExchangeID:         yExchange.GetExchange(),
		isXSpot:             xExchange.IsSpot(),
		isYSpot:             yExchange.IsSpot(),
		xLeverage:           config.XExchange.Leverage,
		yLeverage:           config.YExchange.Leverage,
		xSymbol:             xSymbol,
		ySymbol:             ySymbol,
		config:              config,
		xAccountCh:          xAccountCh,
		yAccountCh:          yAccountCh,
		xPositionCh:         xPositionCh,
		yPositionCh:         yPositionCh,
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
		xDepth:              nil,
		yDepth:              nil,
		xDepthTime:          time.Time{},
		yDepthTime:          time.Time{},
		xAccount:            nil,
		yAccount:            nil,
		xPosition:           nil,
		yPosition:           nil,
		xOrderSilentTime:    time.Time{},
		yOrderSilentTime:    time.Time{},
		xOrder:              nil,
		yOrder:              nil,
		xOrderError:         common.OrderError{},
		yOrderError:         common.OrderError{},
		logSilentTime:       time.Time{},
		saveTimer:           time.NewTimer(config.UpdateTargetSilent),
		stateOutputCh:       nil,
		error:               nil,
		xSizeDiff:           0,
		ySizeDiff:           0,
		xSize:               0,
		ySize:               0,
		xValue:              0,
		yValue:              0,
		xAbsValue:           0,
		yAbsValue:           0,
		midPrice:            0,
		size:                0,
		orderSide:           common.OrderSideUnknown,
		stopped:             0,
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
		logger.Debugf("stopped")
	}
}

func (strat *XYStrategy) startLoop(ctx context.Context) {
	defer strat.saveTimer.Stop()
	defer strat.Stop()
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
			strat.handleSave()
			break
		case strat.xAccount = <-strat.xAccountCh:
			break
		case strat.yAccount = <-strat.yAccountCh:
			break
		case nextXPos = <-strat.xPositionCh:
			strat.handleXPosition(nextXPos)
			break
		case nextYPos = <-strat.yPositionCh:
			strat.handleYPosition(nextYPos)
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
		case strat.xNextDepth = <-strat.xDepthCh:
			strat.handleXDepth()
			break
		case strat.yNextDepth = <-strat.yDepthCh:
			strat.handleYDepth()
			break
		}
	}
}
func (strat *XYStrategy) handleSave() {
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
	strat.saveTimer.Reset(strat.config.InternalInflux.SaveInterval)
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
			if math.Abs(strat.xPosition.GetSize()-nextPos.GetSize()) >= strat.xStepSize {
				logger.Debugf("%s x position change %f -> %f %v", nextPos.GetSymbol(), strat.xPosition.GetSize(), nextPos.GetSize(), nextPos.GetEventTime())
				strat.xPosition = nextPos
				strat.hedgeYPosition()
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
			if math.Abs(strat.yPosition.GetSize()-nextPos.GetSize()) >= strat.xStepSize {
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
