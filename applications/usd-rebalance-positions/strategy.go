package main

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"math/rand"
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

	xTickerCh chan common.Ticker,
	yTickerCh chan common.Ticker,

) (err error) {

	strat := XYStrategy{

		xExchange: xExchange,
		yExchange: yExchange,

		isXSpot: xExchange.IsSpot(),
		isYSpot: yExchange.IsSpot(),

		xSymbol: xSymbol,
		ySymbol: ySymbol,

		config: config,

		xAccountCh: xAccountCh,
		yAccountCh: yAccountCh,

		xPositionCh: xPositionCh,
		yPositionCh: yPositionCh,

		xOrderCh: xOrderCh,
		yOrderCh: yOrderCh,

		xOrderErrorCh: xOrderErrorCh,
		yOrderErrorCh: yOrderErrorCh,

		xOrderRequestCh: xOrderRequestCh,
		yOrderRequestCh: yOrderRequestCh,

		xSystemStatusCh: xSystemStatusCh,
		ySystemStatusCh: ySystemStatusCh,

		xTickerCh: xTickerCh,
		yTickerCh: yTickerCh,

		xPositionUpdateTime: time.Time{},
		yPositionUpdateTime: time.Time{},

		xTicker: nil,
		yTicker: nil,

		xNextTicker: nil,
		yNextTicker: nil,

		xTickerTime: time.Time{},
		yTickerTime: time.Time{},

		xAccount: nil,
		yAccount: nil,

		xPosition: nil,
		yPosition: nil,

		xOrder: nil,
		yOrder: nil,

		xOrderError: common.OrderError{},
		yOrderError: common.OrderError{},

		usdAvailable:  0,
		logSilentTime: time.Time{},
		error:         nil,

		xSize: 0,
		ySize: 0,

		xValue: 0,
		yValue: 0,

		xAbsValue: 0,
		yAbsValue: 0,

		enterValue: 0,

		xOrderSize: 0,
		yOrderSize: 0,

		stopped: 0,
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

	go strat.startLoop(ctx)
	return
}

func (strat *XYStrategy) Stop() {
	if atomic.CompareAndSwapInt32(&strat.stopped, 0, 1) {
		logger.Debugf("stopped %s %s", strat.xSymbol, strat.ySymbol)
	}
}

func (strat *XYStrategy) startLoop(ctx context.Context) {
	defer strat.Stop()
	var nextXPos common.Position
	strat.OrderSilentTime = time.Now().Add(strat.config.RestartSilent)
	updateTimer := time.NewTimer(time.Millisecond*time.Duration(rand.Intn(10000)+5000))
	for {
		select {
		case <-ctx.Done():
			return
		case strat.xSystemStatus = <-strat.xSystemStatusCh:
			if strat.xSystemStatus != common.SystemStatusReady {
				strat.OrderSilentTime = time.Now().Add(strat.config.RestartSilent)
			}
			break
		case strat.ySystemStatus = <-strat.ySystemStatusCh:
			if strat.ySystemStatus != common.SystemStatusReady {
				strat.OrderSilentTime = time.Now().Add(strat.config.RestartSilent)
			}
			break

		case <-updateTimer.C:
			strat.updatePosition()
			updateTimer.Reset(time.Millisecond*time.Duration(rand.Intn(10000)+5000))
			break

		case strat.xNextTicker = <-strat.xTickerCh:
			strat.handleXTicker()
			break

		case strat.yNextTicker = <-strat.yTickerCh:
			strat.handleYTicker()
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

		case nextXPos = <-strat.yPositionCh:
			strat.handleYPosition(nextXPos)
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

		}
	}
}

func (strat *XYStrategy) updateEnterStepAndTarget() {
	if strat.yAccount == nil {
		return
	}
	strat.usdAvailable = strat.yAccount.GetFree() * strat.config.YExchange.Leverage
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
			}
			strat.xPosition = nextPos
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
			if math.Abs(strat.yPosition.GetSize()-nextPos.GetSize()) >= strat.yStepSize {
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
