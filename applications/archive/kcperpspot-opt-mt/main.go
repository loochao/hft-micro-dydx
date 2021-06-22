package main

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/kucoin-usdtfuture"
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
	kcperpAPI, err = kucoin_usdtfuture.NewAPI(
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

	_, kcperpMultipliers, _, _, err = kucoin_usdtfuture.GetOrderLimits(kcGlobalCtx, kcperpAPI, kcperpSymbols)
	if err != nil {
		logger.Fatal(err)
	}
	_, kcspotStepSizes, kcspotTickSizes, kcspotMinNotional, err = kcspot.GetOrderLimits(kcGlobalCtx, kcspotAPI, kcspotSymbols)
	if err != nil {
		logger.Fatal(err)
	}
	for _, spotSymbol := range kcspotSymbols {
		kcMergedStepSizes[spotSymbol] = common.MergedStepSize(kcspotStepSizes[spotSymbol], kcperpMultipliers[kcspSymbolsMap[spotSymbol]])
	}
	logger.Debugf("MERGED STEP SIZE %v", kcMergedStepSizes)

	kcInternalInfluxWriter, err = common.NewInfluxWriter(
		kcGlobalCtx,
		*kcConfig.InternalInflux.Address,
		*kcConfig.InternalInflux.Username,
		*kcConfig.InternalInflux.Password,
		*kcConfig.InternalInflux.Database,
		*kcConfig.InternalInflux.BatchSize,
	)
	if err != nil {
		logger.Fatal(err)
	}
	defer kcInternalInfluxWriter.Stop()

	kcExternalInfluxWriter, err = common.NewInfluxWriter(
		kcGlobalCtx,
		*kcConfig.ExternalInflux.Address,
		*kcConfig.ExternalInflux.Username,
		*kcConfig.ExternalInflux.Password,
		*kcConfig.ExternalInflux.Database,
		*kcConfig.ExternalInflux.BatchSize,
	)
	if err != nil {
		logger.Fatal(err)
	}
	defer kcInternalInfluxWriter.Stop()

	if *kcConfig.ChangeAutoDepositStatus {
		for _, symbol := range kcperpSymbols {
			res, err := kcperpAPI.ChangeAutoDepositStatus(kcGlobalCtx, kucoin_usdtfuture.AutoDepositStatusParam{
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

	kcperpUserWebsocket = kucoin_usdtfuture.NewUserWebsocketAndStart(
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

	go kucoin_usdtfuture.PositionsHttpLoop(
		kcGlobalCtx, kcperpAPI,
		kcperpSymbols, *kcConfig.PullInterval,
		kcperpPositionCh,
	)
	go kucoin_usdtfuture.AccountHttpLoop(
		kcGlobalCtx, kcperpAPI,
		kucoin_usdtfuture.AccountParam{Currency: "USDT"},
		*kcConfig.PullInterval, kcperpAccountCh,
	)
	go kcspot.AccountHttpLoop(
		kcGlobalCtx, kcspotAPI,
		kcspot.AccountsParam{},
		*kcConfig.PullInterval, kcspotAccountCh,
	)
	go kucoin_usdtfuture.FundingRateLoop(
		kcGlobalCtx, kcperpAPI,
		kcperpSymbols,
		*kcConfig.PullInterval*10, kcperpFundingRatesCh,
	)

	spreadReportCh := make(chan common.SpreadReport, 10000)
	go reportsSaveLoop(
		kcGlobalCtx,
		kcInternalInfluxWriter,
		*kcConfig.InternalInflux,
		spreadReportCh,
	)

	makerRawDepthChs := make(map[string]chan *common.DepthRawMessage)
	for start := 0; start < len(kcspotSymbols); start += *kcConfig.OrderBookBatchSize {
		end := start + *kcConfig.OrderBookBatchSize
		if end > len(kcspotSymbols) {
			end = len(kcspotSymbols)
		}
		subMakerRowDepthChs := make(map[string]chan *common.DepthRawMessage)
		for _, symbol := range kcspotSymbols[start:end] {
			makerRawDepthChs[symbol] = make(chan *common.DepthRawMessage, 100)
			subMakerRowDepthChs[symbol] = makerRawDepthChs[symbol]
		}
		go makerDepthWSLoop(
			kcGlobalCtx,
			kcGlobalCancel,
			kcspotAPI,
			*kcConfig.ProxyAddress,
			subMakerRowDepthChs,
		)
	}

	takerRawDepthChs := make(map[string]chan *common.DepthRawMessage)
	for start := 0; start < len(kcperpSymbols); start += *kcConfig.OrderBookBatchSize {
		end := start + *kcConfig.OrderBookBatchSize
		if end > len(kcperpSymbols) {
			end = len(kcperpSymbols)
		}
		subTakerRowDepthChs := make(map[string]chan *common.DepthRawMessage)
		for _, symbol := range kcperpSymbols[start:end] {
			takerRawDepthChs[symbol] = make(chan *common.DepthRawMessage, 100)
			subTakerRowDepthChs[symbol] = takerRawDepthChs[symbol]
		}
		go takerDepthWebsocketLoop(
			kcGlobalCtx,
			kcGlobalCancel,
			kcperpAPI,
			*kcConfig.ProxyAddress,
			subTakerRowDepthChs,
		)
	}

	spreadCh := make(chan *common.ShortSpread, len(kcspotSymbols)*100)
	for makerSymbol, takerSymbol := range kcConfig.SpotPerpPairs {
		go watchMakerTakerSpread(
			kcGlobalCtx,
			makerSymbol, takerSymbol,
			kcperpMultipliers[takerSymbol],
			*kcConfig.OrderBookTakerImpact,
			*kcConfig.OrderBookMakerDecay,
			*kcConfig.OrderBookMakerBias,
			*kcConfig.OrderBookTakerDecay,
			*kcConfig.OrderBookTakerBias,
			*kcConfig.OrderBookMaxAgeDiffBias,
			*kcConfig.ReportCount,
			*kcConfig.SpreadLookbackDuration,
			*kcConfig.SpreadLookbackMinimalWindow,
			makerRawDepthChs[makerSymbol],
			takerRawDepthChs[takerSymbol],
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
		kcperpOrderRequestChs[perpSymbol] = make(chan kucoin_usdtfuture.NewOrderParam, 2)
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

	go kucoin_usdtfuture.WatchSystemStatusHttp(
		kcGlobalCtx,
		kcperpAPI,
		*kcConfig.PullInterval/2,
		kcperpSystemStatusCh,
	)
	go kcspot.WatchSystemStatusHttp(
		kcGlobalCtx,
		kcspotAPI,
		*kcConfig.PullInterval/2,
		kcspotSystemStatusCh,
	)

	defer kcGlobalCancel()

	logger.Debugf("START mainLoop")
	for {
		select {
		case <-done:
			logger.Debugf("EXIT mainLoop")
			return
		case kcspotSystemReady = <-kcspotSystemStatusCh:
			if !kcspotSystemReady {
				kcGlobalSilent = time.Now().Add(*kcConfig.RestartSilent)
			}
			break
		case kcperpSystemReady = <-kcperpSystemStatusCh:
			if !kcperpSystemReady {
				kcGlobalSilent = time.Now().Add(*kcConfig.RestartSilent)
			}
			break
		case <-kcspotUserWebsocket.RestartCh:
			logger.Debugf("kcspotUserWebsocket restart silent %v", *kcConfig.RestartSilent)
			kcGlobalSilent = time.Now().Add(*kcConfig.RestartSilent)
			break
		case <-kcperpUserWebsocket.RestartCh:
			logger.Debugf("kcperpUserWebsocket restart silent %v", *kcConfig.RestartSilent)
			kcGlobalSilent = time.Now().Add(*kcConfig.RestartSilent)
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
				logger.Debugf("SPOT WS ORDER FILLED %s %s SIZE %f PRICE %f", spotOrder.Symbol, spotOrder.Side, spotOrder.FilledSize, spotOrder.Price)
				if openOrder, ok := kcspotOpenOrders[spotOrder.Symbol]; ok && openOrder.ClientOid == spotOrder.ClientOid {
					if spotOrder.FilledSize > 0 {
						if spotOrder.Side == kcspot.OrderSideBuy {
							kcspotLastFilledBuyPrices[spotOrder.Symbol] = spotOrder.Price
						} else {
							kcspotLastFilledSellPrices[spotOrder.Symbol] = spotOrder.Price
						}
					}
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
			if perpOrder.EventType == kucoin_usdtfuture.OrderStatusCanceled ||
				perpOrder.EventType == kucoin_usdtfuture.OrderStatusMatch {
				if perpOrder.EventType == kucoin_usdtfuture.OrderStatusCanceled {
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
					if perpOrder.Side == kucoin_usdtfuture.OrderSideSell {
						if spotSymbol, ok := kcpsSymbolsMap[perpOrder.Symbol]; ok {
							if spotPrice, ok := kcspotLastFilledBuyPrices[spotSymbol]; ok && spotPrice > 0 {
								kcRealisedSpread[spotSymbol] = (perpOrder.MatchPrice - spotPrice) / spotPrice
								logger.Debugf("%s %s REALISED OPEN SPREAD %f", spotSymbol, perpOrder.Symbol, kcRealisedSpread[spotSymbol])
								delete(kcspotLastFilledBuyPrices, spotSymbol)
							}
						}
					} else if perpOrder.Side == kucoin_usdtfuture.OrderSideBuy {
						if spotSymbol, ok := kcpsSymbolsMap[perpOrder.Symbol]; ok {
							if spotPrice, ok := kcspotLastFilledSellPrices[spotSymbol]; ok && spotPrice > 0 {
								kcRealisedSpread[spotSymbol] = (perpOrder.MatchPrice - spotPrice) / spotPrice
								logger.Debugf("%s %s REALISED CLOSE SPREAD %f", spotSymbol, perpOrder.Symbol, kcRealisedSpread[spotSymbol])
								delete(kcspotLastFilledSellPrices, spotSymbol)
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
			if kcperpSystemReady && kcspotSystemReady && time.Now().Sub(kcGlobalSilent) > 0 {
				updatePerpPositions()
				updateSpotOldOrders()
				updateSpotNewOrders()
			} else {
				if time.Now().Sub(time.Now().Truncate(time.Second*15)) < *kcConfig.LoopInterval {
					logger.Debugf(
						"SYSTEM NOT READY kcperpSystemReady %v kcspotSystemReady %v kcGlobalSilent %v",
						kcperpSystemReady, kcspotSystemReady, time.Now().Sub(kcGlobalSilent),
					)
				}
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
