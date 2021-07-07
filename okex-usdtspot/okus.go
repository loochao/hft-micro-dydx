package okex_usdtspot

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

type OkexUsdtSpot struct {
	api      *API
	done     chan interface{}
	stopped  int32
	settings common.ExchangeSettings
}

func (bn *OkexUsdtSpot) IsSpot() bool {
	return true
}

func (bn *OkexUsdtSpot) StreamSymbolStatus(ctx context.Context, channels map[string]chan common.SymbolStatusMsg, batchSize int) {
	panic("implement me")
}

func (bn *OkexUsdtSpot) GenerateClientID() string {
	return fmt.Sprintf("%d%04d", time.Now().Unix(), rand.Intn(10000))
}

func (bn *OkexUsdtSpot) GetMultiplier(symbol string) (float64, error) {
	return 1.0, nil
}

func (bn *OkexUsdtSpot) GetMinNotional(symbol string) (float64, error) {
	return 0, nil
}

func (bn *OkexUsdtSpot) GetMinSize(symbol string) (float64, error) {
	if value, ok := MinSizes[symbol]; ok {
		return value, nil
	} else {
		return 0, fmt.Errorf(common.MinSizeNotFoundError, symbol)
	}
}

func (bn *OkexUsdtSpot) GetStepSize(symbol string) (float64, error) {
	if value, ok := StepSizes[symbol]; ok {
		return value, nil
	} else {
		return 0, fmt.Errorf(common.StepSizeNotFoundError, symbol)
	}
}

func (bn *OkexUsdtSpot) GetTickSize(symbol string) (float64, error) {
	if value, ok := TickSizes[symbol]; ok {
		return value, nil
	} else {
		return 0, fmt.Errorf(common.TickSizeNotFoundError, symbol)
	}
}

