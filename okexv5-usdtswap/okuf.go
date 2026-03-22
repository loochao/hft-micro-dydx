package okexv5_usdtswap

import (
	"context"
	"errors"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math/rand"
	"strings"
	"sync/atomic"
	"time"
)

type OkexV5UsdtSwap struct {
	api      *API
	done     chan interface{}
	stopped  int32
	settings common.ExchangeSettings
}

func (okuf *OkexV5UsdtSwap) GetPriceFactor() float64 {
	return 1.0
}

func (okuf *OkexV5UsdtSwap) StreamSystemStatus(ctx context.Context, statusCh chan common.SystemStatus) {
	panic("implement me")
}

func (okuf *OkexV5UsdtSwap) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (okuf *OkexV5UsdtSwap) WatchBatchOrders(ctx context.Context, requestChannels map[string]chan common.BatchOrderRequest, responseChannels map[string]chan common.Order, errorChannels map[string]chan common.OrderError) {
	panic("implement me")
}

func (okuf *OkexV5UsdtSwap) StartSideLoop() {
	panic("implement me")
}

func (okuf *OkexV5UsdtSwap) IsSpot() bool {
	return false
}

func (okuf *OkexV5UsdtSwap) StreamSymbolStatus(ctx context.Context, channels map[string]chan common.SymbolStatusMsg, batchSize int) {
	panic("implement me")
}

func (okuf *OkexV5UsdtSwap) GenerateClientID() string {
	return fmt.Sprintf("M%d", time.Now().Unix()*10000+int64(rand.Intn(10000)))
}

func (okuf *OkexV5UsdtSwap) GetMultiplier(symbol string) (float64, error) {
	if value, ok := Multipliers[symbol]; ok {
		return value, nil
	} else {
		return 0, fmt.Errorf(common.MultiplierNotFoundError, symbol)
	}
}

func (okuf *OkexV5UsdtSwap) GetMinNotional(symbol string) (float64, error) {
	return 0, nil
}

func (okuf *OkexV5UsdtSwap) GetMinSize(symbol string) (float64, error) {
	if value, ok := MinSizes[symbol]; ok {
		return value, nil
	} else {
		return 0, fmt.Errorf(common.MinSizeNotFoundError, symbol)
	}
}

func (okuf *OkexV5UsdtSwap) GetStepSize(symbol string) (float64, error) {
	if value, ok := StepSizes[symbol]; ok {
		return value, nil
	} else {
		return 0, fmt.Errorf(common.StepSizeNotFoundError, symbol)
	}
}

func (okuf *OkexV5UsdtSwap) GetTickSize(symbol string) (float64, error) {
	if value, ok := TickSizes[symbol]; ok {
		return value, nil
	} else {
		return 0, fmt.Errorf(common.TickSizeNotFoundError, symbol)
	}
}

