package main

import (
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/hbcrossswap"
	"github.com/geometrybase/hft-micro/hbspot"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"math/rand"
	"strings"
	"time"
)

func updatePerpPositions() {
	unHedgedValue := 0.0
	for _, spotSymbol := range hSymbols {
		swapSymbol := bhSymbolsMap[spotSymbol]
		if time.Now().Sub(bPositionsUpdateTimes[spotSymbol]) > *hbConfig.BalancePositionMaxAge {
			continue
		}

		if time.Now().Sub(hPositionsUpdateTimes[swapSymbol]) > *hbConfig.BalancePositionMaxAge {
			continue
		}

		if hOrderSilentTimes[swapSymbol].Sub(time.Now()).Seconds() > 0 {
			continue
		}

		swapPosition, okPerpPosition := hPositions[swapSymbol]
		spotBalance, okSpotBalance := hbspotBalances[spotSymbol]
		spread, okSpread := hbSpreads[spotSymbol]
		if !okPerpPosition || !okSpotBalance || !okSpread {
			continue
		}
		swapOrderBook := spread.PerpOrderBook

		contractSize := hbcrossswapContractSizes[swapSymbol]
		swapTickSize := hbcrossswapTickSizes[swapSymbol]

		positionVolume := swapPosition.Volume
		if swapPosition.Direction == hbcrossswap.OrderDirectionSell {
			positionVolume = -positionVolume
		}
		positionSize := positionVolume * contractSize

		swapSize := -(spotBalance.Balance) - positionSize
		unHedgedValue += math.Abs(swapSize * spread.PerpOrderBook.AskPrice)
		swapSize = math.Round(swapSize / contractSize)

		if math.Abs(swapSize) < 1 {
			continue
		}
		if swapSize > 0 && swapPosition.Direction == hbcrossswap.OrderDirectionBuy {
			logger.Debugf("%s SWAP POSITION ERROR, CAN'T ADD %f TO POS %f", swapSize, positionVolume)
			continue
		}
		if swapSize > 0 && swapSize > -positionVolume {
			swapSize = -positionVolume
		}
		logger.Debugf("updatePerpPositions %s SIZE %f POS %f -> %f", swapSymbol, swapSize, positionVolume, positionVolume+swapSize)
		offset := hbcrossswap.OrderOffsetOpen
		if swapSize*positionVolume < 0 && math.Abs(swapSize) <= math.Abs(positionVolume) {
			offset = hbcrossswap.OrderOffsetClose
		}
		price := math.Round(swapOrderBook.AskPrice*(1.0+*hbConfig.EnterSlippage)/swapTickSize) * swapTickSize
		direction := hbcrossswap.OrderDirectionBuy
		id, _ := common.GenerateShortId()
		clOrdID := fmt.Sprintf(
			"%s%d",
			id,
			time.Now().Unix(),
		)
		clOrdID = strings.ReplaceAll(clOrdID, ".", "_")
		if swapSize < 0 {
			direction = hbcrossswap.OrderDirectionSell
			swapSize = -swapSize
			price = math.Round(swapOrderBook.BidPrice*(1.0-*hbConfig.EnterSlippage)/swapTickSize) * swapTickSize
		}
		order := hbcrossswap.NewOrderParam{
			Symbol:         swapSymbol,
			ClientOrderID:  time.Now().Unix()*10000 + int64(rand.Intn(10000)),
			Price:          common.Float64(price),
			Volume:         int64(swapSize),
			Direction:      direction,
			Offset:         offset,
			LeverRate:      *hbConfig.Leverage,
			OrderPriceType: hbcrossswap.OrderPriceTypeLimit,
		}
		logger.Debugf("SWAP ORDER %v", order)
		bOrderSilentTimes[spotSymbol] = time.Now().Add(*hbConfig.OrderSilent)

		hOrderSilentTimes[swapSymbol] = time.Now().Add(*hbConfig.OrderSilent)
		hPositionsUpdateTimes[swapSymbol] = time.Unix(0, 0)
		bLastOrderTimes[swapSymbol] = time.Now()
		hOrderRequestChs[swapSymbol] <- order
	}
	hbUnHedgeValue = unHedgedValue
}

