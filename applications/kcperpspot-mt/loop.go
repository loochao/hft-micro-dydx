package main

import (
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/kcperp"
	"github.com/geometrybase/hft-micro/kcspot"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"math/rand"
	"time"
)

func updatePerpPositions() {
	unHedgedValue := 0.0
	for _, spotSymbol := range kcspotSymbols {
		perpSymbol := kcspSymbolsMap[spotSymbol]
		if time.Now().Sub(kcspotBalancesUpdateTimes[spotSymbol]) > *kcConfig.BalancePositionMaxAge {
			continue
		}

		if time.Now().Sub(kcperpPositionsUpdateTimes[perpSymbol]) > *kcConfig.BalancePositionMaxAge {
			continue
		}

		if kcperpOrderSilentTimes[perpSymbol].Sub(time.Now()).Seconds() > 0 {
			continue
		}

		perpPosition, okPerpPosition := kcperpPositions[perpSymbol]
		spotBalance, okSpotBalance := kcspotBalances[spotSymbol]
		spread, okSpread := kcSpreads[spotSymbol]
		if !okPerpPosition || !okSpotBalance || !okSpread {
			continue
		}
		perpOrderBook := spread.PerpOrderBook

		multiplier := kcperpMultipliers[perpSymbol]
		perpTickSize := kcperpTickSizes[perpSymbol]
		perpLotSize := kcperpLotSizes[perpSymbol]
		lotSize := kcperpLotSizes[perpSymbol]

		perpSize := -(spotBalance.Holds + spotBalance.Available) - perpPosition.CurrentQty*multiplier
		unHedgedValue += math.Abs(perpSize * spread.PerpOrderBook.AskPrice)
		perpSize = math.Round(perpSize / multiplier)
		perpSize = math.Round(perpSize/lotSize) * lotSize

		//只做空PERP，所以开空是加仓，开多是减仓，减仓大小受当前空仓大小限制, 加仓受MinNotional限制
		if perpSize <= 0 && -perpSize < perpLotSize {
			continue
		}
		if perpSize > 0 && perpPosition.CurrentQty >= 0 {
			logger.Debugf("%s PERP POSITION ERROR, CAN'T ADD %f TO POS %f", perpSize, perpPosition.CurrentQty)
			continue
		}
		if perpSize > 0 && perpSize > -perpPosition.CurrentQty {
			perpSize = -perpPosition.CurrentQty
		}

		logger.Debugf("updatePerpPositions %s SIZE %f POS %f -> %f", perpSymbol, perpSize, perpPosition.CurrentQty, perpPosition.CurrentQty+perpSize)

		reduceOnly := false
		if perpSize*perpPosition.CurrentQty < 0 && math.Abs(perpSize) <= math.Abs(perpPosition.CurrentQty) {
			reduceOnly = true
		}
		price := math.Round(perpOrderBook.AskPrice*(1.0+*kcConfig.EnterSlippage)/perpTickSize) * perpTickSize
		side := kcperp.OrderSideBuy
		if perpSize < 0 {
			side = kcperp.OrderSideSell
			perpSize = -perpSize
			price = math.Round(perpOrderBook.BidPrice*(1.0-*kcConfig.EnterSlippage)/perpTickSize) * perpTickSize
		}
		order := kcperp.NewOrderParam{
			Symbol:      perpSymbol,
			Side:        side,
			Type:        kcperp.OrderTypeLimit,
			Price:       common.Float64(price),
			TimeInForce: kcperp.OrderTimeInForceIOC,
			Size:        int64(perpSize),
			ReduceOnly:  reduceOnly,
			ClientOid:   fmt.Sprintf("%d%04d", time.Now().Unix(), rand.Intn(10000)),
			Leverage:    *kcConfig.Leverage,
		}
		logger.Debugf("PERP ORDER %v", order)
		kcspotOrderSilentTimes[spotSymbol] = time.Now().Add(*kcConfig.OrderSilent)

		kcperpOrderSilentTimes[perpSymbol] = time.Now().Add(*kcConfig.OrderSilent)
		kcperpPositionsUpdateTimes[perpSymbol] = time.Unix(0, 0)
		kcperpHttpPositionUpdateSilentTimes[perpSymbol] = time.Now().Add(*kcConfig.HttpSilent)
		kcperpOrderRequestChs[perpSymbol] <- order
	}
	kcUnHedgeValue = unHedgedValue
}

