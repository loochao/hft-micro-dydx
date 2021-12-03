package okexv5_usdtspot

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

type OkexV5UsdtSpot struct {
	api      *API
	done     chan interface{}
	stopped  int32
	settings common.ExchangeSettings
}

func (okus *OkexV5UsdtSpot) GetPriceFactor() float64 {
	return 1.0
}

func (okus *OkexV5UsdtSpot) StreamSystemStatus(ctx context.Context, statusCh chan common.SystemStatus) {
	panic("implement me")
}

func (okus *OkexV5UsdtSpot) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (okus *OkexV5UsdtSpot) WatchBatchOrders(ctx context.Context, requestChannels map[string]chan common.BatchOrderRequest, responseChannels map[string]chan common.Order, errorChannels map[string]chan common.OrderError) {
	panic("implement me")
}

func (okus *OkexV5UsdtSpot) StartSideLoop() {
	panic("implement me")
}

func (okus *OkexV5UsdtSpot) IsSpot() bool {
	return true
}

func (okus *OkexV5UsdtSpot) StreamSymbolStatus(ctx context.Context, channels map[string]chan common.SymbolStatusMsg, batchSize int) {
	panic("implement me")
}

func (okus *OkexV5UsdtSpot) GenerateClientID() string {
	return fmt.Sprintf("M%d", time.Now().Unix()*10000+int64(rand.Intn(10000)))
}

func (okus *OkexV5UsdtSpot) GetMultiplier(symbol string) (float64, error) {
	return 1.0, nil
}

func (okus *OkexV5UsdtSpot) GetMinNotional(symbol string) (float64, error) {
	return 0, nil
}

func (okus *OkexV5UsdtSpot) GetMinSize(symbol string) (float64, error) {
	if value, ok := MinSizes[symbol]; ok {
		return value, nil
	} else {
		return 0, fmt.Errorf(common.MinSizeNotFoundError, symbol)
	}
}

func (okus *OkexV5UsdtSpot) GetStepSize(symbol string) (float64, error) {
	if value, ok := StepSizes[symbol]; ok {
		return value, nil
	} else {
		return 0, fmt.Errorf(common.StepSizeNotFoundError, symbol)
	}
}

func (okus *OkexV5UsdtSpot) GetTickSize(symbol string) (float64, error) {
	if value, ok := TickSizes[symbol]; ok {
		return value, nil
	} else {
		return 0, fmt.Errorf(common.TickSizeNotFoundError, symbol)
	}
}

