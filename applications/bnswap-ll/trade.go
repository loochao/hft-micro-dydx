package main

import (
	"context"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

func watchTrade(
	ctx context.Context,
	proxyAddress string,
	symbol string,
	fastLookBack, slowLookBack time.Duration,
	outputCh chan Signal,
) {
	ws := bnswap.NewTradeWebsocket(
		ctx,
		[]string{symbol},
		time.Minute,
		proxyAddress,
	)
	defer ws.Stop()

	fastBuyPrices := make([]float64, 0)
	fastSellPrices := make([]float64, 0)
	fastBuyArrivalTimes := make([]time.Time, 0)
	fastSellArrivalTimes := make([]time.Time, 0)
	fastBuyPricesSortedSlices := common.SortedFloatSlice{}
	fastSellPricesSortedSlices := common.SortedFloatSlice{}

	slowBuyPrices := make([]float64, 0)
	slowSellPrices := make([]float64, 0)
	slowBuyArrivalTimes := make([]time.Time, 0)
	slowSellArrivalTimes := make([]time.Time, 0)
	slowBuyPricesSortedSlices := common.SortedFloatSlice{}
	slowSellPricesSortedSlices := common.SortedFloatSlice{}

	hasSlowSell := 0
	hasFastSell := 0

	hasSlowBuy := 0
	hasFastBuy := 0

	signal := Signal{}
	for {
		select {
		case <-ws.Done():
			logger.Fatal("TRADE WS CONTEXT DONE %s", symbol)
		case <-ctx.Done():
			return
		case trade := <-ws.DataCh:
			if trade.Quantity < 0.01 {
				break
			}
			if trade.IsTheBuyerTheMarketMaker {
				//sell trade
				fastSellPrices = append(fastSellPrices, trade.Price)
				slowSellPrices = append(slowSellPrices, trade.Price)

				fastSellArrivalTimes = append(fastSellArrivalTimes, trade.EventTime)
				slowSellArrivalTimes = append(slowSellArrivalTimes, trade.EventTime)

				fastSellPricesSortedSlices = fastSellPricesSortedSlices.Insert(trade.Price)
				slowSellPricesSortedSlices = slowSellPricesSortedSlices.Insert(trade.Price)

				fastSellCutIndex := 0
				for i, t := range fastSellArrivalTimes {
					if time.Now().Sub(t) <= fastLookBack {
						fastSellCutIndex = i
						break
					}
				}
				if fastSellCutIndex > 0 {
					hasFastSell = 1
					for _, p := range fastSellPrices[:fastSellCutIndex] {
						fastSellPricesSortedSlices = fastSellPricesSortedSlices.Delete(p)
					}
				}

				slowSellCutIndex := 0
				for i, t := range slowSellArrivalTimes {
					if time.Now().Sub(t) <= slowLookBack {
						slowSellCutIndex = i
						break
					}
				}
				if slowSellCutIndex > 0 {
					hasSlowSell = 1
					for _, p := range slowSellPrices[:slowSellCutIndex] {
						slowSellPricesSortedSlices = slowSellPricesSortedSlices.Delete(p)
					}
				}

				fastSellArrivalTimes = fastSellArrivalTimes[fastSellCutIndex:]
				fastSellPrices = fastSellPrices[fastSellCutIndex:]
				slowSellArrivalTimes = slowSellArrivalTimes[slowSellCutIndex:]
				slowSellPrices = slowSellPrices[slowSellCutIndex:]
			} else {
				//buy trade

				fastBuyPrices = append(fastBuyPrices, trade.Price)
				slowBuyPrices = append(slowBuyPrices, trade.Price)

				fastBuyArrivalTimes = append(fastBuyArrivalTimes, trade.EventTime)
				slowBuyArrivalTimes = append(slowBuyArrivalTimes, trade.EventTime)

				fastBuyPricesSortedSlices = fastBuyPricesSortedSlices.Insert(trade.Price)
				slowBuyPricesSortedSlices = slowBuyPricesSortedSlices.Insert(trade.Price)

				fastBuyCutIndex := 0
				for i, t := range fastBuyArrivalTimes {
					if time.Now().Sub(t) <= fastLookBack {
						fastBuyCutIndex = i
						break
					}
				}
				if fastBuyCutIndex > 0 {
					hasFastBuy = 1
					for _, p := range fastBuyPrices[:fastBuyCutIndex] {
						fastBuyPricesSortedSlices = fastBuyPricesSortedSlices.Delete(p)
					}
				}

				slowBuyCutIndex := 0
				for i, t := range slowBuyArrivalTimes {
					if time.Now().Sub(t) <= slowLookBack {
						slowBuyCutIndex = i
						break
					}
				}
				if slowBuyCutIndex > 0 {
					hasSlowBuy = 1
					for _, p := range slowBuyPrices[:slowBuyCutIndex] {
						slowBuyPricesSortedSlices = slowBuyPricesSortedSlices.Delete(p)
					}
				}
				fastBuyArrivalTimes = fastBuyArrivalTimes[fastBuyCutIndex:]
				fastBuyPrices = fastBuyPrices[fastBuyCutIndex:]
				slowBuyArrivalTimes = slowBuyArrivalTimes[slowBuyCutIndex:]
				slowBuyPrices = slowBuyPrices[slowBuyCutIndex:]
			}

			if hasFastSell == 0 ||
				hasSlowSell == 0 ||
				hasFastBuy == 0 ||
				hasSlowBuy == 0 {
				break
			}
			signal.FastBuyPrice = fastBuyPricesSortedSlices.Median()
			signal.SlowBuyPrice = slowBuyPricesSortedSlices.Median()
			signal.FastSellPrice = fastSellPricesSortedSlices.Median()
			signal.SlowSellPrice = slowSellPricesSortedSlices.Median()
			if signal.FastBuyPrice < signal.SlowBuyPrice &&
				signal.FastSellPrice < signal.SlowSellPrice &&
				signal.FastSellPrice-signal.SlowSellPrice < 1.0*(signal.FastBuyPrice-signal.SlowBuyPrice) {
				signal.Direction = -1
			} else if signal.FastBuyPrice > signal.SlowBuyPrice &&
				signal.FastSellPrice > signal.SlowSellPrice &&
				1.0*(signal.FastSellPrice-signal.SlowSellPrice) < signal.FastBuyPrice-signal.SlowBuyPrice {
				signal.Direction = 1
			}
			outputCh <- signal
			break
		}
	}
}
