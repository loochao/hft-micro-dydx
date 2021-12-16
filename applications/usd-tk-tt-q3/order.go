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
		strat.yPosition == nil ||
		strat.xyFundingRate == nil ||
		strat.xFundingRateFactor == nil ||
		strat.enterTarget == 0 ||
		strat.xTicker.GetBidPrice() <= 0 || //此处要考虑到交易所盘口被干穿的情况，没有BID 或者 ASK， Price == 0
		strat.xTicker.GetAskPrice() <= 0 ||
		strat.yTicker.GetBidPrice() <= 0 ||
		strat.yTicker.GetAskPrice() <= 0 {
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
	ySize := strat.yPosition.GetSize() * strat.yMultiplier
	xValue := 0.0
	yValue := 0.0
	if strat.isXSpot || strat.xPosition.GetPrice() == 0 {
		xValue = xSize * strat.xMidPrice
	} else {
		xValue = xSize * strat.xPosition.GetPrice()
	}
	if strat.isYSpot || strat.yPosition.GetPrice() == 0 {
		yValue = ySize * strat.yMidPrice
	} else {
		yValue = ySize * strat.yPosition.GetPrice()
	}
	xAbsValue := math.Abs(xValue)
	yAbsValue := math.Abs(yValue)
	xyMidPrice := (strat.xMidPrice + strat.yMidPrice) * 0.5

	strat.offsetFactor = (xAbsValue + yAbsValue/strat.config.HedgeRatio) * 0.5 / strat.enterTarget
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
		strat.yAccount == nil ||
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

	if math.Abs(xSize+ySize/strat.config.HedgeRatio)*xyMidPrice > strat.enterStep*0.8 {
		if time.Now().Sub(strat.logSilentTime) > 0 {
			strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
			logger.Debugf(
				"%s %s unhedged value %f > 0.8*enterStep %f",
				strat.xSymbol, strat.ySymbol, math.Abs(xValue+yValue), strat.enterStep*0.8,
			)
		}
		strat.hedgeXPosition()
		if time.Now().Sub(strat.hedgeCheckStopTime) > 0 {
			strat.hedgeYPosition()
		}
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
		strat.enterValue = math.Min(math.Max(4*strat.enterStep, xAbsValue*0.5), math.Min(xAbsValue, yAbsValue/strat.config.HedgeRatio))

		xSizeDiff := strat.enterValue / xyMidPrice

		if xSizeDiff > strat.maxOrderSize {
			xSizeDiff = strat.maxOrderSize
		}

		if strat.config.BestSizeFactor > 0 {
			//限开仓大小限制到best bid ask size, 主要关心Y的深度，保证Y的深度足够
			xSizeDiff = math.Min(strat.yTicker.GetAskSize()*strat.yMultiplier*strat.config.BestSizeFactor*strat.ySlippageFactor, xSizeDiff)
		}
		xSizeDiff = math.Round(xSizeDiff/strat.xyMergedStepSize) * strat.xyMergedStepSize

		strat.enterValue = xSizeDiff * xyMidPrice
		if xAbsValue-strat.enterValue < strat.xyMergedStepSize*1.005 ||
			yAbsValue-strat.enterValue < strat.xyMergedStepSize*1.005 ||
			xSizeDiff > xSize {
			//两种情况都把x全平，间接y全平
			xSizeDiff = xSize
		}
		strat.enterValue = xSizeDiff * xyMidPrice
		xSizeDiff = math.Floor(xSizeDiff/strat.xMultiplier/strat.xStepSize) * strat.xStepSize
		if xSizeDiff >= strat.xMinSize && strat.enterValue >= 1.2*strat.xMinNotional {

			xPrice := strat.xTicker.GetBidPrice()
			//补偿一个TickSize,防止TickSize太大滑太远
			//同时TickSize比slippage小又是经常存在的，所以折个中，只要0.8个tickSize小于slippage, 就加上slippage
			slippage := (strat.thresholdShortBot - strat.spreadLastLong)*0.5 - strat.xTickSize/xPrice*0.5
			if slippage > strat.config.MaxSlippage {
				slippage = strat.config.MaxSlippage
			}
			if strat.xTickSize/xPrice*0.8 < slippage {
				xPrice = xPrice * (1.0 - slippage)
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
			strat.yLastFilledBuyPrice = nil
			strat.yLastFilledSellPrice = nil
			strat.xOrderSilentTime = time.Now().Add(strat.config.XOrderSilent)
			if strat.config.HedgeDelay > 0 {
				strat.hedgeCheckTimer.Reset(strat.config.HedgeDelay)
			} else {
				strat.hedgeCheckTimer.Reset(strat.config.HedgeCheckInterval)
			}
			strat.hedgeCheckStopTime = time.Now().Add(strat.config.HedgeCheckDuration)
			strat.lastEnterTime = strat.spreadTickerTime.Add(strat.config.XOrderSilent)
			strat.referenceSpread = strat.thresholdShortBot
			strat.referenceXPrice = strat.xTicker.GetBidPrice()
			strat.referenceYPrice = strat.yTicker.GetAskPrice()
			logger.Debugf(
				"%10s %10s SHORT BOT REDUCE %f < %f, %f < %f, PRICE %f SIZE %f, XTickerDiff %v YTickerDiff %v X %f %f Y %f %f, Offsets %f %f Fr %f, Ysf %f",
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
				strat.ySlippageFactor,
			)
		}
	} else if strat.spreadMedianShort > strat.thresholdLongTop &&
		strat.spreadLastShort > strat.spreadMedianShort &&
		xSize <= -strat.xMinSize*strat.xMultiplier {

		strat.enterValue = math.Min(math.Max(4*strat.enterStep, xAbsValue*0.5), math.Min(xAbsValue, yAbsValue/strat.config.HedgeRatio))

		xSizeDiff := strat.enterValue / xyMidPrice

		if xSizeDiff > strat.maxOrderSize {
			xSizeDiff = strat.maxOrderSize
		}

		if strat.config.BestSizeFactor > 0 {
			xSizeDiff = math.Min(strat.yTicker.GetBidSize()*strat.yMultiplier*strat.config.BestSizeFactor*strat.ySlippageFactor, xSizeDiff)
		}

		xSizeDiff = math.Round(xSizeDiff/strat.xyMergedStepSize) * strat.xyMergedStepSize
		strat.enterValue = xSizeDiff * xyMidPrice
		if xAbsValue-strat.enterValue < strat.xyMergedStepSize*1.005 ||
			yAbsValue-strat.enterValue < strat.xyMergedStepSize*1.005 ||
			xSizeDiff > -xSize {
			xSizeDiff = -xSize
		}
		strat.enterValue = xSizeDiff * xyMidPrice
		xSizeDiff = math.Floor(xSizeDiff/strat.xMultiplier/strat.xStepSize) * strat.xStepSize
		if xSizeDiff >= strat.xMinSize && strat.enterValue >= 1.2*strat.xMinNotional {
			xPrice := strat.xTicker.GetAskPrice()
			//补偿一个TickSize,防止TickSize太大滑太远
			//同时TickSize比slippage小又是经常存在的，所以折个中，只要0.8个tickSize小于slippage, 就加上slippage
			slippage := (strat.spreadLastShort - strat.thresholdLongTop)*0.5 - strat.xTickSize/xPrice*0.5
			if slippage > strat.config.MaxSlippage {
				slippage = strat.config.MaxSlippage
			}
			if strat.xTickSize/xPrice*0.8 < slippage {
				xPrice = xPrice * (1.0 + slippage)
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
			strat.yLastFilledBuyPrice = nil
			strat.yLastFilledSellPrice = nil
			strat.xOrderSilentTime = time.Now().Add(strat.config.XOrderSilent)
			if strat.config.HedgeDelay > 0 {
				strat.hedgeCheckTimer.Reset(strat.config.HedgeDelay)
			} else {
				strat.hedgeCheckTimer.Reset(strat.config.HedgeCheckInterval)
			}
			strat.hedgeCheckStopTime = time.Now().Add(strat.config.HedgeCheckDuration)
			strat.lastEnterTime = strat.spreadTickerTime.Add(strat.config.XOrderSilent)
			strat.referenceSpread = strat.thresholdLongTop
			strat.referenceXPrice = strat.xTicker.GetAskPrice()
			strat.referenceYPrice = strat.yTicker.GetBidPrice()
			logger.Debugf(
				"%10s %10s LONG TOP REDUCE %f > %f, %f > %f, PRICE %f SIZE %f, XTickerDiff %v YTickerDiff %v X %f %f Y %f %f, Offsets %f %f, Fr %f, Yfr %f",
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
				strat.ySlippageFactor,
			)
		}
	} else if !strat.reduceOnly &&
		!strat.isYSpot &&
		strat.tdSpreadMiddle > strat.config.SpreadMiddleMin &&
		strat.tdSpreadMiddle < strat.config.SpreadMiddleMax &&
		strat.spreadMedianShort > strat.thresholdShortTop &&
		strat.spreadLastShort > strat.spreadMedianShort &&
		*strat.xyFundingRate > strat.config.FundingRateOpenShortMin &&
		xSize > -strat.xMinSize*strat.xMultiplier &&
		strat.xAccount.GetFree() > strat.config.MinXFree &&
		strat.yAccount.GetFree() > strat.config.MinYFree &&
		xSize < strat.maxPosSize &&
		xAbsValue < strat.maxPosValue &&
		yAbsValue < strat.maxPosValue {

		//if strat.config.EnterWithProfitConfirms &&
		//	!strat.isXSpot &&
		//	xSize > 2*strat.xyMergedStepSize &&
		//	//spot balance has no entry price
		//	strat.xTicker.GetAskPrice() < strat.xPosition.GetPrice() {
		//	return
		//}

		strat.targetValue = math.Max(xAbsValue, yAbsValue/strat.config.HedgeRatio) + strat.enterStep
		if strat.targetValue > strat.enterTarget {
			strat.targetValue = strat.enterTarget
		}
		strat.enterValue = strat.targetValue - math.Max(xAbsValue, yAbsValue/strat.config.HedgeRatio)
		if strat.enterValue < 0 {
			//超过最大仓位了, 不操作
			strat.hedgeXPosition()
			return
		}

		xSizeDiff := strat.enterValue / xyMidPrice
		if xSizeDiff > strat.maxOrderSize {
			xSizeDiff = strat.maxOrderSize
		}

		if strat.config.BestSizeFactor > 0 {
			xSizeDiff = math.Min(strat.yTicker.GetBidSize()*strat.yMultiplier*strat.config.BestSizeFactor*strat.ySlippageFactor, xSizeDiff)
		}

		xSizeDiff = math.Round(xSizeDiff/strat.xyMergedStepSize) * strat.xyMergedStepSize
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
			strat.hedgeXPosition()
			return
		}

		xSizeDiff = math.Floor(xSizeDiff/strat.xMultiplier/strat.xStepSize) * strat.xStepSize
		if strat.enterValue < 1.2*strat.yMinNotional || strat.enterValue < 1.2*strat.xMinNotional || xSizeDiff < strat.xMinSize {
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
			strat.hedgeXPosition()
			return
		}
		xPrice := strat.xTicker.GetAskPrice()
		//补偿一个TickSize,防止TickSize太大滑太远
		//同时TickSize比slippage小又是经常存在的，所以折个中，只要0.8个tickSize小于slippage, 就加上slippage
		slippage := (strat.spreadLastShort - strat.thresholdShortTop)*0.5 - strat.xTickSize/xPrice*0.5
		if slippage > strat.config.MaxSlippage {
			slippage = strat.config.MaxSlippage
		}
		if strat.xTickSize/xPrice*0.8 < slippage {
			xPrice = xPrice * (1.0 + slippage)
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
		strat.yLastFilledBuyPrice = nil
		strat.yLastFilledSellPrice = nil
		strat.xOrderSilentTime = time.Now().Add(strat.config.XOrderSilent)
		if strat.config.HedgeDelay > 0 {
			strat.hedgeCheckTimer.Reset(strat.config.HedgeDelay)
		} else {
			strat.hedgeCheckTimer.Reset(strat.config.HedgeCheckInterval)
		}
		strat.hedgeCheckStopTime = time.Now().Add(strat.config.HedgeCheckDuration)
		strat.lastEnterTime = strat.spreadTickerTime.Add(strat.config.XOrderSilent)
		strat.referenceSpread = strat.thresholdShortTop
		strat.referenceXPrice = strat.xTicker.GetAskPrice()
		strat.referenceYPrice = strat.yTicker.GetBidPrice()
		logger.Debugf(
			"%10s %10s SHORT TOP OPEN %f > %f, %f > %f, PRICE %f SIZE %f, XTickerDiff %v YTickerDiff %v X %f %f Y %f %f, Offsets %f %f, Fr %f, Ysr %f",
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
			strat.ySlippageFactor,
		)
	} else if !strat.reduceOnly &&
		!strat.isXSpot &&
		strat.tdSpreadMiddle > strat.config.SpreadMiddleMin &&
		strat.tdSpreadMiddle < strat.config.SpreadMiddleMax &&
		strat.spreadMedianLong < strat.thresholdLongBot &&
		strat.spreadLastLong < strat.spreadMedianLong &&
		*strat.xyFundingRate < strat.config.FundingRateOpenLongMax &&
		xSize < strat.xMinSize*strat.xMultiplier &&
		strat.xAccount.GetFree() > strat.config.MinXFree &&
		strat.yAccount.GetFree() > strat.config.MinYFree &&
		xSize > -strat.maxPosSize &&
		xAbsValue < strat.maxPosValue &&
		yAbsValue < strat.maxPosValue {

		//if strat.config.EnterWithProfitConfirms &&
		//	xSize < -2*strat.xyMergedStepSize &&
		//	strat.xTicker.GetBidPrice() > strat.xPosition.GetPrice() {
		//	return
		//}

		strat.targetValue = math.Max(xAbsValue, yAbsValue/strat.config.HedgeRatio) + strat.enterStep
		if strat.targetValue > strat.enterTarget {
			strat.targetValue = strat.enterTarget
		}
		strat.enterValue = strat.targetValue - math.Max(xAbsValue, yAbsValue/strat.config.HedgeRatio)
		if strat.enterValue < 0 {
			//超过最大仓位了, 不操作
			strat.hedgeXPosition()
			return
		}
		xSizeDiff := strat.enterValue / xyMidPrice

		if xSizeDiff > strat.maxOrderSize {
			xSizeDiff = strat.maxOrderSize
		}

		if strat.config.BestSizeFactor > 0 {
			xSizeDiff = math.Min(strat.yTicker.GetAskSize()*strat.yMultiplier*strat.config.BestSizeFactor*strat.ySlippageFactor, xSizeDiff)
		}

		xSizeDiff = math.Round(xSizeDiff/strat.xyMergedStepSize) * strat.xyMergedStepSize
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
			strat.hedgeXPosition()
			return
		}
		xSizeDiff = math.Floor(xSizeDiff/strat.xMultiplier/strat.xStepSize) * strat.xStepSize
		if strat.enterValue < 1.2*strat.yMinNotional || strat.enterValue < 1.2*strat.xMinNotional || xSizeDiff < strat.xMinSize {
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
			strat.hedgeXPosition()
			return
		}
		xPrice := strat.xTicker.GetBidPrice()
		//补偿一个TickSize,防止TickSize太大滑太远
		//同时TickSize比slippage小又是经常存在的，所以折个中，只要0.8个tickSize小于slippage, 就加上slippage
		slippage := (strat.thresholdLongBot - strat.spreadLastLong)*0.5 - strat.xTickSize/xPrice*0.5
		if slippage > strat.config.MaxSlippage {
			slippage = strat.config.MaxSlippage
		}
		if strat.xTickSize/xPrice*0.8 < slippage {
			xPrice = xPrice * (1.0 - slippage)
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
		strat.yLastFilledBuyPrice = nil
		strat.yLastFilledSellPrice = nil
		strat.xOrderSilentTime = time.Now().Add(strat.config.XOrderSilent)
		if strat.config.HedgeDelay > 0 {
			strat.hedgeCheckTimer.Reset(strat.config.HedgeDelay)
		} else {
			strat.hedgeCheckTimer.Reset(strat.config.HedgeCheckInterval)
		}
		strat.hedgeCheckStopTime = time.Now().Add(strat.config.HedgeCheckDuration)
		strat.lastEnterTime = strat.spreadTickerTime.Add(strat.config.XOrderSilent)
		strat.referenceSpread = strat.thresholdLongBot
		strat.referenceXPrice = strat.xTicker.GetBidPrice()
		strat.referenceYPrice = strat.yTicker.GetAskPrice()
		logger.Debugf(
			"%10s %10s LONG BOT OPEN %f < %f, %f < %f, PRICE %f SIZE %f, XTickerDiff %v YTickerDiff %v X %f %f Y %f %f, Offsets %f %f, Fr %f, Ysr %f",
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
			strat.ySlippageFactor,
		)

	}

}

func (strat *XYStrategy) hedgeXPosition() {
	//如果lastSpreadEnterTime没有更新，说明没有信号触发，就需要检查对冲的情况
	if time.Now().Sub(strat.lastEnterTime) > strat.config.XEnterTimeout {

		//如果已经没有信号对冲，重新检查x y的仓位，对冲较小的
		if math.Abs(strat.xPosition.GetSize()*strat.xMultiplier) < math.Abs(strat.yPosition.GetSize()*strat.yMultiplier/strat.config.HedgeRatio) {
			//X的size比Y小，不用操作X
			return
		}
		xSizeDiff := -strat.yPosition.GetSize()*strat.yMultiplier/strat.config.HedgeRatio - strat.xPosition.GetSize()*strat.xMultiplier

		//if strat.config.BestSizeFactor > 0 {
		//	if xSizeDiff > 0 {
		//		xSizeDiff = math.Min(strat.xTicker.GetAskSize()*strat.config.BestSizeFactor, xSizeDiff)
		//	} else {
		//		xSizeDiff = math.Max(-strat.xTicker.GetBidSize()*strat.config.BestSizeFactor, xSizeDiff)
		//	}
		//}

		//maxOrderSize是默认的币数量
		if xSizeDiff > strat.maxOrderSize {
			xSizeDiff = strat.maxOrderSize
		} else if xSizeDiff < -strat.maxOrderSize {
			xSizeDiff = -strat.maxOrderSize
		}
		xSizeDiff /= strat.xMultiplier

		if xSizeDiff >= 0 {
			xSizeDiff = math.Floor(xSizeDiff/strat.xStepSize) * strat.xStepSize
		} else {
			xSizeDiff = math.Ceil(xSizeDiff/strat.xStepSize) * strat.xStepSize
		}

		if strat.isXSpot {
			if math.Abs(xSizeDiff) < strat.xStepSize {
				return
			} else if xSizeDiff < 0 && -xSizeDiff*strat.xMultiplier*strat.xTicker.GetBidPrice() < 1.2*strat.xMinNotional {
				return
			} else if xSizeDiff > 0 && xSizeDiff*strat.xMultiplier*strat.xTicker.GetAskPrice() < 1.2*strat.xMinNotional {
				return
			}
		} else {
			//期货可以close仓位，没有minNotional限制
			if math.Abs(xSizeDiff) < strat.xStepSize {
				return
			} else if xSizeDiff < 0 && strat.xPosition.GetSize() <= 0 && -xSizeDiff*strat.xMultiplier*strat.xTicker.GetBidPrice() < 1.2*strat.xMinNotional {
				return
			} else if xSizeDiff > 0 && strat.xPosition.GetSize() >= 0 && xSizeDiff*strat.xMultiplier*strat.xTicker.GetAskPrice() < 1.2*strat.xMinNotional {
				return
			}
		}

		var orderSide common.OrderSide
		if xSizeDiff < 0 {
			orderSide = common.OrderSideSell
			xSizeDiff = -xSizeDiff
		} else {
			orderSide = common.OrderSideBuy
		}
		strat.xNewOrderParam = common.NewOrderParam{
			Symbol:     strat.xSymbol,
			Side:       orderSide,
			Type:       common.OrderTypeMarket,
			Size:       xSizeDiff,
			ReduceOnly: true,
			ClientID:   strat.xExchange.GenerateClientID(),
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
		strat.yLastFilledBuyPrice = nil
		strat.yLastFilledSellPrice = nil
		strat.xOrderSilentTime = time.Now().Add(strat.config.XOrderSilent)
		logger.Debugf(
			"%10s %10s REVERSE HEDGE X BY Y, SIZE X %f Y %f, ORDER SIDE %s SIZE %f",
			strat.xSymbol, strat.ySymbol,
			strat.xPosition.GetSize()*strat.xMultiplier,
			strat.yPosition.GetSize()*strat.yMultiplier,
			orderSide,
			xSizeDiff,
		)
	}
}

func (strat *XYStrategy) hedgeYPosition() {
	if strat.xSystemStatus != common.SystemStatusReady ||
		strat.ySystemStatus != common.SystemStatusReady {
		if time.Now().Sub(strat.logSilentTime) > 0 {
			strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
			logger.Debugf("%10s hedgeYPosition xSystemStatus %v ySystemStatus %v", strat.xSymbol, strat.xSystemStatus, strat.ySystemStatus)
		}
		return
	}
	if strat.yPosition == nil ||
		strat.xPosition == nil ||
		time.Now().Sub(strat.yPositionUpdateTime) > strat.config.AccountMaxAge ||
		time.Now().Sub(strat.yOrderSilentTime) < 0 {
		//if time.Now().Sub(strat.logSilentTime) > 0 {
		//	strat.logSilentTime = time.Now().Add(strat.config.LogInterval)
		//	logger.Debugf("hedgeYPosition skipped order silent time %v positionUpdateTime %v", time.Now().Sub(strat.yOrderSilentTime), time.Now().Sub(strat.yPositionUpdateTime))
		//}
		return
	}
	var ySizeDiff float64
	if time.Now().Sub(strat.lastEnterTime) < strat.config.XEnterTimeout {
		ySizeDiff = -strat.xPosition.GetSize()*strat.xMultiplier*strat.config.HedgeRatio/strat.yMultiplier - strat.yPosition.GetSize()
	} else {
		//其他时间对冲小的size, 防止出现一边爆仓的情况
		if math.Abs(strat.xPosition.GetSize()*strat.xMultiplier) > math.Abs(strat.yPosition.GetSize()*strat.yMultiplier/strat.config.HedgeRatio) {
			//Y的size比X小，不用操作Y
			return
		} else {
			ySizeDiff = -strat.xPosition.GetSize()*strat.xMultiplier*strat.config.HedgeRatio/strat.yMultiplier - strat.yPosition.GetSize()
		}
	}

	//if strat.config.BestSizeFactor > 0 {
	//	if ySizeDiff > 0 {
	//		ySizeDiff = math.Min(strat.yTicker.GetAskSize()*strat.config.BestSizeFactor, ySizeDiff)
	//	} else {
	//		ySizeDiff = math.Max(-strat.yTicker.GetBidSize()*strat.config.BestSizeFactor, ySizeDiff)
	//	}
	//}

	if math.Abs(ySizeDiff) < strat.yStepSize {
		return
	}

	if ySizeDiff > strat.maxOrderSize {
		ySizeDiff = strat.maxOrderSize
	} else if ySizeDiff < -strat.maxOrderSize {
		ySizeDiff = -strat.maxOrderSize
	}

	if ySizeDiff >= 0 {
		ySizeDiff = math.Floor(ySizeDiff/strat.yStepSize) * strat.yStepSize
	} else {
		ySizeDiff = math.Ceil(ySizeDiff/strat.yStepSize) * strat.yStepSize
	}

	if strat.isYSpot {
		if math.Abs(ySizeDiff) < strat.yStepSize || math.Abs(ySizeDiff) < strat.yMinSize {
			return
		} else if ySizeDiff < 0 && -ySizeDiff*strat.yMultiplier*strat.yTicker.GetBidPrice() < 1.2*strat.yMinNotional {
			return
		} else if ySizeDiff > 0 && ySizeDiff*strat.yMultiplier*strat.yTicker.GetAskPrice() < 1.2*strat.yMinNotional {
			return
		}
	} else {
		//期货以close仓位，没有minNotional限制
		if math.Abs(ySizeDiff) < strat.yStepSize || math.Abs(ySizeDiff) < strat.yMinSize {
			return
		} else if ySizeDiff < 0 && strat.yPosition.GetSize() <= 0 && -ySizeDiff*strat.yMultiplier*strat.yTicker.GetBidPrice() < 1.2*strat.yMinNotional {
			return
		} else if ySizeDiff > 0 && strat.yPosition.GetSize() >= 0 && ySizeDiff*strat.yMultiplier*strat.yTicker.GetAskPrice() < 1.2*strat.yMinNotional {
			return
		}
	}

	reduceOnly := false
	if ySizeDiff*strat.yPosition.GetSize() < 0 && math.Abs(ySizeDiff)*0.995 <= math.Abs(strat.yPosition.GetSize()) {
		reduceOnly = true
	}
	orderSide := common.OrderSideBuy
	if ySizeDiff < 0 {
		orderSide = common.OrderSideSell
		ySizeDiff = -ySizeDiff
	}
	strat.yNewOrderParam = common.NewOrderParam{
		Symbol:     strat.ySymbol,
		Side:       orderSide,
		Type:       common.OrderTypeMarket,
		Size:       ySizeDiff,
		ReduceOnly: reduceOnly,
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
}
