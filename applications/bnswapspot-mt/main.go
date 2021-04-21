package main

import (
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/bnspot"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"
	"time"
)

func main() {

	if *bnConfig.CpuProfile != "" {
		f, err := os.Create(*bnConfig.CpuProfile)
		if err != nil {
			logger.Debugf("os.Create error %v", err)
			return
		}
		err = pprof.StartCPUProfile(f)
		if err != nil {
			logger.Debugf("pprof.StartCPUProfile error %v", err)
			return
		}
		defer pprof.StopCPUProfile()
	}

	var err error
	bnswapAPI, err = bnswap.NewAPI(&common.Credentials{
		Key:    *bnConfig.ApiKey,
		Secret: *bnConfig.ApiSecret,
	}, *bnConfig.ProxyAddress)
	if err != nil {
		logger.Debugf("bnswap.NewAPI error %v", err)
		return
	}
	bnspotAPI, err = bnspot.NewAPI(&common.Credentials{
		Key:    *bnConfig.ApiKey,
		Secret: *bnConfig.ApiSecret,
	}, *bnConfig.ProxyAddress)
	if err != nil {
		logger.Debugf("bnspot.NewAPI error %v", err)
		return
	}
	bnGlobalCtx, bnGlobalCancel = context.WithCancel(context.Background())
	defer bnGlobalCancel()

	bnswapTickSizes, bnswapStepSizes, _, bnswapMinNotional, _, _, err = bnswap.GetOrderLimits(bnGlobalCtx, bnswapAPI, bnSymbols)
	if err != nil {
		logger.Debugf("bnswap.GetOrderLimits %v", err)
		return
	}
	bnspotTickSizes, bnspotStepSizes, _, bnspotMinNotional, err = bnspot.GetOrderLimits(bnGlobalCtx, bnspotAPI, bnSymbols)
	if err != nil {
		logger.Debugf("bnspot.GetOrderLimits %v", err)
		return
	}

	for symbol := range bnswapStepSizes {
		bnMergedStepSizes[symbol] = common.MergedStepSize(bnswapStepSizes[symbol], bnspotStepSizes[symbol])
	}

	logger.Debugf("MERGED STEP SIZES %v", bnMergedStepSizes)

	bnInfluxWriter, err = common.NewInfluxWriter(
		*bnConfig.InternalInflux.Address,
		*bnConfig.InternalInflux.Username,
		*bnConfig.InternalInflux.Password,
		*bnConfig.InternalInflux.Database,
		*bnConfig.InternalInflux.BatchSize,
	)
	if err != nil {
		logger.Debugf("common.NewInfluxWriter %v", err)
		return
	}

	bnExternalInfluxWriter, err = common.NewInfluxWriter(
		*bnConfig.ExternalInflux.Address,
		*bnConfig.ExternalInflux.Username,
		*bnConfig.ExternalInflux.Password,
		*bnConfig.ExternalInflux.Database,
		*bnConfig.ExternalInflux.BatchSize,
	)
	if err != nil {
		logger.Debugf("common.NewInfluxWriter %v", err)
		return
	}

	defer func() {
		err := bnInfluxWriter.Stop()
		if err != nil {
			logger.Warnf("stop influx writer error %v", err)
		}
	}()

	defer func() {
		err := bnInfluxWriter.Stop()
		if err != nil {
			logger.Warnf("stop influx writer error %v", err)
		}
	}()

	if *bnConfig.ChangeLeverage {
		for _, symbol := range bnSymbols {
			res, err := bnswapAPI.UpdateLeverage(bnGlobalCtx, bnswap.UpdateLeverageParams{
				Symbol:   symbol,
				Leverage: int64(*bnConfig.Leverage),
			})
			if err != nil {
				logger.Debugf("UPDATE LEVERAGE FOR %s ERROR %v", symbol, err)
			} else {
				logger.Debugf("UPDATE LEVERAGE FOR %s RESPONSE %v", symbol, res)
			}
			time.Sleep(time.Second)
			res, err = bnswapAPI.UpdateMarginType(bnGlobalCtx, bnswap.UpdateMarginTypeParams{
				Symbol:     symbol,
				MarginType: *bnConfig.MarginType,
			})
			if err != nil {
				logger.Debugf("UPDATE MARGIN TYPE FOR %s ERROR %v", symbol, err)
			} else {
				logger.Debugf("UPDATE MARGIN TYPE FOR %s RESPONSE %v", symbol, res)
			}
			time.Sleep(time.Second)
		}
	}

	bnspotUserWebsocket, err = bnspot.NewUserWebsocket(
		bnGlobalCtx,
		bnspotAPI,
		*bnConfig.ProxyAddress,
	)
	if err != nil {
		logger.Debugf("bnspot.NewUserWebsocket error %v", err)
		return
	}
	defer bnspotUserWebsocket.Stop()

	bnswapUserWebsocket, err = bnswap.NewUserWebsocket(
		bnGlobalCtx,
		bnswapAPI,
		*bnConfig.ProxyAddress,
	)
	if err != nil {
		logger.Debugf("bnswap.NewUserWebsocket error %v", err)
		return
	}
	defer bnswapUserWebsocket.Stop()

	influxSaveTimer := time.NewTimer(
		time.Now().Truncate(
			*bnConfig.InternalInflux.SaveInterval,
		).Add(
			*bnConfig.InternalInflux.SaveInterval * 3,
		).Sub(time.Now()),
	)
	externalInfluxSaveTimer := time.NewTimer(
		time.Now().Truncate(
			*bnConfig.ExternalInflux.SaveInterval,
		).Add(
			*bnConfig.ExternalInflux.SaveInterval * 3,
		).Sub(time.Now()),
	)
	bnLoopTimer = time.NewTimer(time.Hour * 24) //先等1分钟
	targetValueUpdateTimer := time.NewTimer(time.Hour * 24)
	resetUnrealisedPnlTimer := time.NewTimer(time.Minute)
	reBalanceTimer := time.NewTimer(time.Second)
	frRankUpdatedTimer := time.NewTimer(time.Second * 60)
	bnbReBalanceTimer := time.NewTimer(*bnConfig.BnbCheckInterval)

	defer bnbReBalanceTimer.Stop()
	defer influxSaveTimer.Stop()
	defer bnLoopTimer.Stop()
	defer targetValueUpdateTimer.Stop()
	defer resetUnrealisedPnlTimer.Stop()
	defer reBalanceTimer.Stop()
	defer frRankUpdatedTimer.Stop()

	go bnswap.WatchPositionsFromHttp(
		bnGlobalCtx, bnswapAPI,
		bnSymbols, *bnConfig.PullInterval, bnswapPositionCh,
	)
	go bnswap.WatchAccountFromHttp(
		bnGlobalCtx, bnswapAPI,
		*bnConfig.PullInterval, bnswapAccountCh,
	)
	go bnspot.WatchAccountFromHttp(
		bnGlobalCtx, bnspotAPI,
		*bnConfig.PullInterval, bnspotAccountCh,
	)

	go bnswap.WatchPremiumIndexesFromHttp(
		bnGlobalCtx, bnswapAPI,
		bnSymbols,
		*bnConfig.PullInterval*10,
		bnswapPremiumIndexesCh,
	)

	go watchSwapBars(
		bnGlobalCtx,
		bnswapAPI,
		bnSymbols,
		*bnConfig.BarsLookback,
		*bnConfig.PullBarsInterval,
		*bnConfig.PullBarsRetryInterval,
		time.Second,
		bnswapBarsMapCh,
	)

	go watchSpotBars(
		bnGlobalCtx,
		bnspotAPI,
		bnSymbols,
		*bnConfig.BarsLookback,
		*bnConfig.PullBarsInterval,
		*bnConfig.PullBarsRetryInterval,
		time.Second,
		bnspotBarsMapCh,
	)

	go watchDeltaQuantile(
		bnGlobalCtx,
		bnSymbols,
		*bnConfig.BotQuantile,
		*bnConfig.TopQuantile,
		*bnConfig.TopBandScale,
		*bnConfig.BotBandScale,
		*bnConfig.MinimalEnterDelta,
		*bnConfig.MaximalExitDelta,
		*bnConfig.MinimalBandOffset,
		bnBarsMapCh,
		bnQuantilesCh,
	)


	spreadCh := make(chan Spread, len(bnSymbols)*10)
	walkedOrderBookChMap := make(map[string]chan*WalkedOrderBook)
	for _, symbol := range bnSymbols {
		walkedOrderBookChMap[symbol] = make(chan *WalkedOrderBook, 100)
		go watchSingleSpread(
			bnGlobalCtx,
			symbol,
			*bnConfig.OrderBookMaxAgeDiff,
			*bnConfig.OrderBookMaxAge,
			*bnConfig.SpreadLookbackDuration,
			*bnConfig.SpreadLookbackMinimalWindow,
			walkedOrderBookChMap[symbol],
			spreadCh,
		)
	}

	for start := 0; start < len(bnSymbols); start += *bnConfig.OrderBookBatchSize {
		end := start + *bnConfig.OrderBookBatchSize
		if end > len(bnSymbols) {
			end = len(bnSymbols)
		}
		go watchSpotWalkedOrderBooks(
			bnGlobalCtx,
			bnGlobalCancel,
			*bnConfig.ProxyAddress,
			*bnConfig.OrderBookTakerImpact,
			*bnConfig.OrderBookMakerImpact,
			bnSymbols[start:end],
			walkedOrderBookChMap,
		)
	}

	for start := 0; start < len(bnSymbols); start += *bnConfig.OrderBookBatchSize {
		end := start + *bnConfig.OrderBookBatchSize
		if end > len(bnSymbols) {
			end = len(bnSymbols)
		}
		go watchSwapWalkedOrderBooks(
			bnGlobalCtx,
			bnGlobalCancel,
			*bnConfig.OrderBookTakerDecay,
			*bnConfig.OrderBookTakerBias,
			*bnConfig.ProxyAddress,
			*bnConfig.OrderBookTakerImpact,
			*bnConfig.OrderBookMakerImpact,
			bnSymbols[start:end],
			walkedOrderBookChMap,
		)
	}



	bnspotCancelOrderResponsesCh = make(chan []bnspot.CancelOrderResponse, len(bnSymbols)*100)
	bnspotNewOrderResponseCh = make(chan bnspot.NewOrderResponse, len(bnSymbols)*100)
	bnspotNewOrderErrorCh = make(chan MakerOrderNewError, len(bnSymbols)*100)
	for _, symbol := range bnSymbols {
		bnspotOrderRequestChs[symbol] = make(chan SpotOrderRequest, 2)
		go watchSpotOrderRequest(
			bnGlobalCtx,
			bnspotAPI,
			*bnConfig.OrderTimeout,
			*bnConfig.DryRun,
			bnspotOrderRequestChs[symbol],
			bnspotNewOrderErrorCh,
			bnspotNewOrderResponseCh,
			bnspotCancelOrderResponsesCh,
		)
		bnspotOrderRequestChs[symbol] <- SpotOrderRequest{
			Cancel: &bnspot.CancelAllOrderParams{Symbol: symbol},
		}
	}
	bnswapNewOrderErrorCh = make(chan TakerOrderNewError, len(bnSymbols)*100)
	for _, symbol := range bnSymbols {
		bnswapOrderRequestChs[symbol] = make(chan bnswap.NewOrderParams, 2)
		go watchTakerOrderRequest(
			bnGlobalCtx,
			bnswapAPI,
			*bnConfig.OrderTimeout,
			*bnConfig.DryRun,
			bnswapOrderRequestChs[symbol],
			bnswapOrderFinishCh,
			bnswapNewOrderErrorCh,
		)
	}

	if *bnConfig.CpuProfile != "" {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			sig := <-sigs
			logger.Debugf("Exit with sig %d,", sig)
			bnGlobalCancel()
		}()
	}

	fundingInterval := time.Hour * 8
	fundingSilent := time.Minute * 3

	for {
		select {
		case <-bnGlobalCtx.Done():
			logger.Debugf("EXIT MAIN LOOP")
			return
		case <-reBalanceTimer.C:
			if bnspotUSDTBalance != nil && bnswapUSDTAsset != nil && bnswapUSDTAsset.AvailableBalance != nil {
				//SWAP WS ACCOUNT 没有AvailableBalance, 为0 HTTP GET无数据
				//SWAP的MarginBalance AvailableBalance WS推送缺失，会造成错误判断
				if bnswapAssetUpdatedForReBalance && bnspotBalanceUpdatedForReBalance {
					bnswapAssetUpdatedForReBalance = false
					bnspotBalanceUpdatedForReBalance = false

					expectedInsuranceFund := *bnConfig.StartValue * (1 - *bnConfig.InsuranceFundingRatio) * *bnConfig.Leverage / (*bnConfig.Leverage + 1) * *bnConfig.InsuranceFundingRatio
					totalFree := (bnspotUSDTBalance.Free + *bnswapUSDTAsset.AvailableBalance) - expectedInsuranceFund
					targetSwap := totalFree/(*bnConfig.Leverage+1) + expectedInsuranceFund
					change := targetSwap - *bnswapUSDTAsset.AvailableBalance
					if change > 0 && change > bnspotUSDTBalance.Free {
						change = bnspotUSDTBalance.Free
					}
					if change < 0 && -change > *bnswapUSDTAsset.AvailableBalance-expectedInsuranceFund {
						change = 0
						if *bnswapUSDTAsset.AvailableBalance-expectedInsuranceFund > 0 {
							change = -(*bnswapUSDTAsset.AvailableBalance - expectedInsuranceFund)
						}
					}
					if math.Abs(change) > *bnConfig.ReBalanceMinimalNotional {
						// 如果有转帐发生最好不要让influx统计数据，转帐在中间过程中会有盈利计算误差
						bnspotBalanceUpdatedForExternalInflux = false
						bnswapAssetUpdatedForExternalInflux = false
						bnspotBalanceUpdatedForInflux = false
						bnswapAssetUpdatedForInflux = false
						bnSaveSilentTime = time.Now().Add(*bnConfig.PullInterval * 2)
						go reBalanceUSDT(
							bnGlobalCtx,
							bnspotAPI,
							*bnConfig.OrderTimeout,
							change,
						)
					}
				}
			}
			reBalanceTimer.Reset(*bnConfig.ReBalanceInterval)
			break
		case p := <-bnswapPositionCh:
			handleSwapHttpPositions(p)
			break
		case account := <-bnswapAccountCh:
			handleSwapHttpAccount(account)
			break
		case account := <-bnspotAccountCh:
			handleSpotHttpAccount(account)
			break
		case msg := <-bnspotUserWebsocket.AccountUpdateEventCh:
			handleSpotWSOutboundAccountPosition(msg)
			break
		case data := <-bnspotUserWebsocket.OrderUpdateEventCh:
			if data.CurrentOrderStatus == bnspot.OrderStatusFilled {
				if data.CumulativeFilledQuantity > 0 && data.CumulativeQuoteAssetTransactedQuantity > 0 {
					filledPrice := data.CumulativeQuoteAssetTransactedQuantity / data.CumulativeFilledQuantity
					if data.Side == bnspot.OrderSideBuy {
						bnspotLastFilledBuyPrices[data.Symbol] = filledPrice
					} else {
						bnspotLastFilledSellPrices[data.Symbol] = filledPrice
					}
					logger.Debugf("SPOT WS ORDER FILLED %s %s SIZE %f PRICE %f", data.Symbol, data.Side, data.CumulativeFilledQuantity, filledPrice)
				}
				if openOrder, ok := bnspotOpenOrders[data.Symbol]; ok && openOrder.NewClientOrderID == data.ClientOrderID {
					delete(bnspotOpenOrders, data.Symbol)
				}
				bnspotHttpBalanceUpdateSilentTimes[data.Symbol] = time.Now().Add(*bnConfig.HttpSilent)
			} else if data.CurrentOrderStatus == bnspot.OrderStatusCancelled {
				if openOrder, ok := bnspotOpenOrders[data.Symbol]; ok && openOrder.NewClientOrderID == data.OriginalClientOrderID {
					delete(bnspotOpenOrders, data.Symbol)
				}
			} else if data.CurrentOrderStatus == bnspot.OrderStatusExpired ||
				data.CurrentOrderStatus == bnspot.OrderStatusReject {
				if openOrder, ok := bnspotOpenOrders[data.Symbol]; ok && openOrder.NewClientOrderID == data.ClientOrderID {
					delete(bnspotOpenOrders, data.Symbol)
				}
			}
			break
		case msg := <-bnswapUserWebsocket.BalanceAndPositionUpdateEventCh:
			handleWSAccountEvent(msg)
			break
		case msg := <-bnswapUserWebsocket.OrderUpdateEventCh:
			handleWSOrder(&msg.Order)
			break
		case spread := <-spreadCh:
			bnSpreads[spread.Symbol] = spread
			break
		case bnswapPremiumIndexes = <-bnswapPremiumIndexesCh:
			break
		case <-resetUnrealisedPnlTimer.C:
			//handleResetPnl()
			resetUnrealisedPnlTimer.Reset(
				time.Now().Truncate(
					*bnConfig.ResetUnrealisedPnlInterval,
				).Add(
					*bnConfig.ResetUnrealisedPnlInterval,
				).Sub(time.Now()),
			)
			break
		case bnswapBarsMap = <-bnswapBarsMapCh:
			if bnBarsMapUpdated["spot"] {
				bnBarsMapCh <- [2]common.KLinesMap{bnspotBarsMap, bnswapBarsMap}
				bnBarsMapUpdated["spot"] = false
				bnBarsMapUpdated["swap"] = false
			} else {
				bnBarsMapUpdated["swap"] = true
			}
			break
		case bnspotBarsMap = <-bnspotBarsMapCh:
			if bnBarsMapUpdated["swap"] {
				bnBarsMapCh <- [2]common.KLinesMap{bnspotBarsMap, bnswapBarsMap}
				bnBarsMapUpdated["spot"] = false
				bnBarsMapUpdated["swap"] = false
			} else {
				bnBarsMapUpdated["spot"] = true
			}
			break
		case bnQuantiles = <-bnQuantilesCh:
			//logger.Debugf("QUANTILE %v", bnQuantiles)
			bnLoopTimer.Reset(time.Second)
			break
		case <-influxSaveTimer.C:
			handleSave()
			influxSaveTimer.Reset(
				time.Now().Truncate(
					*bnConfig.InternalInflux.SaveInterval,
				).Add(
					*bnConfig.InternalInflux.SaveInterval,
				).Sub(time.Now()),
			)
			break
		case <-externalInfluxSaveTimer.C:
			handleExternalInfluxSave()
			externalInfluxSaveTimer.Reset(
				time.Now().Truncate(
					*bnConfig.ExternalInflux.SaveInterval,
				).Add(
					*bnConfig.ExternalInflux.SaveInterval,
				).Sub(time.Now()),
			)
			break

		case newError := <-bnswapOrderNewErrorCh:
			bnswapOrderSilentTimes[newError.Params.Symbol] = time.Now().Add(time.Second * 15)
			break

		case order := <-bnswapOrderFinishCh:
			logStr := fmt.Sprintf("SWAP ORDER %s", order.ToString())
			if order.Status == "REJECTED" || order.Status == "EXPIRED" {
				logStr = fmt.Sprintf("%s RESET TIMEOUT", logStr)
				bnswapOrderSilentTimes[order.Symbol] = time.Now().Add(time.Second)
				bnswapPositionsUpdateTimes[order.Symbol] = time.Unix(0, 0)
			}
			if order.Side == common.OrderSideSell && order.Status == "FILLED" {
				if spotPrice, ok := bnspotLastFilledBuyPrices[order.Symbol]; ok {
					bnRealisedSpread[order.Symbol] = (order.CumQuote - spotPrice) / spotPrice
					logger.Debugf("%s REALISED OPEN SPREAD %f", order.Symbol, bnRealisedSpread[order.Symbol])
				}
				bnswapHttpPositionUpdateSilentTimes[order.Symbol] = time.Now().Add(*bnConfig.HttpSilent)
			} else if order.Side == common.OrderSideBuy && order.Status == "FILLED" {
				if spotPrice, ok := bnspotLastFilledSellPrices[order.Symbol]; ok {
					bnRealisedSpread[order.Symbol] = (order.CumQuote - spotPrice) / spotPrice
					logger.Debugf("%s REALISED CLOSE SPREAD %f", order.Symbol, bnRealisedSpread[order.Symbol])
				}
				bnswapHttpPositionUpdateSilentTimes[order.Symbol] = time.Now().Add(*bnConfig.HttpSilent)
			}
			logger.Debug(logStr)
			break

		case orders := <-bnspotCancelOrderResponsesCh:
			logger.Debugf("CANCEL ALL %v", orders)
			for _, o := range orders {
				if openOrder, ok := bnspotOpenOrders[o.Symbol]; ok && openOrder.NewClientOrderID == o.OrigClientOrderID {
					delete(bnspotOpenOrders, o.Symbol)
				}
			}
		case order := <-bnspotNewOrderResponseCh:
			logStr := fmt.Sprintf("SPOT ORDER %v", order)
			if order.Status == bnspot.OrderStatusReject ||
				order.Status == bnspot.OrderStatusExpired ||
				order.Status == bnspot.OrderStatusCancelled {
				logStr = fmt.Sprintf("%s RESET TIMEOUT", logStr)
				bnspotOrderSilentTimes[order.Symbol] = time.Now()
				bnspotBalancesUpdateTimes[order.Symbol] = time.Unix(0, 0)
				if openOrder, ok := bnspotOpenOrders[order.Symbol]; ok && openOrder.NewClientOrderID == order.ClientOrderID {
					delete(bnspotOpenOrders, order.Symbol)
				}
			} else if order.Status == bnspot.OrderStatusFilled {
				sumQty := 0.0
				sumVal := 0.0
				for _, f := range order.Fills {
					sumQty += f.Qty
					sumVal += f.Price * f.Qty
				}
				if sumQty != 0 && sumVal != 0 {
					filledPrice := sumVal / sumQty
					bnspotLastFilledBuyPrices[order.Symbol] = filledPrice
					logStr = fmt.Sprintf("%s FILLED PRICE %f", logStr, filledPrice)
				}
				if openOrder, ok := bnspotOpenOrders[order.Symbol]; ok && openOrder.NewClientOrderID == order.ClientOrderID {
					delete(bnspotOpenOrders, order.Symbol)
				}
			}
			logger.Debug(logStr)
		case order := <-bnspotNewOrderErrorCh:
			if openOrder, ok := bnspotOpenOrders[order.Params.Symbol]; ok && openOrder.NewClientOrderID == order.Params.NewClientOrderID {
				delete(bnspotOpenOrders, order.Params.Symbol)
			}
			bnspotOrderSilentTimes[order.Params.Symbol] = time.Now().Add(*bnConfig.OrderSilent * 3)
		case <-frRankUpdatedTimer.C:
			frs := make([]float64, len(bnSymbols))
			for i, symbol := range bnSymbols {
				if premiumIndex, ok := bnswapPremiumIndexes[symbol]; ok {
					frs[i] = premiumIndex.FundingRate
				} else {
					logger.Debugf("MISS MARK PRICE %s", symbol)
					return
				}
			}
			bnRankSymbolMap, err = common.RankSymbols(bnSymbols, frs)
			if err != nil {
				logger.Debugf("RankSymbols error %v", err)
			}
			//logger.Debugf("SYMBOLS FR RANK %v", bnRankSymbolMap)
			frRankUpdatedTimer.Reset(time.Minute)
		case <-bnbReBalanceTimer.C:
			handleReBalanceBnb()
			bnbReBalanceTimer.Reset(*bnConfig.BnbCheckInterval)
			break
		case <-bnLoopTimer.C:
			updateSwapPositions()
			updateMakerOldOrders()
			if time.Now().Sub(time.Now().Truncate(fundingInterval)) > fundingSilent &&
				time.Now().Truncate(fundingInterval).Add(fundingInterval).Sub(time.Now()) > fundingSilent {
				updateMakerNewOrders()
			}
			bnLoopTimer.Reset(
				time.Now().Truncate(
					*bnConfig.LoopInterval,
				).Add(
					*bnConfig.LoopInterval,
				).Sub(time.Now()),
			)
			break
		}
	}
}
