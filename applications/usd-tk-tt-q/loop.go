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
		logger.Debugf("%s bad x order symbol not match %s %v", strat.xOrder.GetSymbol(), strat.xSymbol, strat.xOrder)
		return
	}
	if strat.xOrder.GetStatus() == common.OrderStatusExpired ||
		strat.xOrder.GetStatus() == common.OrderStatusReject ||
		strat.xOrder.GetStatus() == common.OrderStatusCancelled ||
		strat.xOrder.GetStatus() == common.OrderStatusFilled ||
		strat.xOrder.GetStatus() == common.OrderStatusPartiallyFilled {

		if strat.xOrder.GetStatus() != common.OrderStatusFilled &&
			strat.xOrder.GetStatus() != common.OrderStatusPartiallyFilled {
			//logger.Debugf("x order ended %s %s %s", strat.xOrder.GetSymbol(), strat.xOrder.GetStatus(), strat.xOrder.GetSide())
			strat.xPositionUpdateTime = time.Unix(0, 0)
		} else {
			logger.Debugf("%s x order filled %s %s size %f price %f value %f", strat.xSymbol, strat.xOrder.GetStatus(), strat.xOrder.GetSide(), strat.xOrder.GetFilledSize(), strat.xOrder.GetFilledPrice(), strat.xOrder.GetFilledSize()*strat.xOrder.GetFilledPrice()*strat.xMultiplier)
			strat.realisedSpreadTimer.Reset(time.Second * 5)
			if strat.xOrder.GetSide() == common.OrderSideBuy {
				if strat.xLastFilledBuyPrice == nil {
					strat.xLastFilledBuyPrice = new(float64)
				}
				*strat.xLastFilledBuyPrice = strat.xOrder.GetFilledPrice()
				strat.realisedSpreadTimer.Reset(strat.config.RealisedSpreadLogDelay)
			} else if strat.xOrder.GetSide() == common.OrderSideSell {
				if strat.xLastFilledSellPrice == nil {
					strat.xLastFilledSellPrice = new(float64)
				}
				*strat.xLastFilledSellPrice = strat.xOrder.GetFilledPrice()
				strat.realisedSpreadTimer.Reset(strat.config.RealisedSpreadLogDelay)
			}
		}
	}
}

func (strat *XYStrategy) handleYOrder() {
	if strat.yOrder.GetSymbol() != strat.ySymbol {
		logger.Debugf("%s bad y order symbol not match %s %v", strat.yOrder.GetSymbol(), strat.ySymbol, strat.yOrder)
	}
	if strat.yOrder.GetStatus() == common.OrderStatusExpired ||
		strat.yOrder.GetStatus() == common.OrderStatusReject ||
		strat.yOrder.GetStatus() == common.OrderStatusCancelled ||
		strat.yOrder.GetStatus() == common.OrderStatusFilled ||
		strat.yOrder.GetStatus() == common.OrderStatusPartiallyFilled {

		if strat.yOrder.GetStatus() != common.OrderStatusFilled &&
			strat.yOrder.GetStatus() != common.OrderStatusPartiallyFilled {
			logger.Debugf("%s y order ended %s %s", strat.yOrder.GetSymbol(), strat.yOrder.GetStatus(), strat.yOrder.GetSide())
			strat.yOrderSilentTime = time.Now().Add(strat.config.YOrderSilent)
			strat.yPositionUpdateTime = time.Time{}
		} else {
			logger.Debugf("%s y order filled %s %s size %f price %f value %f", strat.yOrder.GetSymbol(), strat.yOrder.GetStatus(), strat.yOrder.GetSide(), strat.yOrder.GetFilledSize(), strat.yOrder.GetFilledPrice(), strat.yOrder.GetFilledPrice()*strat.yOrder.GetFilledSize()*strat.yMultiplier)
			if strat.yOrder.GetSide() == common.OrderSideBuy {
				if strat.yLastFilledBuyPrice == nil {
					strat.yLastFilledBuyPrice = new(float64)
				}
				*strat.yLastFilledBuyPrice = strat.yOrder.GetFilledPrice()
				strat.realisedSpreadTimer.Reset(strat.config.RealisedSpreadLogDelay)
			} else if strat.yOrder.GetSide() == common.OrderSideSell {
				if strat.yLastFilledSellPrice == nil {
					strat.yLastFilledSellPrice = new(float64)
				}
				*strat.yLastFilledSellPrice = strat.yOrder.GetFilledPrice()
				strat.realisedSpreadTimer.Reset(strat.config.RealisedSpreadLogDelay)
			}
		}
	}
}

