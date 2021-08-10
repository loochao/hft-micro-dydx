package main

import (
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"time"
)

func (strat *XYStrategy) updatePosition() {
	if strat.xSystemStatus != common.SystemStatusReady {
		if time.Now().Sub(strat.logSilentTime) > 0 {
			strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
			logger.Debugf("updatePosition xSystemStatus %v", strat.xSystemStatus)
		}
		return
	}

	if time.Now().Sub(strat.xPositionUpdateTime) > strat.config.BalancePositionMaxAge ||
		strat.xTicker == nil ||
		strat.xPosition == nil ||
		strat.xAccount == nil {
		if time.Now().Sub(strat.logSilentTime) > 0 {
			strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
			logger.Debugf("%s %v %v %v %v",
				strat.xSymbol, time.Now().Sub(strat.xPositionUpdateTime) > strat.config.BalancePositionMaxAge,
				strat.xTicker,
				strat.xPosition,
				strat.xAccount,
			)
			//logger.Debugf("updatePosition xSystemStatus %v", strat.xSystemStatus)
		}
		return
	}

	if strat.xPosition.GetSize() == 0 {
		return
	}

	if time.Now().Sub(strat.OrderSilentTime) < 0 {
		return
	}

	strat.xSize = strat.xPosition.GetSize() * strat.xMultiplier
	strat.xValue = strat.xSize * strat.xMidPrice
	strat.xAbsValue = math.Abs(strat.xValue)

	strat.enterValue = math.Min(strat.config.OrderValue, strat.xAbsValue)
	strat.xOrderSize = strat.enterValue / strat.xMidPrice

	if strat.xAbsValue-strat.enterValue < strat.xStepSize*strat.xMultiplier*1.005 ||
		strat.xOrderSize > math.Abs(strat.xSize) {
		//两种情况都把x全平
		strat.xOrderSize = math.Abs(strat.xSize)

	}
	strat.enterValue = strat.xOrderSize * strat.xMidPrice

	strat.xOrderSize = math.Floor(strat.xOrderSize/strat.xMultiplier/strat.xStepSize) * strat.xStepSize
	if strat.xSize > 0 {
		strat.xOrderSide = common.OrderSideSell
	} else {
		strat.xOrderSide = common.OrderSideBuy
	}
	if !strat.isXSpot || strat.enterValue > 1.005*strat.xMinNotional {
		strat.xNewOrderParam = common.NewOrderParam{
			Symbol:     strat.xSymbol,
			Side:       strat.xOrderSide,
			Size:       strat.xOrderSize,
			Type:       common.OrderTypeMarket,
			PostOnly:   false,
			ReduceOnly: true,
			ClientID:   strat.xExchange.GenerateClientID(),
		}
		if !strat.config.DryRun {
			select {
			case strat.xOrderRequestCh <- common.OrderRequest{
				New: &strat.xNewOrderParam,
			}:
			}
		}
		strat.OrderSilentTime = time.Now().Add(strat.config.OrderSilent)
		logger.Debugf(
			"%s CLOSE %s %f",
			strat.xSymbol,
			strat.xOrderSide, strat.xOrderSize,
		)
	}
}
