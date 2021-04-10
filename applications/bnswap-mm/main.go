package main

import (
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
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
			logger.Fatal(err)
		}
		err = pprof.StartCPUProfile(f)
		if err != nil {
			logger.Fatal(err)
		}
		defer pprof.StopCPUProfile()
	}

	var err error
	bnswapAPI, err = bnswap.NewAPI(&common.Credentials{
		Key:    *bnConfig.ApiKey,
		Secret: *bnConfig.ApiSecret,
	}, *bnConfig.ProxyAddress)
	if err != nil {
		logger.Fatal(err)
	}

	bnGlobalCtx, bnGlobalCancel = context.WithCancel(context.Background())
	defer bnGlobalCancel()

	bnswapTickSizes, bnswapStepSizes, _, bnswapMinNotional, _, _, err = bnswap.GetOrderLimits(bnGlobalCtx, bnswapAPI, bnSymbols)
	if err != nil {
		logger.Fatal(err)
	}
	//bnspotTickSizes, bnspotStepSizes, _, bnspotMinNotional, err = bnspot.GetOrderLimits(bnGlobalCtx, bnspotAPI, bnSymbols)
	//if err != nil {
	//	logger.Fatal(err)
	//}

	bnInternalInfluxWriter, err = common.NewInfluxWriter(
		*bnConfig.InternalInflux.Address,
		*bnConfig.InternalInflux.Username,
		*bnConfig.InternalInflux.Password,
		*bnConfig.InternalInflux.Database,
		*bnConfig.InternalInflux.BatchSize,
	)
	if err != nil {
		logger.Fatal(err)
	}

	bnExternalInfluxWriter, err = common.NewInfluxWriter(
		*bnConfig.ExternalInflux.Address,
		*bnConfig.ExternalInflux.Username,
		*bnConfig.ExternalInflux.Password,
		*bnConfig.ExternalInflux.Database,
		*bnConfig.ExternalInflux.BatchSize,
	)
	if err != nil {
		logger.Fatal(err)
	}

	defer func() {
		err := bnInternalInfluxWriter.Stop()
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

	//bnspotUserWebsocket = bnspot.NewUserWebsocket(
	//	bnGlobalCtx,
	//	bnspotAPI,
	//	*bnConfig.ProxyAddress,
	//)
	//defer bnspotUserWebsocket.Stop()

	bnswapUserWebsocket = bnswap.NewUserWebsocket(
		bnGlobalCtx,
		bnswapAPI,
		*bnConfig.ProxyAddress,
	)
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
	loopTimer := time.NewTimer(time.Second) //先等1分钟

	defer influxSaveTimer.Stop()
	defer loopTimer.Stop()

	go bnswap.WatchPositionsFromHttp(
		bnGlobalCtx, bnswapAPI,
		bnSymbols, *bnConfig.PullInterval, bnswapPositionCh,
	)
	go bnswap.WatchAccountFromHttp(
		bnGlobalCtx, bnswapAPI,
		*bnConfig.PullInterval, bnswapAccountCh,
	)

	walkedOrderBookCh := make(chan WalkedOrderBook, len(bnSymbols)*10)
	swapMarkPriceCh := make(chan *bnswap.MarkPrice, len(bnSymbols))
	for start := 0; start < len(bnSymbols); start += *bnConfig.OrderBookBatchSize {
		end := start + *bnConfig.OrderBookBatchSize
		if end > len(bnSymbols) {
			end = len(bnSymbols)
		}
		go watchSwapWalkedOrderBooks(
			bnGlobalCtx, *bnConfig.ProxyAddress,
			*bnConfig.OrderBookSmallImpact,
			*bnConfig.OrderBookLargeImpact,
			bnSymbols[start:end],
			walkedOrderBookCh,
		)
		go watchMarkPrice(
			bnGlobalCtx, *bnConfig.ProxyAddress,
			bnSymbols[start:end],
			swapMarkPriceCh,
		)
	}

	spreadCh := make(chan Spread, len(bnSymbols)*10)
	go watchSpread(
		bnGlobalCtx,
		bnSymbols,
		*bnConfig.SpreadLookbackDuration,
		*bnConfig.SpreadLookbackMinimalWindow,
		walkedOrderBookCh,
		spreadCh,
	)

	bnswapCancelOrderResponsesCh = make(chan bnswap.CancelAllOrderResponse, len(bnSymbols)*2)
	bnswapNewOrderResponseCh = make(chan bnswap.Order, len(bnSymbols)*2)
	bnswapNewOrderErrorCh = make(chan SwapOrderNewError, len(bnSymbols)*2)
	for _, symbol := range bnSymbols {
		bnswapOrderRequestChs[symbol] = make(chan SwapOrderRequest, 2)
		go watchSwapOrderRequest(
			bnGlobalCtx,
			bnswapAPI,
			*bnConfig.OrderTimeout,
			*bnConfig.DryRun,
			bnswapOrderRequestChs[symbol],
			bnswapNewOrderErrorCh,
			bnswapNewOrderResponseCh,
			bnswapCancelOrderResponsesCh,
		)
		bnswapOrderRequestChs[symbol] <- SwapOrderRequest{
			Cancel: &bnswap.CancelAllOrderParams{Symbol: symbol},
		}
	}

	done := make(chan bool, 1)
	if *bnConfig.CpuProfile != "" {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			sig := <-sigs
			logger.Debugf("Exit with sig %d, clean *.tmp files", sig)
			done <- true
		}()
	}

	for {
		select {
		case <-done:
			logger.Debugf("Exit")
			return
		//case <-reBalanceTimer.C:
		//	if bnspotUSDTBalance != nil && bnswapUSDTAsset != nil && bnswapUSDTAsset.AvailableBalance != nil {
		//		//SWAP WS ACCOUNT 没有AvailableBalance, 为0 HTTP GET无数据
		//		//SWAP的MarginBalance AvailableBalance WS推送缺失，会造成错误判断
		//		if bnswapAssetUpdatedForReBalance && bnspotBalanceUpdatedForReBalance {
		//			bnswapAssetUpdatedForReBalance = false
		//			bnspotBalanceUpdatedForReBalance = false
		//
		//			expectedInsuranceFund := *bnConfig.StartValue * (1 - *bnConfig.InsuranceFundingRatio) * *bnConfig.Leverage / (*bnConfig.Leverage + 1) * *bnConfig.InsuranceFundingRatio
		//			totalFree := (bnspotUSDTBalance.Free + *bnswapUSDTAsset.AvailableBalance) - expectedInsuranceFund
		//			targetSwap := totalFree/(*bnConfig.Leverage+1) + expectedInsuranceFund
		//			change := targetSwap - *bnswapUSDTAsset.AvailableBalance
		//			if change > 0 && change > bnspotUSDTBalance.Free {
		//				change = bnspotUSDTBalance.Free
		//			}
		//			if change < 0 && -change > *bnswapUSDTAsset.AvailableBalance-expectedInsuranceFund {
		//				change = 0
		//				if *bnswapUSDTAsset.AvailableBalance-expectedInsuranceFund > 0 {
		//					change = -(*bnswapUSDTAsset.AvailableBalance - expectedInsuranceFund)
		//				}
		//			}
		//			if math.Abs(change) > *bnConfig.ReBalanceMinimalNotional {
		//				// 如果有转帐发生最好不要让influx统计数据，转帐在中间过程中会有盈利计算误差
		//				bnspotBalanceUpdatedForExternalInflux = false
		//				bnswapAssetUpdatedForExternalInflux = false
		//				bnspotBalanceUpdatedForInflux = false
		//				bnswapAssetUpdatedForInflux = false
		//				bnSaveSilentTime = time.Now().Add(*bnConfig.PullInterval * 2)
		//				go reBalanceUSDT(
		//					bnGlobalCtx,
		//					bnspotAPI,
		//					*bnConfig.OrderTimeout,
		//					change,
		//				)
		//			}
		//		}
		//	}
		//	reBalanceTimer.Reset(*bnConfig.ReBalanceInterval)
		//	break
		case p := <-bnswapPositionCh:
			handleSwapHttpPositions(p)
			break
		case account := <-bnswapAccountCh:
			handleSwapHttpAccount(account)
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
		case markPrice := <-swapMarkPriceCh:
			bnswapMarkPrices[markPrice.Symbol] = *markPrice
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
			if order.Status == common.OrderStatusReject || order.Status == common.OrderStatusExpired {
				logStr = fmt.Sprintf("%s RESET TIMEOUT", logStr)
				bnswapOrderSilentTimes[order.Symbol] = time.Now().Add(time.Second)
				bnswapPositionsUpdateTimes[order.Symbol] = time.Unix(0, 0)
			}
			if order.Status == common.OrderStatusFilled {
				if order.Side == common.OrderSideSell {
					if lastOrder, ok := bnswapLastFilledOrders[order.Symbol]; ok && lastOrder.Side == common.OrderSideBuy && !lastOrder.ReduceOnly {
						bnRealisedPnl[order.Symbol] = (order.CumQuote - lastOrder.CumQuote) / lastOrder.CumQuote
						logger.Debugf("%s REALISED LONG PNL %f", order.Symbol, bnRealisedPnl[order.Symbol])
					}
				} else if order.Side == common.OrderSideBuy {
					if lastOrder, ok := bnswapLastFilledOrders[order.Symbol]; ok && lastOrder.Side == common.OrderSideSell && !lastOrder.ReduceOnly {
						bnRealisedPnl[order.Symbol] = (lastOrder.CumQuote - order.CumQuote) / lastOrder.CumQuote
						logger.Debugf("%s REALISED SHORT PNL %f", order.Symbol, bnRealisedPnl[order.Symbol])
					}
				}
				bnswapLastFilledOrders[order.Symbol] = order
			}
			delete(bnswapOpenOrders, order.Symbol)
			logger.Debug(logStr)
			break

		case o := <-bnswapCancelOrderResponsesCh:
			logger.Debugf("CANCEL ALL %v", o)
			bnswapCancelSilentTimes[o.Symbol] = time.Now()
			delete(bnswapOpenOrders, o.Symbol)
		case order := <-bnswapNewOrderResponseCh:
			logStr := fmt.Sprintf("SWAP ORDER %v", order)
			if order.Status == common.OrderStatusReject ||
				order.Status == common.OrderStatusExpired ||
				order.Status == common.OrderStatusCancelled {
				logStr = fmt.Sprintf("%s RESET TIMEOUT", logStr)
				bnswapOrderSilentTimes[order.Symbol] = time.Now()
				if openOrder, ok := bnswapOpenOrders[order.Symbol]; ok && openOrder.NewClientOrderId == order.ClientOrderId {
					delete(bnswapOpenOrders, order.Symbol)
				}
			} else if order.Status == common.OrderStatusFilled {
				if order.Side == common.OrderSideSell {
					if lastOrder, ok := bnswapLastFilledOrders[order.Symbol]; ok && lastOrder.Side == common.OrderSideBuy && !lastOrder.ReduceOnly {
						bnRealisedPnl[order.Symbol] = (order.CumQuote - lastOrder.CumQuote) / lastOrder.CumQuote
						logger.Debugf("%s REALISED LONG PNL %f", order.Symbol, bnRealisedPnl[order.Symbol])
					}
				} else if order.Side == common.OrderSideBuy && order.Status == "FILLED" {
					if lastOrder, ok := bnswapLastFilledOrders[order.Symbol]; ok && lastOrder.Side == common.OrderSideSell && !lastOrder.ReduceOnly {
						bnRealisedPnl[order.Symbol] = (lastOrder.CumQuote - order.CumQuote) / lastOrder.CumQuote
						logger.Debugf("%s REALISED SHORT PNL %f", order.Symbol, bnRealisedPnl[order.Symbol])
					}
				}
				bnswapLastFilledOrders[order.Symbol] = order
				if openOrder, ok := bnswapOpenOrders[order.Symbol]; ok && openOrder.NewClientOrderId == order.ClientOrderId {
					delete(bnswapOpenOrders, order.Symbol)
				}
			}
			logger.Debug(logStr)
		case order := <-bnswapNewOrderErrorCh:
			if openOrder, ok := bnswapOpenOrders[order.Params.Symbol]; ok && openOrder.NewClientOrderId == order.Params.NewClientOrderId {
				delete(bnswapOpenOrders, order.Params.Symbol)
			}
			bnswapOrderSilentTimes[order.Params.Symbol] = time.Now().Add(*bnConfig.OrderSilent * 3)
			break
		case <-loopTimer.C:
			updateSwapOldOrders()
			updateSwapNewOrders()
			loopTimer.Reset(
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
