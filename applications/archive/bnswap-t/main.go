package main

import (
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"os/signal"
	"syscall"
	"time"
)

//go:generate go run symbols_gen.go

func main() {
	logger.Debugf("####  BUILD @ %s ####", BUILD_TIME)
	st, err := NewStrategy()
	if err != nil {
		logger.Fatal(err)
	}
	defer st.Stop()

	positionsCh := make(chan []bnswap.Position)
	go bnswap.WatchPositionsFromHttp(
		st.GlobalCtx, st.API,
		SYMBOLS[:], *st.Config.PullInterval, positionsCh,
	)

	accountCh := make(chan bnswap.Account)
	go bnswap.WatchAccountFromHttp(
		st.GlobalCtx, st.API,
		*st.Config.PullInterval, accountCh,
	)

	tradeChs := [SYMBOLS_LEN]chan *bnswap.Trade{}
	markPriceCh := make(chan *bnswap.MarkPrice, SYMBOLS_LEN)
	quantileChs := [SYMBOLS_LEN]chan float64{}
	signalCh := make(chan Signal, SYMBOLS_LEN)
	for i, symbol := range SYMBOLS {
		tradeChs[i] = make(chan *bnswap.Trade, SYMBOLS_LEN*100)
		quantileChs[i] = make(chan float64, 10)
		go startQuantileRoutine(
			st.GlobalCtx,
			*st.Config.Quantile,
			*st.Config.QuantileUpdateInterval,
			tradeChs[i],
			quantileChs[i],
		)
		go startSignalRoutine(
			st.GlobalCtx,
			symbol,
			*st.Config.TradeLookback,
			*st.Config.TradeMinCount,
			*st.Config.Quantile,
			tradeChs[i],
			quantileChs[i],
			signalCh,
		)
		go startOrderRoutine(
			st.GlobalCtx,
			st.API,
			*st.Config.OrderTimeout,
			*st.Config.DryRun,
			st.OrderSubmittingChs[i],
			st.OrderNewErrorCh,
			st.OrderFinishCh,
		)
	}
	for start := 0; start < SYMBOLS_LEN; start += *st.Config.SymbolBatchSize {
		end := start + *st.Config.SymbolBatchSize
		if end > SYMBOLS_LEN {
			end = SYMBOLS_LEN
		}
		go startTradesRoutine(
			st.GlobalCtx, *st.Config.ProxyAddress,
			SYMBOLS[start:end],
			tradeChs,
		)
		go startMarkPriceRoutine(
			st.GlobalCtx, *st.Config.ProxyAddress,
			SYMBOLS[start:end],
			markPriceCh,
		)
	}

	loopTimer := time.NewTimer(time.Minute) //先等1分钟
	targetValueUpdateTimer := time.NewTimer(time.Hour * 24)
	resetUnrealisedPnlTimer := time.NewTimer(time.Minute)
	frRankUpdatedTimer := time.NewTimer(time.Second * 60)
	//bnbReBalanceTimer := time.NewTimer(*bnConfig.BnbCheckInterval)

	influxSaveTimer := time.NewTimer(time.Minute)
	defer influxSaveTimer.Stop()
	defer loopTimer.Stop()
	defer targetValueUpdateTimer.Stop()
	defer resetUnrealisedPnlTimer.Stop()
	defer frRankUpdatedTimer.Stop()

	userWS := bnswap.NewUserWebsocket(
		st.GlobalCtx,
		st.API,
		*st.Config.ProxyAddress,
	)
	defer userWS.Stop()

	done := make(chan bool, 1)
	if *st.Config.CpuProfile != "" {
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
		case ps := <-positionsCh:
			st.handleHttpPosition(ps)
			break
		case account := <-accountCh:
			st.handleSwapHttpAccount(account)
			break
		case markPrice := <-markPriceCh:
			st.handleMarkPrice(markPrice)
			break
		case s := <-signalCh:
			st.handleSignal(s)
			break
		case msg := <-userWS.BalanceAndPositionUpdateEventCh:
			st.handleWSAccountEvent(msg)
			break
		case msg := <-userWS.OrderUpdateEventCh:
			st.handleWSOrder(&msg.Order)
			break
		case newError := <-st.OrderNewErrorCh:
			symbolIndex := GetSymbolIndex(newError.Params.Symbol)
			if symbolIndex == -1 {
				break
			}
			st.OrderSilentTimes[symbolIndex] = time.Now().Add(time.Second * 15)
			break
		case order := <-st.OrderFinishCh:
			symbolIndex := GetSymbolIndex(order.Symbol)
			if symbolIndex == -1 {
				break
			}
			//logStr := fmt.Sprintf("SWAP ORDER %s", order.ToString())
			if order.Status == "REJECTED" || order.Status == "EXPIRED" {
				//logStr = fmt.Sprintf("%s RESET TIMEOUT", logStr)
				st.OrderSilentTimes[symbolIndex] = time.Now().Add(time.Second)
				st.PositionsUpdateTimes[symbolIndex] = time.Unix(0, 0)
			}
			//logger.Debug(logStr)
			break
		case <-influxSaveTimer.C:
			st.handleInternalInfluxSave()
			st.handleExternalInfluxSave()
			influxSaveTimer.Reset(
				time.Now().Truncate(
					*st.Config.InternalInflux.SaveInterval,
				).Add(
					*st.Config.InternalInflux.SaveInterval,
				).Sub(time.Now()),
			)
			break
		}
	}
}