func (okuf *OkexV5UsdtSwap) StreamBasic(ctx context.Context, statusCh chan common.SystemStatus, usdtAccountCh chan common.Balance, commissionAssetValueCh chan float64, positionChMap map[string]chan common.Position, orderChMap map[string]chan common.Order) {
	defer okuf.Stop()
	proxy := okuf.settings.Proxy
	userWS := NewUserWebsocket(
		ctx,
		okuf.settings.ApiKey, okuf.settings.ApiSecret, okuf.settings.ApiPassphrase,
		proxy,
	)
	positionsMap := make(map[string]*Position, 0)
	internalAccountCh := make(chan Account, 4)
	internalPositionsCh := make(chan []Position, 4)
	go okuf.watchAccount(ctx, internalAccountCh)
	go okuf.watchPositions(ctx, internalPositionsCh)
	go okuf.watchSystemStatus(ctx, statusCh)
	logSilentTime := time.Now()
	usdtAccount := Account{
		Ccy: "USDT",
	}

	restartToReadyTimer := time.NewTimer(time.Hour * 9999)
	defer restartToReadyTimer.Stop()
	commissionTimer := time.NewTimer(time.Second)
	defer commissionTimer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-okuf.done:
			return
		case <-userWS.done:
			return
		case <-restartToReadyTimer.C:
			select {
			case statusCh <- common.SystemStatusReady:
			default:
				logger.Debugf("statusCh <- common.SystemStatusRestart failed ch len %d", len(statusCh))
			}
			restartToReadyTimer = time.NewTimer(time.Hour * 9999)
			break
		case <-userWS.RestartCh:
			select {
			case statusCh <- common.SystemStatusRestart:
				restartToReadyTimer.Reset(time.Minute * 3)
			default:
				logger.Debugf("statusCh <- common.SystemStatusRestart failed ch len %d", len(statusCh))
			}
			break
		case <-commissionTimer.C:
			select {
			case commissionAssetValueCh <- 0.0:
			default:
				logger.Debugf("commissionAssetValueCh <- 0.0 failed ch len %d", len(commissionAssetValueCh))
			}
			commissionTimer.Reset(time.Minute)
			break
		case positions := <-userWS.PositionsCh:
			for _, nextPos := range positions {
				if nextPos.Ccy != "USDT" {
					continue
				}
				if nextPos.MgnMode != "cross" {
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("%s bad position margin mode %s %v", nextPos.InstId, nextPos.MgnMode, nextPos)
					}
					continue
				}
				if nextPos.PosSide != "net" {
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("%s bad position side %s %v", nextPos.InstId, nextPos.PosSide, nextPos)
					}
					continue
				}
				if ch, ok1 := positionChMap[nextPos.InstId]; ok1 {
					if lastPos, ok2 := positionsMap[nextPos.InstId]; ok2 && nextPos.UTime.Sub(lastPos.UTime) > 0 {
						position := nextPos
						positionsMap[position.InstId] = &position
						select {
						case ch <- &position:
						default:
							if time.Now().Sub(logSilentTime) > 0 {
								logger.Debugf("ch <- &nextPos failed, ch len %d", len(ch))
								logSilentTime = time.Now().Add(time.Minute)
							}
						}
					}
				}
			}
			break
		case accounts := <-userWS.AccountsCh:
			for _, account := range accounts {
				if account.Ccy == "USDT" {
					if account.UTime.Sub(usdtAccount.UTime) > 0 {
						usdtAccount.AvailEq = account.AvailEq
						usdtAccount.CashBal = account.CashBal
						usdtAccount.DisEq = account.DisEq
						usdtAccount.Eq = account.Eq
						usdtAccount.EqUsd = account.EqUsd
						usdtAccount.FrozenBal = account.FrozenBal
						usdtAccount.MgnRatio = account.MgnRatio
						usdtAccount.NotionalLever = account.NotionalLever
						usdtAccount.Upl = account.Upl
						usdtAccount.UTime = account.UTime
						usdtAccount.ParseTime = account.ParseTime
						outUsdtBalance := usdtAccount
						select {
						case usdtAccountCh <- &outUsdtBalance:
						default:
							if time.Now().Sub(logSilentTime) > 0 {
								logger.Debugf("usdtAccountCh <- &outUsdtBalance failed, ch len %d", len(usdtAccountCh))
								logSilentTime = time.Now().Add(time.Minute)
							}
						}
					}
					break
				}
			}
			break
		case orders := <-userWS.OrdersCh:
			for _, wsOrder := range orders {
				if ch, ok := orderChMap[wsOrder.InstId]; ok {
					order := wsOrder
					select {
					case ch <- &order:
					default:
						if time.Now().Sub(logSilentTime) > 0 {
							logger.Debugf("ch <- &order failed, ch len %d", len(ch))
							logSilentTime = time.Now().Add(time.Minute)
						}
					}
				}
			}
			break
		case balance := <-internalAccountCh:
			usdtAccount = balance
			select {
			case usdtAccountCh <- &balance:
			default:
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("usdtAccountCh <- usdtAccount failed, ch len %d", len(usdtAccountCh))
					logSilentTime = time.Now().Add(time.Minute)
				}
			}
			break
		case positions := <-internalPositionsCh:
			hasPositions := make(map[string]bool)
			for _, nextPos := range positions {
				if nextPos.Ccy != "USDT" {
					continue
				}
				if nextPos.MgnMode != "cross" {
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("%s bad position margin mode %s %v", nextPos.InstId, nextPos.MgnMode, nextPos)
					}
					continue
				}
				if nextPos.PosSide != "net" {
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("%s bad position side %s %v", nextPos.InstId, nextPos.PosSide, nextPos)
					}
					continue
				}
				hasPositions[nextPos.InstId] = true
				lastPos, ok := positionsMap[nextPos.InstId]
				if ok && nextPos.UTime.Sub(lastPos.UTime) < 0 {
					continue
				}
				position := nextPos
				positionsMap[position.InstId] = &position
				if ch, ok := positionChMap[position.InstId]; ok {
					select {
					case ch <- &position:
					default:
						if time.Now().Sub(logSilentTime) > 0 {
							logger.Debugf("ch <- &position failed, ch len %d", len(ch))
							logSilentTime = time.Now().Add(time.Minute)
						}
					}
				}
			}
			for symbol, ch := range positionChMap {
				if _, ok := hasPositions[symbol]; !ok {
					position := &Position{
						InstId:    symbol,
						InstType:  "SWAP",
						MgnMode:   "cross",
						PosSide:   "net",
						Ccy:       "USDT",
						UTime:     time.Now(),
						ParseTime: time.Now(),
					}
					positionsMap[symbol] = position
					select {
					case ch <- position:
					default:
						if time.Now().Sub(logSilentTime) > 0 {
							logger.Debugf("ch <- position failed, ch len %d", len(ch))
							logSilentTime = time.Now().Add(time.Minute)
						}
					}
				}
			}
			break
		}
	}

}

