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
	mAPI, err = hbcrossswap.NewAPI(
		*mtConfig.HbApiKey,
		*mtConfig.HbApiSecret,
		*mtConfig.ProxyAddress,
	)
	if err != nil {
		logger.Debugf("hbcrossswap.NewAPI error %v", err)
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

	//totalDiff := int64(0)
	//requestTime := int64(0)
	//for i := 0; i < 10; i++ {
	//	start := time.Now()
	//	tt, err := tAPI.GetServerTime(context.Background())
	//	if err != nil {
	//		logger.Debugf("bnswap.GetServerTime error %v", err)
	//		return
	//	}
	//	requestTime += time.Now().Sub(start).Milliseconds()
	//	totalDiff += tt.ServerTime - time.Now().UnixNano()/1000000
	//	time.Sleep(time.Second)
	//}
	//logger.Debugf("TAKER ROUTE TIME %d SEVER TIME DIFF %d WITH HALF ROUTE %d", requestTime/10, totalDiff/10, totalDiff/10+requestTime/20)
	//
	//totalDiff = int64(0)
	//requestTime = int64(0)
	//for i := 0; i < 10; i++ {
	//	start := time.Now()
	//	tt, err := mAPI.GetHeartbeat(context.Background())
	//	if err != nil {
	//		logger.Debugf("bnswap.GetServerTime error %v", err)
	//		return
	//	}
	//	requestTime += time.Now().Sub(start).Milliseconds()
	//	totalDiff += tt.Timestamp.Sub(time.Now()).Milliseconds()
	//	time.Sleep(time.Second)
	//}
	//logger.Debugf("MAKER ROUTE TIME %d SEVER TIME DIFF %d WITH HALF ROUTE %d", requestTime/10, totalDiff/10, totalDiff/10+requestTime/20)

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

	mTickSizes, mContractSizes, err = hbcrossswap.GetOrderLimits(mtGlobalCtx, mAPI, mSymbols)
	if err != nil {
		logger.Debugf("hbcrossswap.GetOrderLimits error %v", err)
		return
	}
	tTickSizes, tStepSizes, _, tMinNotional, _, _, err = bnswap.GetOrderLimits(mtGlobalCtx, tAPI, tSymbols)
	if err != nil {
		logger.Debugf("bnswap.GetOrderLimits error %v", err)
		return
	}

	for makerSymbol, makerStepSize := range mContractSizes {
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
		mSymbols, *mtConfig.PullInterval,
		mPositionCh,
	)
	go hbcrossswap.WatchAccountFromHttp(
		mtGlobalCtx, mAPI,
		*mtConfig.PullInterval, mAccountCh,
	)
	go hbcrossswap.WatchFundingRate(
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
		*mtConfig.TopBandScale,
		*mtConfig.BotBandScale,
		*mtConfig.MinimalEnterDelta,
		*mtConfig.MaximalExitDelta,
		*mtConfig.MinimalBandOffset,
		mtBarsMapCh,
		mtQuantilesCh,
	)
	depthReportCh := make(chan common.DepthReport, 10000)
	spreadReportCh := make(chan common.SpreadReport, 10000)
	go watchReports(
		mtGlobalCtx,
		mtInfluxWriter,
		*mtConfig.InternalInflux,
		depthReportCh,
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
			mContractSizes[makerSymbol],
			*mtConfig.OrderBookMakerImpact,
			*mtConfig.OrderBookTakerImpact,
			*mtConfig.OrderBookMaxAgeDiff,
			*mtConfig.OrderBookMaxAge,
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
					Cancel: &hbcrossswap.CancelAllParam{
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
			handleMakerWSPosition(msg)
			break
		case msg := <-mUserWebsocket.AccountCh:
			handleMakerWSAccount(msg)
			break
		case makerOrder := <-mUserWebsocket.OrderCh:
			if makerOrder.Status == hbcrossswap.OrderStatusFilled ||
				makerOrder.Status == hbcrossswap.OrderStatusCancelled ||
				makerOrder.Status == hbcrossswap.OrderStatusPartiallyFilledButCancelledByClient {
				if openOrder, ok := mOpenOrders[makerOrder.Symbol]; ok && openOrder.ClientOrderID == makerOrder.ClientOrderID {
					delete(mOpenOrders, makerOrder.Symbol)
				}
				if makerOrder.Status == hbcrossswap.OrderStatusCancelled {
					logger.Debugf("MAKER WS ORDER CANCELED %v ", makerOrder)
					mOrderSilentTimes[makerOrder.Symbol] = time.Now().Add(time.Second)
					mPositionsUpdateTimes[makerOrder.Symbol] = time.Unix(0, 0)
				} else {
					logger.Debugf(
						"MAKER WS ORDER FILLED %s SIDE %s TRADE SIZE %v TRADE PRICE %f",
						makerOrder.Symbol, makerOrder.Direction, makerOrder.TradeVolume, makerOrder.TradeAvgPrice,
					)
					tOrderSilentTimes[mtSymbolsMap[makerOrder.Symbol]] = time.Now()
					mtLoopTimer.Reset(time.Nanosecond)
					mHttpPositionUpdateSilentTimes[makerOrder.Symbol] = time.Now().Add(*mtConfig.HttpSilent)
					if makerOrder.Direction == hbcrossswap.OrderDirectionSell {
						mLastFilledSellPrices[makerOrder.Symbol] = makerOrder.TradeAvgPrice
					} else if makerOrder.Direction == hbcrossswap.OrderDirectionBuy {
						mLastFilledBuyPrices[makerOrder.Symbol] = makerOrder.TradeAvgPrice
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
		case mFundingRates = <-mFundingRatesCh:
			handleUpdateFundingRates()
			break
		case tPremiumIndexes = <-tPremiumIndexesCh:
			//logger.Debugf("%v", tPremiumIndexes)
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
				//logger.Debugf("QUANTILES %s", d)
			}
			mtQuantiles = qs
			//mtLoopTimer.Reset(time.Millisecond)
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
				} else if makerOpenOrder.NewOrderParam != nil && openOrder.ClientOrderID == makerOpenOrder.ClientOrderID {
					//New的Http回报
					mOpenOrders[makerOpenOrder.Symbol] = makerOpenOrder
				}
			}
		case <-mtLoopTimer.C:
			updateTakerPositions()
			if time.Now().Sub(time.Now().Truncate(fundingInterval)) > fundingSilent &&
				time.Now().Truncate(fundingInterval).Add(fundingInterval).Sub(time.Now()) > fundingSilent {
				updateMakerOldOrders()
				updateMakerNewOrders()
			} else {
				cancelAllMakerOpenOrders()
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
