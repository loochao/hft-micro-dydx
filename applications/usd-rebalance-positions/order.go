package main

import (
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"time"
)

func (strat *XYStrategy) updatePosition() {
	if strat.xSystemStatus != common.SystemStatusReady ||
		strat.ySystemStatus != common.SystemStatusReady {
		if time.Now().Sub(strat.logSilentTime) > 0 {
			strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
			logger.Debugf("updatePosition xSystemStatus %v ySystemStatus %v", strat.xSystemStatus, strat.ySystemStatus)
		}
		return
	}

	if time.Now().Sub(strat.xPositionUpdateTime) > strat.config.BalancePositionMaxAge ||
		time.Now().Sub(strat.yPositionUpdateTime) > strat.config.BalancePositionMaxAge ||
		strat.xTicker == nil ||
		strat.yTicker == nil ||
		strat.xPosition == nil ||
		strat.yPosition == nil ||
		strat.xAccount == nil ||
		strat.yAccount == nil {
		if time.Now().Sub(strat.logSilentTime) > 0 {
			strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
			logger.Debugf("%s %v %v %v %v %v %v %v %v",
				strat.xSymbol,
				time.Now().Sub(strat.xPositionUpdateTime) > strat.config.BalancePositionMaxAge,
				time.Now().Sub(strat.yPositionUpdateTime) > strat.config.BalancePositionMaxAge,
				strat.xTicker == nil,
				strat.yTicker == nil,
				strat.xPosition == nil,
				strat.yPosition == nil,
				strat.xAccount == nil,
				strat.yAccount == nil,
			)
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
	strat.ySize = strat.yPosition.GetSize() * strat.yMultiplier
	strat.xValue = strat.xSize * strat.xMidPrice
	strat.yValue = strat.ySize * strat.yMidPrice
	strat.xAbsValue = math.Abs(strat.xValue)
	strat.yAbsValue = math.Abs(strat.yValue)

	strat.enterValue = math.Min(strat.config.OrderValue, strat.xAbsValue)
	strat.xOrderSize = strat.enterValue * 2.0 / (strat.xMidPrice + strat.yMidPrice)

	if strat.xAbsValue-strat.enterValue < strat.xStepSize*strat.xMultiplier*1.005 ||
		strat.xOrderSize > math.Abs(strat.xSize) {
		//两种情况都把x全平
		strat.xOrderSize = math.Abs(strat.xSize)

	}
	strat.enterValue = strat.xOrderSize * (strat.xMidPrice + strat.yMidPrice) * 0.5

	if strat.enterValue > strat.usdAvailable {
		return
	}
	strat.yOrderSize = strat.xOrderSize
	strat.xOrderSize = math.Floor(strat.xOrderSize/strat.xMultiplier/strat.xStepSize) * strat.xStepSize
	strat.yOrderSize = math.Floor(strat.yOrderSize/strat.yMultiplier/strat.yStepSize) * strat.yStepSize
	if strat.xSize > 0 {
		strat.xOrderSide = common.OrderSideSell
		strat.yOrderSide = common.OrderSideBuy
	} else {
		strat.xOrderSide = common.OrderSideBuy
		strat.yOrderSide = common.OrderSideSell
	}
	if !strat.isXSpot || (strat.enterValue > 1.2*strat.xMinNotional && strat.enterValue > 1.2*strat.yMinNotional) {
		strat.xNewOrderParam = common.NewOrderParam{
			Symbol:     strat.xSymbol,
			Side:       strat.xOrderSide,
			Size:       strat.xOrderSize,
			Type:       common.OrderTypeMarket,
			PostOnly:   false,
			ReduceOnly: true,
			ClientID:   strat.xExchange.GenerateClientID(),
		}
		strat.yNewOrderParam = common.NewOrderParam{
			Symbol:     strat.ySymbol,
			Side:       strat.yOrderSide,
			Size:       strat.yOrderSize,
			Type:       common.OrderTypeMarket,
			PostOnly:   false,
			ReduceOnly: false,
			ClientID:   strat.yExchange.GenerateClientID(),
		}
		if !strat.config.DryRun {
			select {
			case strat.xOrderRequestCh <- common.OrderRequest{
				New: &strat.xNewOrderParam,
			}:
			}
			select {
			case strat.yOrderRequestCh <- common.OrderRequest{
				New: &strat.yNewOrderParam,
			}:
			}
		}
		strat.OrderSilentTime = time.Now().Add(strat.config.OrderSilent)
		logger.Debugf(
			"%s %s SWAP X %s %f Y %s %f",
			strat.xSymbol, strat.ySymbol,
			strat.xOrderSide, strat.xOrderSize,
			strat.yOrderSide, strat.yOrderSize,
		)
	}
}
