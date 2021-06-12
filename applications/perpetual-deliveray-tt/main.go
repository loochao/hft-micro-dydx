package main

import (
	"context"
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

	if xyConfig.CpuProfile != "" {
		f, err := os.Create(xyConfig.CpuProfile)
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

	xyGlobalCtx, xyGlobalCancel = context.WithCancel(context.Background())
	defer xyGlobalCancel()

	var err error
	err = xyExchange.Setup(xyGlobalCtx, xyConfig.Exchange)
	if err != nil {
		logger.Debugf("xyExchange.Setup(xyGlobalCtx, xyConfig.XExchange) error %v", err)
		return
	}
	for _, xySymbol := range xySymbols {
		xyStepSizes[xySymbol], err = xyExchange.GetStepSize(xySymbol)
		if err != nil {
			logger.Debugf("xyExchange.GetStepSize(xySymbol) error %v", err)
		}
		xyMinNotionals[xySymbol], err = xyExchange.GetMinNotional(xySymbol)
		if err != nil {
			logger.Debugf("xyExchange.GetMinNotional(xySymbol) error %v", err)
		}
		xyContractSizes[xySymbol], err = xyExchange.GetContractSize(xySymbol)
		if err != nil {
			logger.Debugf("xyExchange.GetContractSize error %v", err)
		}
	}
	logger.Debugf("%v", xyContractSizes)

	xyInfluxWriter, err = common.NewInfluxWriter(
		xyGlobalCtx,
		xyConfig.InternalInflux.Address,
		xyConfig.InternalInflux.Username,
		xyConfig.InternalInflux.Password,
		xyConfig.InternalInflux.Database,
		xyConfig.InternalInflux.BatchSize,
	)
	if err != nil {
		logger.Debugf("common.NewInfluxWriter error %v", err)
		return
	}
	defer xyInfluxWriter.Stop()

	xyExternalInfluxWriter, err = common.NewInfluxWriter(
		xyGlobalCtx,
		xyConfig.ExternalInflux.Address,
		xyConfig.ExternalInflux.Username,
		xyConfig.ExternalInflux.Password,
		xyConfig.ExternalInflux.Database,
		xyConfig.ExternalInflux.BatchSize,
	)
	if err != nil {
		logger.Debugf("common.NewInfluxWriter error %v", err)
		return
	}
	defer xyExternalInfluxWriter.Stop()

	influxSaveTimer := time.NewTimer(
		time.Now().Truncate(
			xyConfig.InternalInflux.SaveInterval,
		).Add(
			xyConfig.InternalInflux.SaveInterval * 3,
		).Sub(time.Now()),
	)
	defer influxSaveTimer.Stop()

	xyLoopTimer = time.NewTimer(time.Second)
	defer xyLoopTimer.Stop()

	xyPositionChMap := make(map[string]chan common.Position)
	xyBalanceChMap := make(map[string]chan common.Balance)
	xyOrderChMap := make(map[string]chan common.Order)
	xyFundingRateChMap := make(map[string]chan common.FundingRate)
	xyDepthChMap := make(map[string]chan common.Depth)
	xyNewOrderErrorChMap := make(map[string]chan common.OrderError)
	for _, xySymbol := range xySymbols {
		xyPositionChMap[xySymbol] = xyPositionCh
		if asset, ok := xyConfig.SymbolAssetMap[xySymbol]; ok {
			xyBalanceChMap[asset] = xyBalanceCh
		}
		xyOrderChMap[xySymbol] = xyOrderCh
		xyFundingRateChMap[xySymbol] = xFundingRateCh
		xyNewOrderErrorChMap[xySymbol] = xyNewOrderErrorCh
		xyDepthChMap[xySymbol] = make(chan common.Depth, 200)
		xyOrderRequestChMap[xySymbol] = make(chan common.OrderRequest, 200)
	}
	go xyExchange.StreamBasic(
		xyGlobalCtx,
		xySystemStatusCh,
		xyBalanceChMap,
		xyPositionChMap,
		xyOrderChMap,
	)
	go xyExchange.StreamFundingRate(
		xyGlobalCtx,
		xyFundingRateChMap,
		xyConfig.BatchSize,
	)
	go xyExchange.StreamDepth(
		xyGlobalCtx,
		xyDepthChMap,
		xyConfig.BatchSize,
	)
	go xyExchange.WatchOrders(
		xyGlobalCtx,
		xyOrderRequestChMap,
		xyOrderChMap,
		xyNewOrderErrorChMap,
	)

	spreadReportCh := make(chan SpreadReport, 10000)
	go reportsSaveLoop(
		xyGlobalCtx,
		xyInfluxWriter,
		xyConfig.InternalInflux,
		spreadReportCh,
	)

	spreadCh := make(chan *XYSpread, len(xSymbols)*100)
	for xSymbol, ySymbol := range xyConfig.XYPairs {
		go watchXYSpread(
			xyGlobalCtx,
			xSymbol, ySymbol,
			xyConfig.DepthMakerImpact,
			xyConfig.DepthTakerImpact,
			xyConfig.DepthXDecay,
			xyConfig.DepthXBias,
			xyConfig.DepthYDecay,
			xyConfig.DepthYBias,
			xyConfig.DepthTimeDeltaMin,
			xyConfig.DepthTimeDeltaMax,
			xyConfig.DepthMaxAgeDiffBias,
			xyConfig.ReportCount,
			xyConfig.SpreadLookback,
			xyDepthChMap[xSymbol],
			xyDepthChMap[ySymbol],
			spreadReportCh,
			spreadCh,
		)
	}

	if xyConfig.CpuProfile != "" {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			sig := <-sigs
			logger.Debugf("catch exit signal %v", sig)
			xyGlobalCancel()
		}()
	}

	logger.Debugf("start main loop")
	fundingInterval := time.Hour * 8
	fundingSilent := time.Minute * 5

	restartTimer := time.NewTimer(xyConfig.RestartInterval)
	defer restartTimer.Stop()

mainLoop:
	for {
		select {
		case <-xyGlobalCtx.Done():
			logger.Debugf("global ctx done, exit main loop")
			xyGlobalCancel()
			break mainLoop
		case <-xyExchange.Done():
			logger.Debugf("x exchange done, exit main loop")
			xyGlobalCancel()
			break mainLoop
		case <-restartTimer.C:
			logger.Debugf("timed restart in %v", xyConfig.RestartInterval)
			xyGlobalCancel()
			break mainLoop
		case xySystemStatus = <-xySystemStatusCh:
			if xySystemStatus != common.SystemStatusReady {
				logger.Debugf("xySystemStatus %v", xySystemStatus)
			}
			break
		case nextPos := <-xyPositionCh:
			//logger.Debugf("x position %s %v %v %f %f", nextPos.GetSymbol(), nextPos.GetEventTime(), nextPos.GetParseTime(), nextPos.GetPrice(), nextPos.GetSize())
			if _, isX := xySymbolsMap[nextPos.GetSymbol()]; isX {
				if prevPos, ok := xyPositions[nextPos.GetSymbol()]; ok {
					if prevPos == nextPos {
						logger.Debugf("bad prevPos == nextPos pass same pointer")
					}
					if nextPos.GetEventTime().Sub(prevPos.GetEventTime()) >= 0 {
						xyTimedPositionChange.Insert(time.Now(), math.Abs(prevPos.GetSize()-nextPos.GetSize())*xyContractSizes[nextPos.GetSymbol()])
						xyPositions[nextPos.GetSymbol()] = nextPos
						if prevPos.GetSize() != nextPos.GetSize() {
							logger.Debugf("%s x position change %f -> %f %v", nextPos.GetSymbol(), prevPos.GetSize(), nextPos.GetSize(), nextPos.GetEventTime())
						}
					}
					xyPositionsUpdateTimes[nextPos.GetSymbol()] = nextPos.GetParseTime()
				} else {
					xyPositions[nextPos.GetSymbol()] = nextPos
					xyPositionsUpdateTimes[nextPos.GetSymbol()] = nextPos.GetParseTime()
					logger.Debugf("%s x position change nil -> %f", nextPos.GetSymbol(), nextPos.GetSize())
				}
			} else if _, isY := yxSymbolsMap[nextPos.GetSymbol()]; isY {
				if prevPos, ok := xyPositions[nextPos.GetSymbol()]; ok {
					if prevPos == nextPos {
						logger.Debugf("bad prevPos == nextPos pass same pointer")
					}
					if nextPos.GetEventTime().Sub(prevPos.GetEventTime()) >= 0 {
						xyPositions[nextPos.GetSymbol()] = nextPos
						if prevPos.GetSize() != nextPos.GetSize() {
							xyTimedPositionChange.Insert(time.Now(), math.Abs(prevPos.GetSize()-nextPos.GetSize())*xyContractSizes[nextPos.GetSymbol()])
							logger.Debugf("%s y position change %f -> %f %v", nextPos.GetSymbol(), prevPos.GetSize(), nextPos.GetSize(), nextPos.GetEventTime())
						}
					}
					xyPositionsUpdateTimes[nextPos.GetSymbol()] = nextPos.GetParseTime()
				} else {
					xyPositions[nextPos.GetSymbol()] = nextPos
					xyPositionsUpdateTimes[nextPos.GetSymbol()] = nextPos.GetParseTime()
					logger.Debugf("%s y position change nil -> %f", nextPos.GetSymbol(), nextPos.GetSize())
				}
			}
			break
		case balance := <-xyBalanceCh:
			//logger.Debugf("%v", balance)
			if oldBalance, ok := xyBalanceMap[balance.GetCurrency()]; ok {
				if balance.GetTime().Sub(oldBalance.GetTime()) >= 0 {
					if balance.GetBalance() != oldBalance.GetBalance() {
						logger.Debugf("%s BALANCE CHANGE %f -> %f", balance.GetCurrency(), oldBalance.GetBalance(), balance.GetBalance())
					}
					xyBalanceMap[balance.GetCurrency()] = balance
				}
			} else {
				xyBalanceMap[balance.GetCurrency()] = balance
				logger.Debugf("%s balance change nil -> %f", balance.GetCurrency(), balance.GetBalance())
			}
			break
		case xyOrder := <-xyOrderCh:
			if _, isX := xySymbolsMap[xyOrder.GetSymbol()]; isX {
				if xyOrder.GetStatus() == common.OrderStatusExpired ||
					xyOrder.GetStatus() == common.OrderStatusReject ||
					xyOrder.GetStatus() == common.OrderStatusCancelled ||
					xyOrder.GetStatus() == common.OrderStatusFilled {

					xSymbol := xyOrder.GetSymbol()
					if xyOrder.GetStatus() != common.OrderStatusFilled {
						logger.Debugf("x order ended %s %s %s", xyOrder.GetSymbol(), xyOrder.GetStatus(), xyOrder.GetSide())
						xyOrderSilentTimes[xSymbol] = time.Now().Add(time.Second)
						xyPositionsUpdateTimes[xSymbol] = time.Unix(0, 0)
					} else {
						logger.Debugf("x order filled %s %s %s size %f price %f", xyOrder.GetSymbol(), xyOrder.GetStatus(), xyOrder.GetSide(), xyOrder.GetFilledSize(), xyOrder.GetFilledPrice())
						if ySymbol, ok := xySymbolsMap[xSymbol]; ok {
							if xyOrder.GetSide() == common.OrderSideBuy {
								xBuyPrice := xyOrder.GetFilledPrice()
								xLastFilledBuyPrices[xSymbol] = xBuyPrice
								if ySellPrice, ok := yLastFilledSellPrices[ySymbol]; ok {
									xyRealisedSpread[xSymbol] = (ySellPrice - xBuyPrice) / ySellPrice
									logger.Debugf("%s - %s realised short spread %f", xSymbol, ySymbol, xyRealisedSpread[xSymbol])
									delete(xLastFilledBuyPrices, xSymbol)
									delete(xLastFilledSellPrices, xSymbol)
									delete(yLastFilledBuyPrices, ySymbol)
									delete(yLastFilledSellPrices, ySymbol)
								}
							} else if xyOrder.GetSide() == common.OrderSideSell {
								xSellPrice := xyOrder.GetFilledPrice()
								xLastFilledSellPrices[xSymbol] = xSellPrice
								if yBuyPrice, ok := yLastFilledBuyPrices[ySymbol]; ok {
									xyRealisedSpread[xSymbol] = (yBuyPrice - xSellPrice) / yBuyPrice
									logger.Debugf("%s - %s realised long spread %f", ySymbol, xSymbol, xyRealisedSpread[xSymbol])
									delete(xLastFilledBuyPrices, xSymbol)
									delete(xLastFilledSellPrices, xSymbol)
									delete(yLastFilledBuyPrices, ySymbol)
									delete(yLastFilledSellPrices, ySymbol)
								}
							}
						}
					}
				}
			} else if _, isY := xySymbolsMap[xyOrder.GetSymbol()]; isY {
				if xyOrder.GetStatus() == common.OrderStatusExpired ||
					xyOrder.GetStatus() == common.OrderStatusReject ||
					xyOrder.GetStatus() == common.OrderStatusCancelled ||
					xyOrder.GetStatus() == common.OrderStatusFilled {

					ySymbol := xyOrder.GetSymbol()
					if xyOrder.GetStatus() != common.OrderStatusFilled {
						logger.Debugf("y order ended %s %s %s", xyOrder.GetSymbol(), xyOrder.GetStatus(), xyOrder.GetSide())
						xyOrderSilentTimes[ySymbol] = time.Now().Add(time.Second)
						xyPositionsUpdateTimes[ySymbol] = time.Unix(0, 0)
					} else {
						logger.Debugf("y order filled %s %s %s size %f price %f", xyOrder.GetSymbol(), xyOrder.GetStatus(), xyOrder.GetSide(), xyOrder.GetFilledSize(), xyOrder.GetFilledPrice())
						if xSymbol, ok := yxSymbolsMap[ySymbol]; ok {
							if xyOrder.GetSide() == common.OrderSideBuy {
								yBuyPrice := xyOrder.GetFilledPrice()
								yLastFilledBuyPrices[ySymbol] = yBuyPrice
								//logger.Debugf("%s set y buy price %f dir x %s sell %f", ySymbol, yBuyPrice, xySymbol, xLastFilledSellPrices[xySymbol])
								if xSellPrice, ok := xLastFilledSellPrices[xSymbol]; ok {
									xyRealisedSpread[xSymbol] = (yBuyPrice - xSellPrice) / yBuyPrice
									logger.Debugf("%s - %s realised long spread %f", ySymbol, xSymbol, xyRealisedSpread[xSymbol])
									delete(xLastFilledBuyPrices, xSymbol)
									delete(xLastFilledSellPrices, xSymbol)
									delete(yLastFilledBuyPrices, ySymbol)
									delete(yLastFilledSellPrices, ySymbol)
								}
							} else if xyOrder.GetSide() == common.OrderSideSell {
								ySellPrice := xyOrder.GetFilledPrice()
								yLastFilledSellPrices[ySymbol] = ySellPrice
								//logger.Debugf("%s set y sell price %f dir x %s buy %f", ySymbol, ySellPrice, xySymbol, xLastFilledBuyPrices[xySymbol])
								if xBuyPrice, ok := xLastFilledBuyPrices[xSymbol]; ok {
									xyRealisedSpread[xSymbol] = (ySellPrice - xBuyPrice) / ySellPrice
									logger.Debugf("%s - %s realised short spread %f", ySymbol, xSymbol, xyRealisedSpread[xSymbol])
									delete(xLastFilledBuyPrices, xSymbol)
									delete(xLastFilledSellPrices, xSymbol)
									delete(yLastFilledBuyPrices, ySymbol)
									delete(yLastFilledSellPrices, ySymbol)
								}
							}
						}
					}
				}
			}
			break
		case spread := <-spreadCh:
			xySpreads[spread.XSymbol] = spread
			break
		case fr := <-xFundingRateCh:
			xFundingRates[fr.GetSymbol()] = fr
			//handleUpdateFundingRates()
			break
		case <-influxSaveTimer.C:
			handleSave()
			influxSaveTimer.Reset(
				time.Now().Truncate(
					xyConfig.InternalInflux.SaveInterval,
				).Add(
					xyConfig.InternalInflux.SaveInterval + time.Second*15,
				).Sub(time.Now()),
			)
			break
		case xNewError := <-xyNewOrderErrorCh:
			if xNewError.Cancel != nil {
				logger.Debugf("Cancel %v error %v", *xNewError.Cancel, xNewError.Error)
				xyOrderSilentTimes[xNewError.Cancel.Symbol] = time.Now().Add(xyConfig.OrderSilent)
			} else if xNewError.New != nil {
				logger.Debugf("New %v error %v", *xNewError.New, xNewError.Error)
				xyOrderSilentTimes[xNewError.New.Symbol] = time.Now().Add(xyConfig.OrderSilent)
			}
			break

		case <-xyLoopTimer.C:
			if xySystemStatus == common.SystemStatusReady {
				if time.Now().Sub(time.Now().Truncate(fundingInterval)) > fundingSilent &&
					time.Now().Truncate(fundingInterval).Add(fundingInterval).Sub(time.Now()) > fundingSilent {
					updateTargetPositionSizes()
					updateXPositions()
					updateYPositions()
				}
			} else {
				if time.Now().Sub(time.Now().Truncate(time.Second*15)) < xyConfig.LoopInterval {
					logger.Debugf(
						"system not ready xySystemStatus %v",
						xySystemStatus,
					)
				}
			}
			xyLoopTimer.Reset(
				time.Now().Truncate(
					xyConfig.LoopInterval,
				).Add(
					xyConfig.LoopInterval,
				).Sub(time.Now()),
			)
			break
		}
	}
	logger.Debugf("hedge all positions, and wait 30s")
	updateXPositions()
	updateYPositions()
	<-time.After(time.Second * 30)
}