func (okuf *OkexV5UsdtSwap) StreamDepth(ctx context.Context, channels map[string]chan common.Depth, batchSize int) {
	logger.Debugf("START StreamDepth")
	defer logger.Debugf("STOP StreamDepth")
	defer okuf.Stop()

	symbols := make([]string, 0)
	for symbol := range channels {
		symbols = append(symbols, symbol)
	}

	proxy := okuf.settings.Proxy

	for start := 0; start < len(symbols); start += batchSize {
		end := start + batchSize
		if end > len(symbols) {
			end = len(symbols)
		}
		subChannels := make(map[string]chan common.Depth)
		for _, symbol := range symbols[start:end] {
			subChannels[symbol] = channels[symbol]
		}
		go func(ctx context.Context, proxy string, channels map[string]chan common.Depth) {
			defer okuf.Stop()
			ws := NewDepth5WS(ctx, proxy, channels)
			for {
				select {
				case <-ctx.Done():
					return
				case <-ws.Done():
					return
				}
			}
		}(ctx, proxy, subChannels)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-okuf.done:
			return
		}
	}
}

func (okuf *OkexV5UsdtSwap) StreamTrade(ctx context.Context, channels map[string]chan common.Trade, batchSize int) {
	panic("implement me")
}

func (okuf *OkexV5UsdtSwap) StreamTicker(ctx context.Context, channels map[string]chan common.Ticker, batchSize int) {
	logger.Debugf("START StreamTicker")
	defer logger.Debugf("STOP StreamTicker")
	defer okuf.Stop()

	symbols := make([]string, 0)
	for symbol := range channels {
		symbols = append(symbols, symbol)
	}

	proxy := okuf.settings.Proxy

	for start := 0; start < len(symbols); start += batchSize {
		end := start + batchSize
		if end > len(symbols) {
			end = len(symbols)
		}
		subChannels := make(map[string]chan common.Ticker)
		for _, symbol := range symbols[start:end] {
			subChannels[symbol] = channels[symbol]
		}
		go func(ctx context.Context, proxy string, channels map[string]chan common.Ticker) {
			defer okuf.Stop()
			ws1 := NewTickerWS(ctx, proxy, channels)
			ws2 := NewDepth5TickerWS(ctx, proxy, channels)
			for {
				select {
				case <-ctx.Done():
					return
				case <-ws1.Done():
					logger.Debugf("ticker ws done")
					return
				case <-ws2.Done():
					logger.Debugf("depth ws done")
					return
				}
			}
		}(ctx, proxy, subChannels)
	}
	for {
		select {
		case <-ctx.Done():
			logger.Debugf("ctx done")
			return
		case <-okuf.done:
			logger.Debugf("okuf done")
			return
		}
	}
}

