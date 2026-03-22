package bnspot

import (
	"context"
	"errors"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"
)

type Bnspot struct {
	api      *API
	done     chan interface{}
	stopped  bool
	mu       sync.Mutex
	settings common.ExchangeSettings
}

func (bn *Bnspot) IsSpot() bool {
	return true
}

func (bn *Bnspot) StreamSymbolStatus(ctx context.Context, channels map[string]chan common.SymbolStatusMsg, batchSize int) {
	panic("implement me")
}

func (bn *Bnspot) GenerateClientID() string {
	return fmt.Sprintf("%d%04d", time.Now().Unix(), rand.Intn(10000))
}

func (bn *Bnspot) GetMinNotional(symbol string) (float64, error) {
	if value, ok := MinNotionals[symbol]; ok {
		return value, nil
	} else {
		return 0, fmt.Errorf(common.MinNotionalNotFoundError, symbol)
	}
}

func (bn *Bnspot) GetMinSize(symbol string) (float64, error) {
	if value, ok := MinSizes[symbol]; ok {
		return value, nil
	} else {
		return 0, fmt.Errorf(common.MinSizeNotFoundError, symbol)
	}
}

func (bn *Bnspot) GetStepSize(symbol string) (float64, error) {
	if value, ok := StepSizes[symbol]; ok {
		return value, nil
	} else {
		return 0, fmt.Errorf(common.StepSizeNotFoundError, symbol)
	}
}

func (bn *Bnspot) GetTickSize(symbol string) (float64, error) {
	if value, ok := TickSizes[symbol]; ok {
		return value, nil
	} else {
		return 0, fmt.Errorf(common.TickSizeNotFoundError, symbol)
	}
}

