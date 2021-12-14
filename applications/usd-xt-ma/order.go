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
			logger.Debugf("%10s updateXPosition xSystemStatus %v ySystemStatus %v", strat.xSymbol, strat.xSystemStatus, strat.ySystemStatus)
		}
		return
	}

	if !strat.spreadReady ||
		strat.xPosition == nil ||
		strat.xyFundingRate == nil ||
		strat.xFundingRateFactor == nil ||
		strat.enterTarget == 0 {
		//if time.Now().Sub(strat.logSilentTime) > 0 {
		//	strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
		//	logger.Debugf("%s %v %v %v %v %v %v",
		//		strat.xSymbol,
		//		strat.spreadReady,
		//		strat.xPosition == nil,
		//		strat.yPosition == nil,
		//		strat.xyFundingRate == nil,
		//		strat.xFundingRateFactor == nil,
		//		strat.enterTarget == 0,
		//	)
		//}
		return
	}

	xSize := strat.xPosition.GetSize() * strat.xMultiplier
	xValue := xSize * strat.xPosition.GetPrice()
	xAbsValue := math.Abs(xValue)
	xyMidPrice := (strat.xMidPrice + strat.yMidPrice) * 0.5

	strat.offsetFactor = xAbsValue / strat.enterTarget
	strat.offsetStep = math.Min(strat.enterStep/strat.enterTarget, strat.offsetFactor)
	if strat.offsetFactor > 1.0 {
		strat.offsetFactor = 1.0
	}
	if strat.offsetStep > 1.0 {
		strat.offsetStep = 1.0
	}

	strat.tdSpreadMiddle = strat.stats.SpreadMiddle
	strat.tdSpreadEnterOffset = strat.stats.SpreadEnterOffset
	strat.tdSpreadExitOffset = strat.stats.SpreadLeaveOffset

	if xSize >= 0 {
		strat.thresholdShortTop = strat.tdSpreadMiddle + strat.config.ShortEnterThreshold + strat.tdSpreadEnterOffset*strat.offsetFactor - *strat.xyFundingRate**strat.xFundingRateFactor
		strat.thresholdShortBot = strat.tdSpreadMiddle + strat.config.ShortLeaveThreshold + strat.tdSpreadExitOffset*(strat.offsetFactor-strat.offsetStep) - *strat.xyFundingRate**strat.xFundingRateFactor
		strat.thresholdLongBot = strat.tdSpreadMiddle + strat.config.LongEnterThreshold - *strat.xyFundingRate**strat.xFundingRateFactor
		strat.thresholdLongTop = strat.tdSpreadMiddle + strat.config.LongLeaveThreshold - *strat.xyFundingRate**strat.xFundingRateFactor
	} else {
		strat.thresholdShortTop = strat.tdSpreadMiddle + strat.config.ShortEnterThreshold - *strat.xyFundingRate**strat.xFundingRateFactor
		strat.thresholdShortBot = strat.tdSpreadMiddle + strat.config.ShortLeaveThreshold - *strat.xyFundingRate**strat.xFundingRateFactor
		strat.thresholdLongBot = strat.tdSpreadMiddle + strat.config.LongEnterThreshold - strat.tdSpreadEnterOffset*strat.offsetFactor - *strat.xyFundingRate**strat.xFundingRateFactor
		strat.thresholdLongTop = strat.tdSpreadMiddle + strat.config.LongLeaveThreshold - strat.tdSpreadExitOffset*(strat.offsetFactor-strat.offsetStep) - *strat.xyFundingRate**strat.xFundingRateFactor
	}

	//if math.IsNaN(strat.thresholdLongBot) && time.Now().Sub(strat.logSilentTime) > 0 {
	//	strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
	//	logger.Debugf("%s enterTarget %f targetWeight %f enterTargetFactor %f", strat.xSymbol, strat.enterTarget, strat.targetWeight, strat.config.EnterTargetFactor)
	//}

	if time.Now().Sub(strat.xPositionUpdateTime) > strat.config.AccountMaxAge ||
		time.Now().Sub(strat.yPositionUpdateTime) > strat.config.AccountMaxAge ||
		strat.xAccount == nil ||
		strat.fundingRateSettleSilent ||
		time.Now().Sub(strat.spreadTickerTime) > strat.config.SpreadMaxAge ||
		time.Now().Sub(strat.xOrderSilentTime) < 0 {
		//if time.Now().Sub(strat.logSilentTime) > 0 {
		//	strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
		//	logger.Debugf("%s %v %v %v %v %v %v %v %v %v %v",
		//		strat.xSymbol,
		//		strat.config.AccountMaxAge,
		//		strat.xPositionUpdateTime, time.Now().Sub(strat.xPositionUpdateTime) > strat.config.AccountMaxAge,
		//		strat.yPositionUpdateTime, time.Now().Sub(strat.yPositionUpdateTime) > strat.config.AccountMaxAge,
		//		strat.xAccount == nil,
		//		strat.yAccount == nil,
		//		strat.fundingRateSettleSilent,
		//		time.Now().Sub(strat.spreadTickerTime) > strat.config.SpreadMaxAge,
		//		time.Now().Sub(strat.xOrderSilentTime) < 0,
		//	)
		//}
		return
	}

	//if time.Now().Sub(strat.logSilentTime) > 0 {
	//	strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
	//	logger.Debugf("SHORT %20s 1 %v 2 %v 3 %v 4 %v 5 %v 6 %v 7 %v 8 %v 9 %v 10 %v 11 %v 12 %v 13 %v x %.2ff > %.2ff y %.2ff > %.2ff",
	//		strat.xSymbol,
	//		!strat.reduceOnly,
	//		!strat.isYSpot,
	//		strat.tdSpreadMiddle > strat.config.SpreadMiddleMin,
	//		strat.tdSpreadMiddle < strat.config.SpreadMiddleMax,
	//		strat.spreadMedianShort > strat.thresholdShortTop,
	//		strat.spreadLastShort > strat.spreadMedianShort,
	//		*strat.xyFundingRate > strat.config.FundingRateOpenShortMin,
	//		xSize > -strat.xMinSize*strat.xMultiplier,
	//		strat.xAccount.GetFree() > strat.config.MinXFree,
	//		strat.yAccount.GetFree() > strat.config.MinYFree,
	//		xSize < strat.maxPosSize,
	//		xAbsValue < strat.maxPosValue,
	//		yAbsValue < strat.maxPosValue,
	//		strat.xAccount.GetFree() , strat.config.MinXFree,
	//		strat.yAccount.GetFree() , strat.config.MinYFree,
	//	)
	//	logger.Debugf("LONG  %20s 1 %v 2 %v 3 %v 4 %v 5 %v 6 %v 7 %v 8 %v 9 %v 10 %v 11 %v 12 %v 13 %v x %.2ff > %.2ff y %.2ff > %.2ff",
	//		strat.xSymbol,
	//		!strat.reduceOnly,
	//		!strat.isXSpot,
	//		strat.tdSpreadMiddle > strat.config.SpreadMiddleMin,
	//		strat.tdSpreadMiddle < strat.config.SpreadMiddleMax,
	//		strat.spreadMedianLong < strat.thresholdLongBot,
	//		strat.spreadLastLong < strat.spreadMedianLong,
	//		*strat.xyFundingRate < strat.config.FundingRateOpenLongMax,
	//		xSize < strat.xMinSize*strat.xMultiplier,
	//		strat.xAccount.GetFree() > strat.config.MinXFree,
	//		strat.yAccount.GetFree() > strat.config.MinYFree,
	//		xSize > -strat.maxPosSize,
	//		xAbsValue < strat.maxPosValue,
	//		yAbsValue < strat.maxPosValue,
	//		strat.xAccount.GetFree() , strat.config.MinXFree,
	//		strat.yAccount.GetFree() , strat.config.MinYFree,
	//	)
	//}

	if strat.spreadMedianLong < strat.thresholdShortBot &&
		strat.spreadLastLong < strat.spreadMedianLong &&
		xSize >= strat.xMinSize*strat.xMultiplier {
		strat.enterValue = math.Min(math.Max(4*strat.enterStep, xAbsValue*0.5), xAbsValue)

		xSizeDiff := strat.enterValue / xyMidPrice

		if xSizeDiff > strat.maxOrderSize {
			xSizeDiff = strat.maxOrderSize
		}

		if strat.config.BestSizeFactor > 0 {
			//限开仓大小限制到best bid ask size, 主要关心X的深度，保证X的深度足够
			xSizeDiff = math.Min(strat.xTicker.GetAskSize()*strat.xMultiplier*strat.config.BestSizeFactor, xSizeDiff)
		}

		strat.enterValue = xSizeDiff * xyMidPrice
		xSizeDiff = math.Floor(xSizeDiff/strat.xMultiplier/strat.xStepSize) * strat.xStepSize
		if xSizeDiff >= strat.xMinSize && strat.enterValue >= 1.2*strat.xMinNotional {

			xPrice := strat.xTicker.GetBidPrice()
			//防止TickSize太大
			if strat.xTickSize/xPrice < strat.config.EnterSlippage {
				xPrice = xPrice * (1.0 - strat.config.EnterSlippage)
				xPrice = math.Floor(xPrice/strat.xTickSize) * strat.xTickSize
			}

			strat.xNewOrderParam = common.NewOrderParam{
				Symbol:      strat.xSymbol,
				Side:        common.OrderSideSell,
				Type:        common.OrderTypeLimit,
				Price:       xPrice,
				TimeInForce: strat.config.XOrderTimeInForce,
				Size:        xSizeDiff,
				PostOnly:    false,
				ReduceOnly:  true,
				ClientID:    strat.xExchange.GenerateClientID(),
				CancelAfter: strat.config.XOrderCancelAfter,
			}
			if !strat.config.DryRun {
				select {
				case strat.xOrderRequestCh <- common.OrderRequest{
					New: &strat.xNewOrderParam,
				}:
				}
			}
			strat.xLastFilledBuyPrice = nil
			strat.xLastFilledSellPrice = nil
			strat.xOrderSilentTime = time.Now().Add(strat.config.XOrderSilent)
			strat.lastEnterTime = strat.spreadTickerTime.Add(strat.config.XOrderSilent)
			strat.referenceXPrice = strat.xTicker.GetBidPrice()
			logger.Debugf(
				"%10s %10s SHORT BOT REDUCE %f < %f, %f < %f, PRICE %f SIZE %f, XTickerDiff %v YTickerDiff %v X %f %f Y %f %f, Offsets %f %f Fr %f",
				strat.xSymbol, strat.ySymbol,
				strat.spreadLastLong, strat.thresholdShortBot,
				strat.spreadMedianLong, strat.thresholdShortBot,
				xPrice,
				xSizeDiff,
				time.Now().Sub(strat.xTickerTime),
				time.Now().Sub(strat.yTickerTime),
				strat.xTicker.GetBidPrice(),
				strat.xTicker.GetAskPrice(),
				strat.yTicker.GetBidPrice(),
				strat.yTicker.GetAskPrice(),
				strat.tdSpreadEnterOffset,
				strat.tdSpreadExitOffset,
				*strat.xyFundingRate,
			)
		}
	} else if strat.spreadMedianShort > strat.thresholdLongTop &&
		strat.spreadLastShort > strat.spreadMedianShort &&
		xSize <= -strat.xMinSize*strat.xMultiplier {

		strat.enterValue = math.Min(math.Max(4*strat.enterStep, xAbsValue*0.5), xAbsValue)

		xSizeDiff := strat.enterValue / xyMidPrice

		if xSizeDiff > strat.maxOrderSize {
			xSizeDiff = strat.maxOrderSize
		}

		if strat.config.BestSizeFactor > 0 {
			xSizeDiff = math.Min(strat.xTicker.GetBidSize()*strat.xMultiplier*strat.config.BestSizeFactor, xSizeDiff)
		}

		strat.enterValue = xSizeDiff * xyMidPrice
		xSizeDiff = math.Floor(xSizeDiff/strat.xMultiplier/strat.xStepSize) * strat.xStepSize
		if xSizeDiff >= strat.xMinSize && strat.enterValue >= 1.2*strat.xMinNotional {
			xPrice := strat.xTicker.GetAskPrice()
			if strat.xTickSize/xPrice < strat.config.EnterSlippage {
				xPrice = xPrice * (1.0 + strat.config.EnterSlippage)
				xPrice = math.Ceil(xPrice/strat.xTickSize) * strat.xTickSize
			}
			strat.xNewOrderParam = common.NewOrderParam{
				Symbol:      strat.xSymbol,
				Side:        common.OrderSideBuy,
				Type:        common.OrderTypeLimit,
				Price:       xPrice,
				TimeInForce: strat.config.XOrderTimeInForce,
				Size:        xSizeDiff,
				PostOnly:    false,
				ReduceOnly:  true,
				ClientID:    strat.xExchange.GenerateClientID(),
				CancelAfter: strat.config.XOrderCancelAfter,
			}
			if !strat.config.DryRun {
				select {
				case strat.xOrderRequestCh <- common.OrderRequest{
					New: &strat.xNewOrderParam,
				}:
				}
			}
			strat.xLastFilledBuyPrice = nil
			strat.xLastFilledSellPrice = nil
			strat.xOrderSilentTime = time.Now().Add(strat.config.XOrderSilent)
			strat.lastEnterTime = strat.spreadTickerTime.Add(strat.config.XOrderSilent)
			strat.referenceXPrice = strat.xTicker.GetAskPrice()
			logger.Debugf(
				"%10s %10s LONG TOP REDUCE %f > %f, %f > %f, PRICE %f SIZE %f, XTickerDiff %v YTickerDiff %v X %f %f Y %f %f, Offsets %f %f, Fr %f",
				strat.xSymbol, strat.ySymbol,
				strat.spreadLastShort, strat.thresholdLongTop,
				strat.spreadMedianShort, strat.thresholdLongTop,
				xPrice,
				xSizeDiff,
				time.Now().Sub(strat.xTickerTime),
				time.Now().Sub(strat.yTickerTime),
				strat.xTicker.GetBidPrice(),
				strat.xTicker.GetAskPrice(),
				strat.yTicker.GetBidPrice(),
				strat.yTicker.GetAskPrice(),
				strat.tdSpreadEnterOffset,
				strat.tdSpreadExitOffset,
				*strat.xyFundingRate,
			)
		}
	} else if !strat.reduceOnly &&
		strat.tdSpreadMiddle > strat.config.SpreadMiddleMin &&
		strat.tdSpreadMiddle < strat.config.SpreadMiddleMax &&
		strat.spreadMedianShort > strat.thresholdShortTop &&
		strat.spreadLastShort > strat.spreadMedianShort &&
		*strat.xyFundingRate > strat.config.FundingRateOpenShortMin &&
		xSize > -strat.xMinSize*strat.xMultiplier &&
		strat.xAccount.GetFree() > strat.config.MinXFree &&
		xSize < strat.maxPosSize &&
		xAbsValue < strat.maxPosValue {

		if strat.xPosition.GetSize() > strat.xStepSize &&
			strat.xPosition.GetPrice() > 0 &&
			strat.xPosition.GetPrice()*(1.0+strat.offsetFactor*strat.config.AddTargetOffset) > strat.xMidPrice {
			//有多仓，没赚钱
			return
		}

		//if strat.config.EnterWithProfitConfirms &&
		//	!strat.isXSpot &&
		//	xSize > 2*strat.xyMergedStepSize &&
		//	//spot balance has no entry price
		//	strat.xTicker.GetAskPrice() < strat.xPosition.GetPrice() {
		//	return
		//}

		strat.targetValue = xAbsValue + strat.enterStep
		if strat.targetValue > strat.enterTarget {
			strat.targetValue = strat.enterTarget
		}
		strat.enterValue = strat.targetValue - xAbsValue
		if strat.enterValue < 0 {
			//超过最大仓位了, 不操作
			return
		}

		xSizeDiff := strat.enterValue / xyMidPrice
		if xSizeDiff > strat.maxOrderSize {
			xSizeDiff = strat.maxOrderSize
		}

		if strat.config.BestSizeFactor > 0 {
			xSizeDiff = math.Min(strat.xTicker.GetBidSize()*strat.xMultiplier*strat.config.BestSizeFactor, xSizeDiff)
		}

		strat.enterValue = xSizeDiff * xyMidPrice
		if strat.enterValue > strat.usdAvailable {
			if time.Now().Sub(strat.logSilentTime) > 0 {
				strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
				logger.Debugf(
					"%10s %10s FAILED SHORT TOP OPEN, ENTRY VALUE %f MORE THAN usdAvailable %f, %f > %f, %f > %f, SIZE %f",
					strat.xSymbol,
					strat.ySymbol,
					strat.enterValue,
					strat.usdAvailable,
					strat.spreadLastShort, strat.thresholdShortTop,
					strat.spreadMedianShort, strat.thresholdShortTop,
					xSizeDiff,
				)
			}
			return
		}

		xSizeDiff = math.Floor(xSizeDiff/strat.xMultiplier/strat.xStepSize) * strat.xStepSize
		if strat.enterValue < 1.2*strat.xMinNotional || xSizeDiff < strat.xMinSize {
			if time.Now().Sub(strat.logSilentTime) > 0 {
				strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
				logger.Debugf(
					"%10s %10s FAILED SHORT TOP OPEN, ORDER VALUE %f TOO SMALL, %f > %f, %f > %f, SIZE %f",
					strat.xSymbol, strat.ySymbol,
					strat.enterValue,
					strat.spreadLastShort, strat.thresholdShortTop,
					strat.spreadMedianShort, strat.thresholdShortTop,
					xSizeDiff,
				)
			}
			return
		}
		xPrice := strat.xTicker.GetAskPrice()
		if strat.xTickSize/xPrice < strat.config.EnterSlippage {
			xPrice = xPrice * (1.0 + strat.config.EnterSlippage)
			xPrice = math.Ceil(xPrice/strat.xTickSize) * strat.xTickSize
		}
		strat.xNewOrderParam = common.NewOrderParam{
			Symbol:      strat.xSymbol,
			Side:        common.OrderSideBuy,
			Type:        common.OrderTypeLimit,
			Price:       xPrice,
			TimeInForce: strat.config.XOrderTimeInForce,
			Size:        xSizeDiff,
			PostOnly:    false,
			ReduceOnly:  false,
			ClientID:    strat.xExchange.GenerateClientID(),
			CancelAfter: strat.config.XOrderCancelAfter,
		}
		if !strat.config.DryRun {
			select {
			case strat.xOrderRequestCh <- common.OrderRequest{
				New: &strat.xNewOrderParam,
			}:
			}
		}
		strat.xLastFilledBuyPrice = nil
		strat.xLastFilledSellPrice = nil
		strat.xOrderSilentTime = time.Now().Add(strat.config.XOrderSilent)
		strat.lastEnterTime = strat.spreadTickerTime.Add(strat.config.XOrderSilent)
		strat.referenceXPrice = strat.xTicker.GetAskPrice()
		logger.Debugf(
			"%10s %10s SHORT TOP OPEN %f > %f, %f > %f, PRICE %f SIZE %f, XTickerDiff %v YTickerDiff %v X %f %f Y %f %f, Offsets %f %f, Fr %f",
			strat.xSymbol, strat.ySymbol,
			strat.spreadLastShort, strat.thresholdShortTop,
			strat.spreadMedianShort, strat.thresholdShortTop,
			xPrice,
			xSizeDiff,
			time.Now().Sub(strat.xTickerTime),
			time.Now().Sub(strat.yTickerTime),
			strat.xTicker.GetBidPrice(),
			strat.xTicker.GetAskPrice(),
			strat.yTicker.GetBidPrice(),
			strat.yTicker.GetAskPrice(),
			strat.tdSpreadEnterOffset,
			strat.tdSpreadExitOffset,
			*strat.xyFundingRate,
		)
	} else if !strat.reduceOnly &&
		strat.tdSpreadMiddle > strat.config.SpreadMiddleMin &&
		strat.tdSpreadMiddle < strat.config.SpreadMiddleMax &&
		strat.spreadMedianLong < strat.thresholdLongBot &&
		strat.spreadLastLong < strat.spreadMedianLong &&
		*strat.xyFundingRate < strat.config.FundingRateOpenLongMax &&
		xSize < strat.xMinSize*strat.xMultiplier &&
		strat.xAccount.GetFree() > strat.config.MinXFree &&
		xSize > -strat.maxPosSize &&
		xAbsValue < strat.maxPosValue {

		if strat.xPosition.GetSize() < -strat.xStepSize &&
			strat.xPosition.GetPrice() > 0 &&
			strat.xPosition.GetPrice()*(1.0-strat.offsetFactor*strat.config.AddTargetOffset) < strat.xMidPrice {
			//有空仓，没赚钱
			return
		}

		strat.targetValue = xAbsValue + strat.enterStep
		if strat.targetValue > strat.enterTarget {
			strat.targetValue = strat.enterTarget
		}
		strat.enterValue = strat.targetValue - xAbsValue
		if strat.enterValue < 0 {
			//超过最大仓位了, 不操作
			return
		}
		xSizeDiff := strat.enterValue / xyMidPrice

		if xSizeDiff > strat.maxOrderSize {
			xSizeDiff = strat.maxOrderSize
		}

		if strat.config.BestSizeFactor > 0 {
			xSizeDiff = math.Min(strat.yTicker.GetAskSize()*strat.yMultiplier*strat.config.BestSizeFactor, xSizeDiff)
		}

		strat.enterValue = xSizeDiff * xyMidPrice
		if strat.enterValue > strat.usdAvailable {
			if time.Now().Sub(strat.logSilentTime) > strat.config.LogInterval {
				strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
				logger.Debugf(
					"%10s %10s FAILED LONG BOT OPEN, ENTRY VALUE %f MORE THAN usdAvailable %f, %f < %f, %f < %f, SIZE %f",
					strat.xSymbol,
					strat.ySymbol,
					strat.enterValue,
					strat.usdAvailable,
					strat.spreadLastLong, strat.thresholdLongBot,
					strat.spreadMedianLong, strat.thresholdLongBot,
					xSizeDiff,
				)
			}
			return
		}
		xSizeDiff = math.Floor(xSizeDiff/strat.xMultiplier/strat.xStepSize) * strat.xStepSize
		if strat.enterValue < 1.2*strat.xMinNotional || xSizeDiff < strat.xMinSize {
			if time.Now().Sub(strat.logSilentTime) > 0 {
				strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
				logger.Debugf(
					"%10s %10s FAILED LONG BOT OPEN, ORDER VALUE %f TOO SMALL, %f < %f, %f < %f, SIZE %f",
					strat.xSymbol, strat.ySymbol,
					strat.enterValue,
					strat.spreadLastLong, strat.thresholdLongBot,
					strat.spreadMedianLong, strat.thresholdLongBot,
					xSizeDiff,
				)
			}
			return
		}
		xPrice := strat.xTicker.GetBidPrice()
		//防止TickSize太大
		if strat.xTickSize/xPrice < strat.config.EnterSlippage {
			xPrice = xPrice * (1.0 - strat.config.EnterSlippage)
			xPrice = math.Floor(xPrice/strat.xTickSize) * strat.xTickSize
		}
		strat.xNewOrderParam = common.NewOrderParam{
			Symbol:      strat.xSymbol,
			Side:        common.OrderSideSell,
			Type:        common.OrderTypeLimit,
			Price:       xPrice,
			TimeInForce: strat.config.XOrderTimeInForce,
			Size:        xSizeDiff,
			PostOnly:    false,
			ReduceOnly:  false,
			ClientID:    strat.xExchange.GenerateClientID(),
			CancelAfter: strat.config.XOrderCancelAfter,
		}
		if !strat.config.DryRun {
			select {
			case strat.xOrderRequestCh <- common.OrderRequest{
				New: &strat.xNewOrderParam,
			}:
			}
		}
		strat.xLastFilledBuyPrice = nil
		strat.xLastFilledSellPrice = nil
		strat.xOrderSilentTime = time.Now().Add(strat.config.XOrderSilent)
		strat.lastEnterTime = strat.spreadTickerTime.Add(strat.config.XOrderSilent)
		strat.referenceXPrice = strat.xTicker.GetBidPrice()
		logger.Debugf(
			"%10s %10s LONG BOT OPEN %f < %f, %f < %f, PRICE %f SIZE %f, XTickerDiff %v YTickerDiff %v X %f %f Y %f %f, Offsets %f %f, Fr %f",
			strat.xSymbol, strat.ySymbol,
			strat.spreadLastLong, strat.thresholdLongBot,
			strat.spreadMedianLong, strat.thresholdLongBot,
			xPrice,
			xSizeDiff,
			time.Now().Sub(strat.xTickerTime),
			time.Now().Sub(strat.yTickerTime),
			strat.xTicker.GetBidPrice(),
			strat.xTicker.GetAskPrice(),
			strat.yTicker.GetBidPrice(),
			strat.yTicker.GetAskPrice(),
			strat.tdSpreadEnterOffset,
			strat.tdSpreadExitOffset,
			*strat.xyFundingRate,
		)
	}
}