func (okus *OkexV5UsdtSpot) StreamBasic(ctx context.Context, statusCh chan common.SystemStatus, usdtAccountCh chan common.Balance, commissionAssetValueCh chan float64, positionChMap map[string]chan common.Position, orderChMap map[string]chan common.Order) {
	defer okus.Stop()
	proxy := okus.settings.Proxy
	userWS := NewUserWebsocket(
		ctx,
		okus.settings.ApiKey, okus.settings.ApiSecret, okus.settings.ApiPassphrase,
		proxy,
	)
	balancesMap := make(map[string]*Balance, 0)
	internalAccountCh := make(chan Balance, 4)
	internalBalancesCh := make(chan []Balance, 4)
	go okus.watchAccount(ctx, internalAccountCh)
	go okus.watchBalances(ctx, internalBalancesCh)
	go okus.watchSystemStatus(ctx, statusCh)
	logSilentTime := time.Now()
	usdtAccount := Balance{
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
		case <-okus.done:
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
		case cashBalances := <-userWS.CashBalancesCh:
			for _, cashBalance := range cashBalances {
				if cashBalance.Ccy == "USDT" {
					if cashBalance.EventTime.Sub(usdtAccount.EventTime) > 0 {
						usdtAccount.Eq = cashBalance.CashBal
						outUsdtBalance := usdtAccount
						select {
						case usdtAccountCh <- &outUsdtBalance:
						default:
							if time.Now().Sub(logSilentTime) > 0 {
								logger.Debugf("usdtAccountCh <- usdtAccount failed, ch len %d", len(usdtAccountCh))
								logSilentTime = time.Now().Add(time.Minute)
							}
						}
					}
					continue
				}
				symbol := cashBalance.Ccy + "-USDT"
				if ch, ok := positionChMap[symbol]; ok {
					if lastBalance, ok2 := balancesMap[symbol]; ok2 && cashBalance.EventTime.Sub(lastBalance.EventTime) > 0 {
						outBalance := *lastBalance
						outBalance.EventTime = cashBalance.EventTime
						outBalance.ParseTime = cashBalance.ParseTime
						outBalance.Eq = cashBalance.CashBal
						balancesMap[symbol] = &outBalance
						select {
						case ch <- &outBalance:
						default:
							if time.Now().Sub(logSilentTime) > 0 {
								logger.Debugf("ch <- &outBalance failed, ch len %d", len(ch))
								logSilentTime = time.Now().Add(time.Minute)
							}
						}
					}
				}
			}
			break
		case balances := <-userWS.BalancesCh:
			for _, wsBalance := range balances {
				if wsBalance.Ccy == "USDT" {
					if wsBalance.EventTime.Sub(usdtAccount.EventTime) > 0 {
						usdtAccount.Eq = wsBalance.Eq
						usdtAccount.CashBal = wsBalance.CashBal
						usdtAccount.OrdFrozen = wsBalance.OrdFrozen
						usdtAccount.EventTime = wsBalance.EventTime
						usdtAccount.ParseTime = wsBalance.ParseTime
						outUsdtBalance := usdtAccount
						select {
						case usdtAccountCh <- &outUsdtBalance:
						default:
							if time.Now().Sub(logSilentTime) > 0 {
								logger.Debugf("usdtAccountCh <- usdtAccount failed, ch len %d", len(usdtAccountCh))
								logSilentTime = time.Now().Add(time.Minute)
							}
						}
					}
					continue
				}
				symbol := wsBalance.Ccy + "-USDT"
				if ch, ok := positionChMap[symbol]; ok {
					lastBalance, ok2 := balancesMap[symbol]
					if ok2 && wsBalance.EventTime.Sub(lastBalance.EventTime) < 0 {
						continue
					}
					outBalance  := wsBalance
					if ok2 {
						outBalance = *lastBalance
						outBalance.EventTime = wsBalance.EventTime
						outBalance.Ccy = wsBalance.Ccy
						outBalance.CashBal = wsBalance.CashBal
						outBalance.FrozenBal = wsBalance.FrozenBal
						outBalance.OrdFrozen = wsBalance.OrdFrozen
						outBalance.ParseTime = wsBalance.ParseTime
					}
					balancesMap[symbol] = &outBalance
					select {
					case ch <- &outBalance:
					default:
						if time.Now().Sub(logSilentTime) > 0 {
							logger.Debugf("ch <- balance failed, ch len %d", len(ch))
							logSilentTime = time.Now().Add(time.Minute)
						}
					}
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
		case balances := <-internalBalancesCh:
			hasBalances := make(map[string]bool)
			for _, balance := range balances {
				if balance.Ccy == "USDT" {
					continue
				}
				symbol := balance.Ccy + "-USDT"
				hasBalances[symbol] = true
				lastBalance, ok := balancesMap[symbol]
				if ok && balance.EventTime.Sub(lastBalance.EventTime) < 0 {
					continue
				}
				balance := balance
				balancesMap[symbol] = &balance
				if ch, ok := positionChMap[symbol]; ok {
					select {
					case ch <- &balance:
					default:
						if time.Now().Sub(logSilentTime) > 0 {
							logger.Debugf("ch <- balance failed, ch len %d", len(ch))
							logSilentTime = time.Now().Add(time.Minute)
						}
					}
				}
			}
			for symbol, ch := range positionChMap {
				if _, ok := hasBalances[symbol]; !ok {
					balance := &Balance{
						Ccy:       strings.Replace(symbol, "-USDT", "", -1),
						EventTime: time.Now(),
						ParseTime: time.Now(),
					}
					balancesMap[symbol] = balance
					select {
					case ch <- balance:
					default:
						if time.Now().Sub(logSilentTime) > 0 {
							logger.Debugf("ch <- balance failed, ch len %d", len(ch))
							logSilentTime = time.Now().Add(time.Minute)
						}
					}
				}
			}
			break
		}
	}

}

func (okus *OkexV5UsdtSpot) StreamDepth(ctx context.Context, channels map[string]chan common.Depth, batchSize int) {
	logger.Debugf("START StreamDepth")
	defer logger.Debugf("STOP StreamDepth")
	defer okus.Stop()

	symbols := make([]string, 0)
	for symbol := range channels {
		symbols = append(symbols, symbol)
	}

	proxy := okus.settings.Proxy

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
			defer okus.Stop()
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
		case <-okus.done:
			return
		}
	}
}

func (okus *OkexV5UsdtSpot) StreamTrade(ctx context.Context, channels map[string]chan common.Trade, batchSize int) {
	panic("implement me")
}

func (okus *OkexV5UsdtSpot) StreamTicker(ctx context.Context, channels map[string]chan common.Ticker, batchSize int) {
	logger.Debugf("START StreamTicker")
	defer logger.Debugf("STOP StreamTicker")
	defer okus.Stop()

	symbols := make([]string, 0)
	for symbol := range channels {
		symbols = append(symbols, symbol)
	}

	proxy := okus.settings.Proxy

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
			defer okus.Stop()
			ws1 := NewTickerWS(ctx, proxy, channels)
			ws2 := NewDepth5TickerWS(ctx, proxy, channels)
			for {
				select {
				case <-ctx.Done():
					return
				case <-ws1.Done():
					return
				case <-ws2.Done():
					return
				}
			}
		}(ctx, proxy, subChannels)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-okus.done:
			return
		}
	}
}

func (okus *OkexV5UsdtSpot) StreamKLine(ctx context.Context, channels map[string]chan []common.KLine, batchSize int, interval, lookback time.Duration) {
	panic("implement me")
}

