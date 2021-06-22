package main

import (
	"fmt"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"math/rand"
	"time"
)

func updatePositions() {

	if bnAccount == nil || bnAccount.AvailableBalance == nil {
		return
	}
	if bnTimeEmaDelta == nil {
		return
	}

	entryStep := *bnAccount.AvailableBalance * *bnConfig.EnterFreePct
	if entryStep < *bnConfig.EnterMinimalStep {
		entryStep = *bnConfig.EnterMinimalStep
	}
	entryTarget := entryStep * *bnConfig.EnterTargetFactor

	//遍历合约 从最大的rank 开始，能保证FR强的先下单, 优先做空
	for _, symbol := range bnSymbols[:*bnConfig.TradeSymbolIndex] {
		if time.Now().Sub(bnNextLoopTimes[symbol]) > 0 {
			continue
		}
		if time.Now().Sub(bnPositionsUpdateTimes[symbol]) > *bnConfig.PositionMaxAge {
			continue
		}
		if time.Now().Sub(bnOrderSilentTimes[symbol]) < 0 {
			continue
		}
		if time.Now().Sub(bnEnterSilentTimes[symbol]) < 0 {
			continue
		}

		quantile, okQuantile := bnQuantiles[symbol]
		position, okPosition := bnPositions[symbol]
		bidPrice, okBidPrice := bnBidPrices[symbol]
		if !okPosition || !okQuantile || !okBidPrice {
			continue
		}
		tickSize := bnTickSizes[symbol]
		minNotional := bnMinNotional[symbol]
		stepSize := bnStepSizes[symbol]

		if _, ok := bnOpenOrders[symbol]; ok {
			bnOrderRequestChs[symbol] <- OrderRequest{
				Cancel: &bnswap.CancelAllOrderParams{
					Symbol: symbol,
				},
			}
		}
		bnNextLoopTimes[symbol] = time.Now().Add(*bnConfig.SymbolLoopInterval)
		if time.Now().Sub(bnEnterSilentTimes[symbol]) < 0 &&
			bnSystemOverHeated &&
			quantile.Dir < 0 {
			price := math.Floor((bidPrice.Price - *bnTimeEmaDelta / *bnConfig.EnterThreshold * quantile.Top)/tickSize) * tickSize
			targetValue := position.PositionAmt*position.EntryPrice + entryStep
			if targetValue > entryTarget {
				targetValue = entryTarget
			}
			entryValue := targetValue - position.PositionAmt*position.EntryPrice
			if entryValue > *bnAccount.AvailableBalance*0.8 {
				entryValue = *bnAccount.AvailableBalance * 0.8
			}
			size := entryValue / price
			size = math.Round(size/stepSize) * stepSize
			entryValue = size * price

			if entryValue > *bnAccount.AvailableBalance {
				if time.Now().Sub(bnLogSilentTimes[symbol]) > 0 {
					logger.Debugf(
						"%s FAILED OPEN LONG ENTRY VALUE %f MORE THAN AvailableBalance %f, EMA DELTA %f",
						symbol,
						entryValue,
						*bnAccount.AvailableBalance,
						*bnTimeEmaDelta,
					)
					bnLogSilentTimes[symbol] = time.Now().Add(*bnConfig.LogInterval)
				}
				continue
			}
			if entryValue < minNotional {
				if time.Now().Sub(bnLogSilentTimes[symbol]) > 0 {
					logger.Debugf(
						"%s FAILED OPEN LONG ENTRY VALUE %f LESS THAN minNotional %f, EMA DELTA %f",
						symbol,
						entryValue,
						minNotional,
						*bnTimeEmaDelta,
					)
					bnLogSilentTimes[symbol] = time.Now().Add(*bnConfig.LogInterval)
				}
				continue
			}
			if size <= 0 {
				continue
			}
			order := bnswap.NewOrderParams{
				Symbol:           symbol,
				Side:             "BUY",
				Type:             "LIMIT",
				Price:            price,
				TimeInForce:      "GTC",
				Quantity:         size,
				ReduceOnly:       false,
				NewClientOrderId: fmt.Sprintf("%d", time.Now().Unix()*10000+int64(rand.Intn(10000))),
			}
			logger.Debugf("OPEN ORDER %v", order.ToString())
			bnOrderSilentTimes[symbol] = time.Now().Add(*bnConfig.OrderSilent)
			bnPositionsUpdateTimes[symbol] = time.Unix(0, 0)
			bnHttpPositionUpdateSilentTimes[symbol] = time.Now().Add(*bnConfig.HttpSilent)
			bnOrderRequestChs[symbol] <- OrderRequest{
				New: &order,
			}
		} else if !bnSystemOverHeated && position.PositionAmt > 0 {
			price := math.Floor((bidPrice.Price + *bnTimeEmaDelta / *bnConfig.EnterThreshold * quantile.Top)/tickSize) * tickSize
			entryValue := math.Max(4*entryStep, position.PositionAmt*position.PositionAmt*0.5)
			size := entryValue / price
			size = math.Round(size/stepSize) * stepSize
			entryValue = size * price
			if position.PositionAmt*position.PositionAmt-entryValue < entryStep {
				size = position.PositionAmt
			}
			if size > 0 {
				order := bnswap.NewOrderParams{
					Symbol:           symbol,
					Side:             "SELL",
					Type:             "LIMIT",
					Price:            price,
					TimeInForce:      "GTC",
					Quantity:         size,
					ReduceOnly:       true,
					NewClientOrderId: fmt.Sprintf("%d", time.Now().Unix()*10000+int64(rand.Intn(10000))),
				}
				logger.Debugf("CLOSE ORDER %v", order.ToString())
				bnOrderSilentTimes[symbol] = time.Now().Add(*bnConfig.OrderSilent)
				bnOrderSilentTimes[symbol] = time.Now().Add(*bnConfig.OrderSilent)
				bnPositionsUpdateTimes[symbol] = time.Unix(0, 0)
				bnHttpPositionUpdateSilentTimes[symbol] = time.Now().Add(*bnConfig.HttpSilent)
				bnOpenOrders[symbol] = order
				bnOrderRequestChs[symbol] <- OrderRequest{
					New: &order,
				}
			}
		}
	}
}
