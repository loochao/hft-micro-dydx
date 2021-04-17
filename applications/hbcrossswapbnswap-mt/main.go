package main

import (
	"context"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/hbcrossswap"
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
			logger.Fatal(err)
		}
		err = pprof.StartCPUProfile(f)
		if err != nil {
			logger.Fatal(err)
		}
		defer pprof.StopCPUProfile()
	}

	var err error
	mAPI, err = hbcrossswap.NewAPI(
		*mtConfig.HbApiKey,
		*mtConfig.HbApiSecret,
		*mtConfig.ProxyAddress,
	)
	if err != nil {
		logger.Fatal(err)
	}
	tAPI, err = bnswap.NewAPI(
		&common.Credentials{
			Key:    *mtConfig.BnApiKey,
			Secret: *mtConfig.BnApiSecret,
		},
		*mtConfig.ProxyAddress,
	)
	if err != nil {
		logger.Fatal(err)
	}

	mtGlobalCtx, mtGlobalCancel = context.WithCancel(context.Background())
	defer mtGlobalCancel()

	mTickSizes, mContractSizes, err = hbcrossswap.GetOrderLimits(mtGlobalCtx, mAPI, mSymbols)
	if err != nil {
		logger.Fatal(err)
	}
	tTickSizes, tStepSizes, _, tMinNotional, _, _, err = bnswap.GetOrderLimits(mtGlobalCtx, tAPI, tSymbols)
	if err != nil {
		logger.Fatal(err)
	}

	for makerSymbol, makerStepSize := range mContractSizes {
		if takerStepSize, ok := tStepSizes[mtSymbolsMap[makerSymbol]]; !ok {
			logger.Fatalf("TAKER STEP SIZE NOT EXISTS FOR MAKER %s - %s", makerSymbol, mtSymbolsMap[makerSymbol])
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
		logger.Fatal(err)
	}

	mtExternalInfluxWriter, err = common.NewInfluxWriter(
		*mtConfig.ExternalInflux.Address,
		*mtConfig.ExternalInflux.Username,
		*mtConfig.ExternalInflux.Password,
		*mtConfig.ExternalInflux.Database,
		*mtConfig.ExternalInflux.BatchSize,
	)
	if err != nil {
		logger.Fatal(err)
	}

	defer func() {
		err := mtInfluxWriter.Stop()
		if err != nil {
			logger.Warnf("stop influx writer error %v", err)
		}
	}()

	tUserWebsocket = bnswap.NewUserWebsocket(
		mtGlobalCtx,
		tAPI,
		*mtConfig.ProxyAddress,
	)
	defer tUserWebsocket.Stop()

	mUserWebsocket = hbcrossswap.NewUserWebsocket(
		mtGlobalCtx,
		*mtConfig.HbApiKey,
		*mtConfig.HbApiSecret,
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

	go hbcrossswap.WatchPositionsFromHttp(
		mtGlobalCtx, mAPI,
		tSymbols, *mtConfig.PullInterval,
		mPositionCh,
	)
	go hbcrossswap.WatchAccountFromHttp(
		mtGlobalCtx, mAPI,
		*mtConfig.PullInterval, mAccountCh,
	)
	go hbcrossswap.WatchFundingRate(
		mtGlobalCtx, mAPI,
		tSymbols,
		*mtConfig.PullInterval*10,
		mFundingRatesCh,
	)
	go bnswap.WatchAccountFromHttp(
		mtGlobalCtx, tAPI,
		*mtConfig.PullInterval, bAccountCh,
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

	go watchHBars(
		mtGlobalCtx,
		mAPI,
		tSymbols,
		*mtConfig.BarsLookback,
		*mtConfig.PullBarsInterval,
		*mtConfig.PullBarsRetryInterval,
		*mtConfig.RequestInterval,
		mBarsMapCh,
	)

	go watchBBars(
		mtGlobalCtx,
		tAPI,
		mSymbols,
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
		*mtConfig.TopBandScale,
		*mtConfig.BotBandScale,
		*mtConfig.MinimalEnterDelta,
		*mtConfig.MaximalExitDelta,
		*mtConfig.MinimalBandOffset,
		mtBarsMapCh,
		hbQuantilesCh,
	)

	walkedOrderBookCh := make(chan WalkedOrderBook, len(mSymbols)*10)
	for start := 0; start < len(tSymbols); start += *mtConfig.OrderBookBatchSize {
		end := start + *mtConfig.OrderBookBatchSize
		if end > len(tSymbols) {
			end = len(tSymbols)
		}
		go watchBWalkedOrderBooks(
			mtGlobalCtx,
			*mtConfig.ProxyAddress,
			*mtConfig.OrderBookTakerImpact,
			*mtConfig.OrderBookMakerImpact,
			tSymbols[start:end],
			walkedOrderBookCh,
		)
	}

	for start := 0; start < len(mSymbols); start += *mtConfig.OrderBookBatchSize {
		end := start + *mtConfig.OrderBookBatchSize
		if end > len(mSymbols) {
			end = len(mSymbols)
		}
		go watchHWalkedOrderBooks(
			mtGlobalCtx,
			*mtConfig.ProxyAddress,
			mContractSizes,
			*mtConfig.OrderBookTakerImpact,
			*mtConfig.OrderBookMakerImpact,
			mSymbols[start:end],
			walkedOrderBookCh,
		)
	}

	spreadCh := make(chan Spread, len(mSymbols)*100)
	go watchSpread(
		mtGlobalCtx,
		mSymbols,
		mtSymbolsMap,
		*mtConfig.OrderBookMaxAgeDiff,
		*mtConfig.OrderBookMaxAge,
		*mtConfig.SpreadLookbackDuration,
		*mtConfig.SpreadLookbackMinimalWindow,
		walkedOrderBookCh,
		spreadCh,
	)

	mNewOrderErrorCh = make(chan HOrderNewError, len(mSymbols)*2)
	for _, makerSymbol := range mSymbols {
		mOrderRequestChs[makerSymbol] = make(chan MakerOrderRequest, 2)
		go watchMakerOrderRequest(
			mtGlobalCtx,
			mAPI,
			*mtConfig.OrderTimeout,
			*mtConfig.DryRun,
			mOrderRequestChs[makerSymbol],
			mNewOrderErrorCh,
		)
		mOrderRequestChs[makerSymbol] <- MakerOrderRequest{
			Cancel: &hbcrossswap.CancelAllParam{Symbol: makerSymbol},
		}
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

	done := make(chan bool, 1)
	if *mtConfig.CpuProfile != "" {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			sig := <-sigs
			logger.Debugf("Exit with sig %d, clean *.tmp files", sig)
			done <- true
		}()
	}

	logger.Debugf("START")

	for {
		select {
		case <-done:
			logger.Debugf("Exit")
			return
		case <-mUserWebsocket.RestartCh:
			logger.Debugf("mUserWebsocket restart silent %v", *mtConfig.RestartSilent)
			handleRestartSilent()
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
			handleWSPosition(msg)
			break
		case msg := <-mUserWebsocket.AccountCh:
			handleWSAccount(msg)
			break
		case makerOrder := <-mUserWebsocket.OrderCh:
			if makerOrder.Status == hbcrossswap.OrderStatusFilled ||
				makerOrder.Status == hbcrossswap.OrderStatusCancelled ||
				makerOrder.Status == hbcrossswap.OrderStatusPartiallyFilledButCancelledByClient {
				if makerOrder.Status == hbcrossswap.OrderStatusCancelled {
					logger.Debugf("MAKER WS ORDER CANCELED %v ", makerOrder)
					mOrderSilentTimes[makerOrder.Symbol] = time.Now().Add(time.Second)
					mPositionsUpdateTimes[makerOrder.Symbol] = time.Unix(0, 0)
				} else {
					logger.Debugf(
						"MAKER WS ORDER FILLED %s SIDE %s TRADE SIZE %v TRADE PRICE %f",
						makerOrder.Symbol, makerOrder.Direction, makerOrder.TradeVolume, makerOrder.TradeAvgPrice,
					)
					if makerOrder.Direction == hbcrossswap.OrderDirectionSell {
						mLastFilledSellPrices[makerOrder.Symbol] = makerOrder.TradeAvgPrice
					} else if makerOrder.Direction == hbcrossswap.OrderDirectionBuy {
						mLastFilledBuyPrices[makerOrder.Symbol] = makerOrder.TradeAvgPrice
					}
				}
				if openOrder, ok := mOpenOrders[makerOrder.Symbol]; ok && openOrder.ClientOrderID == makerOrder.ClientOrderID {
					delete(mOpenOrders, makerOrder.Symbol)
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
				if makerSymbol, ok := tmSymbolsMap[takerOrder.Symbol]; ok {
					if takerOrder.Side == common.OrderSideSell {
						if makerPrice, ok := mLastFilledBuyPrices[makerSymbol]; ok {
							hbRealisedSpread[makerSymbol] = (takerOrder.AveragePrice - makerPrice) / makerPrice
							logger.Debugf("%s REALISED OPEN SPREAD %f", makerSymbol, hbRealisedSpread[makerSymbol])
						}
					} else if takerOrder.Side == common.OrderSideBuy {
						if makerPrice, ok := mLastFilledSellPrices[makerSymbol]; ok {
							hbRealisedSpread[makerSymbol] = (takerOrder.AveragePrice - makerPrice) / makerPrice
							logger.Debugf("%s REALISED CLOSE SPREAD %f", makerSymbol, hbRealisedSpread[makerSymbol])
						}
					}
				}
			}
			break
		case spread := <-spreadCh:
			mtSpreads[spread.HSymbol] = spread
			mtLoopTimer.Reset(time.Millisecond)
			break
		case mFundingRates = <-mFundingRatesCh:
			handleUpdateTradeDirections()
			break
		case tPremiumIndexes = <-tPremiumIndexesCh:
			handleUpdateTradeDirections()
			break
		case mBarsMap = <-mBarsMapCh:
			if mtMapUpdated[WalkedOrderBookTypeTaker] {
				mtBarsMapCh <- [2]common.KLinesMap{mBarsMap, tBarsMap}
				mtMapUpdated[WalkedOrderBookTypeMaker] = false
				mtMapUpdated[WalkedOrderBookTypeTaker] = false
			} else {
				mtMapUpdated[WalkedOrderBookTypeMaker] = true
			}
			break
		case tBarsMap = <-tBarsMapCh:
			if mtMapUpdated[WalkedOrderBookTypeMaker] {
				mtBarsMapCh <- [2]common.KLinesMap{mBarsMap, tBarsMap}
				mtMapUpdated[WalkedOrderBookTypeMaker] = false
				mtMapUpdated[WalkedOrderBookTypeTaker] = false
			} else {
				mtMapUpdated[WalkedOrderBookTypeTaker] = true
			}
			break
		case qs := <-hbQuantilesCh:
			if mtQuantiles == nil {
				logger.Debugf("QUANTILES %v", qs)
			}
			mtQuantiles = qs
			mtLoopTimer.Reset(time.Millisecond)
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
			mOrderSilentTimes[takerNewError.Params.Symbol] = time.Now().Add(*mtConfig.OrderSilent * 5)
			break

		case makerNewError := <-mNewOrderErrorCh:
			if openOrder, ok := mOpenOrders[makerNewError.Params.Symbol]; ok && openOrder.ClientOrderID == makerNewError.Params.ClientOrderID {
				delete(mOpenOrders, makerNewError.Params.Symbol)
			}
			tOrderSilentTimes[makerNewError.Params.Symbol] = time.Now().Add(*mtConfig.OrderSilent * 5)
		case <-mtLoopTimer.C:
			updateTakerPositions()
			updateMakerOldOrders()
			updateMakerNewOrders()
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