func (okuf *OkexV5UsdtSwap) StreamKLine(ctx context.Context, channels map[string]chan []common.KLine, batchSize int, interval, lookback time.Duration) {
	panic("implement me")
}

func (okuf *OkexV5UsdtSwap) StreamFundingRate(ctx context.Context, channels map[string]chan common.FundingRate, batchSize int) {
	logger.Debugf("START StreamFundingRate")
	defer logger.Debugf("STOP StreamFundingRate")
	defer okuf.Stop()

	symbols := make([]string, 0)
	for symbol := range channels {
		symbols = append(symbols, symbol)
	}

	proxy := okuf.settings.Proxy

	for start := 0; start < len(symbols); start += batchSize {
		end := start + batchSize
		if end > len(symbols) {
			end = len(symbols)
		}
		subChannels := make(map[string]chan common.FundingRate)
		for _, symbol := range symbols[start:end] {
			subChannels[symbol] = channels[symbol]
		}
		go func(ctx context.Context, proxy string, channels map[string]chan common.FundingRate) {
			defer okuf.Stop()
			ws1 := NewFundingRateWS(ctx, proxy, channels)
			for {
				select {
				case <-ctx.Done():
					return
				case <-ws1.Done():
					logger.Debugf("ticker ws done")
					return
				}
			}
		}(ctx, proxy, subChannels)
	}
	for {
		select {
		case <-ctx.Done():
			logger.Debugf("ctx done")
			return
		case <-okuf.done:
			logger.Debugf("okuf done")
			return
		}
	}
}

func (okuf *OkexV5UsdtSwap) WatchOrders(ctx context.Context, requestChannels map[string]chan common.OrderRequest, responseChannels map[string]chan common.Order, errorChannels map[string]chan common.OrderError) {
	defer okuf.Stop()
	for symbol, reqCh := range requestChannels {
		respCh, ok := responseChannels[symbol]
		if !ok {
			logger.Debugf("miss response ch for %s, exit", symbol)
			return
		}
		errCh, ok := errorChannels[symbol]
		if !ok {
			logger.Debugf("miss error ch for %s, exit", symbol)
			return
		}
		go okuf.watchOrder(ctx, symbol, reqCh, respCh, errCh)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-okuf.done:
			return
		}
	}
}

func (okuf *OkexV5UsdtSwap) Setup(ctx context.Context, settings common.ExchangeSettings) (err error) {
	okuf.done = make(chan interface{})
	okuf.stopped = 0
	okuf.settings = settings
	okuf.api, err = NewAPI(&Credentials{
		Key:        settings.ApiKey,
		Secret:     settings.ApiSecret,
		Passphrase: settings.ApiPassphrase,
	}, settings.Proxy)
	if err != nil {
		return
	}
	for _, symbol := range settings.Symbols {
		if _, ok := TickSizes[symbol]; !ok {
			return fmt.Errorf("tick size not found for %s", symbol)
		}
		if _, ok := StepSizes[symbol]; !ok {
			return fmt.Errorf("step size not found for %s", symbol)
		}
		if _, ok := MinSizes[symbol]; !ok {
			return fmt.Errorf("min size not found for %s", symbol)
		}
		if _, ok := Multipliers[symbol]; !ok {
			return fmt.Errorf("multiplier found for %s", symbol)
		}
		if settings.ChangeLeverage {
			err = okuf.api.UpdateLeverage(ctx, Leverage{
				InstId: symbol,
				Lever:  int(settings.Leverage),
			})
			if err != nil {
				logger.Debugf("UPDATE LEVERAGE FOR %s ERROR %v", symbol, err)
			} else {
				logger.Debugf("UPDATE LEVERAGE TO %.0f FOR %s ", settings.Leverage, symbol)
			}
			time.Sleep(time.Second)
		}
	}
	return
}

