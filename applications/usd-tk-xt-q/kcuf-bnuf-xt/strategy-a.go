package main

import (
	"github.com/geometrybase/hft-micro/backtests"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"time"
)

func strategyA(
	params Params,
	data []*common.MatchedSpread,
) *Result {

	result := &Result{
		NetWorth:  make([]float64, 0),
		Positions: make([]float64, 0),
		Costs:     make([]float64, 0),
		MidPrices: make([]float64, 0),
		Params:    params,
	}

	enterSilentTime := time.Time{}
	outputSilentTime := time.Time{}
	currentXValue := params.StartValue
	xPosition := &backtests.Position{}
	eventTime := time.Time{}

	lastXFrTime := time.Time{}
	tradeVolume := 0.0

	startTime := time.Unix(0, data[0].EventTime)
	endTime := startTime

	var lastXAskPrice, lastXBidPrice *float64
	var lastYAskPrice, lastYBidPrice *float64

	for _, spread := range data {
		if spread == nil {
			break
		}

		if spread.EventTime == 0 {
			logger.Debugf("%v", spread)
			break
		}

		eventTime = time.Unix(0, spread.EventTime)
		endTime = eventTime

		if lastXAskPrice == nil {
			lastXAskPrice = new(float64)
			*lastXAskPrice = spread.XAskPrice
		}
		if lastXBidPrice == nil {
			lastXBidPrice = new(float64)
			*lastXBidPrice = spread.XBidPrice
		}
		if lastYAskPrice == nil {
			lastYAskPrice = new(float64)
			*lastYAskPrice = spread.YAskPrice
		}
		if lastYBidPrice == nil {
			lastYBidPrice = new(float64)
			*lastYBidPrice = spread.YBidPrice
		}

		ignore := false
		if math.Abs(*lastXAskPrice-spread.XAskPrice) / *lastXAskPrice > 0.5 {
			//fmt.Printf("%v %s bad XAskPrice %f -> %f\n", eventTime, params.XSymbol, *lastXAskPrice, spread.XAskPrice)
			ignore = true
		}
		if math.Abs(*lastXBidPrice-spread.XBidPrice) / *lastXBidPrice > 0.5 {
			//fmt.Printf("%v %s bad XBidPrice %f -> %f\n", eventTime,params.XSymbol, *lastXBidPrice, spread.XBidPrice)
			ignore = true
		}
		if math.Abs(*lastYAskPrice-spread.YAskPrice) / *lastYAskPrice > 0.5 {
			//fmt.Printf("%v %s bad YAskPrice %f -> %f\n", eventTime,params.YSymbol, *lastYAskPrice, spread.YAskPrice)
			ignore = true
		}
		if math.Abs(*lastYBidPrice-spread.YBidPrice) / *lastYBidPrice > 0.5 {
			//fmt.Printf("%v %s bad YBidPrice %f -> %f\n", eventTime,params.YSymbol, *lastYBidPrice, spread.YBidPrice)
			ignore = true
		}
		if ignore {
			//数据有错，直接平仓
			if xPosition.Size > 0 {
				tradeVolume += math.Abs(xPosition.Size * *lastYBidPrice)
				currentXValue += xPosition.Size * *lastYBidPrice * params.TradeCost
				currentXValue += xPosition.Add(-xPosition.Size, *lastYBidPrice)
			} else if xPosition.Size < 0 {
				tradeVolume += math.Abs(xPosition.Size * *lastYAskPrice)
				currentXValue += -xPosition.Size * *lastYAskPrice * params.TradeCost
				currentXValue += xPosition.Add(-xPosition.Size, *lastYAskPrice)
			}
			continue
		}
		*lastXAskPrice = spread.XAskPrice
		*lastXBidPrice = spread.XBidPrice
		*lastYAskPrice = spread.YAskPrice
		*lastYBidPrice = spread.YBidPrice

		unrealisedXPnl := xPosition.GetUnrealisedPnl((spread.XBidPrice + spread.XAskPrice) * 0.5)

		if eventTime.Sub(enterSilentTime) > 0 {
			shortTop := spread.SpreadQuantile50 + params.EnterOffset - params.FrFactor*(spread.YFundingRate-spread.XFundingRate)
			shortBot := spread.SpreadQuantile50 + params.LeaveOffset - params.FrFactor*(spread.YFundingRate-spread.XFundingRate)

			longTop := spread.SpreadQuantile50 - params.LeaveOffset - params.FrFactor*(spread.YFundingRate-spread.XFundingRate)
			longBot := spread.SpreadQuantile50 - params.EnterOffset - params.FrFactor*(spread.YFundingRate-spread.XFundingRate)

			frClose := math.Abs(spread.XFundingRate) > params.MaxFundingRate ||
				math.Abs(spread.YFundingRate) > params.MaxFundingRate ||
				math.Abs(spread.XFundingRate-spread.YFundingRate) > params.MaxFundingRate

			stopLoss := false
			if xPosition.Size != 0 && unrealisedXPnl/math.Abs(xPosition.Size*xPosition.Price) < params.StopLoss{
				stopLoss = true
			}

			shortBotClose := spread.ShortMedianSpread < shortBot &&
				spread.ShortMedianSpread < spread.SpreadQuantile20 &&
				spread.ShortLastSpread <= spread.ShortMedianSpread

			longTopClose := spread.LongMedianSpread > longTop &&
				spread.LongMedianSpread > spread.SpreadQuantile80 &&
				spread.LongLastSpread >= spread.LongMedianSpread

			shortTopOpen := spread.ShortMedianSpread > shortTop &&
				spread.ShortMedianSpread > spread.SpreadQuantile95 &&
				spread.ShortLastSpread >= spread.ShortMedianSpread

			longBotOpen := spread.LongMedianSpread < longBot &&
				spread.LongMedianSpread < spread.SpreadQuantile05 &&
				spread.LongLastSpread <= spread.LongMedianSpread

			if xPosition.Size > 0 &&
				(frClose || shortBotClose || stopLoss) {
				tradeValue := math.Min(
					spread.XBidPrice*spread.XBidSize*params.BestSizeFactor,
					spread.YAskPrice*spread.YAskSize*params.BestSizeFactor,
				)
				tradeXSize := math.Min(
					tradeValue/spread.XBidPrice,
					xPosition.Size,
				)
				tradeVolume += math.Abs(tradeXSize * spread.XBidPrice)
				currentXValue += tradeXSize * spread.XBidPrice * params.TradeCost
				currentXValue += xPosition.Add(-tradeXSize, spread.XBidPrice)
				enterSilentTime = eventTime.Add(params.enterInterval)
			} else if xPosition.Size < 0 &&
				(frClose || longTopClose || stopLoss) {
				tradeValue := math.Min(
					spread.XAskPrice*spread.XAskSize*params.BestSizeFactor,
					spread.YBidPrice*spread.YBidSize*params.BestSizeFactor,
				)
				tradeXSize := math.Min(
					tradeValue/spread.XAskPrice,
					-xPosition.Size,
				)
				tradeVolume += math.Abs(tradeXSize * spread.XAskPrice)
				currentXValue += tradeXSize * spread.XAskPrice * params.TradeCost
				currentXValue += xPosition.Add(tradeXSize, spread.XAskPrice)
				enterSilentTime = eventTime.Add(params.enterInterval)
			} else if shortTopOpen &&
				!frClose && !stopLoss &&
				xPosition.Size >= 0 &&
				unrealisedXPnl >= 0 {
				freeUSD := currentXValue - math.Abs(xPosition.Size*xPosition.Price/params.Leverage)
				tradeValue := math.Min(
					spread.XAskPrice*spread.XAskSize*params.BestSizeFactor,
					spread.YBidPrice*spread.YBidSize*params.BestSizeFactor,
				)
				tradeValue = math.Min(freeUSD*params.EnterStep, tradeValue)
				if freeUSD < tradeValue/params.Leverage {
					continue
				}
				tradeVolume += tradeValue * 2
				currentXValue += tradeValue * params.TradeCost
				currentXValue += xPosition.Add(tradeValue/spread.XAskPrice, spread.XAskPrice)
				enterSilentTime = eventTime.Add(params.enterInterval)
			} else if longBotOpen &&
				!frClose && !stopLoss &&
				xPosition.Size <= 0 &&
				unrealisedXPnl >= 0 {
				freeUSD := currentXValue - math.Abs(xPosition.Size*xPosition.Price/params.Leverage)
				tradeValue := math.Min(
					spread.XBidPrice*spread.XBidSize*params.BestSizeFactor,
					spread.YAskPrice*spread.YAskSize*params.BestSizeFactor,
				)
				tradeValue = math.Min(freeUSD*params.EnterStep, tradeValue)
				if freeUSD < tradeValue/params.Leverage {
					continue
				}
				tradeVolume += tradeValue * 2
				currentXValue += tradeValue * params.TradeCost
				currentXValue += xPosition.Add(-tradeValue/spread.XBidPrice, spread.XBidPrice)
				enterSilentTime = eventTime.Add(params.enterInterval)
			}
		}

		xFrDiff := eventTime.Sub(eventTime.Add(time.Hour * 4).Truncate(time.Hour * 8).Add(time.Hour * 4))
		if xFrDiff < time.Minute &&
			xFrDiff > -time.Minute &&
			eventTime.Sub(lastXFrTime) > time.Hour {
			frRet := -spread.XFundingRate * xPosition.Size * xPosition.Price
			if frRet != 0 {
				//logger.Debugf("X %v %f %f", eventTime, spread.XFundingRate, frRet)
				currentXValue += frRet
			}
			lastXFrTime = eventTime
		}

		if eventTime.Sub(outputSilentTime) > 0 {
			//logger.Debugf("%f %f %f %f", currentXValue, currentXValue, unrealisedXPnl, unrealisedYPnl)
			result.NetWorth = append(result.NetWorth, (currentXValue+unrealisedXPnl)/params.StartValue)
			result.Positions = append(result.Positions, xPosition.Size*xPosition.Price)
			result.EventTimes = append(result.EventTimes, eventTime)
			if xPosition.Size != 0 {
				result.Costs = append(result.Costs, xPosition.Price)
			} else {
				result.Costs = append(result.Costs, (spread.XBidPrice+spread.XAskPrice)/2)
			}
			result.MidPrices = append(result.MidPrices, (spread.XBidPrice+spread.XAskPrice)/2)
			result.FundingRates = append(result.FundingRates, spread.YFundingRate-spread.XFundingRate)
			outputSilentTime = eventTime.Add(params.OutputInterval)
		}
	}
	result.Turnover = tradeVolume / params.StartValue / float64(endTime.Sub(startTime)/(time.Hour*24))
	return result
}
