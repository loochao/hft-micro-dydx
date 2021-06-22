package main

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {

	//if mConfig.CpuProfile != "" {
	//	f, err := os.Create(mConfig.CpuProfile)
	//	if err != nil {
	//		logger.Debugf("os.Create error %v", err)
	//		return
	//	}
	//	err = pprof.StartCPUProfile(f)
	//	if err != nil {
	//		logger.Debugf("pprof.StartCPUProfile error %v", err)
	//		return
	//	}
	//	defer pprof.StopCPUProfile()
	//}

	mGlobalCtx, mGlobalCancel = context.WithCancel(context.Background())
	defer mGlobalCancel()

	var err error
	err = mExchange.Setup(mGlobalCtx, mConfig.MakerExchange)
	if err != nil {
		logger.Debugf("mExchange.Setup(mGlobalCtx, mConfig.MakerExchange) error %v", err)
		return
	}
	for _, mSymbol := range mSymbols {
		mStepSizes[mSymbol], err = mExchange.GetStepSize(mSymbol)
		if err != nil {
			logger.Debugf("mExchange.GetStepSize(mSymbol) error %v", err)
		}
		mTickSizes[mSymbol], err = mExchange.GetTickSize(mSymbol)
		if err != nil {
			logger.Debugf("mExchange.GetTickSize(mSymbol) error %v", err)
		}
		mMinNotional[mSymbol], err = mExchange.GetMinNotional(mSymbol)
		if err != nil {
			logger.Debugf("mExchange.GetMinNotional(mSymbol) error %v", err)
		}
	}
	logger.Debugf("maker tickSizes %v", mTickSizes)
	logger.Debugf("maker stepSizes %v", mStepSizes)
	logger.Debugf("maker minNotional %v", mMinNotional)

	if mConfig.InternalInflux.Address != "" {
		mInfluxWriter, err = common.NewInfluxWriter(
			mGlobalCtx,
			mConfig.InternalInflux.Address,
			mConfig.InternalInflux.Username,
			mConfig.InternalInflux.Password,
			mConfig.InternalInflux.Database,
			mConfig.InternalInflux.BatchSize,
		)
		if err != nil {
			logger.Debugf("common.NewInfluxWriter error %v", err)
			return
		}
		defer mInfluxWriter.Stop()
	}

	if mConfig.ExternalInflux.Address != "" {
		mExternalInfluxWriter, err = common.NewInfluxWriter(
			mGlobalCtx,
			mConfig.ExternalInflux.Address,
			mConfig.ExternalInflux.Username,
			mConfig.ExternalInflux.Password,
			mConfig.ExternalInflux.Database,
			mConfig.ExternalInflux.BatchSize,
		)
		if err != nil {
			logger.Debugf("common.NewInfluxWriter error %v", err)
			return
		}
		defer mExternalInfluxWriter.Stop()
	}

	internalInfluxSaveTimer := time.NewTimer(
		time.Now().Truncate(
			mConfig.InternalInflux.SaveInterval,
		).Add(
			mConfig.InternalInflux.SaveInterval * 3,
		).Sub(time.Now()),
	)
	externalInfluxSaveTimer := time.NewTimer(
		time.Now().Truncate(
			mConfig.ExternalInflux.SaveInterval,
		).Add(
			mConfig.ExternalInflux.SaveInterval * 3,
		).Sub(time.Now()),
	)
	mLoopTimer = time.NewTimer(time.Second) //先等1分钟
	defer internalInfluxSaveTimer.Stop()
	defer mLoopTimer.Stop()
	defer externalInfluxSaveTimer.Stop()

	makerPositionChMap := make(map[string]chan common.Position)
	makerOrderChMap := make(map[string]chan common.Order)
	makerDepthChMap := make(map[string]chan common.Depth)
	makerNewOrderErrorChMap := make(map[string]chan common.OrderError)
	for _, makerSymbol := range mSymbols {
		makerPositionChMap[makerSymbol] = mPositionCh
		makerOrderChMap[makerSymbol] = mOrderCh
		makerDepthChMap[makerSymbol] = make(chan common.Depth, 200)
		mOrderRequestChMap[makerSymbol] = make(chan common.OrderRequest, 200)
		makerNewOrderErrorChMap[makerSymbol] = mNewOrderErrorCh
	}
	go mExchange.StreamBasic(
		mGlobalCtx,
		mSystemStatusCh,
		mAccountCh,
		makerPositionChMap,
		makerOrderChMap,
	)
	go mExchange.StreamDepth(
		mGlobalCtx,
		makerDepthChMap,
		mConfig.BatchSize,
	)
	go mExchange.WatchOrders(
		mGlobalCtx,
		mOrderRequestChMap,
		makerOrderChMap,
		makerNewOrderErrorChMap,
	)

	wakedDepthCh := make(chan *common.WalkedMakerTakerDepth, len(mSymbols)*100)
	for makerSymbol := range mConfig.MakerOrderOffsets {
		go walkMakerDepth(
			mGlobalCtx,
			makerSymbol,
			mConfig.DepthMakerImpact,
			mConfig.DepthTakerImpact,
			mConfig.DepthMakerDecay,
			mConfig.DepthMakerBias,
			mConfig.DepthTimeDeltaMin,
			mConfig.DepthTimeDeltaMax,
			mConfig.ReportCount,
			makerDepthChMap[makerSymbol],
			wakedDepthCh,
			mFilterRatioCh,
		)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		logger.Debugf("catch exit signal %v", sig)
		mGlobalCancel()
	}()

	if !mConfig.DryRun {
		go func() {
			for _, makerSymbol := range mSymbols {
				select {
				case <-mGlobalCtx.Done():
					return
				case <-time.After(mConfig.RequestInterval):
					logger.Debugf("initial cancel all %s", makerSymbol)
					select {
					case <-mGlobalCtx.Done():
						return
					case mOrderRequestChMap[makerSymbol] <- common.OrderRequest{
						Cancel: &common.CancelOrderParam{
							Symbol: makerSymbol,
						},
					}:
					}
				}
			}
		}()
	}

	logger.Debugf("start main loop")
	for {
		select {
		case <-mGlobalCtx.Done():
			logger.Debugf("global ctx done, exit main loop")
			return
		case <-mExchange.Done():
			logger.Debugf("maker exchange done, exit main loop")
			return
		case mSystemStatus = <-mSystemStatusCh:
			if mSystemStatus != common.SystemStatusReady {
				logger.Debugf("mSystemStatus %v", mSystemStatus)
			}
			break
		case nextPos := <-mPositionCh:
			//logger.Debugf("maker position %s %v %f %f", nextPos.GetSymbol(), nextPos.GetTime(), nextPos.GetPrice(), nextPos.GetSize())
			if prevPos, ok := mPositions[nextPos.GetSymbol()]; ok {
				if nextPos.GetEventTime().Sub(prevPos.GetEventTime()) >= -time.Second {
					mPositions[nextPos.GetSymbol()] = nextPos
					if prevPos.GetSize() != nextPos.GetSize() {
						logger.Debugf("%s POS CHANGE %f -> %f", nextPos.GetSymbol(), prevPos.GetSize(), nextPos.GetSize())
						if walkedDepth, ok := mWalkedDepths[nextPos.GetSymbol()]; ok {
							mTimedPositionChange.Insert(time.Now(), math.Abs(prevPos.GetSize()-nextPos.GetSize())*walkedDepth.MidPrice)
						}
						if nextPos.GetSize() != 0 {
							mEnterSilentTimes[nextPos.GetSymbol()] = time.Now().Add(mConfig.EnterSilent)
						} else {
							mEnterSilentTimes[nextPos.GetSymbol()] = time.Now()
						}
					}
				}
			} else {
				logger.Debugf("%s POS CHANGE nil -> %f", nextPos.GetSymbol(), nextPos.GetSize())
				mPositions[nextPos.GetSymbol()] = nextPos
			}
			mPositionsUpdateTimes[nextPos.GetSymbol()] = time.Now()
			break
		case mAccount = <-mAccountCh:
			break
		case makerOrder := <-mOrderCh:
			if makerOrder.GetStatus() == common.OrderStatusExpired ||
				makerOrder.GetStatus() == common.OrderStatusReject ||
				makerOrder.GetStatus() == common.OrderStatusCancelled ||
				makerOrder.GetStatus() == common.OrderStatusFilled {
				if openOrder, ok := mOpenOrders[makerOrder.GetSymbol()]; ok && openOrder.ClientID == makerOrder.GetClientID() {
					delete(mOpenOrders, makerOrder.GetSymbol())
				}
				if makerOrder.GetStatus() == common.OrderStatusFilled {
					logger.Debugf(
						"ORDER %s FILLED SIDE %s TRADE SIZE %v TRADE PRICE %f",
						makerOrder.GetSymbol(), makerOrder.GetSide(), makerOrder.GetFilledSize(), makerOrder.GetFilledPrice(),
					)
				} else {
					//logger.Debugf("ORDER %s %s %s", makerOrder.GetSymbol(), makerOrder.GetStatus(), makerOrder.GetClientID())
					mOrderSilentTimes[makerOrder.GetSymbol()] = time.Now().Add(time.Second)
					mPositionsUpdateTimes[makerOrder.GetSymbol()] = time.Now()
				}
			}
			break

		case depth := <-wakedDepthCh:
			mWalkedDepths[depth.Symbol] = depth
			break
		case <-internalInfluxSaveTimer.C:
			handleInternalSave()
			internalInfluxSaveTimer.Reset(
				time.Now().Truncate(
					mConfig.InternalInflux.SaveInterval,
				).Add(
					mConfig.InternalInflux.SaveInterval + time.Second*15,
				).Sub(time.Now()),
			)
			break
		case <-externalInfluxSaveTimer.C:
			handleExternalInfluxSave()
			externalInfluxSaveTimer.Reset(
				time.Now().Truncate(
					mConfig.ExternalInflux.SaveInterval,
				).Add(
					mConfig.ExternalInflux.SaveInterval + time.Second*15,
				).Sub(time.Now()),
			)
			break
		case makerNewError := <-mNewOrderErrorCh:
			if makerNewError.Cancel != nil {
				mOrderSilentTimes[makerNewError.Cancel.Symbol] = time.Now().Add(mConfig.OrderSilent)
			} else if makerNewError.New != nil {
				mOrderSilentTimes[makerNewError.New.Symbol] = time.Now().Add(mConfig.OrderSilent)
			}
			break
		case report := <-mFilterRatioCh:
			mFilterRatios[report.Symbol] = report
			if report.Value > mConfig.EnterTriggerFilterRatio {
				mEnterTriggerTimes[report.Symbol] = time.Now()
			}
		case <-mLoopTimer.C:
			if mSystemStatus == common.SystemStatusReady {
				updateMakerOldOrders()
				updateMakerNewOrders()
			} else {
				if time.Now().Sub(time.Now().Truncate(time.Second*15)) < mConfig.LoopInterval {
					logger.Debugf(
						"system not ready mSystemStatus %v",
						mSystemStatus,
					)
				}
				if len(mOpenOrders) > 0 && !mConfig.DryRun {
					cancelAllMakerOpenOrders()
				}
			}
			mLoopTimer.Reset(
				time.Now().Truncate(
					mConfig.LoopInterval,
				).Add(
					mConfig.LoopInterval,
				).Sub(time.Now()),
			)
			break
		}
	}
}
