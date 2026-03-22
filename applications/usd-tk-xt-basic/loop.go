package main

import (
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func (strat *XYStrategy) handleFundingRate() {
	if strat.xFundingRate == nil || strat.yFundingRate == nil {
		return
	}
	if strat.xyFundingRate == nil {
		strat.xyFundingRate = new(float64)
	}
	*strat.xyFundingRate = strat.yFundingRate.GetFundingRate() - strat.xFundingRate.GetFundingRate()
}

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
			logger.Debugf("x order filled %s %s %s size %f price %f value %f", strat.xSymbol, strat.xOrder.GetStatus(), strat.xOrder.GetSide(), strat.xOrder.GetFilledSize(), strat.xOrder.GetFilledPrice(), strat.xOrder.GetFilledSize()*strat.xOrder.GetFilledPrice()*strat.xMultiplier)
		}
	}
}



func (strat *XYStrategy) handleXOrderError() {
	if strat.xOrderError.Cancel != nil {
		logger.Debugf("cancel %v error %v", *strat.xOrderError.Cancel, strat.xOrderError.Error)
		//strat.xOrderSilentTime = time.Now().Add(strat.config.XOrderSilent)
	} else if strat.xOrderError.New != nil {
		logger.Debugf("new %v error %v", *strat.xOrderError.New, strat.xOrderError.Error)
		strat.xOrderSilentTime = time.Now().Add(strat.config.XOrderSilent)
	}
}