func (bn *Bnspot) StreamBasic(ctx context.Context, statusCh chan common.SystemStatus, accountCh chan common.Account, positionChMap map[string]chan common.Position, orderChMap map[string]chan common.Order) {
	defer bn.Stop()
	bn.mu.Lock()
	proxy := bn.settings.Proxy
	bn.mu.Unlock()
	userWS, err := NewUserWebsocket(ctx, bn.api, proxy)
	if err != nil {
		logger.Debugf("NewUserWebsocket(ctx,  bn.api, proxy) error %v", err)
		return
	}
	balancesMap := make(map[string]*Balance, 0)
	for symbol := range positionChMap {
		asset := strings.Replace(symbol, "USDT", "", -1)
		balancesMap[symbol] = &Balance{
			Asset:     asset,
			Free:      0,
			Locked:    0,
			EventTime: time.Time{},
			ParseTime: time.Time{},
		}
	}
	internalAccountCh := make(chan Account, 10)
	go bn.watchAccount(ctx, internalAccountCh)
	go bn.watchSystemStatus(ctx, statusCh)
	logSilentTime := time.Now()

	usdtBalance := Balance{
		Asset:     "USDT",
		Free:      0,
		Locked:    0,
		EventTime: time.Time{},
		ParseTime: time.Time{},
	}

	restartToReadyTimer := time.NewTimer(time.Hour * 9999)
	defer restartToReadyTimer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-bn.done:
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
		case account := <-userWS.AccountUpdateEventCh:
			for _, wsBalance := range account.Balances {
				if wsBalance.Asset == "USDT" {
					if wsBalance.EventTime.Sub(usdtBalance.EventTime) > 0 {
						usdtBalance.Free = wsBalance.FreeAmount
						usdtBalance.Locked = wsBalance.LockedAmount
						usdtBalance.EventTime = wsBalance.EventTime
						usdtBalance.ParseTime = wsBalance.ParseTime
						outUsdtBalance := usdtBalance
						select {
						case accountCh <- &outUsdtBalance:
						default:
							if time.Now().Sub(logSilentTime) > 0 {
								logger.Debugf("accountCh <- usdtBalance failed, ch len %d", len(accountCh))
								logSilentTime = time.Now().Add(time.Minute)
							}
						}
					}
					continue
				}
				symbol := wsBalance.Asset + "USDT"
				lastBalance, ok := balancesMap[wsBalance.Asset]
				if !ok || wsBalance.EventTime.Sub(lastBalance.EventTime) < 0 {
					continue
				}
				balancesMap[symbol].EventTime = wsBalance.EventTime
				balancesMap[symbol].ParseTime = wsBalance.ParseTime
				balancesMap[symbol].Free = wsBalance.FreeAmount
				balancesMap[symbol].Locked = wsBalance.LockedAmount
				if ch, ok := positionChMap[symbol]; ok {
					outBalance := *balancesMap[symbol]
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
		case wsOrder := <-userWS.OrderUpdateEventCh:
			if ch, ok := orderChMap[wsOrder.Symbol]; ok {
				select {
				case ch <- wsOrder:
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("ch <- &wsOrder failed, ch len %d", len(ch))
						logSilentTime = time.Now().Add(time.Minute)
					}
				}
			}
			break
		case account := <-internalAccountCh:
			hasBalances := make(map[string]bool)
			for _, balance := range account.Balances {
				balance := balance
				if balance.Asset == "USDT" {
					if balance.EventTime.Sub(usdtBalance.EventTime) > 0 {
						usdtBalance = balance
						select {
						case accountCh <- &balance:
						default:
							if time.Now().Sub(logSilentTime) > 0 {
								logger.Debugf("accountCh <- usdtBalance failed, ch len %d", len(accountCh))
								logSilentTime = time.Now().Add(time.Minute)
							}
						}
					}
					continue
				}
				symbol := balance.Asset + "USDT"
				lastBalance, ok := balancesMap[balance.Asset]
				if ok && balance.EventTime.Sub(lastBalance.EventTime) < 0 {
					continue
				}
				balancesMap[symbol] = &balance
			}
			for symbol := range balancesMap {
				if ch, ok := positionChMap[symbol]; ok {
					//logger.Debugf("%s %v %v", balancesMap[symbol].Asset, balancesMap[symbol].EventTime, balancesMap[symbol].ParseTime)
					hasBalances[symbol] = true
					outBalance := *balancesMap[symbol]
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
			for symbol, ch := range positionChMap {
				if _, ok := hasBalances[symbol]; !ok {
					select {
					case ch <- &Balance{
						Asset: strings.Replace(symbol, "USDT", "", -1),
						EventTime: account.EventTime,
						ParseTime: account.ParseTime,
					}:
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

func (bn *Bnspot) StreamDepth(ctx context.Context, channels map[string]chan common.Depth, batchSize int) {
	logger.Debugf("START StreamDepth")
	defer logger.Debugf("STOP StreamDepth")
	defer bn.Stop()

	symbols := make([]string, 0)
	for symbol := range channels {
		symbols = append(symbols, symbol)
	}

	bn.mu.Lock()
	proxy := bn.settings.Proxy
	bn.mu.Unlock()

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
		case <-bn.done:
			return
		}
	}
}

func (bn *Bnspot) StreamTrade(ctx context.Context, channels map[string]chan common.Trade, batchSize int) {
	logger.Debugf("START StreamTrade")
	defer logger.Debugf("STOP StreamTrade")
	defer bn.Stop()

	symbols := make([]string, 0)
	for symbol := range channels {
		symbols = append(symbols, symbol)
	}

	bn.mu.Lock()
	proxy := bn.settings.Proxy
	bn.mu.Unlock()

	for start := 0; start < len(symbols); start += batchSize {
		end := start + batchSize
		if end > len(symbols) {
			end = len(symbols)
		}
		subChannels := make(map[string]chan common.Trade)
		for _, symbol := range symbols[start:end] {
			subChannels[symbol] = channels[symbol]
		}
		go func(ctx context.Context, proxy string, channels map[string]chan common.Trade) {
			ws := NewTradeRoutedWS(ctx, proxy, channels)
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
		case <-bn.done:
			return
		}
	}
}

func (bn *Bnspot) StreamTicker(ctx context.Context, channels map[string]chan common.Ticker, batchSize int) {
	panic("implement me")
}

func (bn *Bnspot) StreamKLine(ctx context.Context, channels map[string]chan []common.KLine, batchSize int, interval, lookback time.Duration) {
	panic("implement me")
}

func (bn *Bnspot) StreamFundingRate(ctx context.Context, channels map[string]chan common.FundingRate, batchSize int) {
	bn.mu.Lock()
	pullInterval := bn.settings.PullInterval
	bn.mu.Unlock()
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			for symbol, ch := range channels {
				select {
				case ch <- &FundingRate{
					Symbol: symbol,
				}:
				default:
					logger.Debugf("ch <- &FundingRate failed %s ch len %d", symbol, len(ch))
				}
			}
			timer.Reset(time.Now().Truncate(pullInterval).Add(pullInterval).Sub(time.Now()))
		}
	}
}

func (bn *Bnspot) WatchOrders(ctx context.Context, requestChannels map[string]chan common.OrderRequest, responseChannels map[string]chan common.Order, errorChannels map[string]chan common.OrderError) {
	defer bn.Stop()
	for symbol, reqCh := range requestChannels {
		tickSize, ok := TickSizes[symbol]
		if !ok {
			logger.Debugf("miss price increment for %s, exit", symbol)
			return
		}
		stepSize, ok := StepSizes[symbol]
		if !ok {
			logger.Debugf("miss size increment for %s, exit", symbol)
			return
		}
		logger.Debugf("%v", responseChannels)
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
		go bn.watchOrder(ctx, symbol, tickSize, stepSize, reqCh, respCh, errCh)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-bn.done:
			return
		}
	}
}

func (bn *Bnspot) Setup(ctx context.Context, settings common.ExchangeSettings) (err error) {
	bn.done = make(chan interface{})
	bn.mu = sync.Mutex{}
	bn.stopped = false
	bn.settings = settings
	bn.api, err = NewAPI(&common.Credentials{
		Key:    settings.ApiKey,
		Secret: settings.ApiSecret,
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
		if _, ok := MinNotionals[symbol]; !ok {
			return fmt.Errorf("min notional not found for %s", symbol)
		}
		if _, ok := MultiplierUps[symbol]; !ok {
			return fmt.Errorf("multiplier up not found for %s", symbol)
		}
		if _, ok := MultiplierDowns[symbol]; !ok {
			return fmt.Errorf("multiplier down not found for %s", symbol)
		}
		time.Sleep(time.Second)
	}
	return
}

func (bn *Bnspot) Stop() {
	bn.mu.Lock()
	if !bn.stopped {
		bn.stopped = true
		close(bn.done)
		logger.Debugf("stopped")
	}
	bn.mu.Unlock()
}

func (bn *Bnspot) Done() chan interface{} {
	return bn.done
}

func (bn *Bnspot) watchSystemStatus(
	ctx context.Context,
	output chan common.SystemStatus,
) {
	bn.mu.Lock()
	updateInterval := bn.settings.PullInterval
	bn.mu.Unlock()
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-bn.done:
			return
		case <-timer.C:
			subCtx, _ := context.WithTimeout(ctx, time.Minute)
			_, err := bn.api.PingServer(subCtx)
			if err != nil {
				logger.Debugf("api.PingServer error %v", err)
				select {
				case output <- common.SystemStatusError:
				default:
					logger.Debugf("output <- common.SystemStatusError failed")
				}
			} else {
				select {
				case output <- common.SystemStatusReady:
				default:
					logger.Debugf("output <- common.SystemStatusReady failed")
				}
			}
			timer.Reset(time.Now().Truncate(updateInterval).Add(updateInterval).Sub(time.Now()))
		}
	}
}

func (bn *Bnspot) watchAccount(
	ctx context.Context,
	output chan Account,
) {
	bn.mu.Lock()
	updateInterval := bn.settings.PullInterval
	bn.mu.Unlock()
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			subCtx, _ := context.WithTimeout(ctx, time.Minute)
			account, _, err := bn.api.GetAccount(subCtx)
			if err != nil {
				logger.Debugf("bn.api.GetAccount(subCtx) error %v", err)
			} else {
				output <- *account
			}
			timer.Reset(time.Now().Truncate(updateInterval).Add(updateInterval).Sub(time.Now()))
		}
	}
}

