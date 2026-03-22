package main

import "time"

func strategy1(
	depthDir,
	longThreshold,
	shortThreshold,
	addOffset,
	addValue,
	commission,
	netWorth,
	bestBidPrice,
	bestAskPrice,
	positionSize,
	positionCost,
	lastFilledPrice float64,
	eventTime time.Time,
	tradeInterval time.Duration,
) (newNetWorth float64, newPositionSize float64, newPositionCost float64, newLastFilledPrice float64, nextTradeTime time.Time) {
	nextTradeTime = eventTime
	if depthDir > longThreshold {
		if positionSize < 0 {
			if bestAskPrice > lastFilledPrice*(1-addOffset) {
				//	//做空还有盈利,平半仓
				//	netWorth += positionSize * 0.5 * (bestAskPrice - positionCost) / positionCost
				//	netWorth += -positionSize * 0.5 * commission
				//	positionSize *= 0.5
				//	lastFilledPrice = bestAskPrice
				//} else {
				netWorth += positionSize * (bestAskPrice - positionCost) / positionCost
				netWorth += -positionSize * commission
				netWorth += addValue * commission
				positionSize = addValue
				positionCost = bestAskPrice
				lastFilledPrice = bestAskPrice
			}
			nextTradeTime = eventTime.Add(tradeInterval)
		} else if positionSize == 0 && bestBidPrice > lastFilledPrice{
			netWorth += addValue * commission
			positionSize = addValue
			positionCost = bestAskPrice
			lastFilledPrice = bestAskPrice
			nextTradeTime = eventTime.Add(tradeInterval)
		} else if bestBidPrice > lastFilledPrice*(1+addOffset) {
			//做多已经盈利,加仓
			//netWorth += addValue * commission
			//positionCost = (positionSize*positionCost + addValue*bestAskPrice) / (positionSize + addValue)
			//positionSize += addValue
			lastFilledPrice = bestAskPrice
			nextTradeTime = eventTime.Add(tradeInterval)
		}
	} else if depthDir < shortThreshold {
		if positionSize > 0 {
			if bestBidPrice < lastFilledPrice*(1+addOffset) {
				//	//做多还有盈利,平半仓
				//	netWorth += positionSize * 0.5 * (bestBidPrice - positionCost) / positionCost
				//	netWorth += positionSize * 0.5 * commission
				//	positionSize *= 0.5
				//	lastFilledPrice = bestBidPrice
				//} else {
				netWorth += positionSize * (bestBidPrice - positionCost) / positionCost
				netWorth += positionSize * commission
				netWorth += addValue * commission
				positionSize = -addValue
				positionCost = bestBidPrice
				lastFilledPrice = bestBidPrice
			}
			nextTradeTime = eventTime.Add(tradeInterval)
		} else if positionSize == 0 && bestAskPrice < lastFilledPrice{
			netWorth += addValue * commission
			positionSize = -addValue
			positionCost = bestBidPrice
			lastFilledPrice = bestBidPrice
			nextTradeTime = eventTime.Add(tradeInterval)
		} else if bestAskPrice < lastFilledPrice*(1-addOffset) {
			//做空已经盈利,加仓
			//netWorth += addValue * commission
			//positionCost = (positionSize*positionCost - addValue*bestBidPrice) / (positionSize - addValue)
			//positionSize -= addValue
			lastFilledPrice = bestBidPrice
			nextTradeTime = eventTime.Add(tradeInterval)
		}
	} else {
		if positionSize > 0 && bestBidPrice < lastFilledPrice*(1+addOffset) {
			//做多在一个TradeInterval没有达到AddOffset, 平仓
			netWorth += positionSize * (bestBidPrice - positionCost) / positionCost
			netWorth += positionSize * commission
			positionSize = 0
			lastFilledPrice = bestBidPrice
			nextTradeTime = eventTime.Add(tradeInterval)
		} else if positionSize > 0 && bestBidPrice > lastFilledPrice*(1+addOffset) {
			//netWorth += addValue * commission
			//positionCost = (positionSize*positionCost + addValue*bestAskPrice) / (positionSize + addValue)
			//positionSize += addValue
			lastFilledPrice = bestAskPrice
			nextTradeTime = eventTime.Add(tradeInterval)
		} else if positionSize < 0 && bestAskPrice > lastFilledPrice*(1-addOffset) {
			netWorth += positionSize * (bestBidPrice - positionCost) / positionCost
			netWorth += -positionSize * commission
			positionSize = 0
			lastFilledPrice = bestBidPrice
			nextTradeTime = eventTime.Add(tradeInterval)
		} else if positionSize < 0 && bestAskPrice < lastFilledPrice*(1-addOffset) {
			//netWorth += addValue * commission
			//positionCost = (positionSize*positionCost - addValue*bestBidPrice) / (positionSize - addValue)
			//positionSize -= addValue
			lastFilledPrice = bestBidPrice
			nextTradeTime = eventTime.Add(tradeInterval)
		}
	}
	newNetWorth = netWorth
	newPositionCost = positionCost
	newPositionSize = positionSize
	newLastFilledPrice = lastFilledPrice
	return
}
