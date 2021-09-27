package main

import (
	"github.com/geometrybase/hft-micro/backtests"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"time"
)

func strategyB(
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
	currentYValue := params.StartValue
	yPosition := &backtests.Position{}
	eventTime := time.Time{}

	lastYFrTime := time.Time{}
	tradeVolume := 0.0

	startTime := time.Unix(0, data[0].EventTime)
	endTime := startTime

	//logger.Debugf("%v", endTime)

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

		unrealisedYPnl := yPosition.GetUnrealisedPnl((spread.YBidPrice + spread.YAskPrice) * 0.5)

		if eventTime.Sub(enterSilentTime) > 0 {
			shortTop := spread.SpreadQuantile50 + params.EnterOffset  - params.FrFactor*(spread.YFundingRate-spread.XFundingRate)
			shortBot := spread.SpreadQuantile50 + params.LeaveOffset  - params.FrFactor*(spread.YFundingRate-spread.XFundingRate)

			longTop := spread.SpreadQuantile50 - params.LeaveOffset  - params.FrFactor*(spread.YFundingRate-spread.XFundingRate)
			longBot := spread.SpreadQuantile50 - params.EnterOffset  - params.FrFactor*(spread.YFundingRate-spread.XFundingRate)

			frClose := math.Abs(spread.XFundingRate) > params.MaxFundingRate ||
				math.Abs(spread.YFundingRate) > params.MaxFundingRate ||
				math.Abs(spread.XFundingRate-spread.YFundingRate) > params.MaxFundingRate

			shortBotClose := spread.ShortMedianSpread < shortBot &&
				spread.ShortMedianSpread < spread.SpreadQuantile05 &&
				spread.ShortLastSpread <= spread.ShortMedianSpread

			longTopClose := spread.LongMedianSpread > longTop &&
				spread.LongMedianSpread > spread.SpreadQuantile95 &&
				spread.LongLastSpread >= spread.LongMedianSpread

			shortTopOpen := spread.ShortMedianSpread > shortTop &&
				spread.ShortMedianSpread > spread.SpreadQuantile995 &&
				spread.ShortLastSpread >= spread.ShortMedianSpread

			longBotOpen := spread.LongMedianSpread < longBot &&
				spread.LongMedianSpread < spread.SpreadQuantile005 &&
				spread.LongLastSpread <= spread.LongMedianSpread

			if yPosition.Size < 0 &&
				(frClose || shortBotClose) {
				tradeValue := spread.YBidPrice*spread.YBidSize*params.BestSizeFactor
				tradeYSize := math.Min(
					tradeValue/spread.YAskPrice,
					-yPosition.Size,
				)
				tradeVolume += math.Abs(tradeYSize * spread.YAskPrice)
				currentYValue += tradeYSize * spread.YAskPrice * params.TradeCost
				currentYValue += yPosition.Add(tradeYSize, spread.YAskPrice)
				enterSilentTime = eventTime.Add(params.enterInterval)
			} else if yPosition.Size > 0 &&
				(frClose || longTopClose) {
				tradeValue := spread.YBidPrice*spread.YBidSize*params.BestSizeFactor
				tradeYSize := math.Min(
					tradeValue/spread.YBidPrice,
					yPosition.Size,
				)
				tradeVolume += math.Abs(tradeYSize * spread.YBidPrice)
				currentYValue += tradeYSize * spread.YBidPrice * params.TradeCost
				currentYValue += yPosition.Add(-tradeYSize, spread.YBidPrice)
				enterSilentTime = eventTime.Add(params.enterInterval)
			} else if shortTopOpen &&
				!frClose &&
				yPosition.Size <= 0 &&
				unrealisedYPnl >= 0 {
				freeUSD := currentYValue - math.Abs(yPosition.Size*yPosition.Price/params.Leverage)
				tradeValue := spread.YAskPrice*spread.YAskSize*params.BestSizeFactor
				tradeValue = math.Min(freeUSD*params.EnterStep, tradeValue)
				if freeUSD < tradeValue/params.Leverage {
					continue
				}
				tradeVolume += tradeValue * 2
				currentYValue += tradeValue * params.TradeCost
				currentYValue += yPosition.Add(-tradeValue/spread.YBidPrice, spread.YBidPrice)
				enterSilentTime = eventTime.Add(params.enterInterval)
			} else if longBotOpen &&
				!frClose &&
				yPosition.Size >= 0 &&
				unrealisedYPnl >= 0 {
				freeUSD := currentYValue - math.Abs(yPosition.Size*yPosition.Price/params.Leverage)
				tradeValue := spread.YBidPrice*spread.YBidSize*params.BestSizeFactor
				tradeValue = math.Min(freeUSD*params.EnterStep, tradeValue)
				if freeUSD < tradeValue/params.Leverage {
					continue
				}
				tradeVolume += tradeValue * 2
				currentYValue += tradeValue * params.TradeCost
				currentYValue += yPosition.Add(tradeValue/spread.YAskPrice, spread.YAskPrice)
				enterSilentTime = eventTime.Add(params.enterInterval)
			}
		}

		yFrDiff := eventTime.Sub(eventTime.Truncate(time.Hour * 8).Add(time.Hour * 8))
		if yFrDiff < time.Minute &&
			yFrDiff > -time.Minute &&
			eventTime.Sub(lastYFrTime) > time.Hour {
			frRet := -spread.YFundingRate * yPosition.Size * yPosition.Price
			if frRet != 0 {
				//logger.Debugf("X %v %f %f", eventTime, spread.XFundingRate, frRet)
				currentYValue += frRet
			}
			lastYFrTime = eventTime
		}

		if eventTime.Sub(outputSilentTime) > 0 {
			//logger.Debugf("%f %f %f %f", currentYValue, currentYValue, unrealisedYPnl, unrealisedYPnl)
			result.NetWorth = append(result.NetWorth, (currentYValue+unrealisedYPnl)/params.StartValue)
			result.Positions = append(result.Positions, yPosition.Size*yPosition.Price)
			result.EventTimes = append(result.EventTimes, eventTime)
			if yPosition.Size != 0 {
				result.Costs = append(result.Costs, yPosition.Price)
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
