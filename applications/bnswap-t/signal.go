package main

import (
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"math"
	"strings"
	"time"
)

func startSignalRoutine(
	ctx context.Context,
	symbol string,
	tradeLookback time.Duration,
	tradeMinCount int,
	quantile float64,
	tradeCh chan *bnswap.Trade,
	quantileCh chan float64,
	outputCh chan Signal,
) {
	minTradeAmount := math.MaxFloat64
	timer := time.NewTimer(time.Hour)
	eventTimes := make([]time.Time, 0)
	tradeDirs := make([]bool, 0)
	buyCount := 0.0
	sellCount := 0.0

	signal := Signal{
		Symbol:    symbol,
		EventTime: time.Unix(0, 0),
		Value:     0.0,
	}

	for {
		select {
		case <-ctx.Done():
			return
		case minTradeAmount = <-quantileCh:
			//logger.Debugf("%s QUANTILE %f", symbol, minTradeAmount)
			break
		case trade := <-tradeCh:
			if trade.Quantity < minTradeAmount {
				break
			}
			//logger.Debugf("%v", *trade)
			eventTimes = append(eventTimes, time.Unix(0, trade.EventTime*1000000))
			if trade.IsTheBuyerTheMarketMaker {
				tradeDirs = append(tradeDirs, false)
				sellCount += 1.0
			} else {
				tradeDirs = append(tradeDirs, true)
				buyCount += 1.0
			}
			cutIndex := 0
			for i, t := range eventTimes {
				if time.Since(t) < tradeLookback {
					cutIndex = i
					break
				}
				if tradeDirs[i] {
					buyCount -= 1.0
				} else {
					sellCount -= 1.0
				}
			}
			if cutIndex > 0 {
				eventTimes = eventTimes[cutIndex:]
				tradeDirs = tradeDirs[cutIndex:]
			} else {
				break
			}

			if len(eventTimes) < tradeMinCount {
				break
			}

			signal.Buy = int(buyCount)
			signal.Sell = int(sellCount)
			if buyCount > sellCount*(1+(1-quantile)/2) {
				signal.Value = 1.0
			} else if sellCount > buyCount*(1+(1-quantile)/2) {
				signal.Value = -1.0
			} else {
				signal.Value = 0.0
			}

			//logger.Debugf("%s %d %d %d", trade.Market, len(eventTimes), buyCount, sellCount)
			timer.Reset(time.Microsecond)
			break
		case <-timer.C:
			outputCh <- signal
		}
	}
}

func (st *Strategy) handleSignal(signal Signal) {
	symbol := signal.Symbol
	symbolIndex := GetSymbolIndex(symbol)
	if symbolIndex == -1 {
		return
	}
	st.Signals[symbolIndex] = signal

	if signal.Value == 0 {
		return
	}

	//totalDir := 0.0
	//for _, s := range st.SymbolsMap {
	//	totalDir += s.Value
	//}

	//logger.Debugf("%s BUY %d SELL %d SIGNAL %f TOTAL DIR %f", symbol, signal.Buy, signal.Sell, signal.Value, totalDir)
	//if signal.Value*totalDir >= 0 {
	//	return
	//}

	if time.Since(st.PositionsUpdateTimes[symbolIndex]) > *st.Config.PositionMaxAge {
		return
	}

	if time.Since(st.OrderSilentTimes[symbolIndex]) < 0 {
		return
	}

	position := st.Positions[symbolIndex]
	markPrice := st.MarkPrices[symbolIndex]
	if position.Symbol == "" ||
		markPrice.Symbol == "" {
		return
	}

	stepSize := st.StepSizes[symbolIndex]
	tickSize := st.TickSizes[symbolIndex]
	minNotional := st.MinNotional[symbolIndex]
	submitCh := st.OrderSubmittingChs[symbolIndex]

	targetValue := 0.0
	if signal.Value > 0 {
		targetValue = *st.Config.EnterValue
	} else {
		targetValue = -*st.Config.EnterValue
	}

	size := math.Round((targetValue-position.EntryPrice*position.PositionAmt)/markPrice.MarkPrice/stepSize) * stepSize

	if size == 0 {
		return
	}
	price := markPrice.MarkPrice
	side := common.OrderSideBuy
	if size > 0 {
		price *= 1.0 + *st.Config.EnterSlippage
		price = math.Ceil(price/tickSize) * tickSize
	} else {
		size = -size
		side = common.OrderSideSell
		price *= 1.0 - *st.Config.EnterSlippage
		price = math.Floor(price/tickSize) * tickSize
	}

	if math.Abs(price*size) < minNotional {
		return
	}
	id, _ := common.GenerateShortId()
	clOrdID := fmt.Sprintf(
		"%s-B%dS%d",
		id,
		signal.Buy,
		signal.Sell,
	)
	clOrdID = strings.ReplaceAll(clOrdID, ".", "_")

	submitCh <- bnswap.NewOrderParams{
		Symbol:           symbol,
		Side:             side,
		Type:             "LIMIT",
		Price:            price,
		TimeInForce:      "FOK",
		Quantity:         size,
		ReduceOnly:       false,
		NewClientOrderId: clOrdID,
		NewOrderRespType: common.OrderRespTypeFull,
	}
	st.OrderSilentTimes[symbolIndex] = time.Now().Add(*st.Config.OrderSilent)
	st.PositionsUpdateTimes[symbolIndex] = time.Unix(0, 0)
	st.LastOrderTimes[symbolIndex] = time.Now()
}