func updateSpotNewOrders() {

	if kcspotUSDTBalance == nil {
		return
	}

	if kcperpUSDTAccount == nil {
		return
	}

	if len(kcRankSymbolMap) == 0 {
		return
	}

	if kcUnHedgeValue > *kcConfig.MaxUnHedgeValue {
		if time.Now().Sub(kcUnHedgeLogSilentTime) > 0 {
			kcUnHedgeLogSilentTime = time.Now().Add(*kcConfig.LogInterval)
			logger.Debugf("UN HEDGE VALUE %f > %f", kcUnHedgeValue, *kcConfig.MaxUnHedgeValue)
		}
		return
	}

	entryStep := (kcperpUSDTAccount.AvailableBalance + kcspotUSDTBalance.Available) * *kcConfig.EnterFreePct
	if entryStep < *kcConfig.EnterMinimalStep {
		entryStep = *kcConfig.EnterMinimalStep
	}
	entryTarget := entryStep * *kcConfig.EnterTargetFactor
	usdtAvailable := kcspotUSDTBalance.Available

	//遍历合约 从最大的rank 开始，能保证FR强的先下单
	for rank := len(kcperpSymbols) - 1; rank >= 0; rank-- {

		perpSymbol := kcRankSymbolMap[rank]
		spotSymbol := kcpsSymbolsMap[perpSymbol]
		//需要保证期货和现货都有仓位更新，才调整现货仓位
		if time.Now().Sub(kcspotBalancesUpdateTimes[spotSymbol]) > *kcConfig.BalancePositionMaxAge {
			continue
		}
		if time.Now().Sub(kcperpPositionsUpdateTimes[perpSymbol]) > *kcConfig.BalancePositionMaxAge {
			continue
		}
		if _, ok := kcspotOpenOrders[spotSymbol]; ok {
			//如果还有订单不操作
			continue
		}
		if time.Now().Sub(kcspotOrderSilentTimes[spotSymbol]) < 0 {
			continue
		}
		if time.Now().Sub(kcspotSilentTimes[spotSymbol]) < 0 {
			continue
		}
		quantile, okQuantile := kcQuantiles[spotSymbol]
		spread, okSpread := kcSpreads[spotSymbol]
		spotBalance, okSpotBalance := kcspotBalances[spotSymbol]
		fundingRate, okFundingRate := kcperpFundingRates[perpSymbol]
		if !okSpread || !okQuantile || !okSpotBalance || !okFundingRate {
			continue
		}
		if time.Now().Sub(spread.LastUpdateTime) > *kcConfig.SpreadTimeToLive {
			continue
		}
		perpStepSize := kcperpLotSizes[perpSymbol] * kcperpMultipliers[perpSymbol]
		spotStepSize := kcspotStepSizes[spotSymbol]
		spotTickSize := kcspotTickSizes[spotSymbol]
		spotMinNotional := kcspotMinNotional[spotSymbol]

		currentSpotSize := spotBalance.Available + spotBalance.Holds
		if spread.LastEnter > quantile.Top &&
			spread.MedianEnter > quantile.Top &&
			fundingRate.Value > *kcConfig.MinimalEnterFundingRate &&
			rank >= len(kcspotSymbols)-*kcConfig.TradeCount {
			price := spread.SpotOrderBook.MakerBidVWAP
			price = math.Floor(price/spotTickSize) * spotTickSize
			targetValue := currentSpotSize*price + entryStep
			if targetValue > entryTarget {
				targetValue = entryTarget
			}
			entryValue := targetValue - currentSpotSize*price

			if entryValue > usdtAvailable*0.8 {
				entryValue = usdtAvailable * 0.8
			}

			entryValue = math.Max(entryValue, spotMinNotional)

			quantity := entryValue / price
			quantity = math.Round(quantity/spotStepSize) * spotStepSize
			quantity = math.Round(quantity/perpStepSize) * perpStepSize

			entryValue = quantity * price

			//不及一个0.8*EntryStep, 不操作
			if entryValue < entryStep*0.8 {
				if time.Now().Sub(kcOpenLogSilentTimes[spotSymbol]) > 0 {
					logger.Debugf(
						"FAILED TOP OPEN, ENTRY VALUE %f LESS THAN 0.8*ENTRY_STEP %f, %s %f > %f, %f > %f, SIZE %f",
						entryValue,
						entryStep*0.8,
						spotSymbol,
						spread.LastEnter, quantile.Top,
						spread.MedianEnter, quantile.Top,
						quantity,
					)
					kcOpenLogSilentTimes[spotSymbol] = time.Now().Add(*kcConfig.LogInterval)
				}
				continue
			}
			if entryValue > usdtAvailable {
				if time.Now().Sub(kcOpenLogSilentTimes[spotSymbol]) > 0 {
					logger.Debugf(
						"FAILED TOP OPEN, ENTRY VALUE %f MORE THAN FREE USDT %f, %s %f > %f, %f > %f, SIZE %f",
						entryValue,
						usdtAvailable,
						spotSymbol,
						spread.LastEnter, quantile.Top,
						spread.MedianEnter, quantile.Top,
						quantity,
					)
					kcOpenLogSilentTimes[spotSymbol] = time.Now().Add(*kcConfig.LogInterval)
				}
				continue
			}
			if quantity*price < spotMinNotional ||
				quantity < perpStepSize {
				if time.Now().Sub(kcOpenLogSilentTimes[spotSymbol]) > 0 {
					logger.Debugf(
						"FAILED TOP OPEN, ORDER VALUE %f LESS THAN NOTIONAL %f, %s %f > %f, %f > %f, SIZE %f",
						quantity*price,
						spotMinNotional,
						spotSymbol,
						spread.LastEnter, quantile.Top,
						spread.MedianEnter, quantile.Top,
						quantity,
					)
					kcOpenLogSilentTimes[spotSymbol] = time.Now().Add(*kcConfig.LogInterval)
				}
				continue
			}
			kcOpenLogSilentTimes[spotSymbol] = time.Now()
			logger.Debugf(
				"TOP OPEN %s %f > %f, %f > %f, SIZE %f",
				spotSymbol,
				spread.LastEnter, quantile.Top,
				spread.MedianEnter, quantile.Top,
				quantity,
			)
			order := kcspot.NewOrderParam{
				Symbol:      spotSymbol,
				Price:       common.Float64(price),
				Size:        common.Float64(quantity),
				TimeInForce: kcspot.OrderTimeInForceGTC,
				Side:        kcspot.OrderSideBuy,
				Type:        kcspot.OrderTypeLimit,
				ClientOid:   fmt.Sprintf("%d%04d", time.Now().Unix(), rand.Intn(10000)),
			}
			kcspotOrderSilentTimes[spotSymbol] = time.Now().Add(*kcConfig.OrderSilent)
			kcspotOrderCancelCounts[spotSymbol] = 0
			kcspotOpenOrders[spotSymbol] = order
			kcspotOrderRequestChs[spotSymbol] <- SpotOrderRequest{New: &order}
			return
		} else if spread.LastExit < quantile.Bot &&
			spread.MedianExit < quantile.Bot &&
			fundingRate.Value < *kcConfig.MinimalKeepFundingRate {
			price := spread.SpotOrderBook.MakerAskVWAP
			price = math.Ceil(price/spotTickSize) * spotTickSize
			if spotBalance.Available*price > spotMinNotional {
				entryValue := math.Min(-4*entryStep, -spotBalance.Available*price*0.5)
				if fundingRate.Value > *kcConfig.MinimalKeepFundingRate/2 {
					entryValue = math.Min(-2*entryStep, -spotBalance.Available*price*0.5)
				}
				quantity := entryValue / price
				quantity = math.Round(quantity/spotStepSize) * spotStepSize
				quantity = math.Round(quantity/perpStepSize) * perpStepSize
				if spotBalance.Available*price+entryValue < entryStep {
					quantity = math.Ceil(-spotBalance.Available/spotStepSize) * spotStepSize
					quantity = math.Ceil(quantity/perpStepSize) * perpStepSize
				}
				if quantity < 0 {
					logger.Debugf(
						"BOT REDUCE %s %f < %f, %f < %f, SIZE %f",
						spotSymbol,
						spread.LastExit, quantile.Bot,
						spread.MedianExit, quantile.Bot,
						quantity,
					)
					order := kcspot.NewOrderParam{
						Symbol:      spotSymbol,
						Price:       common.Float64(price),
						Size:        common.Float64(-quantity),
						TimeInForce: kcspot.OrderTimeInForceGTC,
						Side:        kcspot.OrderSideSell,
						Type:        kcspot.OrderTypeLimit,
						ClientOid:   fmt.Sprintf("%d%04d", time.Now().Unix(), rand.Intn(10000)),
					}
					kcspotOrderSilentTimes[spotSymbol] = time.Now().Add(*kcConfig.OrderSilent)
					kcspotOrderCancelCounts[spotSymbol] = 0
					kcspotOpenOrders[spotSymbol] = order
					kcspotOrderRequestChs[spotSymbol] <- SpotOrderRequest{New: &order}
					return
				}
			}
		}
	}
}

func handleWebsocketRestart() {
	for _, spotSymbol := range kcspotSymbols {
		kcspotSilentTimes[spotSymbol] = time.Now().Add(*kcConfig.RestartSilent)
	}
}
