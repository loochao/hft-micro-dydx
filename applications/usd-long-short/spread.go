package main

import (
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"time"
)



func (strat *XYStrategy) handleXDepth() {
	if strat.xDepth == strat.xNextDepth {
		return
	}
	if strat.xNextDepth.GetTime().Sub(strat.xDepthTime) < 0 {
		return
	}
	strat.xDepth = strat.xNextDepth
	strat.xDepthTime = strat.xDepth.GetTime()
	strat.updateTargetPositionSize()
	strat.hedgeXPosition()
	strat.hedgeYPosition()
}

func (strat *XYStrategy) handleYDepth() {
	if strat.yDepth == strat.yNextDepth {
		return
	}
	if strat.yNextDepth.GetTime().Sub(strat.yDepthTime) < 0 {
		return
	}
	strat.yDepth = strat.yNextDepth
	strat.yDepthTime = strat.yDepth.GetTime()
	strat.updateTargetPositionSize()
	strat.hedgeXPosition()
	strat.hedgeYPosition()
}

func (strat *XYStrategy) updateTargetPositionSize() {
	if strat.xSystemStatus != common.SystemStatusReady ||
		strat.ySystemStatus != common.SystemStatusReady {
		if time.Now().Sub(strat.logSilentTime) > 0 {
			strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
			logger.Debugf("updateTargetPositionSize failed xSystemStatus %v ySystemStatus %v", strat.xSystemStatus, strat.ySystemStatus)
		}
		return
	}
	if time.Now().Sub(strat.xOrderSilentTime) < 0 ||
		time.Now().Sub(strat.yOrderSilentTime) < 0 ||
		time.Now().Sub(strat.xPositionUpdateTime) > strat.config.BalancePositionMaxAge ||
		time.Now().Sub(strat.yPositionUpdateTime) > strat.config.BalancePositionMaxAge ||
		time.Now().Sub(strat.updateTargetSilentTime) < 0 ||
		strat.xAccount == nil ||
		strat.yAccount == nil ||
		strat.xPosition == nil ||
		strat.yPosition == nil ||
		strat.xDepth == nil ||
		strat.yDepth == nil {
		if time.Now().Sub(strat.logSilentTime) > 0 {
			strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
			logger.Debugf(
				"%v %v %v %v %v %v %v %v %v %v %v",
				time.Now().Sub(strat.xOrderSilentTime) < 0,
				time.Now().Sub(strat.yOrderSilentTime) < 0,
				time.Now().Sub(strat.xPositionUpdateTime) > strat.config.BalancePositionMaxAge,
				time.Now().Sub(strat.yPositionUpdateTime) > strat.config.BalancePositionMaxAge,
				time.Now().Sub(strat.updateTargetSilentTime) < 0,
				strat.xAccount == nil,
				strat.yAccount == nil,
				strat.xPosition == nil,
				strat.yPosition == nil,
				strat.xDepth == nil,
				strat.yDepth == nil,
			)
		}
		return
	}
	strat.midPrice = (strat.xDepth.GetBids()[0][0] + strat.xDepth.GetAsks()[0][0] + strat.yDepth.GetBids()[0][0] + strat.yDepth.GetAsks()[0][0]) * 0.25
	strat.xSize = strat.xPosition.GetSize() * strat.xMultiplier
	strat.ySize = strat.yPosition.GetSize() * strat.yMultiplier
	strat.xFreeSize = strat.xAccount.GetFree() * strat.xLeverage * 0.8 / strat.midPrice
	strat.yFreeSize = strat.yAccount.GetFree() * strat.yLeverage * 0.8 / strat.midPrice
	strat.xAbsSize = math.Abs(strat.xSize) + strat.xFreeSize
	strat.yAbsSize = math.Abs(strat.ySize) + strat.yFreeSize
	strat.xValue = strat.xSize * strat.midPrice
	strat.yValue = strat.ySize * strat.midPrice
	strat.xAbsValue = math.Abs(strat.xValue)
	strat.yAbsValue = math.Abs(strat.yValue)
	if strat.xyTargetSize == nil {
		strat.xyTargetSize = new(float64)
	}
	*strat.xyTargetSize = strat.config.EnterTarget / strat.midPrice
	*strat.xyTargetSize = math.Min(*strat.xyTargetSize, strat.xAbsSize)
	*strat.xyTargetSize = math.Min(*strat.xyTargetSize, strat.yAbsSize)
	strat.updateTargetSilentTime = time.Now().Add(strat.config.UpdateTargetSilent)
	logger.Debugf("xyTargetSize %v %f", *strat.xyTargetSize, strat.midPrice)
}