func (bn *OkexUsdtSpot) StreamBasic(ctx context.Context, statusCh chan common.SystemStatus, usdtAccountCh chan common.Balance, commissionAssetValueCh chan float64, positionChMap map[string]chan common.Position, orderChMap map[string]chan common.Order) {
	defer bn.Stop()
	proxy := bn.settings.Proxy
	userWS := NewUserWebsocket(
		ctx,
		bn.settings.ApiKey, bn.settings.ApiSecret, bn.settings.ApiPassphrase,
		bn.settings.Symbols,
		proxy,
	)
	balancesMap := make(map[string]*Balance, 0)
	internalAccountCh := make(chan []Balance, 4)
	go bn.watchAccount(ctx, internalAccountCh)
	go bn.watchSystemStatus(ctx, statusCh)
	logSilentTime := time.Now()
	usdtAccount := Balance{
		Currency: "USDT",
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
		case balances := <-userWS.BalancesCh:
			for _, wsBalance := range balances {
				if wsBalance.Currency == "USDT" {
					if wsBalance.EventTime.Sub(usdtAccount.EventTime) > 0 {
						usdtAccount.Balance = wsBalance.Balance
						usdtAccount.Hold = wsBalance.Hold
						usdtAccount.Available = wsBalance.Available
						usdtAccount.EventTime = wsBalance.EventTime
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
				symbol := wsBalance.Currency + "-USDT"
				if ch, ok := positionChMap[symbol]; ok {
					lastBalance, ok := balancesMap[symbol]
					if ok && wsBalance.EventTime.Sub(lastBalance.EventTime) < 0 {
						continue
					}
					if !ok {
						balancesMap[symbol] = &Balance{
							Asset:     wsBalance.Asset,
							Free:      wsBalance.FreeAmount,
							Locked:    wsBalance.LockedAmount,
							EventTime: wsBalance.EventTime,
							ParseTime: wsBalance.ParseTime,
						}
					} else {
						balancesMap[symbol].EventTime = wsBalance.EventTime
						balancesMap[symbol].ParseTime = wsBalance.ParseTime
						balancesMap[symbol].Free = wsBalance.FreeAmount
						balancesMap[symbol].Locked = wsBalance.LockedAmount
					}
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
		case wsOrders := <-userWS.OrdersCh:
			for _, wsOrder := range wsOrders {
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
			}
			break
		case balances := <-internalAccountCh:
			for _, balance := range balances {
				balance := balance
				if balance.Currency == "USDT" {
					if balance.EventTime.Sub(usdtAccount.EventTime) >= 0 {
						usdtAccount = balance
						select {
						case usdtAccountCh <- &balance:
						default:
							if time.Now().Sub(logSilentTime) > 0 {
								logger.Debugf("usdtAccountCh <- usdtAccount failed, ch len %d", len(usdtAccountCh))
								logSilentTime = time.Now().Add(time.Minute)
							}
						}
					}
					continue
				}
				symbol := balance.Currency + "-USDT"
				lastBalance, ok := balancesMap[symbol]
				if ok && balance.EventTime.Sub(lastBalance.EventTime) < 0 {
					continue
				}
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
				if _, ok := balancesMap[symbol]; !ok {
					balance := &Balance{
						Currency:  strings.Replace(symbol, "-USDT", "", -1),
						EventTime: time.Now(),
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

func (bn *OkexUsdtSpot) StreamDepth(ctx context.Context, channels map[string]chan common.Depth, batchSize int) {
	logger.Debugf("START StreamDepth")
	defer logger.Debugf("STOP StreamDepth")
	defer bn.Stop()

	symbols := make([]string, 0)
	for symbol := range channels {
		symbols = append(symbols, symbol)
	}

	proxy := bn.settings.Proxy

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
			defer bn.Stop()
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

func (bn *OkexUsdtSpot) StreamTrade(ctx context.Context, channels map[string]chan common.Trade, batchSize int) {
	logger.Debugf("START StreamTrade")
	defer logger.Debugf("STOP StreamTrade")
	defer bn.Stop()

	symbols := make([]string, 0)
	for symbol := range channels {
		symbols = append(symbols, symbol)
	}

	proxy := bn.settings.Proxy

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
			defer bn.Stop()
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

func (bn *OkexUsdtSpot) StreamTicker(ctx context.Context, channels map[string]chan common.Ticker, batchSize int) {
	panic("implement me")
}

func (bn *OkexUsdtSpot) StreamKLine(ctx context.Context, channels map[string]chan []common.KLine, batchSize int, interval, lookback time.Duration) {
	panic("implement me")
}

func (bn *OkexUsdtSpot) StreamFundingRate(ctx context.Context, channels map[string]chan common.FundingRate, batchSize int) {
	pullInterval := bn.settings.PullInterval
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			for symbol, ch := range channels {
				select {
				case ch <- &common.ZeroFundingRate{
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

func (bn *OkexUsdtSpot) WatchOrders(ctx context.Context, requestChannels map[string]chan common.OrderRequest, responseChannels map[string]chan common.Order, errorChannels map[string]chan common.OrderError) {
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

func (bn *OkexUsdtSpot) Setup(ctx context.Context, settings common.ExchangeSettings) (err error) {
	bn.done = make(chan interface{})
	bn.stopped = 0
	bn.settings = settings
	bn.api, err = NewAPI(&Credentials{
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

func (bn *OkexUsdtSpot) Stop() {
	if atomic.CompareAndSwapInt32(&bn.stopped, 0, 1) {
		close(bn.done)
		logger.Debugf("stopped")
	}
}

func (bn *OkexUsdtSpot) Done() chan interface{} {
	return bn.done
}

func (bn *OkexUsdtSpot) watchSystemStatus(
	ctx context.Context,
	output chan common.SystemStatus,
) {
	updateInterval := bn.settings.PullInterval
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
			statuses, err := bn.api.GetStatus(subCtx)
			if err != nil {
				logger.Debugf("api.GetStatus(subCtx) error %v", err)
				if !strings.Contains(err.Error(), "Too Many Requests") {
					select {
					case output <- common.SystemStatusError:
					default:
						logger.Debugf("output <- common.SystemStatusError, failed ch len %d", len(output))
					}
				}
			} else {
				ready := true
				for _, s := range statuses {
					if (s.ProductType == "0" || s.ProductType == "1") && s.Status == "1" {
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
			timer.Reset(time.Now().Truncate(updateInterval).Add(updateInterval).Sub(time.Now()))
		}
	}
}

func (bn *OkexUsdtSpot) watchAccount(
	ctx context.Context,
	output chan []Balance,
) {
	updateInterval := bn.settings.PullInterval
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			subCtx, _ := context.WithTimeout(ctx, time.Minute)
			balances, err := bn.api.GetAccounts(subCtx)
			for i := range balances {
				balances[i].EventTime = time.Now()
			}
			if err != nil {
				logger.Debugf("api.GetAccounts error %v", err)
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

func (bn *OkexUsdtSpot) watchOrder(
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
				bn.submitOrder(ctx, *req.New, responseCh, errorCh)
			} else if req.Cancel != nil {
				bn.cancelOrder(ctx, *req.Cancel, errorCh)
			}
		}
	}
}

func (bn *OkexUsdtSpot) submitOrder(ctx context.Context, param common.NewOrderParam, respCh chan common.Order, errCh chan common.OrderError) {
	newOrderParam := NewOrderParam{}
	newOrderParam.Symbol = param.Symbol
	newOrderParam.Size = new(float64)
	*newOrderParam.Size = param.Size
	if param.Side == common.OrderSideBuy {
		newOrderParam.Side = OrderSideBuy
	} else {
		newOrderParam.Side = OrderSideSell
	}
	if param.Type == common.OrderTypeMarket {
		newOrderParam.Type = OrderMarket
	} else {
		newOrderParam.Type = OrderLimit
	}
	if param.PostOnly {
		newOrderParam.OrderType = OrderTypePostOnly
	} else {
		switch param.TimeInForce {
		case common.OrderTimeInForceIOC:
			newOrderParam.OrderType = OrderTypeImmediateOrCancel
		case common.OrderTimeInForceFOK:
			newOrderParam.OrderType = OrderTypeFillOrKill
		default:
			newOrderParam.OrderType = OrderTypeNormalOrder
		}
	}
	if param.Price != 0 {
		newOrderParam.Price = new(float64)
		*newOrderParam.Price = param.Price
	}
	newOrderParam.ClientOID = param.ClientID
	_, err := bn.api.SubmitOrder(ctx, newOrderParam)
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

func (bn *OkexUsdtSpot) cancelOrder(ctx context.Context, param common.CancelOrderParam, errCh chan common.OrderError) {
	if param.Symbol != "" {
		_, err := bn.api.CancelOrders(ctx, CancelOrderParam{
			ClientOid: param.ClientID,
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
