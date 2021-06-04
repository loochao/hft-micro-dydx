package main

import (
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"time"
)

func updateMakerNewOrders() {

	if mAccount == nil {
		if time.Now().Sub(time.Now().Truncate(mConfig.LogInterval)) < mConfig.LoopInterval {
			logger.Debugf("mACCOUNT not ready")
		}
		return
	}

	entryStep := mAccount.GetFree() * mConfig.EnterFreePct
	if entryStep < mConfig.EnterMinimalStep {
		entryStep = mConfig.EnterMinimalStep
	}
	entryTarget := entryStep * mConfig.EnterTargetFactor

	//得是两个市场的最小可用资金, 以防有一边用完了钱, 开不了仓
	makerUSDTAvailable := mAccount.GetFree() * mConfig.MakerExchange.Leverage

	for makerSymbol := range mConfig.MakerOrderOffsets {

		walkedDepth, okDepth := mWalkedDepths[makerSymbol]
		makerPosition, okMakerPosition := mPositions[makerSymbol]

		if !okDepth || !okMakerPosition {
			//if time.Now().Sub(time.Now().Truncate(mConfig.LogInterval)) < mConfig.LoopInterval {
			//	logger.Debugf("walkedDepth %v maker position %v %s", okDepth, okMakerPosition, makerSymbol)
			//}
			continue
		}

		offset := mOrderOffsets[makerSymbol]
		makerTickSize := mTickSizes[makerSymbol]
		makerStepSize := mStepSizes[makerSymbol]
		makerMinNotional := mMinNotional[makerSymbol]
		makerValue := makerPosition.GetSize() * makerPosition.GetPrice()

		//if time.Now().Sub(walkedDepth.Time) > mConfig.DepthTimeToLive {
		//	//if time.Now().Sub(time.Now().Truncate(mConfig.LogInterval)) < mConfig.LoopInterval {
		//	//	logger.Debugf("walkedDepth too old %s %v", makerSymbol, walkedDepth.Time)
		//	//}
		//	continue
		//}

		//需要保证两边都有仓位更新，才调整现货仓位
		if time.Now().Sub(mPositionsUpdateTimes[makerSymbol]) > mConfig.BalancePositionMaxAge {
			if time.Now().Sub(time.Now().Truncate(mConfig.LogInterval)) < mConfig.LoopInterval {
				logger.Debugf("maker position too old %s", makerSymbol)
			}
			continue
		}
		if time.Now().Sub(mOrderSilentTimes[makerSymbol]) < 0 {
			if time.Now().Sub(time.Now().Truncate(mConfig.LogInterval)) < mConfig.LoopInterval {
				logger.Debugf("taker order silent %s", makerSymbol)
			}
			continue
		}

		if (mConfig.TradeDir < 0 && makerPosition.GetSize() > 0) ||
			(makerPosition.GetSize() > 0 && walkedDepth.MakerBid > makerPosition.GetPrice()*(1.0+offset.Top)) {
			order := common.NewOrderParam{
				Symbol:     makerSymbol,
				Side:       common.OrderSideSell,
				Type:       common.OrderTypeMarket,
				Size:       makerPosition.GetSize(),
				ReduceOnly: true,
				ClientID:   mExchange.GenerateClientID(),
			}
			mOrderSilentTimes[makerSymbol] = time.Now().Add(mConfig.OrderSilent)
			mPositionsUpdateTimes[makerSymbol] = time.Unix(0, 0)
			if !mConfig.DryRun {
				mOrderRequestChMap[makerSymbol] <- common.OrderRequest{New: &order}
			}
			continue
		} else if (mConfig.TradeDir > 0 && makerPosition.GetSize() < 0) ||
			(makerPosition.GetSize() < 0 && walkedDepth.MakerAsk < makerPosition.GetPrice()*(1.0+offset.Bot)) {
			order := common.NewOrderParam{
				Symbol:     makerSymbol,
				Side:       common.OrderSideBuy,
				Type:       common.OrderTypeMarket,
				Size:       -makerPosition.GetSize(),
				ReduceOnly: true,
				ClientID:   mExchange.GenerateClientID(),
			}
			mOrderSilentTimes[makerSymbol] = time.Now().Add(mConfig.OrderSilent)
			mPositionsUpdateTimes[makerSymbol] = time.Unix(0, 0)
			if !mConfig.DryRun {
				mOrderRequestChMap[makerSymbol] <- common.OrderRequest{New: &order}
			}
			continue
		}

		if time.Now().Sub(mEnterSilentTimes[makerSymbol]) < 0 {
			//if time.Now().Sub(time.Now().Truncate(mConfig.LogInterval)) < mConfig.LoopInterval {
			//	logger.Debugf("maker enter silent %s", makerSymbol)
			//}
			continue
		}
		if _, ok := mOpenOrders[makerSymbol]; ok {
			//if time.Now().Sub(time.Now().Truncate(mConfig.LogInterval)) < mConfig.LoopInterval {
			//	logger.Debugf("has open order %s", makerSymbol)
			//}
			continue
		}

		//if time.Now().Sub(time.Now().Truncate(mConfig.LogInterval)) < mConfig.LoopInterval {
		//	logger.Debugf("loop %s", makerSymbol)
		//}

		buyPrice := math.Floor(walkedDepth.MakerBid*(1.0+offset.Bot)/makerTickSize) * makerTickSize
		sellPrice := math.Ceil(walkedDepth.MakerAsk*(1.0+offset.Top)/makerTickSize) * makerTickSize
		if mConfig.TradeDir > 0 &&
			(makerPosition.GetSize() == 0 || buyPrice < makerPosition.GetPrice()) {
			targetValue := makerValue + entryStep
			if targetValue > entryTarget {
				targetValue = entryTarget
			}
			entryValue := targetValue - makerValue
			if entryValue > makerUSDTAvailable*0.8 {
				entryValue = makerUSDTAvailable * 0.8
			}
			size := entryValue / buyPrice
			size = math.Round(size/makerStepSize) * makerStepSize

			entryValue = size * buyPrice

			if entryValue > makerUSDTAvailable {
				//if time.Now().Sub(mLogSilentTimes[makerSymbol]) > 0 {
				//	logger.Debugf(
				//		"FAILED OPEN LONG, ENTRY VALUE %f MORE THAN Available USDT %f, %s, SIZE %f PRICE %f",
				//		entryValue,
				//		makerUSDTAvailable,
				//		makerSymbol,
				//		size, buyPrice,
				//	)
				//	mLogSilentTimes[makerSymbol] = time.Now().Add(mConfig.LogInterval)
				//}
				continue
			}
			if entryValue < makerMinNotional {
				//if time.Now().Sub(mLogSilentTimes[makerSymbol]) > 0 {
				//	logger.Debugf(
				//		"FAILED OPEN LONG, ORDER VALUE %f LESS THAN MIN NOTIONAL %f, %s SIZE %f PRICE %f",
				//		entryValue,
				//		makerMinNotional,
				//		makerSymbol,
				//		size, buyPrice,
				//	)
				//	mLogSilentTimes[makerSymbol] = time.Now().Add(mConfig.LogInterval)
				//}
				continue
			}
			mLogSilentTimes[makerSymbol] = time.Now()
			makerUSDTAvailable -= entryValue
			order := common.NewOrderParam{
				Symbol:      makerSymbol,
				Side:        common.OrderSideBuy,
				Type:        common.OrderTypeLimit,
				Price:       buyPrice,
				TimeInForce: common.OrderTimeInForceGTC,
				Size:        size,
				PostOnly:    true,
				ReduceOnly:  false,
				ClientID:    mExchange.GenerateClientID(),
			}
			mOpenOrders[makerSymbol] = order
			mOrderSilentTimes[makerSymbol] = time.Now().Add(mConfig.OrderSilent)
			if !mConfig.DryRun {
				//logger.Debugf("ORDER %s BUY %s", order.Symbol, order.ClientID)
				mOrderRequestChMap[makerSymbol] <- common.OrderRequest{New: &order}
			}
		} else if mConfig.TradeDir < 0 &&
			(makerPosition.GetSize() == 0 || sellPrice > makerPosition.GetPrice()) {
			targetValue := makerValue + entryStep
			if targetValue > entryTarget {
				targetValue = entryTarget
			}
			entryValue := targetValue - makerValue
			if entryValue > makerUSDTAvailable*0.8 {
				entryValue = makerUSDTAvailable * 0.8
			}
			size := entryValue / sellPrice
			size = math.Round(size/makerStepSize) * makerStepSize

			entryValue = size * sellPrice

			if entryValue > makerUSDTAvailable {
				//if time.Now().Sub(mLogSilentTimes[makerSymbol]) > 0 {
				//	logger.Debugf(
				//		"FAILED OPEN SHORT, ENTRY VALUE %f MORE THAN Available USDT %f, %s SIZE %f PRICE %f",
				//		entryValue,
				//		makerUSDTAvailable,
				//		makerSymbol,
				//		size, sellPrice,
				//	)
				//	mLogSilentTimes[makerSymbol] = time.Now().Add(mConfig.LogInterval)
				//}
				continue
			}
			if entryValue < makerMinNotional {
				//if time.Now().Sub(mLogSilentTimes[makerSymbol]) > 0 {
				//	logger.Debugf(
				//		"FAILED OPEN SHORT, ORDER VALUE %f LESS THAN MIN NOTIONAL %f, %s SIZE %f PRICE %f",
				//		entryValue,
				//		makerMinNotional,
				//		makerSymbol,
				//		size, sellPrice,
				//	)
				//	mLogSilentTimes[makerSymbol] = time.Now().Add(mConfig.LogInterval)
				//}
				continue
			}
			mLogSilentTimes[makerSymbol] = time.Now()
			makerUSDTAvailable -= entryValue
			order := common.NewOrderParam{
				Symbol:      makerSymbol,
				Side:        common.OrderSideSell,
				Type:        common.OrderTypeLimit,
				Price:       sellPrice,
				TimeInForce: common.OrderTimeInForceGTC,
				Size:        size,
				PostOnly:    true,
				ReduceOnly:  false,
				ClientID:    mExchange.GenerateClientID(),
			}
			mOpenOrders[makerSymbol] = order
			mOrderSilentTimes[makerSymbol] = time.Now().Add(mConfig.OrderSilent)
			if !mConfig.DryRun {
				//logger.Debugf("ORDER %s SELL %s", order.Symbol, order.ClientID)
				mOrderRequestChMap[makerSymbol] <- common.OrderRequest{New: &order}
			}
		}
	}
}
