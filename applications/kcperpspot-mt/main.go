package main

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/kcperp"
	"github.com/geometrybase/hft-micro/kcspot"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"
	"time"
)

func main() {

	if *kcConfig.CpuProfile != "" {
		f, err := os.Create(*kcConfig.CpuProfile)
		if err != nil {
			logger.Fatal(err)
		}
		err = pprof.StartCPUProfile(f)
		if err != nil {
			logger.Fatal(err)
		}
		defer pprof.StopCPUProfile()
	}

	var err error
	kcperpAPI, err = kcperp.NewAPI(
		*kcConfig.PerpApiKey,
		*kcConfig.PerpApiSecret,
		*kcConfig.PerpApiPassphrase,
		*kcConfig.ProxyAddress,
	)
	if err != nil {
		logger.Fatal(err)
	}
	kcspotAPI, err = kcspot.NewAPI(
		*kcConfig.SpotApiKey,
		*kcConfig.SpotApiSecret,
		*kcConfig.SpotApiPassphrase,
		*kcConfig.ProxyAddress,
	)
	if err != nil {
		logger.Fatal(err)
	}

	kcGlobalCtx, kcGlobalCancel = context.WithCancel(context.Background())
	defer kcGlobalCancel()

	kcperpLotSizes, kcperpMultipliers, kcperpTickSizes, _, err = kcperp.GetOrderLimits(kcGlobalCtx, kcperpAPI, kcperpSymbols)
	if err != nil {
		logger.Fatal(err)
	}
	_, kcspotStepSizes, kcspotTickSizes, kcspotMinNotional, err = kcspot.GetOrderLimits(kcGlobalCtx, kcspotAPI, kcspotSymbols)
	if err != nil {
		logger.Fatal(err)
	}

	kcInfluxWriter, err = common.NewInfluxWriter(
		*kcConfig.InternalInflux.Address,
		*kcConfig.InternalInflux.Username,
		*kcConfig.InternalInflux.Password,
		*kcConfig.InternalInflux.Database,
		*kcConfig.InternalInflux.BatchSize,
	)
	if err != nil {
		logger.Fatal(err)
	}

	kcExternalInfluxWriter, err = common.NewInfluxWriter(
		*kcConfig.ExternalInflux.Address,
		*kcConfig.ExternalInflux.Username,
		*kcConfig.ExternalInflux.Password,
		*kcConfig.ExternalInflux.Database,
		*kcConfig.ExternalInflux.BatchSize,
	)
	if err != nil {
		logger.Fatal(err)
	}

	defer func() {
		err := kcInfluxWriter.Stop()
		if err != nil {
			logger.Warnf("stop influx writer error %v", err)
		}
	}()

	if *kcConfig.ChangeAutoDepositStatus {
		for _, symbol := range kcperpSymbols {
			res, err := kcperpAPI.ChangeAutoDepositStatus(kcGlobalCtx, kcperp.AutoDepositStatusParam{
				Symbol: symbol,
				Status: true,
			})
			if err != nil {
				logger.Debugf("ChangeAutoDepositStatus FOR %s ERROR %v", symbol, err)
			} else {
				logger.Debugf("ChangeAutoDepositStatus FOR %s RESPONSE %v", symbol, res)
			}
			time.Sleep(time.Second)
		}
	}

	kcspotUserWebsocket = kcspot.NewUserWebsocket(
		kcGlobalCtx,
		kcspotAPI,
		*kcConfig.ProxyAddress,
	)
	defer kcspotUserWebsocket.Stop()

	kcperpUserWebsocket = kcperp.NewUserWebsocket(
		kcGlobalCtx,
		kcperpAPI,
		kcperpSymbols,
		*kcConfig.ProxyAddress,
	)
	defer kcperpUserWebsocket.Stop()

	influxSaveTimer := time.NewTimer(
		time.Now().Truncate(
			*kcConfig.InternalInflux.SaveInterval,
		).Add(
			*kcConfig.InternalInflux.SaveInterval * 3,
		).Sub(time.Now()),
	)
	externalInfluxSaveTimer := time.NewTimer(
		time.Now().Truncate(
			*kcConfig.ExternalInflux.SaveInterval,
		).Add(
			*kcConfig.ExternalInflux.SaveInterval * 3,
		).Sub(time.Now()),
	)
	loopTimer := time.NewTimer(time.Hour * 24) //先等1分钟
	//targetValueUpdateTimer := time.NewTimer(time.Hour * 24)
	//resetUnrealisedPnlTimer := time.NewTimer(time.Minute)
	//reBalanceTimer := time.NewTimer(time.Second)
	frRankUpdatedTimer := time.NewTimer(time.Second * 180)

	defer influxSaveTimer.Stop()
	defer loopTimer.Stop()
	//defer targetValueUpdateTimer.Stop()
	//defer resetUnrealisedPnlTimer.Stop()
	//defer reBalanceTimer.Stop()
	defer frRankUpdatedTimer.Stop()
	//
	go kcperp.WatchPositionsFromHttp(
		kcGlobalCtx, kcperpAPI,
		kcperpSymbols, *kcConfig.PullInterval,
		kcperpPositionCh,
	)
	go kcperp.WatchAccountFromHttp(
		kcGlobalCtx, kcperpAPI,
		kcperp.AccountParam{Currency: "USDT"},
		*kcConfig.PullInterval, kcperpAccountCh,
	)
	go kcspot.WatchAccountFromHttp(
		kcGlobalCtx, kcspotAPI,
		kcspot.AccountsParam{},
		*kcConfig.PullInterval, kcspotAccountCh,
	)

	go watchPerpBars(
		kcGlobalCtx,
		kcperpAPI,
		kcperpSymbols,
		*kcConfig.BarsLookback,
		*kcConfig.PullBarsInterval,
		*kcConfig.PullBarsRetryInterval,
		kcperpBarsMapCh,
	)

	go watchSpotBars(
		kcGlobalCtx,
		kcspotAPI,
		kcspotSymbols,
		*kcConfig.BarsLookback,
		*kcConfig.PullBarsInterval,
		*kcConfig.PullBarsRetryInterval,
		kcspotBarsMapCh,
	)

	go watchDeltaQuantile(
		kcGlobalCtx,
		kcspotSymbols,
		kcspSymbolsMap,
		*kcConfig.BotQuantile,
		*kcConfig.TopQuantile,
		*kcConfig.TopBandScale,
		*kcConfig.BotBandScale,
		*kcConfig.MinimalEnterDelta,
		*kcConfig.MaximalExitDelta,
		*kcConfig.MinimalBandOffset,
		kcBarsMapCh,
		kcQuantilesCh,
	)

	walkedOrderBookCh := make(chan WalkedOrderBook, len(kcspotSymbols)*10)
	for start := 0; start < len(kcspotSymbols); start += *kcConfig.OrderBookBatchSize {
		end := start + *kcConfig.OrderBookBatchSize
		if end > len(kcspotSymbols) {
			end = len(kcspotSymbols)
		}
		go watchSpotWalkedOrderBooks(
			kcGlobalCtx,
			kcspotAPI,
			*kcConfig.ProxyAddress,
			*kcConfig.OrderBookTakerImpact,
			*kcConfig.OrderBookMakerImpact,
			kcspotSymbols[start:end],
			walkedOrderBookCh,
		)
	}

	perpMarkPriceCh := make(chan *kcperp.MarkPrice, len(kcperpSymbols)*100)
	perpFundingRateCh := make(chan *kcperp.FundingRate, len(kcperpSymbols)*100)
	for start := 0; start < len(kcperpSymbols); start += *kcConfig.OrderBookBatchSize {
		end := start + *kcConfig.OrderBookBatchSize
		if end > len(kcspotSymbols) {
			end = len(kcspotSymbols)
		}
		go watchPerpWalkedOrderBooks(
			kcGlobalCtx,
			kcperpAPI,
			*kcConfig.ProxyAddress,
			kcperpMultipliers,
			*kcConfig.OrderBookTakerImpact,
			*kcConfig.OrderBookMakerImpact,
			kcperpSymbols[start:end],
			walkedOrderBookCh,
		)
		go watchInstrument(
			kcGlobalCtx,
			kcperpAPI,
			*kcConfig.ProxyAddress,
			kcperpSymbols[start:end],
			perpMarkPriceCh,
			perpFundingRateCh,
		)
	}

	spreadCh := make(chan Spread, len(kcspotSymbols)*100)
	go watchSpread(
		kcGlobalCtx,
		kcspotSymbols,
		kcpsSymbolsMap,
		*kcConfig.OrderBookMaxAgeDiff,
		*kcConfig.OrderBookMaxAge,
		*kcConfig.SpreadLookbackDuration,
		*kcConfig.SpreadLookbackMinimalWindow,
		walkedOrderBookCh,
		spreadCh,
	)

	kcspotNewOrderErrorCh = make(chan SpotOrderNewError, len(kcspotSymbols)*2)
	for _, spotSymbol := range kcspotSymbols {
		kcspotOrderRequestChs[spotSymbol] = make(chan SpotOrderRequest, 2)
		go watchSpotOrderRequest(
			kcGlobalCtx,
			kcspotAPI,
			*kcConfig.OrderTimeout,
			*kcConfig.DryRun,
			kcspotOrderRequestChs[spotSymbol],
			kcspotNewOrderErrorCh,
		)
		kcspotOrderRequestChs[spotSymbol] <- SpotOrderRequest{
			Cancel: &kcspot.CancelAllOrdersParam{Symbol: spotSymbol},
		}
	}

	kcperpNewOrderErrorCh = make(chan PerpOrderNewError, len(kcspotSymbols)*2)
	for _, perpSymbol := range kcperpSymbols {
		kcperpOrderRequestChs[perpSymbol] = make(chan kcperp.NewOrderParam, 2)
		go watchPerpOrderRequest(
			kcGlobalCtx,
			kcperpAPI,
			*kcConfig.OrderTimeout,
			*kcConfig.DryRun,
			kcperpOrderRequestChs[perpSymbol],
			kcperpNewOrderErrorCh,
		)
	}

	done := make(chan bool, 1)
	if *kcConfig.CpuProfile != "" {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			sig := <-sigs
			logger.Debugf("Exit with sig %d, clean *.tmp files", sig)
			done <- true
		}()
	}

	defer kcGlobalCancel()

	for {
		select {
		case <-done:
			logger.Debugf("Exit")
			return
		//case <-reBalanceTimer.C:
		//	if kcspotUSDTBalance != nil && kcperpUSDTAccount != nil && kcperpUSDTAccount.AvailableBalance != nil {
		//		//PERP WS ACCOUNT 没有AvailableBalance, 为0 HTTP GET无数据
		//		//PERP的MarginBalance AvailableBalance WS推送缺失，会造成错误判断
		//		if kcperpAssetUpdatedForReBalance && kcspotBalanceUpdatedForReBalance {
		//			kcperpAssetUpdatedForReBalance = false
		//			kcspotBalanceUpdatedForReBalance = false
		//
		//			expectedInsuranceFund := *kcConfig.StartValue * (1 - *kcConfig.InsuranceFundingRatio) * *kcConfig.Leverage / (*kcConfig.Leverage + 1) * *kcConfig.InsuranceFundingRatio
		//			totalFree := (kcspotUSDTBalance.Free + *kcperpUSDTAccount.AvailableBalance) - expectedInsuranceFund
		//			targetPerp := totalFree/(*kcConfig.Leverage+1) + expectedInsuranceFund
		//			change := targetPerp - *kcperpUSDTAccount.AvailableBalance
		//			if change > 0 && change > kcspotUSDTBalance.Free {
		//				change = kcspotUSDTBalance.Free
		//			}
		//			if change < 0 && -change > *kcperpUSDTAccount.AvailableBalance-expectedInsuranceFund {
		//				change = 0
		//				if *kcperpUSDTAccount.AvailableBalance-expectedInsuranceFund > 0 {
		//					change = -(*kcperpUSDTAccount.AvailableBalance - expectedInsuranceFund)
		//				}
		//			}
		//			if math.Abs(change) > *kcConfig.ReBalanceMinimalNotional {
		//				// 如果有转帐发生最好不要让influx统计数据，转帐在中间过程中会有盈利计算误差
		//				kcspotBalanceUpdatedForExternalInflux = false
		//				kcperpAssetUpdatedForExternalInflux = false
		//				kcspotBalanceUpdatedForInflux = false
		//				kcperpAssetUpdatedForInflux = false
		//				kcSaveSilentTime = time.Now().Add(*kcConfig.PullInterval * 2)
		//				go reBalanceUSDT(
		//					kcGlobalCtx,
		//					kcspotAPI,
		//					*kcConfig.OrderTimeout,
		//					change,
		//				)
		//			}
		//		}
		//	}
		//	reBalanceTimer.Reset(*kcConfig.ReBalanceInterval)
		//	break
		case p := <-kcperpPositionCh:
			handlePerpHttpPositions(p)
			break
		case account := <-kcperpAccountCh:
			handlePerpHttpAccount(account)
			break
		case account := <-kcspotAccountCh:
			handleSpotHttpAccount(account)
			break
		case msg := <-kcspotUserWebsocket.BalanceCh:
			handleSpotWSBalance(msg)
			break
		case spotOrder := <-kcspotUserWebsocket.OrderCh:
			if spotOrder.Type == kcspot.OrderTypeFilled {
				if spotOrder.FilledSize > 0 {
					if spotOrder.Side == kcspot.OrderSideBuy {
						kcspotLastFilledBuyPrices[spotOrder.Symbol] = spotOrder.Price
					} else {
						kcspotLastFilledSellPrices[spotOrder.Symbol] = spotOrder.Price
					}
					logger.Debugf("SPOT WS ORDER FILLED %s %s SIZE %f PRICE %f", spotOrder.Symbol, spotOrder.Side, spotOrder.FilledSize, spotOrder.Price)
				}
				if openOrder, ok := kcspotOpenOrders[spotOrder.Symbol]; ok && openOrder.ClientOid == spotOrder.ClientOid {
					delete(kcspotOpenOrders, spotOrder.Symbol)
				}
			} else if spotOrder.Type == kcspot.OrderTypeCanceled {
				if openOrder, ok := kcspotOpenOrders[spotOrder.Symbol]; ok && openOrder.ClientOid == spotOrder.ClientOid {
					delete(kcspotOpenOrders, spotOrder.Symbol)
				}
			} else if spotOrder.Status == kcspot.OrderStatusDone {
				if openOrder, ok := kcspotOpenOrders[spotOrder.Symbol]; ok && openOrder.ClientOid == spotOrder.ClientOid {
					delete(kcspotOpenOrders, spotOrder.Symbol)
				}
			}
			break
		case msg := <-kcperpUserWebsocket.PositionCh:
			handleWSPosition(msg)
			break
		case msg := <-kcperpUserWebsocket.BalanceCh:
			handleWSBalance(msg)
			break
		case order := <-kcperpUserWebsocket.OrderCh:
			if order.Type == kcperp.OrderTypeCanceled ||
				order.Type == kcperp.OrderTypeMatch {
				if order.Type == kcperp.OrderTypeCanceled {
					logger.Debugf("PERP WS ORDER CANCELED %v ", order)
					kcperpOrderSilentTimes[order.Symbol] = time.Now().Add(time.Second)
					kcperpPositionsUpdateTimes[order.Symbol] = time.Unix(0, 0)
				} else {
					logger.Debugf(
						"PERP WS ORDER MATCHED %s SIDE %s MATCHED SIZE %v MATCHED PRICE %f",
						order.Symbol, order.Side, order.MatchSize, order.MatchPrice,
					)
					if order.Side == common.OrderSideSell {
						if spotSymbol, ok := kcpsSymbolsMap[order.Symbol]; ok {
							if spotPrice, ok := kcspotLastFilledBuyPrices[spotSymbol]; ok {
								kcRealisedSpread[spotSymbol] = (order.MatchPrice - spotPrice) / spotPrice
								logger.Debugf("%s %s REALISED OPEN SPREAD %f", spotSymbol, order.Symbol, kcRealisedSpread[order.Symbol])
							}
						}
					} else if order.Side == common.OrderSideBuy  {
						if spotSymbol, ok := kcpsSymbolsMap[order.Symbol]; ok {
							if spotPrice, ok := kcspotLastFilledSellPrices[spotSymbol]; ok {
								kcRealisedSpread[order.Symbol] = (order.MatchPrice - spotPrice) / spotPrice
								logger.Debugf("%s %s REALISED CLOSE SPREAD %f", spotSymbol, order.Symbol, kcRealisedSpread[order.Symbol])
							}
						}
					}
				}
			}
			break
		case spread := <-spreadCh:
			kcSpreads[spread.Symbol] = spread
			break
		case markPrice := <-perpMarkPriceCh:
			kcperpMarkPrices[markPrice.Symbol] = markPrice
			break
		case fr := <-perpFundingRateCh:
			kcperpFundingRates[fr.Symbol] = fr
			break
		//case <-resetUnrealisedPnlTimer.C:
		//	//handleResetPnl()
		//	resetUnrealisedPnlTimer.Reset(
		//		time.Now().Truncate(
		//			*kcConfig.ResetUnrealisedPnlInterval,
		//		).Add(
		//			*kcConfig.ResetUnrealisedPnlInterval,
		//		).Sub(time.Now()),
		//	)
		//	break
		case kcperpBarsMap = <-kcperpBarsMapCh:
			if kcBarsMapUpdated["spot"] {
				kcBarsMapCh <- [2]common.KLinesMap{kcspotBarsMap, kcperpBarsMap}
				kcBarsMapUpdated["spot"] = false
				kcBarsMapUpdated["swap"] = false
			} else {
				kcBarsMapUpdated["swap"] = true
			}
			break
		case kcspotBarsMap = <-kcspotBarsMapCh:
			if kcBarsMapUpdated["swap"] {
				kcBarsMapCh <- [2]common.KLinesMap{kcspotBarsMap, kcperpBarsMap}
				kcBarsMapUpdated["spot"] = false
				kcBarsMapUpdated["swap"] = false
			} else {
				kcBarsMapUpdated["spot"] = true
			}
			break
		case kcQuantiles = <-kcQuantilesCh:
			logger.Debugf("QUANTILES %v", kcQuantiles)
			loopTimer.Reset(time.Second)
			break
		case <-influxSaveTimer.C:
			handleSave()
			influxSaveTimer.Reset(
				time.Now().Truncate(
					*kcConfig.InternalInflux.SaveInterval,
				).Add(
					*kcConfig.InternalInflux.SaveInterval,
				).Sub(time.Now()),
			)
			break
		case <-externalInfluxSaveTimer.C:
			handleExternalInfluxSave()
			externalInfluxSaveTimer.Reset(
				time.Now().Truncate(
					*kcConfig.ExternalInflux.SaveInterval,
				).Add(
					*kcConfig.ExternalInflux.SaveInterval,
				).Sub(time.Now()),
			)
			break

		case newError := <-kcperpNewOrderErrorCh:
			kcperpOrderSilentTimes[newError.Params.Symbol] = time.Now().Add(time.Second * 15)
			break

		case order := <-kcspotNewOrderErrorCh:
			if openOrder, ok := kcspotOpenOrders[order.Params.Symbol]; ok && openOrder.ClientOid == order.Params.ClientOid {
				delete(kcspotOpenOrders, order.Params.Symbol)
			}
			kcspotOrderSilentTimes[order.Params.Symbol] = time.Now().Add(*kcConfig.OrderSilent * 3)
		case <-frRankUpdatedTimer.C:
			frs := make([]float64, len(kcperpSymbols))
			for i, symbol := range kcperpSymbols {
				if markPrice, ok := kcperpFundingRates[symbol]; ok {
					frs[i] = markPrice.FundingRate
				} else {
					logger.Debugf("MISS FUNDING RATE %s", symbol)
					break
				}
			}
			if len(kcRankSymbolMap) == 0 {
				logger.Debugf("RANK FR...")
			}
			kcRankSymbolMap, err = common.RankSymbols(kcperpSymbols, frs)
			if err != nil {
				logger.Debugf("RankSymbols error %v", err)
			}
			//logger.Debugf("SYMBOLS FR RANK %v", kcRankSymbolMap)
			frRankUpdatedTimer.Reset(time.Minute)
		case <-loopTimer.C:
			updatePerpPositions()
			updateSpotOldOrders()
			updateSpotNewOrders()
			loopTimer.Reset(
				time.Now().Truncate(
					*kcConfig.LoopInterval,
				).Add(
					*kcConfig.LoopInterval,
				).Sub(time.Now()),
			)
			break
		}
	}
}
