package main

import (
	"time"
)

func strategy1(
	tradeBookRatio,
	tradeDir,
	addOffset,
	addValue,
	commission,
	netWorth,
	bestBidPrice,
	bestAskPrice,
	meanPrice,
	positionSize,
	positionCost,
	lastMarkedPrice float64,
	eventTime time.Time,
	tradeInterval time.Duration,
) (newNetWorth float64, newPositionSize float64, newPositionCost float64, newLastMarkedPrice float64, nextTradeTime time.Time) {
	nextTradeTime = eventTime
	newLastMarkedPrice = lastMarkedPrice
	if tradeBookRatio > 0.1  {
		if tradeDir > 0 {
			if positionSize < 0 {
				if meanPrice > lastMarkedPrice*(1-addOffset) {
					netWorth += positionSize * (bestAskPrice - positionCost) / positionCost
					netWorth += -positionSize * commission
					netWorth += addValue * commission
					positionSize = addValue
					positionCost = bestAskPrice
					newLastMarkedPrice = meanPrice
					nextTradeTime = eventTime.Add(tradeInterval)
				}
			} else if positionSize == 0 {
				netWorth += addValue * commission
				positionSize = addValue
				positionCost = bestAskPrice
				newLastMarkedPrice = meanPrice
				nextTradeTime = eventTime.Add(tradeInterval)
			} else if meanPrice > lastMarkedPrice*(1+addOffset) {
				//做多已经盈利,加仓
				netWorth += addValue * commission
				positionCost = (positionSize*positionCost + addValue*bestAskPrice) / (positionSize + addValue)
				positionSize += addValue
				newLastMarkedPrice = meanPrice
				nextTradeTime = eventTime.Add(tradeInterval)
			}
		} else {
			if positionSize > 0 {
				if meanPrice < lastMarkedPrice*(1+addOffset) {

					netWorth += positionSize * (bestBidPrice - positionCost) / positionCost
					netWorth += positionSize * commission

					netWorth += addValue * commission
					positionSize = -addValue
					positionCost = bestBidPrice
					newLastMarkedPrice = meanPrice
					nextTradeTime = eventTime.Add(tradeInterval)
				}
			} else if positionSize == 0 {
				netWorth += addValue * commission
				positionSize = -addValue
				positionCost = bestBidPrice
				newLastMarkedPrice = meanPrice
				nextTradeTime = eventTime.Add(tradeInterval)
			} else if meanPrice < lastMarkedPrice*(1-addOffset) {
				//做空已经盈利,加仓
				netWorth += addValue * commission
				positionCost = (positionSize*positionCost - addValue*bestBidPrice) / (positionSize - addValue)
				positionSize -= addValue
				newLastMarkedPrice = meanPrice
				nextTradeTime = eventTime.Add(tradeInterval)
			}
		}
	} else {
		if positionSize > 0 && meanPrice < lastMarkedPrice*(1+addOffset) {
			//做多在一个TradeInterval没有达到AddOffset, 平仓
			netWorth += positionSize * (bestBidPrice - positionCost) / positionCost
			netWorth += positionSize * commission
			positionSize = 0
			newLastMarkedPrice = meanPrice
			nextTradeTime = eventTime.Add(tradeInterval)
		} else if positionSize > 0 && meanPrice > lastMarkedPrice*(1+addOffset) {
			//netWorth += addValue * commission
			//positionCost = (positionSize*positionCost + addValue*bestAskPrice) / (positionSize + addValue)
			//positionSize += addValue
			newLastMarkedPrice = meanPrice
			nextTradeTime = eventTime.Add(tradeInterval)
		} else if positionSize < 0 && meanPrice > lastMarkedPrice*(1-addOffset) {
			//做空在一个TradeInterval没有达到AddOffset, 平仓
			netWorth += positionSize * (bestBidPrice - positionCost) / positionCost
			netWorth += -positionSize * commission
			positionSize = 0
			newLastMarkedPrice = meanPrice
			nextTradeTime = eventTime.Add(tradeInterval)
		} else if positionSize < 0 && meanPrice < lastMarkedPrice*(1-addOffset) {
			//netWorth += addValue * commission
			//positionCost = (positionSize*positionCost - addValue*bestBidPrice) / (positionSize - addValue)
			//positionSize -= addValue
			newLastMarkedPrice = meanPrice
			nextTradeTime = eventTime.Add(tradeInterval)
		}
	}

	newNetWorth = netWorth
	newPositionCost = positionCost
	newPositionSize = positionSize
	return
}
