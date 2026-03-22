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
		Params:    params,
	}

	enterSilentTime := time.Time{}
	outputSilentTime := time.Time{}
	currentXValue := params.startValue * 0.5
	currentYValue := params.startValue * 0.5
	xPosition := &backtests.Position{}
	yPosition := &backtests.Position{}
	eventTime := time.Time{}

	lastXFrTime := time.Time{}
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

		unrealisedXPnl := xPosition.GetUnrealisedPnl((spread.XBidPrice + spread.XAskPrice) * 0.5)
		unrealisedYPnl := yPosition.GetUnrealisedPnl((spread.YBidPrice + spread.YAskPrice) * 0.5)

		if math.Abs(currentYValue-currentYValue) > params.startValue*0.25 {
			currentYValue = (currentYValue + currentXValue) * 0.5
			currentXValue = currentYValue
		}

		if eventTime.Sub(enterSilentTime) > 0 {
			shortTop := spread.SpreadQuantile50 + params.enterOffset - params.frFactor*(spread.YFundingRate - spread.XFundingRate)
			shortBot := spread.SpreadQuantile50 - params.leaveOffset - params.frFactor*(spread.YFundingRate - spread.XFundingRate)
			longTop := spread.SpreadQuantile50 + params.leaveOffset - params.frFactor*(spread.YFundingRate - spread.XFundingRate)
			longBot := spread.SpreadQuantile50 - params.enterOffset - params.frFactor*(spread.YFundingRate - spread.XFundingRate)

			if spread.ShortMedianSpread < shortBot &&
				spread.ShortLastSpread <= spread.ShortMedianSpread &&
				xPosition.Size > 0 {
				tradeValue := math.Min(
					spread.XBidPrice*spread.XBidSize*params.bestSizeFactor,
					spread.YAskPrice*spread.YAskSize*params.bestSizeFactor,
				)
				tradeXSize := math.Min(
					tradeValue/spread.XBidPrice,
					xPosition.Size,
				)
				tradeYSize := math.Min(
					tradeValue/spread.YAskPrice,
					math.Abs(yPosition.Size),
				)
				tradeVolume += math.Abs(tradeXSize * spread.XBidPrice) + math.Abs(tradeYSize * spread.YAskPrice)
				currentXValue += tradeXSize * spread.XBidPrice * params.tradeCost
				currentYValue += tradeYSize * spread.YAskPrice * params.tradeCost
				currentXValue += xPosition.Add(-tradeXSize, spread.XBidPrice)
				currentYValue += yPosition.Add(tradeYSize, spread.YAskPrice)
				enterSilentTime = eventTime.Add(params.enterInterval)
				//logger.Debugf("%v SHORT CLOSE %f %f", eventTime, spread.ShortLastSpread-spread.SpreadQuantile50, spread.ShortMedianSpread-spread.SpreadQuantile50)
			} else if spread.LongMedianSpread > longTop &&
				spread.LongLastSpread >= spread.LongMedianSpread &&
				xPosition.Size < 0 {
				tradeValue := math.Min(
					spread.XAskPrice*spread.XAskSize*params.bestSizeFactor,
					spread.YBidPrice*spread.YBidSize*params.bestSizeFactor,
				)
				tradeXSize := math.Min(
					tradeValue/spread.XAskPrice,
					-xPosition.Size,
				)
				tradeYSize := math.Min(
					tradeValue/spread.YBidPrice,
					math.Abs(yPosition.Size),
				)
				tradeVolume += math.Abs(tradeXSize * spread.XAskPrice) + math.Abs(tradeYSize * spread.YBidPrice)
				currentXValue += tradeXSize * spread.XAskPrice * params.tradeCost
				currentYValue += tradeYSize * spread.YBidPrice * params.tradeCost
				currentXValue += xPosition.Add(tradeXSize, spread.XAskPrice)
				currentYValue += yPosition.Add(-tradeYSize, spread.YBidPrice)
				enterSilentTime = eventTime.Add(params.enterInterval)
				//logger.Debugf("%v LONG CLOSE %f %f", eventTime, spread.LongLastSpread-spread.SpreadQuantile50, spread.LongMedianSpread-spread.SpreadQuantile50)
			} else if spread.ShortMedianSpread > shortTop &&
				spread.ShortLastSpread >= spread.ShortMedianSpread &&
				xPosition.Size >= 0 {
				freeUSD := math.Min(
					currentXValue-math.Abs(xPosition.Size*xPosition.Price/params.leverage),
					currentYValue-math.Abs(yPosition.Size*yPosition.Price/params.leverage),
				)
				tradeValue := math.Min(
					spread.XAskPrice*spread.XAskSize*params.bestSizeFactor,
					spread.YBidPrice*spread.YBidSize*params.bestSizeFactor,
				)
				tradeValue = math.Min(freeUSD*params.enterStep, tradeValue)
				if freeUSD < tradeValue/params.leverage {
					continue
				}
				tradeVolume += tradeValue*2
				currentXValue += tradeValue * params.tradeCost
				currentYValue += tradeValue * params.tradeCost
				currentXValue += xPosition.Add(tradeValue/spread.XAskPrice, spread.XAskPrice)
				currentYValue += yPosition.Add(-tradeValue/spread.YBidPrice, spread.YBidPrice)
				enterSilentTime = eventTime.Add(params.enterInterval)
				//logger.Debugf("%v SHORT OPEN %f %f", eventTime,spread.ShortLastSpread-spread.SpreadQuantile50, spread.ShortMedianSpread-spread.SpreadQuantile50)
			} else if spread.LongMedianSpread < longBot &&
				spread.LongLastSpread <= spread.LongMedianSpread &&
				xPosition.Size <= 0 {
				freeUSD := math.Min(
					currentXValue-math.Abs(xPosition.Size*xPosition.Price/params.leverage),
					currentYValue-math.Abs(yPosition.Size*yPosition.Price/params.leverage),
				)
				tradeValue := math.Min(
					spread.XBidPrice*spread.XBidSize*params.bestSizeFactor,
					spread.YAskPrice*spread.YAskSize*params.bestSizeFactor,
				)
				tradeValue = math.Min(freeUSD*params.enterStep, tradeValue)
				if freeUSD < tradeValue/params.leverage {
					continue
				}
				tradeVolume += tradeValue*2
				currentXValue += tradeValue * params.tradeCost
				currentYValue += tradeValue * params.tradeCost
				currentXValue += xPosition.Add(-tradeValue/spread.XBidPrice, spread.XBidPrice)
				currentYValue += yPosition.Add(tradeValue/spread.YAskPrice, spread.YAskPrice)
				enterSilentTime = eventTime.Add(params.enterInterval)
				//logger.Debugf("%v LONG OPEN %f %f", eventTime,spread.LongLastSpread-spread.SpreadQuantile50, spread.LongMedianSpread-spread.SpreadQuantile50)
			}
		}

		xFrDiff := eventTime.Sub(eventTime.Add(time.Hour * 4).Truncate(time.Hour * 8).Add(time.Hour * 4))
		yFrDiff := eventTime.Sub(eventTime.Truncate(time.Hour * 8).Add(time.Hour * 8))
		if xFrDiff < time.Minute &&
			xFrDiff > -time.Minute &&
			eventTime.Sub(lastXFrTime) > time.Hour {
			 frRet := -spread.XFundingRate*xPosition.Size*xPosition.Price
			 if frRet != 0 {
			 	//logger.Debugf("X %v %f %f", eventTime, spread.XFundingRate, frRet)
			 	currentXValue += frRet
			 }
			 lastXFrTime = eventTime
		}
		if yFrDiff < time.Minute &&
			yFrDiff > -time.Minute &&
			eventTime.Sub(lastYFrTime) > time.Hour {
			frRet := -spread.YFundingRate*yPosition.Size*yPosition.Price
			if frRet != 0 {
				//logger.Debugf("Y %v %f %f", eventTime, spread.YFundingRate, frRet)
				currentYValue += frRet
			}
			lastYFrTime = eventTime
		}


		if eventTime.Sub(outputSilentTime) > 0 {
			//logger.Debugf("%f %f %f %f", currentXValue, currentYValue, unrealisedXPnl, unrealisedYPnl)
			result.NetWorth = append(result.NetWorth, (currentXValue+currentYValue+unrealisedXPnl+unrealisedYPnl)/params.startValue)
			result.Positions = append(result.Positions, xPosition.Size*xPosition.Price)
			outputSilentTime = eventTime.Add(params.outputInterval)
		}
	}
	result.Turnover = tradeVolume/params.startValue/float64(endTime.Sub(startTime)/(time.Hour*24))
	return result
}
