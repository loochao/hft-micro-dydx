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
	bnspotAPI, err = bnspot.NewAPI(&common.Credentials{
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
	bnspotTickSizes, bnspotStepSizes, _, bnspotMinNotional, err = bnspot.GetOrderLimits(bnGlobalCtx, bnspotAPI, bnSymbols)
	if err != nil {
		logger.Fatal(err)
	}

	bnInfluxWriter, err = common.NewInfluxWriter(
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
		err := bnInfluxWriter.Stop()
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

	bnspotUserWebsocket, err = bnspot.NewUserWebsocket(
		bnGlobalCtx,
		bnspotAPI,
		*bnConfig.ProxyAddress,
	)
	defer bnspotUserWebsocket.Stop()

	bnswapUserWebsocket, err = bnswap.NewUserWebsocket(
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
	loopTimer := time.NewTimer(time.Hour * 24) //先等1分钟
	resetUnrealisedPnlTimer := time.NewTimer(time.Minute)
	reBalanceTimer := time.NewTimer(time.Second)
	frRankUpdatedTimer := time.NewTimer(time.Second * 60)
	bnbReBalanceTimer := time.NewTimer(*bnConfig.BnbCheckInterval)

	defer bnbReBalanceTimer.Stop()
	defer influxSaveTimer.Stop()
	defer loopTimer.Stop()
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

	go watchSwapBars(
		bnGlobalCtx,
		bnswapAPI,
		bnSymbols,
		*bnConfig.BarsLookback,
		*bnConfig.PullBarsInterval,
		*bnConfig.PullBarsRetryInterval,
		bnswapBarsMapCh,
	)

	go watchSpotBars(
		bnGlobalCtx,
		bnspotAPI,
		bnSymbols,
		*bnConfig.BarsLookback,
		*bnConfig.PullBarsInterval,
		*bnConfig.PullBarsRetryInterval,
		bnspotBarsMapCh,
	)

	go watchDeltaQuantile(
		bnGlobalCtx,
		bnSymbols,
		*bnConfig.BotQuantile,
		*bnConfig.TopQuantile,
		*bnConfig.TopBandScale,
		*bnConfig.BotBandScale,
		*bnConfig.MinimalEnterDelta,
		*bnConfig.MaximalExitDelta,
		*bnConfig.MinimalBandOffset,
		bnBarsMapCh,
		bnQuantilesCh,
	)

	walkedOrderBookCh := make(chan WalkedOrderBook, len(bnSymbols)*10)
	for start := 0; start < len(bnSymbols); start += *bnConfig.OrderBookBatchSize {
		end := start + *bnConfig.OrderBookBatchSize
		if end > len(bnSymbols) {
			end = len(bnSymbols)
		}
		go watchSpotWalkedOrderBooks(
			bnGlobalCtx, *bnConfig.ProxyAddress,
			*bnConfig.OrderBookTakerImpact,
			*bnConfig.OrderBookMakerImpact,
			bnSymbols[start:end],
			walkedOrderBookCh,
		)
	}

	swapMarkPriceCh := make(chan *bnswap.MarkPrice, len(bnSymbols))
	for start := 0; start < len(bnSymbols); start += *bnConfig.OrderBookBatchSize {
		end := start + *bnConfig.OrderBookBatchSize
		if end > len(bnSymbols) {
			end = len(bnSymbols)
		}
		go watchSwapWalkedOrderBooks(
			bnGlobalCtx, *bnConfig.ProxyAddress,
			*bnConfig.OrderBookTakerImpact,
			*bnConfig.OrderBookMakerImpact,
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
		*bnConfig.OrderBookMaxAgeDiff,
		*bnConfig.OrderBookMaxAge,
		*bnConfig.SpreadLookbackDuration,
		*bnConfig.SpreadLookbackMinimalWindow,
		walkedOrderBookCh,
		spreadCh,
	)

	bnspotOrderFinishCh = make(chan bnspot.NewOrderResponse, len(bnSymbols)*2)
	bnspotNewOrderErrorCh = make(chan SpotOrderNewError, len(bnSymbols)*2)

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
		case <-reBalanceTimer.C:
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
		case data := <-bnspotUserWebsocket.OrderUpdateEventCh:
			if data.CurrentOrderStatus == bnspot.OrderStatusFilled {
				if data.CumulativeFilledQuantity > 0 && data.CumulativeQuoteAssetTransactedQuantity > 0 {
					filledPrice := data.CumulativeQuoteAssetTransactedQuantity / data.CumulativeFilledQuantity
					if data.Side == common.OrderSideBuy {
						bnspotLastFilledBuyPrices[data.Symbol] = filledPrice
					} else {
						bnspotLastFilledSellPrices[data.Symbol] = filledPrice
					}
					logger.Debugf("SPOT WS ORDER FILLED %s %s SIZE %f PRICE %f", data.Symbol, data.Side, data.CumulativeFilledQuantity, filledPrice)
				}
			} else if data.CurrentOrderStatus == bnspot.OrderStatusCancelled {
			} else if data.CurrentOrderStatus == bnspot.OrderStatusExpired ||
				data.CurrentOrderStatus == bnspot.OrderStatusReject {
			}
			break
		case msg := <-bnswapUserWebsocket.BalanceAndPositionUpdateEventCh:
			handleWSAccountEvent(msg)
			break
		case msg := <-bnswapUserWebsocket.OrderUpdateEventCh:
			handleWSOrder(&msg.Order)
			break
		case spread := <-spreadCh:
			bnSpreads[spread.Symbol] = spread
			//logger.Debugf("%s", spread.ToString())
			break
		case markPrice := <-swapMarkPriceCh:
			bnswapMarkPrices[markPrice.Symbol] = *markPrice
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
			if bnBarsMapUpdated["spot"] {
				bnBarsMapCh <- [2]common.KLinesMap{bnspotBarsMap, bnswapBarsMap}
				bnBarsMapUpdated["spot"] = false
				bnBarsMapUpdated["swap"] = false
			} else {
				bnBarsMapUpdated["swap"] = true
			}
			break
		case bnspotBarsMap = <-bnspotBarsMapCh:
			if bnBarsMapUpdated["swap"] {
				bnBarsMapCh <- [2]common.KLinesMap{bnspotBarsMap, bnswapBarsMap}
				bnBarsMapUpdated["spot"] = false
				bnBarsMapUpdated["swap"] = false
			} else {
				bnBarsMapUpdated["spot"] = true
			}
			break
		case bnQuantiles = <-bnQuantilesCh:
			//logger.Debugf("QUANTILE %v", bnQuantiles)
			loopTimer.Reset(time.Second)
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
			if order.Status == "REJECTED" || order.Status == "EXPIRED" {
				logStr = fmt.Sprintf("%s RESET TIMEOUT", logStr)
				bnswapOrderSilentTimes[order.Symbol] = time.Now().Add(time.Second)
				bnswapPositionsUpdateTimes[order.Symbol] = time.Unix(0, 0)
			}
			if order.Status == "FILLED" && order.Side == common.OrderSideSell {
				if spotPrice, ok := bnspotLastFilledBuyPrices[order.Symbol]; ok {
					bnRealisedSpread[order.Symbol] = (order.CumQuote - spotPrice) / spotPrice
					logger.Debugf("%s REALISED OPEN SPREAD %f", order.Symbol, bnRealisedSpread[order.Symbol])
				}
			}else if order.Status == "FILLED" && order.Side == common.OrderSideBuy {
				if spotPrice, ok := bnspotLastFilledSellPrices[order.Symbol]; ok {
					bnRealisedSpread[order.Symbol] = (order.CumQuote - spotPrice) / spotPrice
					logger.Debugf("%s REALISED CLOSE SPREAD %f", order.Symbol, bnRealisedSpread[order.Symbol])
				}
			}
			logger.Debug(logStr)
			break

		case order := <-bnspotOrderFinishCh:
			logStr := fmt.Sprintf("SPOT ORDER %v", order)
			if order.Status == bnspot.OrderStatusReject ||
				order.Status == bnspot.OrderStatusExpired ||
				order.Status == bnspot.OrderStatusCancelled {
				logStr = fmt.Sprintf("%s RESET TIMEOUT", logStr)
				bnspotOrderSilentTimes[order.Symbol] = time.Now()
				bnspotBalancesUpdateTimes[order.Symbol] = time.Unix(0, 0)
			} else if order.Status == bnspot.OrderStatusFilled {
				sumQty := 0.0
				sumVal := 0.0
				for _, f := range order.Fills {
					sumQty += f.Qty
					sumVal += f.Price * f.Qty
				}
				if sumQty != 0 && sumVal != 0 {
					filledPrice := sumVal / sumQty
					if order.Side == common.OrderSideBuy {
						bnspotLastFilledBuyPrices[order.Symbol] = filledPrice
					} else {
						bnspotLastFilledSellPrices[order.Symbol] = filledPrice
					}
					logStr = fmt.Sprintf("%s FILLED PRICE %f", logStr, filledPrice)
				}
			}
			logger.Debug(logStr)
		case order := <-bnspotNewOrderErrorCh:
			bnspotOrderSilentTimes[order.Params.Symbol] = time.Now().Add(*bnConfig.OrderSilent * 3)
		case <-frRankUpdatedTimer.C:
			frs := make([]float64, len(bnSymbols))
			for i, symbol := range bnSymbols {
				if markPrice, ok := bnswapMarkPrices[symbol]; ok {
					frs[i] = markPrice.FundingRate
				} else {
					logger.Debugf("MISS MARK PRICE %s", symbol)
					return
				}
			}
			bnRankSymbolMap, err = common.RankSymbols(bnSymbols, frs)
			if err != nil {
				logger.Debugf("RankSymbols error %v", err)
			}
			//logger.Debugf("SYMBOLS FR RANK %v", bnRankSymbolMap)
			frRankUpdatedTimer.Reset(time.Minute)
		case <-bnbReBalanceTimer.C:
			handleReBalanceBnb()
			bnbReBalanceTimer.Reset(*bnConfig.BnbCheckInterval)
			break
		case <-loopTimer.C:
			updateSwapPositions()
			updateSpotPositions()
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
