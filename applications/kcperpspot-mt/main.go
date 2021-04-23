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

	_, kcperpMultipliers, kcperpTickSizes, _, err = kcperp.GetOrderLimits(kcGlobalCtx, kcperpAPI, kcperpSymbols)
	if err != nil {
		logger.Fatal(err)
	}
	_, kcspotStepSizes, kcspotTickSizes, kcspotMinNotional, err = kcspot.GetOrderLimits(kcGlobalCtx, kcspotAPI, kcspotSymbols)
	if err != nil {
		logger.Fatal(err)
	}
	for _, spotSymbol := range kcspotSymbols {
		kcMergedStepSizes[spotSymbol] =common.MergedStepSize(kcspotStepSizes[spotSymbol], kcperpMultipliers[kcspSymbolsMap[spotSymbol]])
	}
	logger.Debugf("MERGED STEP SIZE %v", kcMergedStepSizes)

	kcInternalInfluxWriter, err = common.NewInfluxWriter(
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
		err := kcInternalInfluxWriter.Stop()
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

	frRankUpdatedTimer := time.NewTimer(time.Second * 180)

	defer influxSaveTimer.Stop()
	defer kcLoopTimer.Stop()
	defer frRankUpdatedTimer.Stop()

	go kcperp.PositionsHttpLoop(
		kcGlobalCtx, kcperpAPI,
		kcperpSymbols, *kcConfig.PullInterval,
		kcperpPositionCh,
	)
	go kcperp.AccountHttpLoop(
		kcGlobalCtx, kcperpAPI,
		kcperp.AccountParam{Currency: "USDT"},
		*kcConfig.PullInterval, kcperpAccountCh,
	)
	go kcspot.AccountHttpLoop(
		kcGlobalCtx, kcspotAPI,
		kcspot.AccountsParam{},
		*kcConfig.PullInterval, kcspotAccountCh,
	)
	go kcperp.FundingRateLoop(
		kcGlobalCtx, kcperpAPI,
		kcperpSymbols,
		*kcConfig.PullInterval*10, kcperpFundingRatesCh,
	)

	go perpBarsPullingLoop(
		kcGlobalCtx,
		kcperpAPI,
		kcperpSymbols,
		*kcConfig.BarsLookback,
		*kcConfig.PullBarsInterval,
		*kcConfig.PullBarsRetryInterval,
		kcperpBarsMapCh,
	)

	go spotBarsPullingLoop(
		kcGlobalCtx,
		kcspotAPI,
		kcspotSymbols,
		*kcConfig.BarsLookback,
		*kcConfig.PullBarsInterval,
		*kcConfig.PullBarsRetryInterval,
		kcspotBarsMapCh,
	)

	go deltaQuantileLoop(
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

	depthReportCh := make(chan common.DepthReport, 10000)
	spreadReportCh := make(chan common.SpreadReport, 10000)
	go reportsSaveLoop(
		kcGlobalCtx,
		kcInternalInfluxWriter,
		*kcConfig.InternalInflux,
		depthReportCh,
		spreadReportCh,
	)

	makerRowDepthChs := make(map[string]chan *common.DepthRawMessage)
	for start := 0; start < len(kcspotSymbols); start += *kcConfig.OrderBookBatchSize {
		end := start + *kcConfig.OrderBookBatchSize
		if end > len(kcspotSymbols) {
			end = len(kcspotSymbols)
		}
		subMakerRowDepthChs := make(map[string]chan *common.DepthRawMessage)
		for _, symbol := range kcspotSymbols[start:end] {
			makerRowDepthChs[symbol] = make(chan *common.DepthRawMessage, 100)
			subMakerRowDepthChs[symbol] = makerRowDepthChs[symbol]
		}
		go makerDepthWSLoop(
			kcGlobalCtx,
			kcGlobalCancel,
			kcspotAPI,
			*kcConfig.ProxyAddress,
			*kcConfig.OrderBookMakerDecay,
			*kcConfig.OrderBookMakerBias,
			*kcConfig.ReportCount,
			depthReportCh,
			subMakerRowDepthChs,
		)
	}

	takerRowDepthChs := make(map[string]chan *common.DepthRawMessage)
	for start := 0; start < len(kcperpSymbols); start += *kcConfig.OrderBookBatchSize {
		end := start + *kcConfig.OrderBookBatchSize
		if end > len(kcperpSymbols) {
			end = len(kcperpSymbols)
		}
		subTakerRowDepthChs := make(map[string]chan *common.DepthRawMessage)
		for _, symbol := range kcperpSymbols[start:end] {
			takerRowDepthChs[symbol] = make(chan *common.DepthRawMessage, 100)
			subTakerRowDepthChs[symbol] = takerRowDepthChs[symbol]
		}
		go takerDepthWSLoop(
			kcGlobalCtx,
			kcGlobalCancel,
			kcperpAPI,
			*kcConfig.ProxyAddress,
			*kcConfig.OrderBookTakerDecay,
			*kcConfig.OrderBookTakerBias,
			*kcConfig.ReportCount,
			depthReportCh,
			subTakerRowDepthChs,
		)
	}

	spreadCh := make(chan *common.MakerTakerSpread, len(kcspotSymbols)*100)
	for makerSymbol, takerSymbol := range kcConfig.SpotPerpPairs {
		go watchMakerTakerSpread(
			kcGlobalCtx,
			makerSymbol, takerSymbol,
			kcperpMultipliers[makerSymbol],
			*kcConfig.OrderBookMakerImpact,
			*kcConfig.OrderBookTakerImpact,
			*kcConfig.OrderBookMaxAgeDiff,
			*kcConfig.OrderBookMaxAge,
			*kcConfig.SpreadLookbackDuration,
			*kcConfig.SpreadLookbackMinimalWindow,
			makerRowDepthChs[makerSymbol],
			takerRowDepthChs[takerSymbol],
			spreadReportCh,
			spreadCh,
		)
	}

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

	go kcperp.WatchSystemStatusHttp(
		kcGlobalCtx,
		kcperpAPI,
		*kcConfig.PullInterval/2,
		kcPerpSystemStatusCh,
	)
	go kcspot.WatchSystemStatusHttp(
		kcGlobalCtx,
		kcspotAPI,
		*kcConfig.PullInterval/2,
		kcSpotSystemStatusCh,
	)

	defer kcGlobalCancel()

	logger.Debugf("START mainLoop")
	for {
		select {
		case <-done:
			logger.Debugf("EXIT mainLoop")
			return
		case kcspotSystemReady = <-kcSpotSystemStatusCh:
			if !kcspotSystemReady {
				kcSystemReadyTime = time.Now().Add(*kcConfig.RestartSilent)
			}
			break
		case kcperpSystemReady = <-kcPerpSystemStatusCh:
			if !kcperpSystemReady {
				kcSystemReadyTime = time.Now().Add(*kcConfig.RestartSilent)
			}
			break
		case <-kcspotUserWebsocket.RestartCh:
			logger.Debugf("kcspotUserWebsocket restart silent %v", *kcConfig.RestartSilent)
			handleWebsocketRestart()
			break
		case <-kcperpUserWebsocket.RestartCh:
			logger.Debugf("kcperpUserWebsocket restart silent %v", *kcConfig.RestartSilent)
			handleWebsocketRestart()
			break
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
				kcLoopTimer.Reset(time.Nanosecond)
				kcspotHttpBalanceUpdateSilentTimes[spotOrder.Symbol] = time.Now().Add(*kcConfig.HttpSilent)
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
			handlePerpWSPosition(msg)
			break
		case msg := <-kcperpUserWebsocket.BalanceCh:
			handlePerpWSBalance(msg)
			break
		case perpOrder := <-kcperpUserWebsocket.OrderCh:
			if perpOrder.Type == kcperp.OrderTypeCanceled ||
				perpOrder.Type == kcperp.OrderTypeMatch {
				if perpOrder.Type == kcperp.OrderTypeCanceled {
					logger.Debugf("PERP WS ORDER CANCELED %v ", perpOrder)
					kcperpOrderSilentTimes[perpOrder.Symbol] = time.Now().Add(time.Second)
					kcperpPositionsUpdateTimes[perpOrder.Symbol] = time.Unix(0, 0)
				} else {
					logger.Debugf(
						"PERP WS ORDER MATCHED %s SIDE %s MATCHED SIZE %v MATCHED PRICE %f",
						perpOrder.Symbol, perpOrder.Side, perpOrder.MatchSize, perpOrder.MatchPrice,
					)
					kcLoopTimer.Reset(time.Nanosecond)
					kcperpHttpPositionUpdateSilentTimes[perpOrder.Symbol] = time.Now().Add(*kcConfig.HttpSilent)
					if perpOrder.Side == kcperp.OrderSideSell {
						if spotSymbol, ok := kcpsSymbolsMap[perpOrder.Symbol]; ok {
							if spotPrice, ok := kcspotLastFilledBuyPrices[spotSymbol]; ok {
								kcRealisedSpread[spotSymbol] = (perpOrder.MatchPrice - spotPrice) / spotPrice
								logger.Debugf("%s %s REALISED OPEN SPREAD %f", spotSymbol, perpOrder.Symbol, kcRealisedSpread[spotSymbol])
							}
						}
					} else if perpOrder.Side == kcperp.OrderSideBuy {
						if spotSymbol, ok := kcpsSymbolsMap[perpOrder.Symbol]; ok {
							if spotPrice, ok := kcspotLastFilledSellPrices[spotSymbol]; ok {
								kcRealisedSpread[spotSymbol] = (perpOrder.MatchPrice - spotPrice) / spotPrice
								logger.Debugf("%s %s REALISED CLOSE SPREAD %f", spotSymbol, perpOrder.Symbol, kcRealisedSpread[spotSymbol])
							}
						}
					}
				}
			}
			break
		case spread := <-spreadCh:
			kcSpreads[spread.MakerSymbol] = spread
			break
		case fr := <-kcperpFundingRatesCh:
			kcperpFundingRates[fr.Symbol] = fr
			break
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
			kcLoopTimer.Reset(time.Second)
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
					frs[i] = markPrice.Value
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
			frRankUpdatedTimer.Reset(time.Minute)
		case <-kcLoopTimer.C:
			if kcperpSystemReady && kcspotSystemReady && time.Now().Sub(kcSystemReadyTime) > 0 {
				updatePerpPositions()
				updateSpotOldOrders()
				updateSpotNewOrders()
			} else {
				if len(kcspotOpenOrders) > 0 {
					for symbol := range kcspotOpenOrders {
						kcspotOrderRequestChs[symbol] <- SpotOrderRequest{
							Cancel: &kcspot.CancelAllOrdersParam{Symbol: symbol},
						}
						delete(kcspotOpenOrders, symbol)
					}
				}
			}
			kcLoopTimer.Reset(
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
