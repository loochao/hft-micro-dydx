package main

func (strat *XYStrategy) handleXTicker() {
	if strat.xTicker == strat.xNextTicker {
		return
	}
	if strat.xNextTicker.GetTime().Sub(strat.xTickerTime) < 0 {
		return
	}
	strat.xTicker = strat.xNextTicker
	strat.xMidPrice = 0.5 * (strat.xTicker.GetAskPrice() + strat.xTicker.GetBidPrice())
	strat.xTickerTime = strat.xTicker.GetTime()
}

func (strat *XYStrategy) handleYTicker() {
	if strat.yTicker == strat.yNextTicker {
		return
	}
	if strat.yNextTicker.GetTime().Sub(strat.yTickerTime) < 0 {
		return
	}
	strat.yTicker = strat.yNextTicker
	strat.yMidPrice = 0.5 * (strat.yTicker.GetAskPrice() + strat.yTicker.GetBidPrice())
	strat.yTickerTime = strat.yTicker.GetTime()
}