func updateSpotNewOrders() {

	if hbspotUSDTBalance == nil {
		return
	}

	if hbcrossswapAccount == nil {
		return
	}

	if len(hbRankSymbolMap) == 0 {
		return
	}

	if hbUnHedgeValue > *hbConfig.MaxUnHedgeValue {
		logger.Debugf("UN HEDGE VALUE %f > %f", hbUnHedgeValue, *hbConfig.MaxUnHedgeValue)
		return
	}

	entryStep := (hbcrossswapAccount.WithdrawAvailable + hbspotUSDTBalance.Available) * *hbConfig.EnterFreePct
	if entryStep < *hbConfig.EnterMinimalStep {
		entryStep = *hbConfig.EnterMinimalStep
	}
	entryTarget := entryStep * *hbConfig.EnterTargetFactor

	//遍历合约 从最大的rank 开始，能保证FR强的先下单
	for rank := len(bSymbols) - 1; rank >= 0; rank-- {

		swapSymbol := hbRankSymbolMap[rank]
		spotSymbol := hbSymbolsMap[swapSymbol]
		//需要保证期货和现货都有仓位更新，才调整现货仓位
		if time.Now().Sub(bPositionsUpdateTimes[spotSymbol]) > *hbConfig.BalancePositionMaxAge {
			continue
		}
		if time.Now().Sub(hPositionsUpdateTimes[swapSymbol]) > *hbConfig.BalancePositionMaxAge {
			continue
		}
		if _, ok := hOpenOrders[spotSymbol]; ok {
			//如果还有订单不操作
			continue
		}
		if time.Now().Sub(bOrderSilentTimes[spotSymbol]) < 0 {
			continue
		}
		if time.Now().Sub(hSilentTimes[spotSymbol]) < 0 {
			continue
		}
		quantile, okQuantile := hbQuantiles[spotSymbol]
		spread, okSpread := hbSpreads[spotSymbol]
		spotBalance, okSpotBalance := hbspotBalances[spotSymbol]
		fundingRate, okFundingRate := hFundingRates[swapSymbol]
		//logger.Debugf("%v %v %v %v %v", okSpread, okQuantile, okSpotBalance, okFundingRate, time.Now().Sub(spread.LastUpdateTime))
		if !okSpread || !okQuantile || !okSpotBalance || !okFundingRate {
			continue
		}
		if time.Now().Sub(spread.LastUpdateTime) > *hbConfig.SpreadTimeToLive {
			continue
		}
		swapContractSize := hbcrossswapContractSizes[swapSymbol]
		spotStepSize := hbspotStepSizes[spotSymbol]
		spotTickSize := hbspotTickSizes[spotSymbol]
		spotMinNotional := hbspotMinNotional[spotSymbol]
		amountPrecision := hbspotAmountPrecisions[spotSymbol]
		pricePrecision := hbspotPricePrecisions[spotSymbol]

		currentSpotSize := spotBalance.Balance
		if spread.LastEnter > quantile.Top &&
			spread.MedianEnter > quantile.Top &&
			fundingRate.FundingRate > *hbConfig.MinimalEnterFundingRate &&
			rank >= len(hSymbols)-*hbConfig.TradeCount {
			price := spread.SpotOrderBook.MakerBidVWAP
			price = math.Floor(price/spotTickSize) * spotTickSize
			targetValue := currentSpotSize*price + entryStep
			if targetValue > entryTarget {
				targetValue = entryTarget
			}
			entryValue := targetValue - currentSpotSize*price

			if entryValue > hbspotUSDTBalance.Available*0.8 {
				entryValue = hbspotUSDTBalance.Available * 0.8
			}

			entryValue = math.Max(entryValue, spotMinNotional)

			amount := entryValue / price
			amount = math.Round(amount/spotStepSize) * spotStepSize
			amount = math.Round(amount/swapContractSize) * swapContractSize

			//不及一个0.8*EntryStep, 不操作
			if entryValue < entryStep*0.8 {
				if time.Now().Sub(hOpenLogSilentTimes[spotSymbol]) > 0 {
					logger.Debugf(
						"FAILED TOP OPEN, ENTRY VALUE %f LESS THAN 0.8*ENTRY_STEP %f, %s %f > %f, %f > %f, SIZE %f",
						entryValue,
						entryStep*0.8,
						spotSymbol,
						spread.LastEnter, quantile.Top,
						spread.MedianEnter, quantile.Top,
						amount,
					)
					hOpenLogSilentTimes[spotSymbol] = time.Now().Add(time.Minute * 5)
				}
				continue
			}
			if entryValue > hbspotUSDTBalance.Available {
				if time.Now().Sub(hOpenLogSilentTimes[spotSymbol]) > 0 {
					logger.Debugf(
						"FAILED TOP OPEN, ENTRY VALUE %f MORE THAN FREE USDT %f, %s %f > %f, %f > %f, SIZE %f",
						entryValue,
						hbspotUSDTBalance.Available,
						spotSymbol,
						spread.LastEnter, quantile.Top,
						spread.MedianEnter, quantile.Top,
						amount,
					)
					hOpenLogSilentTimes[spotSymbol] = time.Now().Add(time.Minute * 5)
				}
				continue
			}
			if amount*price > hbspotUSDTBalance.Available {
				if time.Now().Sub(hOpenLogSilentTimes[spotSymbol]) > 0 {
					logger.Debugf(
						"FAILED TOP OPEN, ORDER VALUE %f MORE THAN FREE USDT %f, %s %f > %f, %f > %f, SIZE %f",
						amount*price,
						hbspotUSDTBalance.Available,
						spotSymbol,
						spread.LastEnter, quantile.Top,
						spread.MedianEnter, quantile.Top,
						amount,
					)
					hOpenLogSilentTimes[spotSymbol] = time.Now().Add(time.Minute * 5)
				}
				continue
			}
			if amount*price < spotMinNotional ||
				amount < swapContractSize {
				if time.Now().Sub(hOpenLogSilentTimes[spotSymbol]) > 0 {
					logger.Debugf(
						"FAILED TOP OPEN, ORDER VALUE %f LESS THAN NOTIONAL %f, %s %f > %f, %f > %f, SIZE %f",
						amount*price,
						spotMinNotional,
						spotSymbol,
						spread.LastEnter, quantile.Top,
						spread.MedianEnter, quantile.Top,
						amount,
					)
					hOpenLogSilentTimes[spotSymbol] = time.Now().Add(time.Minute * 5)
				}
				continue
			}
			hOpenLogSilentTimes[spotSymbol] = time.Now()
			logger.Debugf(
				"TOP OPEN %s %f > %f, %f > %f, SIZE %f",
				spotSymbol,
				spread.LastEnter, quantile.Top,
				spread.MedianEnter, quantile.Top,
				amount,
			)
			order := hbspot.NewOrderParam{
				Symbol:        spotSymbol,
				AccountId:     hbspotAccountID,
				ClientOrderID: fmt.Sprintf("%d%04d", time.Now().Unix(), rand.Intn(10000)),
				Price:         common.FormatByPrecision(price, pricePrecision),
				Amount:        common.FormatByPrecision(amount, amountPrecision),
				OriginPrice:   price,
				OriginAmount:  amount,
				Type:          hbspot.OrderTypeBuyLimit,
			}

			bOrderSilentTimes[spotSymbol] = time.Now().Add(*hbConfig.OrderSilent)
			hOrderCancelCounts[spotSymbol] = 0
			hOpenOrders[spotSymbol] = order
			hbspotOrderRequestChs[spotSymbol] <- SpotOrderRequest{New: &order}
			return
		} else if spread.LastExit < quantile.Bot &&
			spread.MedianExit < quantile.Bot &&
			fundingRate.FundingRate < *hbConfig.MinimalKeepFundingRate {
			price := spread.SpotOrderBook.MakerAskVWAP
			price = math.Ceil(price/spotTickSize) * spotTickSize
			if spotBalance.Available*price > spotMinNotional {
				entryValue := math.Min(-4*entryStep, -spotBalance.Available*price*0.5)
				if fundingRate.FundingRate > *hbConfig.MinimalKeepFundingRate/2 {
					entryValue = math.Min(-2*entryStep, -spotBalance.Available*price*0.5)
				}
				amount := entryValue / price
				amount = math.Round(amount/spotStepSize) * spotStepSize
				amount = math.Round(amount/swapContractSize) * swapContractSize
				if spotBalance.Available*price+entryValue < entryStep {
					amount = math.Ceil(-spotBalance.Available/spotStepSize) * spotStepSize
					amount = math.Ceil(amount/swapContractSize) * swapContractSize
				}
				logger.Debugf(
					"BOT REDUCE %s %f < %f, %f < %f, SIZE %f",
					spotSymbol,
					spread.LastExit, quantile.Bot,
					spread.MedianExit, quantile.Bot,
					amount,
				)
				if amount < 0 {
					order := hbspot.NewOrderParam{
						Symbol:        spotSymbol,
						AccountId:     hbspotAccountID,
						ClientOrderID: fmt.Sprintf("%d%04d", time.Now().Unix(), rand.Intn(10000)),
						Price:         common.FormatByPrecision(price, pricePrecision),
						Amount:        common.FormatByPrecision(-amount, amountPrecision),
						OriginPrice:   price,
						OriginAmount:  amount,
						Type:          hbspot.OrderTypeSellLimit,
					}
					bOrderSilentTimes[spotSymbol] = time.Now().Add(*hbConfig.OrderSilent)
					hOrderCancelCounts[spotSymbol] = 0
					hOpenOrders[spotSymbol] = order
					hbspotOrderRequestChs[spotSymbol] <- SpotOrderRequest{New: &order}
					return
				}
			}
		}
	}
}
