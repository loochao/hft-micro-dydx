package main

import (
	"context"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/kcperp"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"
	"time"
)

func main() {

	if *mtConfig.CpuProfile != "" {
		f, err := os.Create(*mtConfig.CpuProfile)
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
	mAPI, err = kcperp.NewAPI(
		*mtConfig.KcApiKey,
		*mtConfig.KcApiSecret,
		*mtConfig.KcApiPassphrase,
		*mtConfig.ProxyAddress,
	)
	if err != nil {
		logger.Debugf("kcperp.NewAPI error %v", err)
		return
	}
	tAPI, err = bnswap.NewAPI(
		&common.Credentials{
			Key:    *mtConfig.BnApiKey,
			Secret: *mtConfig.BnApiSecret,
		},
		*mtConfig.ProxyAddress,
	)
	if err != nil {
		logger.Debugf("bnswap.NewAPI error %v", err)
		return
	}

	mtGlobalCtx, mtGlobalCancel = context.WithCancel(context.Background())
	defer mtGlobalCancel()

	if *mtConfig.ChangeLeverage {
		for _, takerSymbol := range tSymbols {
			res, err := tAPI.UpdateLeverage(mtGlobalCtx, bnswap.UpdateLeverageParams{
				Symbol:   takerSymbol,
				Leverage: int64(*mtConfig.Leverage),
			})
			if err != nil {
				logger.Debugf("UPDATE LEVERAGE FOR %s ERROR %v", takerSymbol, err)
			} else {
				logger.Debugf("UPDATE LEVERAGE FOR %s RESPONSE %v", takerSymbol, res)
			}
			time.Sleep(time.Second)
			res, err = tAPI.UpdateMarginType(mtGlobalCtx, bnswap.UpdateMarginTypeParams{
				Symbol:     takerSymbol,
				MarginType: *mtConfig.MarginType,
			})
			if err != nil {
				logger.Debugf("UPDATE MARGIN TYPE FOR %s ERROR %v", takerSymbol, err)
			} else {
				logger.Debugf("UPDATE MARGIN TYPE FOR %s RESPONSE %v", takerSymbol, res)
			}
			time.Sleep(time.Second)
		}
	}
	_, mMultipliers, mTickSizes, _, err = kcperp.GetOrderLimits(mtGlobalCtx, mAPI, mSymbols)
	if err != nil {
		logger.Debugf("kcperp.GetOrderLimits error %v", err)
		return
	}
	tTickSizes, tStepSizes, _, tMinNotional, _, _, err = bnswap.GetOrderLimits(mtGlobalCtx, tAPI, tSymbols)
	if err != nil {
		logger.Debugf("bnswap.GetOrderLimits error %v", err)
		return
	}

	for makerSymbol, makerStepSize := range mMultipliers {
		if takerStepSize, ok := tStepSizes[mtSymbolsMap[makerSymbol]]; !ok {
			logger.Debugf("TAKER STEP SIZE NOT EXISTS FOR MAKER %s - %s", makerSymbol, mtSymbolsMap[makerSymbol])
			return
		} else {
			mtStepSizes[makerSymbol] = common.MergedStepSize(makerStepSize, takerStepSize)
		}
	}
	logger.Debugf("MERGED STEP SIZES: %v", mtStepSizes)

	mtInfluxWriter, err = common.NewInfluxWriter(
		*mtConfig.InternalInflux.Address,
		*mtConfig.InternalInflux.Username,
		*mtConfig.InternalInflux.Password,
		*mtConfig.InternalInflux.Database,
		*mtConfig.InternalInflux.BatchSize,
	)
	if err != nil {
		logger.Debugf("common.NewInfluxWriter error %v", err)
		return
	}
	defer func() {
		err := mtInfluxWriter.Stop()
		if err != nil {
			logger.Warnf("stop influx writer error %v", err)
		}
	}()

	mtExternalInfluxWriter, err = common.NewInfluxWriter(
		*mtConfig.ExternalInflux.Address,
		*mtConfig.ExternalInflux.Username,
		*mtConfig.ExternalInflux.Password,
		*mtConfig.ExternalInflux.Database,
		*mtConfig.ExternalInflux.BatchSize,
	)
	if err != nil {
		logger.Debugf("common.NewInfluxWriter error %v", err)
		return
	}
	defer func() {
		err := mtExternalInfluxWriter.Stop()
		if err != nil {
			logger.Warnf("stop influx writer error %v", err)
		}
	}()

	tUserWebsocket, err = bnswap.NewUserWebsocket(
		mtGlobalCtx,
		tAPI,
		*mtConfig.ProxyAddress,
	)
	if err != nil {
		logger.Debugf("bnswap.NewUserWebsocket error %v", err)
		return
	}
	defer tUserWebsocket.Stop()

	mUserWebsocket = kcperp.NewUserWebsocket(
		mtGlobalCtx,
		mAPI,
		mSymbols,
		*mtConfig.ProxyAddress,
	)
	defer mUserWebsocket.Stop()

	influxSaveTimer := time.NewTimer(
		time.Now().Truncate(
			*mtConfig.InternalInflux.SaveInterval,
		).Add(
			*mtConfig.InternalInflux.SaveInterval * 3,
		).Sub(time.Now()),
	)
	externalInfluxSaveTimer := time.NewTimer(
		time.Now().Truncate(
			*mtConfig.ExternalInflux.SaveInterval,
		).Add(
			*mtConfig.ExternalInflux.SaveInterval * 3,
		).Sub(time.Now()),
	)
	mtLoopTimer = time.NewTimer(time.Second) //先等1分钟
	defer influxSaveTimer.Stop()
	defer mtLoopTimer.Stop()
	defer externalInfluxSaveTimer.Stop()

	go kcperp.PositionsHttpLoop(
		mtGlobalCtx, mAPI,
		mSymbols, *mtConfig.PullInterval,
		mPositionCh,
	)
	go kcperp.AccountHttpLoop(
		mtGlobalCtx, mAPI,
		kcperp.AccountParam{Currency: "USDT"},
		*mtConfig.PullInterval, mAccountCh,
	)
	go kcperp.FundingRateLoop(
		mtGlobalCtx, mAPI,
		mSymbols,
		*mtConfig.PullInterval*10,
		mFundingRatesCh,
	)
	go bnswap.WatchAccountFromHttp(
		mtGlobalCtx, tAPI,
		*mtConfig.PullInterval, tAccountCh,
	)
	go bnswap.WatchPositionsFromHttp(
		mtGlobalCtx, tAPI,
		tSymbols,
		*mtConfig.PullInterval, tPositionsCh,
	)
	go bnswap.WatchPremiumIndexesFromHttp(
		mtGlobalCtx, tAPI,
		tSymbols,
		*mtConfig.PullInterval*10, tPremiumIndexesCh,
	)

	go watchMakerBars(
		mtGlobalCtx,
		mAPI,
		mSymbols,
		*mtConfig.BarsLookback,
		*mtConfig.PullBarsInterval,
		*mtConfig.PullBarsRetryInterval,
		*mtConfig.RequestInterval,
		mBarsMapCh,
	)

	go watchTakerBars(
		mtGlobalCtx,
		tAPI,
		tSymbols,
		*mtConfig.BarsLookback,
		*mtConfig.PullBarsInterval,
		*mtConfig.PullBarsRetryInterval,
		*mtConfig.RequestInterval,
		tBarsMapCh,
	)

	go watchDeltaQuantile(
		mtGlobalCtx,
		mSymbols,
		mtSymbolsMap,
		*mtConfig.BotQuantile,
		*mtConfig.TopQuantile,
		*mtConfig.MinimalEnterDelta,
		*mtConfig.MaximalExitDelta,
		*mtConfig.MinimalBandOffset,
		mtBarsMapCh,
		mtQuantilesCh,
	)
	spreadReportCh := make(chan common.SpreadReport, 10000)
	go reportsSaveLoop(
		mtGlobalCtx,
		mtInfluxWriter,
		*mtConfig.InternalInflux,
		spreadReportCh,
	)

	makerRowDepthChs := make(map[string]chan *common.DepthRawMessage)
	for start := 0; start < len(mSymbols); start += *mtConfig.OrderBookBatchSize {
		end := start + *mtConfig.OrderBookBatchSize
		if end > len(mSymbols) {
			end = len(mSymbols)
		}
		subMakerRowDepthChs := make(map[string]chan *common.DepthRawMessage)
		for _, symbol := range mSymbols[start:end] {
			makerRowDepthChs[symbol] = make(chan *common.DepthRawMessage, 100)
			subMakerRowDepthChs[symbol] = makerRowDepthChs[symbol]
		}
		go makerRoutedDepthLoop(
			mtGlobalCtx,
			mtGlobalCancel,
			mAPI,
			*mtConfig.ProxyAddress,
			subMakerRowDepthChs,
		)
	}

	takerRowDepthChs := make(map[string]chan *common.DepthRawMessage)
	for start := 0; start < len(tSymbols); start += *mtConfig.OrderBookBatchSize {
		end := start + *mtConfig.OrderBookBatchSize
		if end > len(tSymbols) {
			end = len(tSymbols)
		}
		subTakerRowDepthChs := make(map[string]chan *common.DepthRawMessage)
		for _, symbol := range tSymbols[start:end] {
			takerRowDepthChs[symbol] = make(chan *common.DepthRawMessage, 100)
			subTakerRowDepthChs[symbol] = takerRowDepthChs[symbol]
		}
		go takerRoutedDepthLoop(
			mtGlobalCtx,
			mtGlobalCancel,
			*mtConfig.ProxyAddress,
			subTakerRowDepthChs,
		)
	}

	spreadCh := make(chan *common.MakerTakerSpread, len(mSymbols)*100)
	for makerSymbol, takerSymbol := range mtConfig.MakerTakerSymbolsMap {
		go watchMakerTakerSpread(
			mtGlobalCtx,
			makerSymbol, takerSymbol,
			mMultipliers[makerSymbol],
			*mtConfig.OrderBookMakerImpact,
			*mtConfig.OrderBookTakerImpact,
			*mtConfig.OrderBookMakerDecay,
			*mtConfig.OrderBookMakerBias,
			*mtConfig.OrderBookTakerDecay,
			*mtConfig.OrderBookTakerBias,
			*mtConfig.OrderBookMaxAgeDiffBias,
			*mtConfig.ReportCount,
			*mtConfig.SpreadLookbackDuration,
			*mtConfig.SpreadLookbackMinimalWindow,
			makerRowDepthChs[makerSymbol],
			takerRowDepthChs[takerSymbol],
			spreadReportCh,
			spreadCh,
		)
	}

	mNewOrderErrorCh = make(chan MakerOrderNewError, len(mSymbols)*2)
	for _, makerSymbol := range mSymbols {
		mOrderRequestChs[makerSymbol] = make(chan MakerOrderRequest, 2)
		go watchMakerOrderRequest(
			mtGlobalCtx,
			mAPI,
			*mtConfig.OrderTimeout,
			*mtConfig.DryRun,
			mOrderRequestChs[makerSymbol],
			mOpenOrderCh,
			mNewOrderErrorCh,
		)
	}

	tNewOrderErrorCh = make(chan TakerOrderNewError, len(mSymbols)*2)
	for _, takerSymbol := range tSymbols {
		tOrderRequestChs[takerSymbol] = make(chan TakerOrderRequest, 2)
		go watchTakerOrderRequest(
			mtGlobalCtx,
			tAPI,
			*mtConfig.OrderTimeout,
			*mtConfig.DryRun,
			tOrderRequestChs[takerSymbol],
			tOpenOrderCh,
			tNewOrderErrorCh,
		)
	}

	go kcperp.WatchSystemStatusHttp(
		mtGlobalCtx,
		mAPI,
		*mtConfig.PullInterval/2,
		mSystemStatusCh,
	)

	go bnswap.SystemStatusLoop(
		mtGlobalCtx,
		tAPI,
		*mtConfig.PullInterval/2,
		tSystemStatusCh,
	)

	if *mtConfig.CpuProfile != "" {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			sig := <-sigs
			logger.Debugf("CATCH EXIT SIGNAL %v", sig)
			mtGlobalCancel()
		}()
	}

	go func() {
		for _, makerSymbol := range mSymbols {
			select {
			case <-mtGlobalCtx.Done():
				return
			case <-time.After(*mtConfig.RequestInterval):
				logger.Debugf("INITIAL CANCEL ALL %s", makerSymbol)
				select {
				case <-mtGlobalCtx.Done():
					return
				case mOrderRequestChs[makerSymbol] <- MakerOrderRequest{
					Cancel: &kcperp.CancelAllOrdersParam{
						Symbol: makerSymbol,
					},
				}:
				}
				select {
				case <-mtGlobalCtx.Done():
					return
				case tOrderRequestChs[makerSymbol] <- TakerOrderRequest{
					Cancel: &bnswap.CancelAllOrderParams{
						Symbol: mtSymbolsMap[makerSymbol],
					},
				}:
				}
			}
		}
	}()

	logger.Debugf("START MAIN LOOP")
	for {
		select {
		case <-mtGlobalCtx.Done():
			logger.Debugf("GLOBAL CTX DONE, EXIT MAIN LOOP")
			return
		case <-mUserWebsocket.Done():
			logger.Debugf("MAKER USER WS DONE, EXIT MAIN LOOP")
			return
		case <-tUserWebsocket.Done():
			logger.Debugf("MAKER USER WS DONE, EXIT MAIN LOOP")
			return
		case mSystemReady = <-mSystemStatusCh:
			if !mSystemReady {
				mtGlobalSilent = time.Now().Add(*mtConfig.RestartSilent)
			}
			break
		case tSystemReady = <-tSystemStatusCh:
			if !tSystemReady {
				mtGlobalSilent = time.Now().Add(*mtConfig.RestartSilent)
			}
			break
		case <-mUserWebsocket.RestartCh:
			logger.Debugf("<-mUserWebsocket.RestartCh restart silent %v", *mtConfig.RestartSilent)
			mtGlobalSilent = time.Now().Add(*mtConfig.RestartSilent)
		case <-tUserWebsocket.RestartCh:
			logger.Debugf("<-tUserWebsocket.RestartCh restart silent %v", *mtConfig.RestartSilent)
			mtGlobalSilent = time.Now().Add(*mtConfig.RestartSilent)
			break
		case ps := <-mPositionCh:
			handleMakerHttpPositions(ps)
			break
		case account := <-mAccountCh:
			handleMakerHttpAccount(account)
			break
		case account := <-tAccountCh:
			handleTakerHttpAccount(account)
			break
		case ps := <-tPositionsCh:
			handleTakerHttpPositions(ps)
			break
		case msg := <-tUserWebsocket.BalanceAndPositionUpdateEventCh:
			handleTakerWSAccount(msg)
			break
		case msg := <-mUserWebsocket.PositionCh:
			handleMakerWSPosition(msg)
			break
		case msg := <-mUserWebsocket.BalanceCh:
			handleMakerWSAccount(msg)
			break
		case makerOrder := <-mUserWebsocket.OrderCh:
			if makerOrder.Type == kcperp.OrderTypeCanceled ||
				makerOrder.Type == kcperp.OrderTypeMatch {
				if openOrder, ok := mOpenOrders[makerOrder.Symbol]; ok && openOrder.ClientOid == makerOrder.ClientOid {
					delete(mOpenOrders, makerOrder.Symbol)
				}
				if makerOrder.Type == kcperp.OrderTypeCanceled {
					mOrderSilentTimes[makerOrder.Symbol] = time.Now().Add(time.Second)
					mPositionsUpdateTimes[makerOrder.Symbol] = time.Unix(0, 0)
				} else {
					logger.Debugf(
						"MAKER ORDER FILLED %s SIDE %s TRADE SIZE %v TRADE PRICE %f",
						makerOrder.Symbol, makerOrder.Side, makerOrder.MatchSize, makerOrder.MatchPrice,
					)
					if takerSymbol, ok := mtSymbolsMap[makerOrder.Symbol]; ok {
						mtLimitHedgeTimeouts[takerSymbol] = time.Now().Add(*mtConfig.HedgeTimeout)
					}
					mtLimitHedgeTimeouts[mtSymbolsMap[makerOrder.Symbol]] = time.Now().Add(*mtConfig.HedgeTimeout)
					tOrderSilentTimes[mtSymbolsMap[makerOrder.Symbol]] = time.Now()
					mtLoopTimer.Reset(time.Nanosecond)
					mHttpPositionUpdateSilentTimes[makerOrder.Symbol] = time.Now().Add(*mtConfig.HttpSilent)
					if makerOrder.Side == kcperp.OrderSideSell {
						mLastFilledSellPrices[makerOrder.Symbol] = makerOrder.MatchPrice
					} else if makerOrder.Side == kcperp.OrderSideBuy {
						mLastFilledBuyPrices[makerOrder.Symbol] = makerOrder.MatchPrice
					}
				}
			}
			break
		case takerOrderEvent := <-tUserWebsocket.OrderUpdateEventCh:
			takerOrder := takerOrderEvent.Order
			if takerOrder.Status == "REJECTED" || takerOrder.Status == "EXPIRED" {
				tOrderSilentTimes[takerOrder.Symbol] = time.Now()
				delete(tOpenOrders, takerOrder.Symbol)
			} else if takerOrder.Status == "FILLED" {
				delete(tOpenOrders, takerOrder.Symbol)
				logger.Debugf("TAKER FILLED ORDER %s %s %f %f", takerOrder.Symbol, takerOrder.Status, takerOrder.FilledAccumulatedQuantity, takerOrder.AveragePrice)
				tHttpPositionUpdateSilentTimes[takerOrder.Symbol] = time.Now().Add(*mtConfig.HttpSilent)
				if makerSymbol, ok := tmSymbolsMap[takerOrder.Symbol]; ok {
					if takerOrder.Side == common.OrderSideSell {
						if makerPrice, ok := mLastFilledBuyPrices[makerSymbol]; ok {
							mtRealisedSpread[makerSymbol] = (takerOrder.AveragePrice - makerPrice) / makerPrice
							if takerOrder.ReduceOnly {
								logger.Debugf("%s REALISED CLOSE LONG SPREAD %f", makerSymbol, mtRealisedSpread[makerSymbol])
							} else {
								logger.Debugf("%s REALISED OPEN SHORT SPREAD %f", makerSymbol, mtRealisedSpread[makerSymbol])
							}
						}
					} else if takerOrder.Side == common.OrderSideBuy {
						if makerPrice, ok := mLastFilledSellPrices[makerSymbol]; ok {
							mtRealisedSpread[makerSymbol] = (takerOrder.AveragePrice - makerPrice) / makerPrice
							if takerOrder.ReduceOnly {
								logger.Debugf("%s REALISED CLOSE SHORT SPREAD %f", makerSymbol, mtRealisedSpread[makerSymbol])
							} else {
								logger.Debugf("%s REALISED OPEN LONG SPREAD %f", makerSymbol, mtRealisedSpread[makerSymbol])
							}
						}
					}
				}
			}
			break
		case spread := <-spreadCh:
			mtSpreads[spread.MakerSymbol] = spread
			break
		case fr := <-mFundingRatesCh:
			mFundingRates[fr.Symbol] = fr
			handleUpdateFundingRates()
			break
		case tPremiumIndexes = <-tPremiumIndexesCh:
			handleUpdateFundingRates()
			break
		case mBarsMap = <-mBarsMapCh:
			if mtMapUpdated[TakerName] {
				mtBarsMapCh <- [2]common.KLinesMap{mBarsMap, tBarsMap}
				mtMapUpdated[TakerName] = false
				mtMapUpdated[MakerName] = false
			} else {
				mtMapUpdated[MakerName] = true
			}
			break
		case tBarsMap = <-tBarsMapCh:
			if mtMapUpdated[MakerName] {
				mtBarsMapCh <- [2]common.KLinesMap{mBarsMap, tBarsMap}
				mtMapUpdated[MakerName] = false
				mtMapUpdated[TakerName] = false
			} else {
				mtMapUpdated[TakerName] = true
			}
			break
		case qs := <-mtQuantilesCh:
			if mtQuantiles == nil {
				logger.Debugf("QUANTILES %v", qs)
			}
			mtQuantiles = qs
			break
		case <-influxSaveTimer.C:
			handleSave()
			influxSaveTimer.Reset(
				time.Now().Truncate(
					*mtConfig.InternalInflux.SaveInterval,
				).Add(
					*mtConfig.InternalInflux.SaveInterval,
				).Sub(time.Now()),
			)
			break
		case <-externalInfluxSaveTimer.C:
			handleExternalInfluxSave()
			externalInfluxSaveTimer.Reset(
				time.Now().Truncate(
					*mtConfig.ExternalInflux.SaveInterval,
				).Add(
					*mtConfig.ExternalInflux.SaveInterval,
				).Sub(time.Now()),
			)
			break
		case takerNewError := <-tNewOrderErrorCh:
			tOrderSilentTimes[takerNewError.Params.Symbol] = time.Now().Add(*mtConfig.OrderSilent * 5)
			break
		case makerNewError := <-mNewOrderErrorCh:
			mOrderSilentTimes[makerNewError.Params.Symbol] = time.Now().Add(*mtConfig.OrderSilent * 5)
			break
		case makerOpenOrder := <-mOpenOrderCh:
			if openOrder, ok := mOpenOrders[makerOpenOrder.Symbol]; ok {
				if makerOpenOrder.NewOrderParam == nil && openOrder.ResponseOrderID == makerOpenOrder.ResponseOrderID {
					//Cancel的Http回报
					delete(mOpenOrders, makerOpenOrder.Symbol)
				} else if makerOpenOrder.NewOrderParam != nil && openOrder.ClientOid == makerOpenOrder.ClientOid {
					//New的Http回报
					mOpenOrders[makerOpenOrder.Symbol] = makerOpenOrder
				}
			}
		case takerOpenOrder := <-tOpenOrderCh:
			if openOrder, ok := tOpenOrders[takerOpenOrder.Symbol]; ok {
				if takerOpenOrder.NewOrderParams == nil {
					//Cancel的Http回报
					delete(tOpenOrders, takerOpenOrder.Symbol)
				} else if takerOpenOrder.NewOrderParams != nil && openOrder.NewClientOrderId == takerOpenOrder.NewClientOrderId {
					//New的Http回报
					tOpenOrders[takerOpenOrder.Symbol] = takerOpenOrder
				}
			}
		case <-mtLoopTimer.C:
			if mSystemReady && tSystemReady && time.Now().Sub(mtGlobalSilent) > 0 {
				updateTakerOldOrders()
				updateTakerOrders()
				updateMakerOldOrders()
				updateMakerNewOrders()
			} else {
				if len(mOpenOrders) > 0 {
					for makerSymbol := range mOpenOrders {
						select {
						case mOrderRequestChs[makerSymbol] <- MakerOrderRequest{
							Cancel: &kcperp.CancelAllOrdersParam{
								Symbol: makerSymbol,
							},
						}:
							delete(mOpenOrders, makerSymbol)
						default:
						}
					}
				}
			}
			mtLoopTimer.Reset(
				time.Now().Truncate(
					*mtConfig.LoopInterval,
				).Add(
					*mtConfig.LoopInterval,
				).Sub(time.Now()),
			)
			break
		}
	}
}