func (okuf *OkexV5UsdtSwap) Stop() {
	if atomic.CompareAndSwapInt32(&okuf.stopped, 0, 1) {
		logger.Debugf("stopped")
		close(okuf.done)
	}
}

func (okuf *OkexV5UsdtSwap) Done() chan interface{} {
	return okuf.done
}

func (okuf *OkexV5UsdtSwap) watchSystemStatus(
	ctx context.Context,
	output chan common.SystemStatus,
) {
	timer := time.NewTimer(time.Second*time.Duration(90 - rand.Intn(30)))
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-okuf.done:
			return
		case <-timer.C:
			subCtx, _ := context.WithTimeout(ctx, time.Minute)
			statuses, err := okuf.api.GetStatus(subCtx)
			if err != nil {
				logger.Debugf("api.GetStatus(subCtx) error %v", err)
				if !strings.Contains(err.Error(), "Requests too frequent") {
					select {
					case output <- common.SystemStatusError:
					default:
						logger.Debugf("output <- common.SystemStatusError, failed ch len %d", len(output))
					}
				}
			} else {
				ready := true
				for _, s := range statuses {
					if s.State == StateOngoing && (s.ServiceType == 0 || s.ServiceType == 3 || s.ServiceType == 5) {
						ready = false
					}
				}
				if ready {
					select {
					case output <- common.SystemStatusReady:
					default:
						logger.Debugf("output <- common.SystemStatusReady %v, failed ch len %d", ready, len(output))
					}
				} else {
					select {
					case output <- common.SystemStatusNotReady:
					default:
						logger.Debugf("output <- common.SystemStatusNotReady %v, failed ch len %d", ready, len(output))
					}
				}
			}
			timer.Reset(time.Second*time.Duration(90 - rand.Intn(30)))
		}
	}
}

func (okuf *OkexV5UsdtSwap) watchAccount(
	ctx context.Context,
	output chan Account,
) {
	updateInterval := okuf.settings.PullInterval
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			subCtx, _ := context.WithTimeout(ctx, time.Minute)
			balance, err := okuf.api.GetAccount(subCtx)
			if err != nil {
				logger.Debugf("api.GetAccount error %v", err)
			} else {
				select {
				case output <- *balance:
				default:
					logger.Debugf("output <- balance failed, ch len %d", len(output))
				}
			}
			timer.Reset(time.Now().Truncate(updateInterval).Add(updateInterval).Sub(time.Now()))
		}
	}
}

func (okuf *OkexV5UsdtSwap) watchPositions(
	ctx context.Context,
	output chan []Position,
) {
	updateInterval := okuf.settings.PullInterval
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			subCtx, _ := context.WithTimeout(ctx, time.Minute)
			positions, err := okuf.api.GetPositions(subCtx)
			if err != nil {
				logger.Debugf("api.GetBalances error %v", err)
			} else {
				select {
				case output <- positions:
				default:
					logger.Debugf("output <- positions failed, ch len %d", len(output))
				}
			}
			timer.Reset(time.Now().Truncate(updateInterval).Add(updateInterval).Sub(time.Now()))
		}
	}
}

func (okuf *OkexV5UsdtSwap) watchOrder(
	ctx context.Context,
	market string,
	requestCh chan common.OrderRequest,
	responseCh chan common.Order,
	errorCh chan common.OrderError,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-okuf.Done():
			return
		case req := <-requestCh:
			if req.New != nil {
				if req.New.Symbol != market {
					select {
					case errorCh <- common.OrderError{
						New:   req.New,
						Error: errors.New(fmt.Sprintf("bad create request market not match %s %s", req.New.Symbol, market)),
					}:
					default:
						logger.Debugf("errorCh <- common.OrderError failed, ch len %d", len(errorCh))
					}
					continue
				}
				okuf.submitOrder(ctx, *req.New, responseCh, errorCh)
			} else if req.Cancel != nil && req.Cancel.ClientID != "" {
				//cancel need client order id
				okuf.cancelOrder(ctx, *req.Cancel, errorCh)
			}
		}
	}
}

