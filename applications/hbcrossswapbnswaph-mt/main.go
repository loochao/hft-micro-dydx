package main

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/hbcrossswap"
	"github.com/geometrybase/hft-micro/hbspot"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"os/signal"
	"runtime/pprof"
	"strings"
	"syscall"
	"time"
)

func main() {

	if *hbConfig.CpuProfile != "" {
		f, err := os.Create(*hbConfig.CpuProfile)
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
	hAPI, err = hbcrossswap.NewAPI(
		*hbConfig.HbApiKey,
		*hbConfig.HbApiSecret,
		*hbConfig.ProxyAddress,
	)
	if err != nil {
		logger.Fatal(err)
	}
	bAPI, err = hbspot.NewAPI(
		*hbConfig.HbApiKey,
		*hbConfig.HbApiSecret,
		*hbConfig.ProxyAddress,
	)
	if err != nil {
		logger.Fatal(err)
	}

	hbGlobalCtx, hbGlobalCancel = context.WithCancel(context.Background())
	defer hbGlobalCancel()

	accounts, err := bAPI.GetAccounts(hbGlobalCtx)
	if err != nil {
		logger.Fatal(err)
	}
	for _, a := range accounts {
		if a.Type == "spot" {
			hbspotAccountID = a.ID
		}
	}
	if hbspotAccountID == 0 {
		logger.Fatal("HB SPOT ACCOUNT ID NOT EXISTS!!!")
	}

	hbcrossswapTickSizes, hbcrossswapContractSizes, err = hbcrossswap.GetOrderLimits(hbGlobalCtx, hAPI, bSymbols)
	if err != nil {
		logger.Fatal(err)
	}
	hbspotTickSizes, hbspotStepSizes, hbspotMinSizes, hbspotMinNotional, hbspotPricePrecisions, hbspotAmountPrecisions, err = hbspot.GetOrderLimits(hbGlobalCtx, bAPI, hSymbols)
	if err != nil {
		logger.Fatal(err)
	}

	hbInfluxWriter, err = common.NewInfluxWriter(
		*hbConfig.InternalInflux.Address,
		*hbConfig.InternalInflux.Username,
		*hbConfig.InternalInflux.Password,
		*hbConfig.InternalInflux.Database,
		*hbConfig.InternalInflux.BatchSize,
	)
	if err != nil {
		logger.Fatal(err)
	}

	hbExternalInfluxWriter, err = common.NewInfluxWriter(
		*hbConfig.ExternalInflux.Address,
		*hbConfig.ExternalInflux.Username,
		*hbConfig.ExternalInflux.Password,
		*hbConfig.ExternalInflux.Database,
		*hbConfig.ExternalInflux.BatchSize,
	)
	if err != nil {
		logger.Fatal(err)
	}

	defer func() {
		err := hbInfluxWriter.Stop()
		if err != nil {
			logger.Warnf("stop influx writer error %v", err)
		}
	}()

	bUserWebsocket = hbspot.NewUserWebsocket(
		hbGlobalCtx,
		*hbConfig.HbApiKey,
		*hbConfig.HbApiSecret,
		hSymbols,
		*hbConfig.ProxyAddress,
	)
	defer bUserWebsocket.Stop()

	hUserWebsocket = hbcrossswap.NewUserWebsocket(
		hbGlobalCtx,
		*hbConfig.HbApiKey,
		*hbConfig.HbApiSecret,
		bSymbols,
		*hbConfig.ProxyAddress,
	)
	defer hUserWebsocket.Stop()

	influxSaveTimer := time.NewTimer(
		time.Now().Truncate(
			*hbConfig.InternalInflux.SaveInterval,
		).Add(
			*hbConfig.InternalInflux.SaveInterval * 3,
		).Sub(time.Now()),
	)
	externalInfluxSaveTimer := time.NewTimer(
		time.Now().Truncate(
			*hbConfig.ExternalInflux.SaveInterval,
		).Add(
			*hbConfig.ExternalInflux.SaveInterval * 3,
		).Sub(time.Now()),
	)
	hbLoopTimer = time.NewTimer(time.Second) //先等1分钟
	frRankUpdatedTimer := time.NewTimer(time.Second * 15)

	defer influxSaveTimer.Stop()
	defer hbLoopTimer.Stop()
	defer frRankUpdatedTimer.Stop()
	defer externalInfluxSaveTimer.Stop()

	go hbcrossswap.WatchPositionsFromHttp(
		hbGlobalCtx, hAPI,
		bSymbols, *hbConfig.PullInterval,
		hPositionCh,
	)
	go hbcrossswap.WatchAccountFromHttp(
		hbGlobalCtx, hAPI,
		*hbConfig.PullInterval, hbcrossswapAccountCh,
	)
	go hbspot.WatchAccountFromHttp(
		hbGlobalCtx, bAPI, hbspotAccountID,
		*hbConfig.PullInterval, hbspotAccountCh,
	)
	go hbcrossswap.WatchFundingRate(
		hbGlobalCtx, hAPI,
		bSymbols,
		*hbConfig.PullInterval*10,
		hFundingRatesCh,
	)

	go watchSwapBars(
		hbGlobalCtx,
		hAPI,
		bSymbols,
		*hbConfig.BarsLookback,
		*hbConfig.PullBarsInterval,
		*hbConfig.PullBarsRetryInterval,
		*hbConfig.RequestInterval,
		hbcrossswapBarsMapCh,
	)

	go watchSpotBars(
		hbGlobalCtx,
		bAPI,
		hSymbols,
		*hbConfig.BarsLookback,
		*hbConfig.PullBarsInterval,
		*hbConfig.PullBarsRetryInterval,
		*hbConfig.RequestInterval,
		hbspotBarsMapCh,
	)

	go watchDeltaQuantile(
		hbGlobalCtx,
		hSymbols,
		bhSymbolsMap,
		*hbConfig.BotQuantile,
		*hbConfig.TopQuantile,
		*hbConfig.TopBandScale,
		*hbConfig.BotBandScale,
		*hbConfig.MinimalEnterDelta,
		*hbConfig.MaximalExitDelta,
		*hbConfig.MinimalBandOffset,
		hbBarsMapCh,
		hbQuantilesCh,
	)

	walkedOrderBookCh := make(chan WalkedOrderBook, len(hSymbols)*10)
	for start := 0; start < len(hSymbols); start += *hbConfig.OrderBookBatchSize {
		end := start + *hbConfig.OrderBookBatchSize
		if end > len(hSymbols) {
			end = len(hSymbols)
		}
		go watchSpotWalkedOrderBooks(
			hbGlobalCtx,
			*hbConfig.ProxyAddress,
			*hbConfig.OrderBookTakerImpact,
			*hbConfig.OrderBookMakerImpact,
			hSymbols[start:end],
			walkedOrderBookCh,
		)
	}

	for start := 0; start < len(bSymbols); start += *hbConfig.OrderBookBatchSize {
		end := start + *hbConfig.OrderBookBatchSize
		if end > len(hSymbols) {
			end = len(hSymbols)
		}
		go watchSwapWalkedOrderBooks(
			hbGlobalCtx,
			*hbConfig.ProxyAddress,
			hbcrossswapContractSizes,
			*hbConfig.OrderBookTakerImpact,
			*hbConfig.OrderBookMakerImpact,
			bSymbols[start:end],
			walkedOrderBookCh,
		)
	}

	spreadCh := make(chan Spread, len(hSymbols)*100)
	go watchSpread(
		hbGlobalCtx,
		hSymbols,
		hbSymbolsMap,
		*hbConfig.OrderBookMaxAgeDiff,
		*hbConfig.OrderBookMaxAge,
		*hbConfig.SpreadLookbackDuration,
		*hbConfig.SpreadLookbackMinimalWindow,
		walkedOrderBookCh,
		spreadCh,
	)

	hbspotNewOrderErrorCh = make(chan SpotOrderNewError, len(hSymbols)*2)
	for _, spotSymbol := range hSymbols {
		hbspotOrderRequestChs[spotSymbol] = make(chan SpotOrderRequest, 2)
		go watchSpotOrderRequest(
			hbGlobalCtx,
			bAPI,
			*hbConfig.OrderTimeout,
			*hbConfig.DryRun,
			hbspotOrderRequestChs[spotSymbol],
			hbspotNewOrderErrorCh,
		)
		hbspotOrderRequestChs[spotSymbol] <- SpotOrderRequest{
			Cancel: &hbspot.CancelAllParam{Symbol: spotSymbol},
		}
	}

	hNewOrderErrorCh = make(chan SwapOrderNewError, len(hSymbols)*2)
	for _, swapSymbol := range bSymbols {
		hOrderRequestChs[swapSymbol] = make(chan hbcrossswap.NewOrderParam, 2)
		go watchSwapOrderRequest(
			hbGlobalCtx,
			hAPI,
			*hbConfig.OrderTimeout,
			*hbConfig.DryRun,
			hOrderRequestChs[swapSymbol],
			hNewOrderErrorCh,
		)
	}

	done := make(chan bool, 1)
	if *hbConfig.CpuProfile != "" {
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
		case p := <-hPositionCh:
			handleSwapHttpPositions(p)
			break
		case account := <-hbcrossswapAccountCh:
			handleSwapHttpAccount(account)
			break
		case account := <-hbspotAccountCh:
			handleSpotHttpAccount(account)
			break
		case msg := <-bUserWebsocket.BalanceCh:
			handleSpotWSBalance(msg)
			break
		case spotOrder := <-bUserWebsocket.OrderCh:
			if spotOrder.OrderStatus != nil {
				if *spotOrder.OrderStatus == hbspot.OrderStatusFilled {
					if spotOrder.TradeVolume != nil && spotOrder.TradePrice != nil && spotOrder.Type != nil {
						if strings.Contains(*spotOrder.Type, "buy") {
							hLastFilledBuyPrices[spotOrder.Symbol] = *spotOrder.TradePrice
						} else {
							hLastFilledSellPrices[spotOrder.Symbol] = *spotOrder.TradePrice
						}
						logger.Debugf("SPOT WS ORDER FILLED %s %s SIZE %f PRICE %f", spotOrder.Symbol, *spotOrder.Type, *spotOrder.TradeVolume, *spotOrder.TradePrice)
					}
					if openOrder, ok := hOpenOrders[spotOrder.Symbol]; ok && openOrder.ClientOrderID == spotOrder.ClientOrderID {
						delete(hOpenOrders, spotOrder.Symbol)
					}
				} else if *spotOrder.OrderStatus == hbspot.OrderStatusCanceled {
					if openOrder, ok := hOpenOrders[spotOrder.Symbol]; ok && openOrder.ClientOrderID == spotOrder.ClientOrderID {
						delete(hOpenOrders, spotOrder.Symbol)
					}
				} else if *spotOrder.OrderStatus == hbspot.OrderStatusRejected {
					if openOrder, ok := hOpenOrders[spotOrder.Symbol]; ok && openOrder.ClientOrderID == spotOrder.ClientOrderID {
						delete(hOpenOrders, spotOrder.Symbol)
					}
				}
			}
			break
		case msg := <-hUserWebsocket.PositionCh:
			handleWSPosition(msg)
			break
		case msg := <-hUserWebsocket.AccountCh:
			handleWSAccount(msg)
			break
		case swapOrder := <-hUserWebsocket.OrderCh:
			if swapOrder.Status == hbcrossswap.OrderStatusFilled ||
				swapOrder.Status == hbcrossswap.OrderStatusCancelled ||
				swapOrder.Status == hbcrossswap.OrderStatusPartiallyFilledButCancelledByClient {
				if swapOrder.Status == hbcrossswap.OrderStatusCancelled {
					logger.Debugf("SWAP WS ORDER CANCELED %v ", swapOrder)
					hOrderSilentTimes[swapOrder.Symbol] = time.Now().Add(time.Second)
					hPositionsUpdateTimes[swapOrder.Symbol] = time.Unix(0, 0)
				} else {
					logger.Debugf(
						"SWAP WS ORDER FILLED %s SIDE %s TRADE SIZE %v TRADE PRICE %f",
						swapOrder.Symbol, swapOrder.Direction, swapOrder.TradeVolume, swapOrder.TradeAvgPrice,
					)
					if swapOrder.Direction == hbcrossswap.OrderDirectionSell {
						if spotSymbol, ok := hbSymbolsMap[swapOrder.Symbol]; ok {
							if spotPrice, ok := hLastFilledBuyPrices[spotSymbol]; ok {
								hbRealisedSpread[spotSymbol] = (swapOrder.TradeAvgPrice - spotPrice) / spotPrice
								logger.Debugf("%s %s REALISED OPEN SPREAD %f", spotSymbol, swapOrder.Symbol, hbRealisedSpread[spotSymbol])
							}
						}
					} else if swapOrder.Direction == hbcrossswap.OrderDirectionBuy {
						if spotSymbol, ok := hbSymbolsMap[swapOrder.Symbol]; ok {
							if spotPrice, ok := hLastFilledSellPrices[spotSymbol]; ok {
								hbRealisedSpread[spotSymbol] = (swapOrder.TradeAvgPrice - spotPrice) / spotPrice
								logger.Debugf("%s %s REALISED CLOSE SPREAD %f", spotSymbol, swapOrder.Symbol, hbRealisedSpread[spotSymbol])
							}
						}
					}
				}
			}
			break
		case spread := <-spreadCh:
			hbSpreads[spread.Symbol] = spread
			hbLoopTimer.Reset(time.Nanosecond)
			break
		case hFundingRates = <-hFundingRatesCh:
			//logger.Debugf("FRS %v", hFundingRates)
			break
		case hbcrossswapBarsMap = <-hbcrossswapBarsMapCh:
			if hBarsMapUpdated["spot"] {
				hbBarsMapCh <- [2]common.KLinesMap{hbspotBarsMap, hbcrossswapBarsMap}
				hBarsMapUpdated["spot"] = false
				hBarsMapUpdated["swap"] = false
			} else {
				hBarsMapUpdated["swap"] = true
			}
			break
		case hbspotBarsMap = <-hbspotBarsMapCh:
			if hBarsMapUpdated["swap"] {
				hbBarsMapCh <- [2]common.KLinesMap{hbspotBarsMap, hbcrossswapBarsMap}
				hBarsMapUpdated["spot"] = false
				hBarsMapUpdated["swap"] = false
			} else {
				hBarsMapUpdated["spot"] = true
			}
			break
		case qs := <-hbQuantilesCh:
			if hbQuantiles == nil {
				logger.Debugf("QUANTILES %v", qs)
			}
			hbQuantiles = qs
			hbLoopTimer.Reset(time.Millisecond)
			break
		case <-influxSaveTimer.C:
			handleSave()
			influxSaveTimer.Reset(
				time.Now().Truncate(
					*hbConfig.InternalInflux.SaveInterval,
				).Add(
					*hbConfig.InternalInflux.SaveInterval,
				).Sub(time.Now()),
			)
			break
		case <-externalInfluxSaveTimer.C:
			handleExternalInfluxSave()
			externalInfluxSaveTimer.Reset(
				time.Now().Truncate(
					*hbConfig.ExternalInflux.SaveInterval,
				).Add(
					*hbConfig.ExternalInflux.SaveInterval,
				).Sub(time.Now()),
			)
			break

		case newError := <-hNewOrderErrorCh:
			hOrderSilentTimes[newError.Params.Symbol] = time.Now().Add(time.Second * 15)
			break

		case order := <-hbspotNewOrderErrorCh:
			if openOrder, ok := hOpenOrders[order.Params.Symbol]; ok && openOrder.ClientOrderID == order.Params.ClientOrderID {
				delete(hOpenOrders, order.Params.Symbol)
			}
			bOrderSilentTimes[order.Params.Symbol] = time.Now().Add(*hbConfig.OrderSilent * 3)
		case <-frRankUpdatedTimer.C:
			frs := make([]float64, len(bSymbols))
			for i, symbol := range bSymbols {
				if fr, ok := hFundingRates[symbol]; ok {
					frs[i] = fr.FundingRate
				} else {
					logger.Debugf("MISS FUNDING RATE %s", symbol)
					break
				}
			}
			if len(hbRankSymbolMap) == 0 {
				logger.Debugf("RANK FR...")
			}
			hbRankSymbolMap, err = common.RankSymbols(bSymbols, frs)
			if err != nil {
				logger.Debugf("RankSymbols error %v", err)
			}
			//logger.Debugf("SYMBOLS FR RANK %v", hbRankSymbolMap)
			frRankUpdatedTimer.Reset(time.Minute)
			break
		case <-hbLoopTimer.C:
			updatePerpPositions()
			updateSpotOldOrders()
			updateSpotNewOrders()
			hbLoopTimer.Reset(
				time.Now().Truncate(
					*hbConfig.LoopInterval,
				).Add(
					*hbConfig.LoopInterval,
				).Sub(time.Now()),
			)
			break
		}
	}
}
