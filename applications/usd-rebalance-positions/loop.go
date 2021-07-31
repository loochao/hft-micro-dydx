package main

import (
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func (strat *XYStrategy) handleXOrder() {
	if strat.xOrder.GetSymbol() != strat.xSymbol {
		logger.Debugf("bad x order symbol not match %s %s %v", strat.xOrder.GetSymbol(), strat.xSymbol, strat.xOrder)
		return
	}
	if strat.xOrder.GetStatus() == common.OrderStatusExpired ||
		strat.xOrder.GetStatus() == common.OrderStatusReject ||
		strat.xOrder.GetStatus() == common.OrderStatusCancelled ||
		strat.xOrder.GetStatus() == common.OrderStatusFilled {
		if strat.xOrder.GetStatus() != common.OrderStatusFilled {
			logger.Debugf("x order ended %s %s %s", strat.xOrder.GetSymbol(), strat.xOrder.GetStatus(), strat.xOrder.GetSide())
			strat.xPositionUpdateTime = time.Unix(0, 0)
		} else {
			logger.Debugf("x order filled %s %s %s xOrderSize %f price %f value %f", strat.xSymbol, strat.xOrder.GetStatus(), strat.xOrder.GetSide(), strat.xOrder.GetFilledSize(), strat.xOrder.GetFilledPrice(), strat.xOrder.GetFilledSize()*strat.xOrder.GetFilledPrice()*strat.xMultiplier)
		}
	}
}

func (strat *XYStrategy) handleXOrderError() {
	if strat.xOrderError.Cancel != nil {
		logger.Debugf("cancel %v error %v", *strat.xOrderError.Cancel, strat.xOrderError.Error)
	} else if strat.xOrderError.New != nil {
		logger.Debugf("new %v error %v", *strat.xOrderError.New, strat.xOrderError.Error)
		strat.OrderSilentTime = time.Now().Add(strat.config.OrderSilent)
	}
}

func (strat *XYStrategy) handleYOrder() {
	if strat.yOrder.GetSymbol() != strat.ySymbol {
		logger.Debugf("bad y order symbol not match %s %s %v", strat.yOrder.GetSymbol(), strat.ySymbol, strat.yOrder)
		return
	}
	if strat.yOrder.GetStatus() == common.OrderStatusExpired ||
		strat.yOrder.GetStatus() == common.OrderStatusReject ||
		strat.yOrder.GetStatus() == common.OrderStatusCancelled ||
		strat.yOrder.GetStatus() == common.OrderStatusFilled {
		if strat.yOrder.GetStatus() != common.OrderStatusFilled {
			logger.Debugf("y order ended %s %s %s", strat.yOrder.GetSymbol(), strat.yOrder.GetStatus(), strat.yOrder.GetSide())
			strat.yPositionUpdateTime = time.Unix(0, 0)
		} else {
			logger.Debugf("y order filled %s %s %s yOrderSize %f price %f value %f", strat.xSymbol, strat.yOrder.GetStatus(), strat.yOrder.GetSide(), strat.yOrder.GetFilledSize(), strat.yOrder.GetFilledPrice(), strat.yOrder.GetFilledSize()*strat.yOrder.GetFilledPrice()*strat.xMultiplier)
		}
	}
}

func (strat *XYStrategy) handleYOrderError() {
	if strat.yOrderError.Cancel != nil {
		logger.Debugf("y cancel %v error %v", *strat.yOrderError.Cancel, strat.yOrderError.Error)
	} else if strat.yOrderError.New != nil {
		logger.Debugf("y new %v error %v", *strat.yOrderError.New, strat.yOrderError.Error)
		strat.OrderSilentTime = time.Now().Add(strat.config.OrderSilent)
	}
}
