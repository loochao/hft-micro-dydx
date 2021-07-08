package main

import (
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"time"
)

func (strat *XYStrategy) hedgeXPosition() {
	if strat.xSystemStatus != common.SystemStatusReady ||
		strat.ySystemStatus != common.SystemStatusReady {
		if time.Now().Sub(strat.logSilentTime) > 0 {
			strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
			logger.Debugf("hedgeYPosition failed xSystemStatus %v ySystemStatus %v", strat.xSystemStatus, strat.ySystemStatus)
		}
		return
	}
	if strat.yPosition == nil ||
		strat.xPosition == nil ||
		strat.xyTargetSize == nil ||
		time.Now().Sub(strat.yPositionUpdateTime) > strat.config.BalancePositionMaxAge ||
		time.Now().Sub(strat.xPositionUpdateTime) > strat.config.BalancePositionMaxAge ||
		time.Now().Sub(strat.xOrderSilentTime) < 0 {
		return
	}
	strat.xSizeDiff = *strat.xyTargetSize/strat.xMultiplier - strat.xPosition.GetSize()
	if math.Abs(strat.xSizeDiff) < strat.xStepSize {
		return
	}
	strat.xSizeDiff = math.Round(strat.xSizeDiff/strat.xStepSize) * strat.xStepSize

	if strat.isXSpot {
		if math.Abs(strat.xSizeDiff) < strat.xStepSize {
			return
		} else if strat.xSizeDiff < 0 && -strat.xSizeDiff*strat.xMultiplier*strat.xDepth.GetAsks()[0][0]< strat.xMinNotional {
			return
		} else if strat.xSizeDiff > 0 && strat.xSizeDiff*strat.xMultiplier*strat.xDepth.GetAsks()[0][0] < strat.xMinNotional {
			return
		}
	} else {
		//期货以close仓位，没有minNotional限制
		if math.Abs(strat.xSizeDiff) < strat.xStepSize {
			return
		} else if strat.xSizeDiff < 0 && strat.xPosition.GetSize() <= 0 &&
			-strat.xSizeDiff*strat.xMultiplier*strat.xDepth.GetAsks()[0][0] < strat.xMinNotional {
			return
		} else if strat.xSizeDiff > 0 && strat.xPosition.GetSize() >= 0 &&
			strat.xSizeDiff*strat.xMultiplier*strat.xDepth.GetAsks()[0][0] < strat.xMinNotional {
			return
		}
	}

	strat.reduceOnly = false
	if strat.xSizeDiff*strat.xPosition.GetSize() < 0 && math.Abs(strat.xSizeDiff) <= math.Abs(strat.xPosition.GetSize()) {
		strat.reduceOnly = true
	}
	strat.orderSide = common.OrderSideBuy
	if strat.xSizeDiff < 0 {
		strat.orderSide = common.OrderSideSell
		strat.xSizeDiff = -strat.xSizeDiff
	}
	strat.xNewOrderParam = common.NewOrderParam{
		Symbol:     strat.xSymbol,
		Side:       strat.orderSide,
		Type:       common.OrderTypeMarket,
		Size:       strat.xSizeDiff,
		ReduceOnly: strat.reduceOnly,
		ClientID:   strat.xExchange.GenerateClientID(),
	}
	if !strat.config.DryRun {
		select {
		case strat.xOrderRequestCh <- common.OrderRequest{
			New: &strat.xNewOrderParam,
		}:
			strat.xOrderSilentTime = time.Now().Add(strat.config.XOrderSilent)
			strat.xPositionUpdateTime = time.Unix(0, 0)
		}
	} else {
		strat.xOrderSilentTime = time.Now().Add(strat.config.XOrderSilent)
		strat.xPositionUpdateTime = time.Unix(0, 0)
	}
}

func (strat *XYStrategy) hedgeYPosition() {
	//logger.Debugf("hedgeYPosition %s", strat.ySymbol)
	if strat.xSystemStatus != common.SystemStatusReady ||
		strat.ySystemStatus != common.SystemStatusReady {
		if time.Now().Sub(strat.logSilentTime) > 0 {
			strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
			logger.Debugf("hedgeYPosition failed xSystemStatus %v ySystemStatus %v", strat.xSystemStatus, strat.ySystemStatus)
		}
		return
	}
	if strat.yPosition == nil ||
		strat.xPosition == nil ||
		strat.xyTargetSize == nil ||
		time.Now().Sub(strat.yPositionUpdateTime) > strat.config.BalancePositionMaxAge ||
		time.Now().Sub(strat.xPositionUpdateTime) > strat.config.BalancePositionMaxAge ||
		time.Now().Sub(strat.yOrderSilentTime) < 0 {
		return
	}
	strat.ySizeDiff = -*strat.xyTargetSize/strat.yMultiplier - strat.yPosition.GetSize()
	if math.Abs(strat.ySizeDiff) < strat.yStepSize {
		return
	}
	strat.ySizeDiff = math.Round(strat.ySizeDiff/strat.yStepSize) * strat.yStepSize

	if strat.isYSpot {
		if math.Abs(strat.ySizeDiff) < strat.yStepSize {
			return
		} else if strat.ySizeDiff < 0 && -strat.ySizeDiff*strat.yMultiplier*strat.yDepth.GetBids()[0][0] < strat.yMinNotional {
			return
		} else if strat.ySizeDiff > 0 && strat.ySizeDiff*strat.yMultiplier*strat.yDepth.GetBids()[0][0] < strat.yMinNotional {
			return
		}
	} else {
		//期货以close仓位，没有minNotional限制
		if math.Abs(strat.ySizeDiff) < strat.yStepSize {
			return
		} else if strat.ySizeDiff < 0 && strat.yPosition.GetSize() <= 0 && -strat.ySizeDiff*strat.yMultiplier*strat.yDepth.GetBids()[0][0] < strat.yMinNotional {
			return
		} else if strat.ySizeDiff > 0 && strat.yPosition.GetSize() >= 0 && strat.ySizeDiff*strat.yMultiplier*strat.yDepth.GetBids()[0][0] < strat.yMinNotional {
			return
		}
	}

	strat.reduceOnly = false
	if strat.ySizeDiff*strat.yPosition.GetSize() < 0 && math.Abs(strat.ySizeDiff) <= math.Abs(strat.yPosition.GetSize()) {
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
	return
}