func (okus *OkexV5UsdtSpot) StreamFundingRate(ctx context.Context, channels map[string]chan common.FundingRate, batchSize int) {
	pullInterval := okus.settings.PullInterval
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			for symbol, ch := range channels {
				select {
				case ch <- FundingRate{
					Symbol: symbol,
				}:
				default:
					logger.Debugf("ch <- &common.ZeroFundingRate failed %s ch len %d", symbol, len(ch))
				}
			}
			timer.Reset(time.Now().Truncate(pullInterval).Add(pullInterval).Sub(time.Now()))
		}
	}
}

func (okus *OkexV5UsdtSpot) WatchOrders(ctx context.Context, requestChannels map[string]chan common.OrderRequest, responseChannels map[string]chan common.Order, errorChannels map[string]chan common.OrderError) {
	defer okus.Stop()
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
		go okus.watchOrder(ctx, symbol, reqCh, respCh, errCh)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-okus.done:
			return
		}
	}
}

func (okus *OkexV5UsdtSpot) Setup(ctx context.Context, settings common.ExchangeSettings) (err error) {
	okus.done = make(chan interface{})
	okus.stopped = 0
	okus.settings = settings
	okus.api, err = NewAPI(&Credentials{
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
	}
	return
}

func (okus *OkexV5UsdtSpot) Stop() {
	if atomic.CompareAndSwapInt32(&okus.stopped, 0, 1) {
		logger.Debugf("stopped")
		close(okus.done)
	}
}

func (okus *OkexV5UsdtSpot) Done() chan interface{} {
	return okus.done
}

func (okus *OkexV5UsdtSpot) watchSystemStatus(
	ctx context.Context,
	output chan common.SystemStatus,
) {
	timer := time.NewTimer(time.Second*time.Duration(180 - rand.Intn(120)))
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-okus.done:
			return
		case <-timer.C:
			subCtx, _ := context.WithTimeout(ctx, time.Minute)
			statuses, err := okus.api.GetStatus(subCtx)
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
					if s.State == StateOngoing && (s.ServiceType == 0 || s.ServiceType == 1 || s.ServiceType == 5) {
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
			timer.Reset(time.Second*time.Duration(180 - rand.Intn(120)))
		}
	}
}

func (okus *OkexV5UsdtSpot) watchAccount(
	ctx context.Context,
	output chan Balance,
) {
	updateInterval := okus.settings.PullInterval
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			subCtx, _ := context.WithTimeout(ctx, time.Minute)
			balance, err := okus.api.GetAccount(subCtx)
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

func (okus *OkexV5UsdtSpot) watchBalances(
	ctx context.Context,
	output chan []Balance,
) {
	updateInterval := okus.settings.PullInterval
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			subCtx, _ := context.WithTimeout(ctx, time.Minute)
			balances, err := okus.api.GetBalances(subCtx)
			if err != nil {
				logger.Debugf("api.GetBalances error %v", err)
			} else {
				select {
				case output <- balances:
				default:
					logger.Debugf("output <- balances failed, ch len %d", len(output))
				}
			}
			timer.Reset(time.Now().Truncate(updateInterval).Add(updateInterval).Sub(time.Now()))
		}
	}
}

func (okus *OkexV5UsdtSpot) watchOrder(
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
		case <-okus.Done():
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
				okus.submitOrder(ctx, *req.New, responseCh, errorCh)
			} else if req.Cancel != nil && req.Cancel.ClientID != "" {
				//cancel need client order id
				okus.cancelOrder(ctx, *req.Cancel, errorCh)
			}
		}
	}
}

func (okus *OkexV5UsdtSpot) submitOrder(ctx context.Context, param common.NewOrderParam, respCh chan common.Order, errCh chan common.OrderError) {
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
		newOrderParam.OrderType = OrderTypeMarket
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
	_, err := okus.api.SubmitOrder(ctx, newOrderParam)
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
}

func (okus *OkexV5UsdtSpot) cancelOrder(ctx context.Context, param common.CancelOrderParam, errCh chan common.OrderError) {
	if param.Symbol != "" {
		_, err := okus.api.CancelOrders(ctx, CancelOrderParam{
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

type OkexV5UsdtSpotWithWalkedDepth5 struct {
	OkexV5UsdtSpot
}

func (okus *OkexV5UsdtSpotWithWalkedDepth5) StreamTicker(ctx context.Context, channels map[string]chan common.Ticker, batchSize int) {
	logger.Debugf("START StreamTicker")
	defer logger.Debugf("STOP StreamTicker")
	defer okus.Stop()

	walkImpact := okus.settings.DepthWalkValue
	if walkImpact <= 0 {
		walkImpact = 1.0
	}

	symbols := make([]string, 0)
	for symbol := range channels {
		symbols = append(symbols, symbol)
	}

	proxy := okus.settings.Proxy

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
			defer okus.Stop()
			ws1 := NewWalkedDepth5WS(ctx, proxy, walkImpact, channels)
			for {
				select {
				case <-ctx.Done():
					return
				case <-ws1.Done():
					return
				}
			}
		}(ctx, proxy, subChannels)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-okus.done:
			return
		}
	}
}