func (strat *XYStrategy) handleRealisedSpread() {
	if strat.xLastFilledBuyPrice != nil && strat.yLastFilledSellPrice != nil {
		if strat.realisedSpread == nil {
			strat.realisedSpread = new(float64)
		}
		*strat.realisedSpread = (*strat.yLastFilledSellPrice - *strat.xLastFilledBuyPrice) / *strat.yLastFilledSellPrice
		if strat.quantileMiddle != nil {
			if strat.adjustedRealisedSpread == nil {
				strat.adjustedRealisedSpread = new(float64)
			}
			*strat.adjustedRealisedSpread = *strat.realisedSpread - *strat.quantileMiddle
		}
		strat.xLastFilledBuyPrice = nil
		strat.yLastFilledBuyPrice = nil
		strat.xLastFilledSellPrice = nil
		strat.yLastFilledSellPrice = nil
		if strat.quantileMiddle != nil &&
			strat.xyFundingRate != nil &&
			strat.fundingRateFactor != nil {
			logger.Debugf("%s - %s realised short abs spread %f quantile middle %f funding rate offset %f adjusted spread %f", strat.ySymbol, strat.xSymbol, *strat.realisedSpread, *strat.quantileMiddle, *strat.xyFundingRate**strat.fundingRateFactor, *strat.realisedSpread-*strat.quantileMiddle+*strat.xyFundingRate**strat.fundingRateFactor)
		} else {
			logger.Debugf("%s - %s realised short abs spread %f", strat.ySymbol, strat.xSymbol, *strat.realisedSpread)
		}
		strat.xOrderSilentTime = time.Now().Add(strat.config.enterSilent)
	} else if strat.xLastFilledSellPrice != nil && strat.yLastFilledBuyPrice != nil {
		if strat.realisedSpread == nil {
			strat.realisedSpread = new(float64)
		}
		*strat.realisedSpread = (*strat.yLastFilledBuyPrice - *strat.xLastFilledSellPrice) / *strat.yLastFilledBuyPrice
		if strat.quantileMiddle != nil {
			if strat.adjustedRealisedSpread == nil {
				strat.adjustedRealisedSpread = new(float64)
			}
			*strat.adjustedRealisedSpread = *strat.realisedSpread - *strat.quantileMiddle
		}
		strat.xLastFilledBuyPrice = nil
		strat.yLastFilledBuyPrice = nil
		strat.xLastFilledSellPrice = nil
		strat.yLastFilledSellPrice = nil
		if strat.quantileMiddle != nil &&
			strat.xyFundingRate != nil &&
			strat.fundingRateFactor != nil {
			logger.Debugf("%s - %s realised long abs spread %f quantile middle %f funding rate offset %f adjusted spread %f", strat.ySymbol, strat.xSymbol, *strat.realisedSpread, *strat.quantileMiddle, *strat.xyFundingRate**strat.fundingRateFactor, *strat.realisedSpread-*strat.quantileMiddle+*strat.xyFundingRate**strat.fundingRateFactor)
		} else {
			logger.Debugf("%s - %s realised long abs spread %f", strat.ySymbol, strat.xSymbol, *strat.realisedSpread)
		}
		strat.xOrderSilentTime = time.Now().Add(strat.config.enterSilent)
	}
}

func (strat *XYStrategy) handleXOrderError() {
	if strat.xOrderError.Cancel != nil {
		logger.Debugf("%s cancel %v error %v", strat.xSymbol, *strat.xOrderError.Cancel, strat.xOrderError.Error)
		//strat.xOrderSilentTime = time.Now().Add(strat.config.XOrderSilent)
	} else if strat.xOrderError.New != nil {
		logger.Debugf("%s new %v error %v", strat.xSymbol, *strat.xOrderError.New, strat.xOrderError.Error)
		strat.xOrderSilentTime = time.Now().Add(strat.config.XOrderSilent)
	}
}

func (strat *XYStrategy) handleYOrderError() {
	if strat.yOrderError.Cancel != nil {
		logger.Debugf("%s cancel %v error %v", strat.xSymbol, *strat.yOrderError.Cancel, strat.yOrderError.Error)
		strat.yOrderSilentTime = time.Now().Add(strat.config.YOrderSilent)
	} else if strat.yOrderError.New != nil {
		logger.Debugf("%s new %v error %v", strat.xSymbol, *strat.yOrderError.New, strat.yOrderError.Error)
		strat.yOrderSilentTime = time.Now().Add(strat.config.YOrderSilent)
	}
}