func (bn *Bnspot) watchOrder(
	ctx context.Context,
	market string,
	tickSize, stepSize float64,
	requestCh chan common.OrderRequest,
	responseCh chan common.Order,
	errorCh chan common.OrderError,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-bn.Done():
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
				bn.submitOrder(ctx, *req.New, tickSize, stepSize, responseCh, errorCh)
			} else if req.Cancel != nil {
				bn.cancelOrder(ctx, *req.Cancel, errorCh)
			}
		}
	}
}

func (bn *Bnspot) submitOrder(ctx context.Context, param common.NewOrderParam, tickSize, stepSize float64, respCh chan common.Order, errCh chan common.OrderError) {
	newOrderParam := NewOrderParams{}
	newOrderParam.Symbol = param.Symbol
	newOrderParam.Quantity = math.Round(param.Size/stepSize) * stepSize
	if param.Side == common.OrderSideBuy {
		newOrderParam.Side = OrderSideBuy
	} else {
		newOrderParam.Side = OrderSideSell
	}
	if param.Type == common.OrderTypeMarket {
		newOrderParam.Type = OrderTypeMarket
	} else {
		if param.PostOnly {
			newOrderParam.Type = OrderTypeLimitMarker
		} else {
			newOrderParam.Type = OrderTypeLimit
		}
	}
	switch param.TimeInForce {
	case common.OrderTimeInForceIOC:
		newOrderParam.TimeInForce = OrderTimeInForceIOC
	case common.OrderTimeInForceGTC:
		newOrderParam.TimeInForce = OrderTimeInForceGTC
	case common.OrderTimeInForceFOK:
		newOrderParam.TimeInForce = OrderTimeInForceFOK
	}
	if param.Price != 0 {
		newOrderParam.Price = math.Round(param.Price/tickSize) * tickSize
	}
	newOrderParam.NewClientOrderID = param.ClientID
	order, _, err := bn.api.SubmitOrder(ctx, newOrderParam)
	if err != nil {
		select {
		case errCh <- common.OrderError{
			New:   &param,
			Error: err,
		}:
		default:
			logger.Debugf("errCh <- common.OrderError failed, ch len %d", len(errCh))
		}
	} else {
		select {
		case respCh <- order:
		default:
			logger.Debugf("respCh <- order failed, ch len %d", len(respCh))
		}
	}
}

func (bn *Bnspot) cancelOrder(ctx context.Context, param common.CancelOrderParam, errCh chan common.OrderError) {
	if param.Symbol != "" {
		_, _, err := bn.api.CancelAllOrder(ctx, CancelAllOrderParams{
			Symbol: param.Symbol,
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
