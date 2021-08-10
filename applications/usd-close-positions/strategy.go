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

	xSymbol string,

	config Config,

	xExchange common.UsdExchange,

	xAccountCh chan common.Balance,

	xPositionCh chan common.Position,

	xOrderRequestCh chan common.OrderRequest,

	xOrderCh chan common.Order,

	xOrderErrorCh chan common.OrderError,

	xSystemStatusCh chan common.SystemStatus,

	xTickerCh chan common.Ticker,

) (err error) {

	strat := XYStrategy{

		xExchange: xExchange,

		isXSpot: xExchange.IsSpot(),

		xSymbol: xSymbol,

		config: config,

		xAccountCh: xAccountCh,

		xPositionCh: xPositionCh,

		xOrderCh: xOrderCh,

		xOrderErrorCh: xOrderErrorCh,

		xOrderRequestCh: xOrderRequestCh,

		xSystemStatusCh: xSystemStatusCh,

		xTickerCh: xTickerCh,

		xPositionUpdateTime: time.Time{},

		xTicker: nil,

		xNextTicker: nil,

		xTickerTime: time.Time{},

		xAccount: nil,

		xPosition: nil,

		xOrder: nil,

		xOrderError: common.OrderError{},

		logSilentTime: time.Time{},
		error:         nil,

		xSize: 0,

		xValue: 0,

		xAbsValue: 0,

		enterValue: 0,

		xOrderSize: 0,

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

	go strat.startLoop(ctx)
	return
}

func (strat *XYStrategy) Stop() {
	if atomic.CompareAndSwapInt32(&strat.stopped, 0, 1) {
		logger.Debugf("stopped %s", strat.xSymbol)
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

		case <-updateTimer.C:
			strat.updatePosition()
			updateTimer.Reset(time.Millisecond*time.Duration(rand.Intn(10000)+5000))
			break

		case strat.xNextTicker = <-strat.xTickerCh:
			strat.handleXTicker()
			break

		case strat.xAccount = <-strat.xAccountCh:
			break

		case nextXPos = <-strat.xPositionCh:
			strat.handleXPosition(nextXPos)
			break

		case strat.xOrder = <-strat.xOrderCh:
			strat.handleXOrder()
			break

		case strat.xOrderError = <-strat.xOrderErrorCh:
			strat.handleXOrderError()
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