func (okuf *OkexV5UsdtSwap) submitOrder(ctx context.Context, param common.NewOrderParam, respCh chan common.Order, errCh chan common.OrderError) {
	newOrderParam := NewOrderParam{
		TdMode: TdModeCash,
	}
	newOrderParam.InstId = param.Symbol
	newOrderParam.Size = common.FormatFloat(param.Size, StepPrecisions[param.Symbol])
	if param.Side == common.OrderSideBuy {
		newOrderParam.Side = OrderSideBuy
	} else {
		newOrderParam.Side = OrderSideSell
	}
	if param.Type == common.OrderTypeMarket {
		newOrderParam.OrderType = OrderTypeOptimalLimitIoc
	} else {
		newOrderParam.OrderType = OrderTypeLimit
	}
	if param.PostOnly {
		newOrderParam.OrderType = OrderTypePostOnly
	} else {
		switch param.TimeInForce {
		case common.OrderTimeInForceIOC:
			newOrderParam.OrderType = OrderTypeIOC
		case common.OrderTimeInForceFOK:
			newOrderParam.OrderType = OrderTypeFOK
		}
	}
	if param.Price != 0 {
		newOrderParam.Price = new(string)
		*newOrderParam.Price = common.FormatFloat(param.Price, TickPrecisions[param.Symbol])
	}
	newOrderParam.ClOrdId = param.ClientID
	_, err := okuf.api.SubmitOrder(ctx, newOrderParam)
	if err != nil {
		select {
		case errCh <- common.OrderError{
			New:   &param,
			Error: err,
		}:
		default:
			logger.Debugf("errCh <- common.OrderError failed, ch len %d", len(errCh))
		}
	}
	//if newOrderParam.OrderType ==  OrderTypeMarket {
	//	_, err = okuf.api.CancelOrders(ctx, CancelOrderParam{
	//		InstId: newOrderParam.InstId,
	//		ClOrdId: newOrderParam.ClOrdId,
	//	})
	//	if err != nil {
	//		logger.Debugf("okuf.api.CancelOrders %v", err)
	//	}
	//}
}

func (okuf *OkexV5UsdtSwap) cancelOrder(ctx context.Context, param common.CancelOrderParam, errCh chan common.OrderError) {
	if param.Symbol != "" {
		_, err := okuf.api.CancelOrders(ctx, CancelOrderParam{
			ClOrdId: param.ClientID,
			InstId:  param.Symbol,
		})
		if err != nil {
			select {
			case errCh <- common.OrderError{
				Cancel: &param,
				Error:  err,
			}:
			default:
				logger.Debugf("errCh <- common.OrderError failed, ch len %d", len(errCh))
			}
		}
	}
}

type OkexV5UsdtSwapWithWalkedDepth5 struct {
	OkexV5UsdtSwap
}

func (okuf *OkexV5UsdtSwapWithWalkedDepth5) StreamTicker(ctx context.Context, channels map[string]chan common.Ticker, batchSize int) {
	logger.Debugf("START StreamTicker")
	defer logger.Debugf("STOP StreamTicker")
	defer okuf.Stop()

	walkImpact := okuf.settings.DepthWalkValue
	if walkImpact <= 0 {
		walkImpact = 1.0
	}
	symbols := make([]string, 0)
	for symbol := range channels {
		symbols = append(symbols, symbol)
	}

	proxy := okuf.settings.Proxy

	for start := 0; start < len(symbols); start += batchSize {
		end := start + batchSize
		if end > len(symbols) {
			end = len(symbols)
		}
		subChannels := make(map[string]chan common.Ticker)
		for _, symbol := range symbols[start:end] {
			subChannels[symbol] = channels[symbol]
		}
		go func(ctx context.Context, proxy string, channels map[string]chan common.Ticker) {
			defer okuf.Stop()
			ws1 := NewWalkedDepth5WS(ctx, proxy, walkImpact, channels)
			for {
				select {
				case <-ctx.Done():
					return
				case <-ws1.Done():
					logger.Debugf("ticker ws done")
					return
				}
			}
		}(ctx, proxy, subChannels)
	}
	for {
		select {
		case <-ctx.Done():
			logger.Debugf("ctx done")
			return
		case <-okuf.done:
			logger.Debugf("okuf done")
			return
		}
	}
}
