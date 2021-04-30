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

	bnInternalInfluxWriter, err = common.NewInfluxWriter(
		bnGlobalCtx,
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
		bnGlobalCtx,
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
	defer bnInternalInfluxWriter.Stop()
	defer bnExternalInfluxWriter.Stop()

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
		*bnConfig.MinimalEnterDelta,
		*bnConfig.MaximalExitDelta,
		*bnConfig.MinimalBandOffset,
		bnswapAvgFundingRateCh,
		bnBarsMapCh,
		bnQuantilesCh,
	)

	makerRowDepthChs := make(map[string]chan *common.DepthRawMessage)
	for start := 0; start < len(bnSymbols); start += *bnConfig.OrderBookBatchSize {
		end := start + *bnConfig.OrderBookBatchSize
		if end > len(bnSymbols) {
			end = len(bnSymbols)
		}
		subMakerRowDepthChs := make(map[string]chan *common.DepthRawMessage)
		for _, symbol := range bnSymbols[start:end] {
			makerRowDepthChs[symbol] = make(chan *common.DepthRawMessage, 100)
			subMakerRowDepthChs[symbol] = makerRowDepthChs[symbol]
		}
		go makerDepthWebsocketLoop(
			bnGlobalCtx,
			bnGlobalCancel,
			*bnConfig.ProxyAddress,
			subMakerRowDepthChs,
		)
	}

	takerRowDepthChs := make(map[string]chan *common.DepthRawMessage)
	for start := 0; start < len(bnSymbols); start += *bnConfig.OrderBookBatchSize {
		end := start + *bnConfig.OrderBookBatchSize
		if end > len(bnSymbols) {
			end = len(bnSymbols)
		}
		subTakerRowDepthChs := make(map[string]chan *common.DepthRawMessage)
		for _, symbol := range bnSymbols[start:end] {
			takerRowDepthChs[symbol] = make(chan *common.DepthRawMessage, 100)
			subTakerRowDepthChs[symbol] = takerRowDepthChs[symbol]
		}
		go takerDepthWebsocketLoop(
			bnGlobalCtx,
			bnGlobalCancel,
			*bnConfig.ProxyAddress,
			subTakerRowDepthChs,
		)
	}

	spreadReportCh := make(chan common.SpreadReport, 10000)
	go reportsSaveLoop(
		bnGlobalCtx,
		bnInternalInfluxWriter,
		*bnConfig.InternalInflux,
		spreadReportCh,
	)

	spreadCh := make(chan *common.MakerTakerSpread, len(bnSymbols)*100)
	for _, symbol := range bnSymbols {
		go watchMakerTakerSpread(
			bnGlobalCtx,
			symbol,
			*bnConfig.OrderBookMakerImpact,
			*bnConfig.OrderBookTakerImpact,
			*bnConfig.OrderBookMakerDecay,
			*bnConfig.OrderBookMakerBias,
			*bnConfig.OrderBookTakerDecay,
			*bnConfig.OrderBookTakerBias,
			*bnConfig.OrderBookMaxAgeDiffBias,
			*bnConfig.ReportCount,
			*bnConfig.SpreadLookbackDuration,
			*bnConfig.SpreadLookbackMinimalWindow,
			makerRowDepthChs[symbol],
			takerRowDepthChs[symbol],
			spreadReportCh,
			spreadCh,
		)
	}

	bnspotCancelOrderResponsesCh = make(chan []bnspot.CancelOrderResponse, len(bnSymbols))
	bnspotNewOrderResponseCh = make(chan bnspot.NewOrderResponse, len(bnSymbols))
	bnspotNewOrderErrorCh = make(chan MakerOrderNewError, len(bnSymbols))
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
	bnswapNewOrderErrorCh = make(chan TakerOrderNewError, len(bnSymbols))
	for _, symbol := range bnSymbols {
		bnswapOrderRequestChs[symbol] = make(chan bnswap.NewOrderParams, 2)
		go watchTakerOrderRequest(
			bnGlobalCtx,
			bnswapAPI,
			*bnConfig.OrderTimeout,
			*bnConfig.DryRun,
			bnswapOrderRequestChs[symbol],
			bnswapOrderResponseCh,
			bnswapNewOrderErrorCh,
		)
	}

	go bnspot.HttpPingLoop(
		bnGlobalCtx,
		bnspotAPI,
		*bnConfig.PullInterval/2,
		bnspotSystemStatusCh,
	)

	go bnswap.SystemStatusLoop(
		bnGlobalCtx,
		bnswapAPI,
		*bnConfig.PullInterval/2,
		bnswapSystemStatusCh,
	)

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
		case bnspotSystemReady = <-bnspotSystemStatusCh:
			if !bnspotSystemReady {
				logger.Debugf("bnspotSystemReady %v", bnspotSystemReady)
				bnGlobalSilent = time.Now().Add(*bnConfig.RestartSilent)
			}
			break
		case bnswapSystemReady = <-bnswapSystemStatusCh:
			if !bnswapSystemReady {
				logger.Debugf("bnswapSystemReady %v", bnswapSystemReady)
				bnGlobalSilent = time.Now().Add(*bnConfig.RestartSilent)
			}
			break
		case <-bnspotUserWebsocket.RestartCh:
			bnGlobalSilent = time.Now().Add(*bnConfig.RestartSilent)
			logger.Debugf("<-bnspotUserWebsocket.RestartCh silent in %v", *bnConfig.RestartSilent)
			break
		case <-bnswapUserWebsocket.RestartCh:
			bnGlobalSilent = time.Now().Add(*bnConfig.RestartSilent)
			logger.Debugf("<-bnswapUserWebsocket.RestartCh silent in %v", *bnConfig.RestartSilent)
			break
		case <-reBalanceTimer.C:
			if time.Now().Sub(bnGlobalSilent) < 0 {
				break
			}
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
		case spotWSOrder := <-bnspotUserWebsocket.OrderUpdateEventCh:
			logger.Debugf("SPOT WS ORDER %v", spotWSOrder)
			if spotWSOrder.CurrentOrderStatus == bnspot.OrderStatusFilled {
				logger.Debugf("SPOT WS ORDER %s %s %s", spotWSOrder.Symbol, spotWSOrder.CurrentOrderStatus, spotWSOrder.ClientOrderID)
				if spotWSOrder.CumulativeFilledQuantity > 0 && spotWSOrder.CumulativeQuoteAssetTransactedQuantity > 0 {
					logger.Debugf("SPOT WS ORDER FILLED %s %s CumulativeFilledQuantity %f CumulativeQuoteAssetTransactedQuantity %f", spotWSOrder.Symbol, spotWSOrder.Side, spotWSOrder.CumulativeFilledQuantity, spotWSOrder.CumulativeQuoteAssetTransactedQuantity)
				}
				if openOrder, ok := bnspotOpenOrders[spotWSOrder.Symbol]; ok && openOrder.NewClientOrderID == spotWSOrder.ClientOrderID {
					delete(bnspotOpenOrders, spotWSOrder.Symbol)
				}
				bnspotHttpBalanceUpdateSilentTimes[spotWSOrder.Symbol] = time.Now().Add(*bnConfig.HttpSilent)
			} else if spotWSOrder.CurrentOrderStatus == bnspot.OrderStatusCancelled {
				logger.Debugf("SPOT WS ORDER %s %s %s", spotWSOrder.Symbol, spotWSOrder.CurrentOrderStatus, spotWSOrder.ClientOrderID)
				if openOrder, ok := bnspotOpenOrders[spotWSOrder.Symbol]; ok && openOrder.NewClientOrderID == spotWSOrder.OriginalClientOrderID {
					delete(bnspotOpenOrders, spotWSOrder.Symbol)
				}
			} else if spotWSOrder.CurrentOrderStatus == bnspot.OrderStatusExpired ||
				spotWSOrder.CurrentOrderStatus == bnspot.OrderStatusReject {
				bnspotBalancesUpdateTimes[spotWSOrder.Symbol] = time.Now()
				logger.Debugf("SPOT WS ORDER %s %s %s", spotWSOrder.Symbol, spotWSOrder.CurrentOrderStatus, spotWSOrder.ClientOrderID)
				if openOrder, ok := bnspotOpenOrders[spotWSOrder.Symbol]; ok && openOrder.NewClientOrderID == spotWSOrder.ClientOrderID {
					delete(bnspotOpenOrders, spotWSOrder.Symbol)
				}
			}
			break
		case msg := <-bnswapUserWebsocket.BalanceAndPositionUpdateEventCh:
			handleWSAccountEvent(msg)
			break
		case orderEvent := <-bnswapUserWebsocket.OrderUpdateEventCh:
			wsOrder := orderEvent.Order
			if wsOrder.Status == common.OrderStatusExpired ||
				wsOrder.Status == common.OrderStatusReject ||
				wsOrder.Status == common.OrderStatusCancelled {
				logger.Debugf("SWAP WS ORDER %s %s %s", wsOrder.Symbol, wsOrder.ClientOrderId, wsOrder.Status)
			} else if wsOrder.Status == common.OrderStatusFilled {
				logger.Debugf("SWAP WS ORDER %s %s %s %f %f", wsOrder.Symbol, wsOrder.ClientOrderId, wsOrder.Status, wsOrder.FilledAccumulatedQuantity, wsOrder.AveragePrice)
				if wsOrder.Side == common.OrderSideBuy {
					if spotPrice, ok := bnspotLastLimitSellPrices[wsOrder.Symbol]; ok {
						bnRealisedSpread[wsOrder.Symbol] = (wsOrder.AveragePrice - spotPrice) / spotPrice
						logger.Debugf("%s REALISED OPEN SPREAD %f", wsOrder.Symbol, bnRealisedSpread[wsOrder.Symbol])
					}
				} else {
					if spotPrice, ok := bnspotLastLimitBuyPrices[wsOrder.Symbol]; ok {
						bnRealisedSpread[wsOrder.Symbol] = (wsOrder.AveragePrice - spotPrice) / spotPrice
						logger.Debugf("%s REALISED OPEN SPREAD %f", wsOrder.Symbol, bnRealisedSpread[wsOrder.Symbol])
					}
				}
			}
			break
		case spread := <-spreadCh:
			bnSpreads[spread.MakerSymbol] = spread
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
			break
		case bnspotBarsMap = <-bnspotBarsMapCh:
			break
		case bnQuantiles = <-bnQuantilesCh:
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

		case order := <-bnswapOrderResponseCh:
			if order.Status == common.OrderStatusReject ||
				order.Status == common.OrderStatusExpired ||
				order.Status == common.OrderStatusCancelled {
				logger.Debugf("SWAP ORDER %s %s %s", order.Symbol, order.Status, order.ClientOrderId)
				bnswapOrderSilentTimes[order.Symbol] = time.Now().Add(time.Second)
				bnswapPositionsUpdateTimes[order.Symbol] = time.Unix(0, 0)
			} else if order.Side == common.OrderSideSell &&
				order.Status == common.OrderStatusFilled &&
				order.CumQuote != 0 && order.CumQty != 0 {
				filledPrice := order.CumQuote / order.CumQty
				logger.Debugf(
					"SWAP ORDER %s %s %s %f %f %f",
					order.Symbol, order.Status, order.ClientOrderId,
					order.CumQty, order.CumQuote, filledPrice,
				)
				if spotPrice, ok := bnspotLastLimitBuyPrices[order.Symbol]; ok {
					bnRealisedSpread[order.Symbol] = (filledPrice - spotPrice) / spotPrice
					logger.Debugf("%s REALISED OPEN SPREAD %f", order.Symbol, bnRealisedSpread[order.Symbol])
				}
				bnswapHttpPositionUpdateSilentTimes[order.Symbol] = time.Now().Add(*bnConfig.HttpSilent)
			} else if order.Side == common.OrderSideBuy &&
				order.Status == common.OrderStatusFilled &&
				order.CumQuote != 0 && order.CumQty != 0 {
				filledPrice := order.CumQuote / order.CumQty
				logger.Debugf(
					"SWAP ORDER %s %s %s %f %f %f",
					order.Symbol, order.Status, order.ClientOrderId,
					order.CumQty, order.CumQuote, filledPrice,
				)
				if spotPrice, ok := bnspotLastLimitSellPrices[order.Symbol]; ok {
					bnRealisedSpread[order.Symbol] = (filledPrice - spotPrice) / spotPrice
					logger.Debugf("%s REALISED CLOSE SPREAD %f", order.Symbol, bnRealisedSpread[order.Symbol])
				}
				bnswapHttpPositionUpdateSilentTimes[order.Symbol] = time.Now().Add(*bnConfig.HttpSilent)
			}
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
				bnspotBalancesUpdateTimes[order.Symbol] = time.Now()
				if openOrder, ok := bnspotOpenOrders[order.Symbol]; ok && openOrder.NewClientOrderID == order.ClientOrderID {
					delete(bnspotOpenOrders, order.Symbol)
				}
			} else if order.Status == bnspot.OrderStatusFilled {
				logStr = fmt.Sprintf("%s FILLED PRICE %f", logStr, order.Price)
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
			frSum := 0.0
			for i, symbol := range bnSymbols {
				if premiumIndex, ok := bnswapPremiumIndexes[symbol]; ok {
					frs[i] = premiumIndex.FundingRate
					frSum += premiumIndex.FundingRate
				} else {
					logger.Debugf("MISS MARK PRICE %s", symbol)
					return
				}
			}
			frSum /= float64(len(bnSymbols))
			bnswapAvgFundingRate = &frSum
			if bnspotBarsMap != nil && bnswapBarsMap != nil {
				select {
				case bnswapAvgFundingRateCh <- frSum:
				default:
				}
				select {
				case bnBarsMapCh <- [2]common.KLinesMap{bnspotBarsMap, bnswapBarsMap}:
				default:
				}
			}
			bnRankSymbolMap, err = common.RankSymbols(bnSymbols, frs)
			if err != nil {
				logger.Debugf("RankSymbols error %v", err)
			}
			frRankUpdatedTimer.Reset(time.Minute)
		case <-bnbReBalanceTimer.C:
			handleReBalanceBnb()
			bnbReBalanceTimer.Reset(*bnConfig.BnbCheckInterval)
			break
		case <-bnLoopTimer.C:
			if bnswapSystemReady && bnspotSystemReady && time.Now().Sub(bnGlobalSilent) > 0 {
				updateSwapPositions()
				updateMakerOldOrders()
				if time.Now().Sub(time.Now().Truncate(fundingInterval)) > fundingSilent &&
					time.Now().Truncate(fundingInterval).Add(fundingInterval).Sub(time.Now()) > fundingSilent {
					updateMakerNewOrders()
				}
			} else {
				if time.Now().Sub(bnGlobalLogSilentTime) > 0 {
					logger.Debugf("SYSTEM NOT READY SPOT %v SWAP %v SILENT TIME %v",
						bnswapSystemReady, bnspotSystemReady, time.Now().Sub(bnGlobalSilent),
					)
					bnGlobalLogSilentTime = time.Now().Add(time.Second*5)
				}
				if len(bnspotOpenOrders) > 0 {
					for symbol := range bnspotOpenOrders {
						bnspotOrderRequestChs[symbol] <- SpotOrderRequest{
							Cancel: &bnspot.CancelAllOrderParams{Symbol: symbol},
						}
						delete(bnspotOpenOrders, symbol)
					}
				}
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
