package main

import (
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func (strat *XYStrategy) handleFundingRate() {
	if strat.xFundingRate == nil ||
		strat.yFundingRate == nil ||
		strat.xFundingRateFactor == nil ||
		strat.yFundingRateFactor == nil {
		return
	}
	if strat.xAdjustedFundingRate == nil {
		strat.xAdjustedFundingRate = new(float64)
	}
	if strat.yAdjustedFundingRate == nil {
		strat.yAdjustedFundingRate = new(float64)
	}
	if strat.xyFundingRate == nil {
		strat.xyFundingRate = new(float64)
	}
	*strat.yAdjustedFundingRate = strat.yFundingRate.GetFundingRate() * strat.config.YFundingRateWeight * *strat.yFundingRateFactor
	*strat.xAdjustedFundingRate = strat.xFundingRate.GetFundingRate() * strat.config.XFundingRateWeight * *strat.xFundingRateFactor
	*strat.xyFundingRate = *strat.yAdjustedFundingRate - *strat.xAdjustedFundingRate
}

func (strat *XYStrategy) handleXOrder() {
	if strat.xOrder.GetSymbol() != strat.xSymbol {
		logger.Debugf("%10s bad x order symbol not match %s %v", strat.xOrder.GetSymbol(), strat.xSymbol, strat.xOrder)
		return
	}
	if strat.xOrder.GetStatus() == common.OrderStatusExpired ||
		strat.xOrder.GetStatus() == common.OrderStatusReject ||
		strat.xOrder.GetStatus() == common.OrderStatusCancelled ||
		strat.xOrder.GetStatus() == common.OrderStatusFilled ||
		strat.xOrder.GetStatus() == common.OrderStatusPartiallyFilledAndCanceled ||
		strat.xOrder.GetStatus() == common.OrderStatusPartiallyFilled {

		//order silent after order end
		strat.xOrderSilentTime = time.Now().Add(strat.config.XOrderSilent)

		if strat.xOrder.GetStatus() != common.OrderStatusFilled &&
			strat.xOrder.GetStatus() != common.OrderStatusPartiallyFilled &&
			strat.xOrder.GetStatus() != common.OrderStatusPartiallyFilledAndCanceled {
			logger.Debugf("%10s x %s order %s %s", strat.xOrder.GetSymbol(), strat.xOrder.GetSide(), strat.xOrder.GetClientID(), strat.xOrder.GetStatus())
			//strat.xPositionUpdateTime = time.Unix(0, 0)
		} else {
			logger.Debugf("%10s x order filled %s %s size %f price %f value %f", strat.xSymbol, strat.xOrder.GetStatus(), strat.xOrder.GetSide(), strat.xOrder.GetFilledSize(), strat.xOrder.GetFilledPrice(), strat.xOrder.GetFilledSize()*strat.xOrder.GetFilledPrice()*strat.xMultiplier)
			strat.realisedSpreadTimer.Reset(strat.config.RealisedSpreadLogDelay)
			if strat.xOrder.GetSide() == common.OrderSideBuy {
				if strat.xLastFilledBuyPrice == nil {
					strat.xLastFilledBuyPrice = new(float64)
				}
				*strat.xLastFilledBuyPrice = strat.xOrder.GetFilledPrice()
			} else if strat.xOrder.GetSide() == common.OrderSideSell {
				if strat.xLastFilledSellPrice == nil {
					strat.xLastFilledSellPrice = new(float64)
				}
				*strat.xLastFilledSellPrice = strat.xOrder.GetFilledPrice()
			}
		}
	}
}

func (strat *XYStrategy) handleRealisedSpread() {
	if strat.xLastFilledBuyPrice != nil {
		strat.xSlippage = 0
		if strat.referenceXPrice != 0 {
			strat.xSlippage = (*strat.xLastFilledBuyPrice - strat.referenceXPrice) / strat.referenceXPrice
			strat.xSlippageTM.Insert(time.Now(), strat.xSlippage)
		}
		strat.xLastFilledBuyPrice = nil
		strat.xLastFilledSellPrice = nil
		if strat.tdSpreadMiddle != 0 &&
			strat.xyFundingRate != nil &&
			strat.xFundingRateFactor != nil {
			logger.Debugf("%10s - %10s realised short abs spread %f slippage x %.6f quantile middle %f funding rate offset %f adjusted spread %f", strat.ySymbol, strat.xSymbol, strat.xSlippage, strat.tdSpreadMiddle, *strat.xyFundingRate)
		} else {
			logger.Debugf("%10s - %10s realised short abs spread %f slippage x %.6f", strat.ySymbol, strat.xSymbol, strat.xSlippage)
		}
		strat.xOrderSilentTime = time.Now().Add(strat.config.XEnterSilent)
	} else if strat.xLastFilledSellPrice != nil {
		strat.xSlippage = 0
		if strat.referenceXPrice != 0 {
			strat.xSlippage = (strat.referenceXPrice - *strat.xLastFilledSellPrice) / strat.referenceXPrice
			strat.xSlippageTM.Insert(time.Now(), strat.xSlippage)
		}
		strat.xLastFilledBuyPrice = nil
		strat.xLastFilledSellPrice = nil
		if strat.tdSpreadMiddle != 0 &&
			strat.xyFundingRate != nil &&
			strat.xFundingRateFactor != nil {
			logger.Debugf("%10s - %10s realised long abs spread %f slippage x %.6f quantile middle %f funding rate offset %f adjusted spread %f", strat.ySymbol, strat.xSymbol, strat.xSlippage, strat.tdSpreadMiddle, *strat.xyFundingRate)
		} else {
			logger.Debugf("%10s - %10s realised long abs spread %f slippage x %.6f", strat.ySymbol, strat.xSymbol, strat.xSlippage)
		}
		strat.xOrderSilentTime = time.Now().Add(strat.config.XEnterSilent)
	}
}

func (strat *XYStrategy) handleXOrderError() {
	if strat.xOrderError.Cancel != nil {
		logger.Debugf("%10s cancel %v error %v", strat.xSymbol, *strat.xOrderError.Cancel, strat.xOrderError.Error)
		//strat.xOrderSilentTime = time.Now().Add(strat.config.XOrderSilent)
	} else if strat.xOrderError.New != nil {
		logger.Debugf("%10s new %v error %v", strat.xSymbol, *strat.xOrderError.New, strat.xOrderError.Error)
		strat.xOrderSilentTime = time.Now().Add(strat.config.XOrderSilent)
	}
}

func (strat *XYStrategy) updateSignal(ctx ) {
	api, err := binance_usdtfuture.NewAPI(&common.Credentials{}, strat.config.XExchange.Proxy)
	if err != nil {
		logger.Fatal(err)
	}
	api.GetHistoryKLines()
}
