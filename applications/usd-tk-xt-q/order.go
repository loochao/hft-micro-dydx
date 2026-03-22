package main

import (
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"time"
)

func (strat *XYStrategy) updateXPosition() {
	if strat.xSystemStatus != common.SystemStatusReady ||
		strat.ySystemStatus != common.SystemStatusReady {
		if time.Now().Sub(strat.logSilentTime) > 0 {
			strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
			logger.Debugf("updateXPosition xSystemStatus %v ySystemStatus %v", strat.xSystemStatus, strat.ySystemStatus)
		}
		return
	}

	if time.Now().Sub(strat.xPositionUpdateTime) > strat.config.BalancePositionMaxAge ||
		strat.xAccount == nil ||
		strat.xPosition == nil ||
		strat.spread == nil ||
		strat.xyFundingRate == nil ||
		strat.quantile50 == nil ||
		strat.fundingRateSettleSilent ||
		time.Now().Sub(strat.spread.EventTime) > strat.config.SpreadTimeToEnter {
		return
	}

	strat.xSize = strat.xPosition.GetSize() * strat.xMultiplier
	strat.xValue = strat.xSize * strat.xMidPrice
	strat.xAbsValue = math.Abs(strat.xValue)

	strat.shortTop = *strat.quantile50 + strat.config.ShortEnterDelta - *strat.xyFundingRate*strat.config.FrOffsetFactor
	strat.shortBot = *strat.quantile50 + strat.config.ShortExitDelta - *strat.xyFundingRate*strat.config.FrOffsetFactor
	strat.longBot = *strat.quantile50 + strat.config.LongEnterDelta - *strat.xyFundingRate*strat.config.FrOffsetFactor
	strat.longTop = *strat.quantile50 + strat.config.LongExitDelta - *strat.xyFundingRate*strat.config.FrOffsetFactor

	strat.midPrice = (strat.xMidPrice + strat.yMidPrice) * 0.5
	if math.IsNaN(strat.longBot) && time.Now().Sub(strat.logSilentTime) > 0 {
		strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
		logger.Debugf("%s long bot is nan", strat.xSymbol)
	}

	if time.Now().Sub(strat.xOrderSilentTime) < 0 {
		return
	}

	frClose := math.Abs(*strat.xyFundingRate) > strat.config.MaximalHoldFundingRate ||
		math.Abs(strat.xFundingRate.GetFundingRate()) > strat.config.MaximalHoldFundingRate ||
		math.Abs(strat.yFundingRate.GetFundingRate()) > strat.config.MaximalHoldFundingRate

	pnlPct := 0.0
	if strat.xPosition.GetSize() > 0 {
		pnlPct = (strat.midPrice - strat.xPosition.GetPrice()) / strat.xPosition.GetPrice()
	} else if strat.xPosition.GetSize() < 0 {
		pnlPct = -(strat.midPrice - strat.xPosition.GetPrice()) / strat.xPosition.GetPrice()
	}
	stopLoss := pnlPct <= strat.config.StopLoss
	if stopLoss {
		logger.Debugf(
			"%s STOP LOSS %f < %f, POS %f %f MID PRICE %f",
			strat.xSymbol,
			pnlPct, strat.config.StopLoss,
			strat.xPosition.GetSize(),
			strat.xPosition.GetPrice(),
			strat.midPrice,
		)
	}


	shortBotClose := strat.spread.ShortMedianLeave < strat.shortBot &&
		strat.spread.ShortMedianLeave < *strat.quantile20 &&
		strat.spread.ShortLastLeave <= strat.spread.ShortMedianLeave

	longTopClose := strat.spread.LongMedianLeave > strat.longTop &&
		strat.spread.LongMedianLeave > *strat.quantile80 &&
		strat.spread.LongLastLeave >= strat.spread.LongMedianLeave

	shortTopOpen := strat.spread.ShortMedianEnter > strat.shortTop &&
		strat.spread.ShortMedianEnter > *strat.quantile95 &&
		strat.spread.ShortLastEnter >= strat.spread.ShortMedianEnter

	longBotOpen := strat.spread.LongMedianEnter < strat.longBot &&
		strat.spread.LongMedianEnter < *strat.quantile05 &&
		strat.spread.LongLastEnter <= strat.spread.LongMedianEnter

	if (frClose || shortBotClose || stopLoss) &&
		strat.xSize >= strat.xStepSize*strat.xMultiplier &&
		strat.xSize*strat.xTicker.GetBidPrice()*strat.xMultiplier > 1.2*strat.xMinNotional {

		strat.enterValue = math.Min(math.Max(4*strat.enterStep, strat.xAbsValue*0.5), strat.xAbsValue)
		strat.size = strat.enterValue / strat.midPrice

		//限开仓大小限制到best bid ask size
		strat.size = math.Min(strat.xTicker.GetBidSize()*strat.xMultiplier*strat.config.BestSizeFactor*2.0, strat.size)

		strat.enterValue = strat.size * strat.midPrice
		if strat.xAbsValue-strat.enterValue < strat.xStepSize*1.005 ||
			strat.size > strat.xSize {
			//两种情况都把x全平
			strat.size = strat.xSize
		}
		strat.size = math.Floor(strat.size/strat.xMultiplier/strat.xStepSize) * strat.xStepSize
		if strat.size > 0 && (!strat.isXSpot || strat.enterValue > 1.2*strat.xMinNotional) {

			strat.price = strat.xTicker.GetBidPrice()
			//防止TickSize太大
			if strat.xTickSize/strat.price < strat.config.EnterSlippage*2.0 {
				strat.price = strat.price * (1.0 - strat.config.EnterSlippage*2.0)
				strat.price = math.Floor(strat.price/strat.xTickSize) * strat.xTickSize
			}
			strat.xNewOrderParam = common.NewOrderParam{
				Symbol:      strat.xSymbol,
				Side:        common.OrderSideSell,
				Type:        common.OrderTypeLimit,
				Price:       strat.price,
				TimeInForce: strat.config.XOrderTimeInForce,
				Size:        strat.size,
				PostOnly:    false,
				ReduceOnly:  true,
				ClientID:    strat.xExchange.GenerateClientID(),
			}
			if !strat.config.DryRun {
				select {
				case strat.xOrderRequestCh <- common.OrderRequest{
					New: &strat.xNewOrderParam,
				}:
				}
			}
			strat.xOrderSilentTime = time.Now().Add(strat.config.XOrderSilent)
			logger.Debugf(
				"%s %s SHORT BOT REDUCE %f < %f, %f < %f, PRICE %f SIZE %f, XTickerDiff %v YTickerDiff %v PnlPct %f",
				strat.xSymbol, strat.ySymbol,
				strat.spread.ShortLastLeave, strat.shortBot,
				strat.spread.ShortMedianLeave, strat.shortBot,
				strat.price,
				strat.size,
				time.Now().Sub(strat.xTickerTime),
				time.Now().Sub(strat.yTickerTime),
				pnlPct,
			)
		}
	} else if (frClose || longTopClose || stopLoss) &&
		strat.xSize <= -strat.xStepSize*strat.xMultiplier {

		strat.enterValue = math.Min(math.Max(4*strat.enterStep, strat.xAbsValue*0.5), strat.xAbsValue)
		strat.size = strat.enterValue / strat.midPrice
		//限开仓大小限制到best bid ask size
		strat.size = math.Min(strat.xTicker.GetAskSize()*strat.xMultiplier*strat.config.BestSizeFactor*2.0, strat.size)

		strat.enterValue = strat.size * strat.midPrice
		if strat.xAbsValue-strat.enterValue < strat.xStepSize*1.005 ||
			strat.size > -strat.xSize {
			strat.size = -strat.xSize
		}
		strat.size = math.Floor(strat.size/strat.xMultiplier/strat.xStepSize) * strat.xStepSize
		if strat.size > 0 && (!strat.isXSpot || strat.enterValue > 1.2*strat.xMinNotional) {
			strat.price = strat.xTicker.GetAskPrice()
			if strat.xTickSize/strat.price < strat.config.EnterSlippage*2.0 {
				strat.price = strat.price * (1.0 + strat.config.EnterSlippage*2.0)
				strat.price = math.Ceil(strat.price/strat.xTickSize) * strat.xTickSize
			}
			strat.xNewOrderParam = common.NewOrderParam{
				Symbol:      strat.xSymbol,
				Side:        common.OrderSideBuy,
				Type:        common.OrderTypeLimit,
				Price:       strat.price,
				TimeInForce: strat.config.XOrderTimeInForce,
				Size:        strat.size,
				PostOnly:    false,
				ReduceOnly:  true,
				ClientID:    strat.xExchange.GenerateClientID(),
			}
			if !strat.config.DryRun {
				select {
				case strat.xOrderRequestCh <- common.OrderRequest{
					New: &strat.xNewOrderParam,
				}:
				}
			}
			strat.xOrderSilentTime = time.Now().Add(strat.config.XOrderSilent)
			logger.Debugf(
				"%s %s LONG TOP REDUCE %f > %f, %f > %f, PRICE %f SIZE %f, XTickerDiff %v YTickerDiff %v X %f %f Y %f %f, PnlPct %f",
				strat.xSymbol, strat.ySymbol,
				strat.spread.LongLastLeave, strat.longTop,
				strat.spread.LongMedianLeave, strat.longTop,
				strat.price,
				strat.size,
				time.Now().Sub(strat.xTickerTime),
				time.Now().Sub(strat.yTickerTime),
				strat.xTicker.GetBidPrice(),
				strat.xTicker.GetAskPrice(),
				strat.yTicker.GetBidPrice(),
				strat.yTicker.GetAskPrice(),
				pnlPct,
			)
		}
	} else if !strat.config.ReduceOnly &&
		!strat.isYSpot &&
		!frClose &&
		shortTopOpen &&
		pnlPct >= 0 &&
		strat.xAbsValue < strat.config.MaxPositionValue &&
		strat.xSize > -strat.xStepSize*strat.xMultiplier {

		strat.enterValue = strat.enterStep
		strat.size = strat.enterValue / strat.midPrice
		strat.size = math.Min(strat.xTicker.GetAskSize()*strat.xMultiplier*strat.config.BestSizeFactor, strat.size)

		strat.enterValue = strat.size * strat.midPrice
		if strat.enterValue > strat.usdAvailable {
			if time.Now().Sub(strat.logSilentTime) > 0 {
				strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
				logger.Debugf(
					"%s %s FAILED SHORT TOP OPEN, ENTRY VALUE %f MORE THAN usdAvailable %f, %f > %f, %f > %f, SIZE %f",
					strat.xSymbol,
					strat.ySymbol,
					strat.enterValue,
					strat.usdAvailable,
					strat.spread.ShortLastEnter, strat.shortTop,
					strat.spread.ShortMedianEnter, strat.shortTop,
					strat.size,
				)
			}
			return
		}
		strat.size = math.Floor(strat.size/strat.xMultiplier/strat.xStepSize) * strat.xStepSize
		if strat.size <= 0 || strat.enterValue < 1.2*strat.xMinNotional {
			if time.Now().Sub(strat.logSilentTime) > 0 {
				strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
				logger.Debugf(
					"%s %s FAILED SHORT TOP OPEN, ORDER VALUE %f TOO SMALL, %f > %f, %f > %f, SIZE %f, PnlPct %f",
					strat.xSymbol, strat.ySymbol,
					strat.enterValue,
					strat.spread.ShortLastEnter, strat.shortTop,
					strat.spread.ShortMedianEnter, strat.shortTop,
					strat.size,
					pnlPct,
				)
			}
			return
		}
		strat.price = strat.xTicker.GetAskPrice()
		if strat.xTickSize/strat.price < strat.config.EnterSlippage {
			strat.price = strat.price * (1.0 + strat.config.EnterSlippage)
			strat.price = math.Ceil(strat.price/strat.xTickSize) * strat.xTickSize
		}
		strat.xNewOrderParam = common.NewOrderParam{
			Symbol:      strat.xSymbol,
			Side:        common.OrderSideBuy,
			Type:        common.OrderTypeLimit,
			Price:       strat.price,
			TimeInForce: strat.config.XOrderTimeInForce,
			Size:        strat.size,
			PostOnly:    false,
			ReduceOnly:  false,
			ClientID:    strat.xExchange.GenerateClientID(),
		}
		if !strat.config.DryRun {
			select {
			case strat.xOrderRequestCh <- common.OrderRequest{
				New: &strat.xNewOrderParam,
			}:
			}
		}
		strat.xOrderSilentTime = time.Now().Add(strat.config.XOrderSilent)
		logger.Debugf(
			"%s %s SHORT TOP OPEN %f > %f, %f > %f, PRICE %f SIZE %f, XTickerDiff %v YTickerDiff %v X %f %f Y %f %f PnlPct %f",
			strat.xSymbol, strat.ySymbol,
			strat.spread.ShortLastEnter, strat.shortTop,
			strat.spread.ShortMedianEnter, strat.shortTop,
			strat.price,
			strat.size,
			time.Now().Sub(strat.xTickerTime),
			time.Now().Sub(strat.yTickerTime),
			strat.xTicker.GetBidPrice(),
			strat.xTicker.GetAskPrice(),
			strat.yTicker.GetBidPrice(),
			strat.yTicker.GetAskPrice(),
			pnlPct,
		)
	} else if !strat.config.ReduceOnly &&
		!strat.isXSpot &&
		!frClose &&
		longBotOpen &&
		pnlPct >= 0 &&
		strat.xAbsValue < strat.config.MaxPositionValue &&
		strat.xSize < strat.xStepSize*strat.xMultiplier {

		strat.enterValue = strat.enterStep
		strat.size = strat.enterValue / strat.midPrice
		strat.size = math.Min(strat.xTicker.GetBidSize()*strat.xMultiplier*strat.config.BestSizeFactor, strat.size)

		strat.enterValue = strat.size * strat.midPrice
		if strat.enterValue > strat.usdAvailable {
			if time.Now().Sub(strat.logSilentTime) > strat.config.LogInterval {
				strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
				logger.Debugf(
					"%s %s FAILED LONG BOT OPEN, ENTRY VALUE %f MORE THAN usdAvailable %f, %f < %f, %f < %f, SIZE %f, PnlPct %f",
					strat.xSymbol,
					strat.ySymbol,
					strat.enterValue,
					strat.usdAvailable,
					strat.spread.LongLastEnter, strat.longBot,
					strat.spread.LongMedianEnter, strat.longBot,
					strat.size,
					pnlPct,
				)
			}
			return
		}
		strat.size = math.Floor(strat.size/strat.xMultiplier/strat.xStepSize) * strat.xStepSize
		if strat.size <= 0 || strat.enterValue < 1.2*strat.xMinNotional {
			if time.Now().Sub(strat.logSilentTime) > 0 {
				strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
				logger.Debugf(
					"%s %s FAILED LONG BOT OPEN, ORDER VALUE %f TOO SMALL, %f < %f, %f < %f, SIZE %f",
					strat.xSymbol, strat.ySymbol,
					strat.enterValue,
					strat.spread.LongLastEnter, strat.longBot,
					strat.spread.LongMedianEnter, strat.longBot,
					strat.size,
				)
			}
			return
		}
		strat.price = strat.xTicker.GetBidPrice()
		//防止TickSize太大
		if strat.xTickSize/strat.price < strat.config.EnterSlippage {
			strat.price = strat.price * (1.0 - strat.config.EnterSlippage)
			strat.price = math.Floor(strat.price/strat.xTickSize) * strat.xTickSize
		}
		strat.xNewOrderParam = common.NewOrderParam{
			Symbol:      strat.xSymbol,
			Side:        common.OrderSideSell,
			Type:        common.OrderTypeLimit,
			Price:       strat.price,
			TimeInForce: strat.config.XOrderTimeInForce,
			Size:        strat.size,
			PostOnly:    false,
			ReduceOnly:  false,
			ClientID:    strat.xExchange.GenerateClientID(),
		}
		if !strat.config.DryRun {
			select {
			case strat.xOrderRequestCh <- common.OrderRequest{
				New: &strat.xNewOrderParam,
			}:
			}
		}
		strat.xOrderSilentTime = time.Now().Add(strat.config.XOrderSilent)
		logger.Debugf(
			"%s %s LONG BOT OPEN %f < %f, %f < %f, PRICE %f SIZE %f, XTickerDiff %v YTickerDiff %v X %f %f Y %f %f PnlPct %f",
			strat.xSymbol, strat.ySymbol,
			strat.spread.LongLastEnter, strat.longBot,
			strat.spread.LongMedianEnter, strat.longBot,
			strat.price,
			strat.size,
			time.Now().Sub(strat.xTickerTime),
			time.Now().Sub(strat.yTickerTime),
			strat.xTicker.GetBidPrice(),
			strat.xTicker.GetAskPrice(),
			strat.yTicker.GetBidPrice(),
			strat.yTicker.GetAskPrice(),
			pnlPct,
		)
	}
}
