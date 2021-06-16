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
			strat.xOrderSilentTime = time.Now().Add(time.Second)
			strat.xPositionUpdateTime = time.Unix(0, 0)
		} else {
			logger.Debugf("x order filled %s %s %s size %f price %f", strat.xSymbol, strat.xOrder.GetStatus(), strat.xOrder.GetSide(), strat.xOrder.GetFilledSize(), strat.xOrder.GetFilledPrice())
			if strat.xOrder.GetSide() == common.OrderSideBuy {
				strat.xLastFilledBuyPrice = new(float64)
				*strat.xLastFilledBuyPrice = strat.xOrder.GetFilledPrice()
				if strat.yLastFilledSellPrice != nil {
					if strat.realisedSpread == nil {
						strat.realisedSpread = new(float64)
					}
					*strat.realisedSpread = (*strat.yLastFilledSellPrice - *strat.xLastFilledBuyPrice) / *strat.yLastFilledSellPrice
					strat.xLastFilledBuyPrice = nil
					strat.yLastFilledBuyPrice = nil
					strat.xLastFilledSellPrice = nil
					strat.yLastFilledSellPrice = nil
					logger.Debugf("%s - %s realised short spread %f", strat.xSymbol, strat.ySymbol, *strat.realisedSpread)
				}
			} else if strat.xOrder.GetSide() == common.OrderSideSell {
				strat.xLastFilledSellPrice = new(float64)
				*strat.xLastFilledSellPrice = strat.xOrder.GetFilledPrice()
				if strat.yLastFilledBuyPrice != nil {
					if strat.realisedSpread == nil {
						strat.realisedSpread = new(float64)
					}
					*strat.realisedSpread = (*strat.yLastFilledBuyPrice - *strat.xLastFilledSellPrice) / *strat.yLastFilledBuyPrice
					strat.xLastFilledBuyPrice = nil
					strat.yLastFilledBuyPrice = nil
					strat.xLastFilledSellPrice = nil
					strat.yLastFilledSellPrice = nil
					logger.Debugf("%s - %s realised long spread %f", strat.ySymbol, strat.xSymbol, *strat.realisedSpread)
				}
			}
		}
	}
}

func (strat *XYStrategy) handleYOrder() {
	if strat.yOrder.GetSymbol() != strat.ySymbol {
		logger.Debugf("bad y order symbol not match %s %s %v", strat.yOrder.GetSymbol(), strat.ySymbol, strat.yOrder)
	}
	if strat.yOrder.GetStatus() == common.OrderStatusExpired ||
		strat.yOrder.GetStatus() == common.OrderStatusReject ||
		strat.yOrder.GetStatus() == common.OrderStatusCancelled ||
		strat.yOrder.GetStatus() == common.OrderStatusFilled {
		if strat.yOrder.GetStatus() != common.OrderStatusFilled {
			logger.Debugf("y order ended %s %s %s", strat.yOrder.GetSymbol(), strat.yOrder.GetStatus(), strat.yOrder.GetSide())
			strat.yOrderSilentTime = time.Now().Add(time.Second)
			strat.yPositionUpdateTime = time.Time{}
		} else {
			logger.Debugf("y order filled %s %s %s size %f price %f", strat.yOrder.GetSymbol(), strat.yOrder.GetStatus(), strat.yOrder.GetSide(), strat.yOrder.GetFilledSize(), strat.yOrder.GetFilledPrice())
			if strat.yOrder.GetSide() == common.OrderSideBuy {
				strat.yLastFilledBuyPrice = new(float64)
				*strat.yLastFilledBuyPrice = strat.yOrder.GetFilledPrice()
				if strat.xLastFilledSellPrice != nil {
					if strat.realisedSpread == nil {
						strat.realisedSpread = new(float64)
					}
					*strat.realisedSpread = (*strat.yLastFilledBuyPrice - *strat.xLastFilledSellPrice) / *strat.yLastFilledBuyPrice
					logger.Debugf("%s - %s realised long spread %f", strat.ySymbol, strat.xSymbol, *strat.realisedSpread)
					strat.xLastFilledBuyPrice = nil
					strat.yLastFilledBuyPrice = nil
					strat.xLastFilledSellPrice = nil
					strat.yLastFilledSellPrice = nil
				}
			} else if strat.yOrder.GetSide() == common.OrderSideSell {
				strat.yLastFilledSellPrice = new(float64)
				*strat.yLastFilledSellPrice = strat.yOrder.GetFilledPrice()
				if strat.xLastFilledSellPrice != nil {
					if strat.realisedSpread == nil {
						strat.realisedSpread = new(float64)
					}
					*strat.realisedSpread = (*strat.yLastFilledSellPrice - *strat.xLastFilledBuyPrice)/ *strat.yLastFilledSellPrice
					logger.Debugf("%s - %s realised short spread %f", strat.ySymbol, strat.xSymbol, *strat.realisedSpread)
					strat.xLastFilledBuyPrice = nil
					strat.yLastFilledBuyPrice = nil
					strat.xLastFilledSellPrice = nil
					strat.yLastFilledSellPrice = nil
				}
			}
		}
	}
}

func (strat *XYStrategy) handleXOrderError() {
	if strat.xOrderError.Cancel != nil {
		logger.Debugf("cancel %v error %v", *strat.xOrderError.Cancel, strat.xOrderError.Error)
		strat.xOrderSilentTime = time.Now().Add(strat.params.orderSilent)
	} else if strat.xOrderError.New != nil {
		logger.Debugf("new %v error %v", *strat.xOrderError.New, strat.xOrderError.Error)
		strat.xOrderSilentTime = time.Now().Add(strat.params.orderSilent)
	}
}

func (strat *XYStrategy) handleYOrderError() {
	if strat.yOrderError.Cancel != nil {
		logger.Debugf("cancel %v error %v", *strat.yOrderError.Cancel, strat.yOrderError.Error)
		strat.yOrderSilentTime = time.Now().Add(strat.params.orderSilent)
	} else if strat.yOrderError.New != nil {
		logger.Debugf("new %v error %v", *strat.yOrderError.New, strat.yOrderError.Error)
		strat.yOrderSilentTime = time.Now().Add(strat.params.orderSilent)
	}
}
