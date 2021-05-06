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


	if *mtConfig.ChangeAutoDepositStatus {
		for _, symbol := range mSymbols {
			res, err := mAPI.ChangeAutoDepositStatus(mtGlobalCtx, kcperp.AutoDepositStatusParam{
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
	_, tStepSizes, _, tMinNotional, _, _, err = bnswap.GetOrderLimits(mtGlobalCtx, tAPI, tSymbols)
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
		mtGlobalCtx,
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
	defer mtInfluxWriter.Stop()

	mtExternalInfluxWriter, err = common.NewInfluxWriter(
		mtGlobalCtx,
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
	defer mtExternalInfluxWriter.Stop()

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
		tOrderRequestChs[takerSymbol] = make(chan bnswap.NewOrderParams, 2)
		go watchTakerOrderRequest(
			mtGlobalCtx,
			tAPI,
			*mtConfig.OrderTimeout,
			*mtConfig.DryRun,
			tOrderRequestChs[takerSymbol],
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
			}
		}
	}()

	logger.Debugf("START MAIN LOOP")
	fundingInterval := time.Hour * 8
	fundingSilent := time.Minute * 5
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
			//logger.Debugf("mSystemStatusCh %v", mSystemReady)
			if !mSystemReady {
				mtGlobalSilent = time.Now().Add(*mtConfig.RestartSilent)
			}
			break
		case tSystemReady = <-tSystemStatusCh:
			//logger.Debugf("tSystemStatusCh %v", tSystemReady)
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
					logger.Debugf("MAKER WS ORDER CANCELED %v ", makerOrder)
					mOrderSilentTimes[makerOrder.Symbol] = time.Now().Add(time.Second)
					mPositionsUpdateTimes[makerOrder.Symbol] = time.Unix(0, 0)
				} else {
					logger.Debugf(
						"MAKER WS ORDER FILLED %s SIDE %s TRADE SIZE %v TRADE PRICE %f",
						makerOrder.Symbol, makerOrder.Side, makerOrder.MatchSize, makerOrder.MatchPrice,
					)
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
				logger.Debugf("TAKER WS ORDER %s %s", takerOrder.Symbol, takerOrder.Status)
				tOrderSilentTimes[takerOrder.Symbol] = time.Now().Add(time.Second)
				tPositionsUpdateTimes[takerOrder.Symbol] = time.Unix(0, 0)
			} else if takerOrder.Status == "FILLED" {
				logger.Debugf("TAKER WS ORDER %s %s %f %f", takerOrder.Symbol, takerOrder.Status, takerOrder.FilledAccumulatedQuantity, takerOrder.AveragePrice)
				tHttpPositionUpdateSilentTimes[takerOrder.Symbol] = time.Now().Add(*mtConfig.HttpSilent)
				if makerSymbol, ok := tmSymbolsMap[takerOrder.Symbol]; ok {
					if takerOrder.Side == common.OrderSideSell {
						if makerPrice, ok := mLastFilledBuyPrices[makerSymbol]; ok {
							mtRealisedSpread[makerSymbol] = (takerOrder.AveragePrice - makerPrice) / makerPrice
							logger.Debugf("%s REALISED SHORT SPREAD %f", makerSymbol, mtRealisedSpread[makerSymbol])
						}
					} else if takerOrder.Side == common.OrderSideBuy {
						if makerPrice, ok := mLastFilledSellPrices[makerSymbol]; ok {
							mtRealisedSpread[makerSymbol] = (takerOrder.AveragePrice - makerPrice) / makerPrice
							logger.Debugf("%s REALISED LONG SPREAD %f", makerSymbol, mtRealisedSpread[makerSymbol])
						}
					}
				}
			}
			break
		case spread := <-spreadCh:
			mtSpreads[spread.MakerSymbol] = spread
			//mtLoopTimer.Reset(time.Millisecond)
			break
		case fr := <-mFundingRatesCh:
			mFundingRates[fr.Symbol] = fr
			handleUpdateFundingRates()
			break
		case tPremiumIndexes = <-tPremiumIndexesCh:
			//logger.Debugf("%v", tPremiumIndexes)
			handleUpdateFundingRates()
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
		case <-mtLoopTimer.C:
			if mSystemReady && tSystemReady && time.Now().Sub(mtGlobalSilent) > 0 {
				updateTakerPositions()
				if time.Now().Sub(time.Now().Truncate(fundingInterval)) > fundingSilent &&
					time.Now().Truncate(fundingInterval).Add(fundingInterval).Sub(time.Now()) > fundingSilent {
					updateMakerOldOrders()
					updateMakerNewOrders()
				} else {
					cancelAllMakerOpenOrders()
				}
			} else {
				if time.Now().Sub(time.Now().Truncate(time.Second*15)) < *mtConfig.LoopInterval {
					logger.Debugf(
						"SYSTEM NOT READY mSystemReady %v tSystemReady %v mtGlobalSilent %v",
						mSystemReady, tSystemReady, time.Now().Sub(mtGlobalSilent),
					)
				}
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
