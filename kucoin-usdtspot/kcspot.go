package kucoin_usdtspot

import (
	"context"
	"errors"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"math/rand"
	"strings"
	"sync/atomic"
	"time"
)

type KucoinUsdtSpot struct {
	done     chan interface{}
	stopped  int32
	api      *API
	settings common.ExchangeSettings
}

func (k *KucoinUsdtSpot) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (k *KucoinUsdtSpot) GetMultiplier(symbol string) (float64, error) {
	return 1.0, nil
}

func (k *KucoinUsdtSpot) WatchBatchOrders(ctx context.Context, requestChannels map[string]chan common.BatchOrderRequest, responseChannels map[string]chan common.Order, errorChannels map[string]chan common.OrderError) {
	panic("implement me")
}

func (k *KucoinUsdtSpot) StartSideLoop() {
	panic("implement me")
}

func (k *KucoinUsdtSpot) IsSpot() bool {
	return true
}

func (k *KucoinUsdtSpot) Done() chan interface{} {
	return k.done
}

func (k *KucoinUsdtSpot) Stop() {
	if atomic.CompareAndSwapInt32(&k.stopped, 0, 1) {
		close(k.done)
		logger.Debugf("stopped")
	}
}

func (k *KucoinUsdtSpot) Setup(ctx context.Context, settings common.ExchangeSettings) error {
	var err error
	k.stopped = 0
	k.done = make(chan interface{})
	k.settings = settings
	k.api, err = NewAPI(settings.ApiKey, settings.ApiSecret, settings.ApiPassphrase, settings.Proxy)
	if err != nil {
		return err
	}
	for _, symbol := range settings.Symbols {
		if _, err = k.GetStepSize(symbol); err != nil {
			return err
		}
		if _, err = k.GetTickSize(symbol); err != nil {
			return err
		}
		if _, err = k.GetMinNotional(symbol); err != nil {
			return err
		}
		if _, err = k.GetMinSize(symbol); err != nil {
			return err
		}
	}
	return nil
}

func (k *KucoinUsdtSpot) GetMinNotional(symbol string) (float64, error) {
	if v, ok := MinNotionals[symbol]; ok {
		return v, nil
	} else {
		return 0.0, fmt.Errorf(common.MinNotionalNotFoundError, symbol)
	}
}

func (k *KucoinUsdtSpot) GetMinSize(symbol string) (float64, error) {
	if v, ok := MinSizes[symbol]; ok {
		return v, nil
	} else {
		return 0.0, fmt.Errorf(common.MinSizeNotFoundError, symbol)
	}
}

func (k *KucoinUsdtSpot) GetStepSize(symbol string) (float64, error) {
	if v, ok := StepSizes[symbol]; ok {
		return v, nil
	} else {
		return 0.0, fmt.Errorf(common.StepSizeNotFoundError, symbol)
	}
}
func (k *KucoinUsdtSpot) GetTickSize(symbol string) (float64, error) {
	if v, ok := TickSizes[symbol]; ok {
		return v, nil
	} else {
		return 0.0, fmt.Errorf(common.TickSizeNotFoundError, symbol)
	}
}

func (k *KucoinUsdtSpot) StreamBasic(ctx context.Context, statusCh chan common.SystemStatus, accountCh chan common.Balance, commissionAssetValueCh chan float64, positionChMap map[string]chan common.Position, orderChMap map[string]chan common.Order) {
	defer k.Stop()
	settings := k.settings
	var err error
	k.api, err = NewAPI(settings.ApiKey, settings.ApiSecret, settings.ApiPassphrase, settings.Proxy)
	if err != nil {
		return
	}
	userWS := NewUserWebsocket(
		ctx,
		k.api,
		settings.Proxy,
	)
	go k.StreamSystemStatus(ctx, statusCh)
	httpAccountsCh := make(chan []Account, 100)
	go k.accountLoop(ctx, httpAccountsCh)
	logSilentTime := time.Now()
	restartResetTimer := time.NewTimer(time.Hour * 9999)
	defer restartResetTimer.Stop()
	var usdtAccount *Account
	accountsMap := make(map[string]Account)
	kcsPriceCh := make(chan float64, 4)
	matchedOrders := make(map[string]*WSOrder)
	go k.watchKcsPrice(ctx, kcsPriceCh)
	var kcsBalance *float64
	var rebalancedKcsSilentTime = time.Now()

	for {
		select {
		case <-userWS.Done():
			return
		case <-k.done:
			return
		case <-restartResetTimer.C:
			select {
			case statusCh <- common.SystemStatusReady:
				restartResetTimer.Reset(time.Hour * 9999)
				break
			default:
				restartResetTimer.Reset(time.Minute * 3)
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("statusCh <- common.SystemStatusReady failed, ch len %d", len(statusCh))
					logSilentTime = time.Now().Add(time.Minute)
				}
			}
		case accounts := <-httpAccountsCh:
			for _, account := range accounts {
				account := account
				if account.Currency == "USDT" {
					if usdtAccount == nil || account.EventTime.Sub(usdtAccount.EventTime) > 0 {
						usdtAccount = &account
						//logger.Debugf("http account %v", account)
						select {
						case accountCh <- &account:
						default:
							if time.Now().Sub(logSilentTime) > 0 {
								logger.Debugf("accountCh <- usdtBalance failed, ch len %d", len(accountCh))
								logSilentTime = time.Now().Add(time.Minute)
							}
						}
					}
					continue
				} else if account.Currency == "KCS" {
					if kcsBalance == nil {
						kcsBalance = new(float64)
					}
					*kcsBalance = account.Available
					continue
				}
				symbol := account.Currency + "-USDT"
				lastBalance, ok := accountsMap[symbol]
				if ok && account.EventTime.Sub(lastBalance.EventTime) < 0 {
					continue
				}
				accountsMap[symbol] = account
			}
			if kcsBalance == nil {
				kcsBalance = new(float64)
				*kcsBalance = 0.0
			}
			hasBalances := make(map[string]bool)
			for symbol, account := range accountsMap {
				if ch, ok := positionChMap[symbol]; ok {
					hasBalances[symbol] = true
					account := account
					select {
					case ch <- &account:
					default:
						if time.Now().Sub(logSilentTime) > 0 {
							logger.Debugf("ch <- account failed, ch len %d", len(ch))
							logSilentTime = time.Now().Add(time.Minute)
						}
					}
				}
			}
			for symbol, ch := range positionChMap {
				if _, ok := hasBalances[symbol]; !ok {
					select {
					case ch <- &Account{
						Currency:  strings.Replace(symbol, "-USDT", "", -1),
						EventTime: time.Now(),
						ParseTime: time.Now(),
					}:
					default:
						if time.Now().Sub(logSilentTime) > 0 {
							logger.Debugf("ch <- account failed, ch len %d", len(ch))
							logSilentTime = time.Now().Add(time.Minute)
						}
					}
				}
			}
		case <-userWS.RestartCh:
			select {
			case statusCh <- common.SystemStatusRestart:
				restartResetTimer.Reset(time.Minute * 3)
				break
			default:
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("statusCh <- common.SystemStatusRestart failed, ch len %d", len(statusCh))
					logSilentTime = time.Now().Add(time.Minute)
				}
			}
		case price := <-kcsPriceCh:
			//logger.Debugf("kcs price %f %v", price, kcsBalance)
			if k.settings.AutoAddCommissionDiscountAsset && price > 0 {
				if kcsBalance != nil {
					select {
					case commissionAssetValueCh <- *kcsBalance * price:
					default:
						logger.Debugf("commissionAssetValueCh <- *kcsBalance * price failed ch len %d", len(commissionAssetValueCh))
					}
					//logger.Debugf("KCS %f %f", *kcsBalance, price)
					if !k.settings.DryRun &&
						k.settings.MinimalCommissionDiscountAssetValue*0.5 > *kcsBalance*price &&
						time.Now().Sub(rebalancedKcsSilentTime) > 0 {
						deltaValue := k.settings.MinimalCommissionDiscountAssetValue - *kcsBalance*price
						go k.buyKcs(ctx, deltaValue, price)
						rebalancedKcsSilentTime = time.Now().Add(time.Hour)
					}
				}
			} else {
				select {
				case commissionAssetValueCh <- 0.0:
				default:
					logger.Debugf("commissionAssetValueCh <- 0.0 failed ch len %d", len(commissionAssetValueCh))
				}
			}
		case balance := <-userWS.BalanceCh:
			//logger.Debugf("%s %f", balance.Currency, balance.Total)
			if balance.Currency == "USDT" {
				usdtAccount = &Account{
					Currency:  balance.Currency,
					Available: balance.Available,
					Balance:   balance.Total,
					Holds:     balance.Hold,
					EventTime: balance.EventTime,
					ParseTime: balance.ParseTime,
				}
				//logger.Debugf("ws balance %v", balance)
				select {
				case accountCh <- usdtAccount:
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("ch <- &order failed, ch len %d", len(accountCh))
						logSilentTime = time.Now().Add(time.Minute)
					}
				}
			} else if balance.Currency == "KCS" {
				if kcsBalance == nil {
					kcsBalance = new(float64)
				}
				*kcsBalance = balance.Available
				//logger.Debugf("kcs %f", *kcsBalance)
			} else {
				if ch, ok := positionChMap[balance.Currency+"-USDT"]; ok {
					if lastBalance, ok := accountsMap[balance.Currency+"-USDT"]; !ok || balance.EventTime.Sub(lastBalance.EventTime) > 0 {
						accountsMap[balance.Currency+"-USDT"] = Account{
							Currency:  balance.Currency,
							Available: balance.Available,
							Balance:   balance.Total,
							Holds:     balance.Hold,
							EventTime: balance.EventTime,
							ParseTime: balance.ParseTime,
						}
						select {
						case ch <- &Account{
							Currency:  balance.Currency,
							Available: balance.Available,
							Balance:   balance.Total,
							Holds:     balance.Hold,
							EventTime: balance.EventTime,
							ParseTime: balance.ParseTime,
						}:
						default:
							if time.Now().Sub(logSilentTime) > 0 {
								logger.Debugf("ch <- &Account{ failed, %s ch len %d", balance.Currency+"-USDT", len(ch))
								logSilentTime = time.Now().Add(time.Minute)
							}
						}
					}
				}
			}
			break

		case order := <-userWS.OrderCh:

			if order.Status == OrderStatusDone {
				if oldOrder, ok := matchedOrders[order.ClientOid]; ok {
					if order.Type == "filled" {
						order.FilledPrice = oldOrder.FilledPrice
					}
					delete(matchedOrders, order.ClientOid)
				}
			}

			//滤掉没有价格的事件
			if order.MatchSize != 0 && order.MatchPrice != 0 {
				if oldOrder, ok := matchedOrders[order.ClientOid]; ok {
					if oldOrder.FilledSize+order.MatchSize != 0 {
						order.FilledPrice = (order.MatchPrice*order.MatchSize + oldOrder.FilledSize*oldOrder.FilledPrice) / (oldOrder.FilledSize + order.MatchSize)
					} else {
						order.FilledPrice = order.MatchPrice
					}
				} else {
					order.FilledPrice = order.MatchPrice
				}
				matchedOrders[order.ClientOid] = order
				if pos, ok := accountsMap[order.Symbol]; ok {
					size := order.MatchSize
					if order.Side != OrderSideBuy {
						size = -order.MatchSize
					}
					logger.Debugf("ORDER %s %s %v MatchSize %f MatchPrice %f POS %f -> %f %v", order.Symbol, order.Type, order.EventTime, order.MatchSize, order.MatchPrice, pos.Balance, pos.Balance+size, pos.EventTime)
					pos.Balance += size
					if pos.Balance < 0 {
						pos.Balance = 0
					}
					//这儿需要防止和order更新的仓位挨得太近, 重复变更仓位的问题，所以ws的仓位默认需要有一个delay
					//一分钟内不要更新仓位
					pos.ParseTime = order.ParseTime.Add(time.Second * 5)
					pos.EventTime = order.EventTime.Add(time.Second * 5)
					accountsMap[order.Symbol] = pos
					if ch, ok := positionChMap[order.Symbol]; ok {
						select {
						case ch <- &pos:
						default:
							if time.Now().Sub(logSilentTime) > 0 {
								logger.Debugf("ch <- &pos failed, ch len %d", len(ch))
								logSilentTime = time.Now().Add(time.Minute)
							}
						}
					}
				}
			}

			if ch, ok := orderChMap[order.Symbol]; ok {
				select {
				case ch <- order:
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("ch <- &order failed, ch len %d", len(ch))
						logSilentTime = time.Now().Add(time.Minute)
					}
				}
			}
			break
		}
	}
}

func (k *KucoinUsdtSpot) StreamSymbolStatus(ctx context.Context, channels map[string]chan common.SymbolStatusMsg, batchSize int) {
}

func (k *KucoinUsdtSpot) StreamDepth(ctx context.Context, channels map[string]chan common.Depth, batchSize int) {
	logger.Debugf("START StreamDepth")
	defer logger.Debugf("STOP StreamDepth")
	defer k.Stop()
	symbols := make([]string, 0)
	for symbol := range channels {
		symbols = append(symbols, symbol)
	}
	proxy := k.settings.Proxy
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
			defer k.Stop()
			ws := NewDepth5WS(ctx, k.api, proxy, channels)
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
		case <-k.done:
			return
		}
	}
}

func (k *KucoinUsdtSpot) StreamTrade(ctx context.Context, channels map[string]chan common.Trade, batchSize int) {
	panic("implement me")
}

func (k *KucoinUsdtSpot) StreamTicker(ctx context.Context, channels map[string]chan common.Ticker, batchSize int) {
	logger.Debugf("START StreamTicker")
	defer logger.Debugf("STOP StreamTicker")
	defer k.Stop()
	symbols := make([]string, 0)
	for symbol := range channels {
		symbols = append(symbols, symbol)
	}
	proxy := k.settings.Proxy
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
			defer k.Stop()
			ws := NewTickerWS(ctx, k.api, proxy, channels)
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
		case <-k.done:
			return
		}
	}
}

func (k *KucoinUsdtSpot) StreamKLine(ctx context.Context, channels map[string]chan []common.KLine, batchSize int, interval, lookback time.Duration) {
	panic("implement me")
}

func (k *KucoinUsdtSpot) StreamFundingRate(ctx context.Context, channels map[string]chan common.FundingRate, batchSize int) {
	interval := k.settings.PullInterval
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-k.done:
			return
		case <-timer.C:
			for symbol, ch := range channels {
				select {
				case ch <- FundingRate{
					Symbol: symbol,
				}:
				default:
					logger.Debugf("ch <- fr failed, %s ch len %d", symbol, len(ch))
				}
			}
			timer.Reset(time.Now().Truncate(interval).Add(interval).Sub(time.Now()))
		}
	}
}

func (k *KucoinUsdtSpot) WatchOrders(ctx context.Context, requestChannels map[string]chan common.OrderRequest, responseChannels map[string]chan common.Order, errorChannels map[string]chan common.OrderError) {
	defer k.Stop()
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
		go k.watchOrder(ctx, symbol, reqCh, respCh, errCh)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-k.done:
			return
		}
	}
}

func (k *KucoinUsdtSpot) GenerateClientID() string {
	return fmt.Sprintf("%d%04d", time.Now().Unix(), rand.Intn(10000))
}

func (k *KucoinUsdtSpot) watchKcsPrice(
	ctx context.Context,
	priceCh chan float64,
) {
	pullTimer := time.NewTimer(time.Second * 30)
	defer pullTimer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-pullTimer.C:
			ticker, err := k.api.GetTicker(ctx, TickerParam{Symbol: "KCS-USDT"})
			if err != nil {
				logger.Debugf("k.api.GetTicker KCS-USDT error %v", err)
				pullTimer.Reset(time.Minute)
			} else {
				select {
				case priceCh <- (ticker.BestAskPrice + ticker.BestBidPrice) * 0.5:
				default:
					logger.Debugf("priceCh <- ticker.Price failed ch len %d", len(priceCh))
				}
				pullTimer.Reset(time.Minute * 5)
			}
		}
	}
}

func (k *KucoinUsdtSpot) StreamSystemStatus(
	ctx context.Context, output chan common.SystemStatus,
) {
	pullInterval := k.settings.PullInterval
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-k.done:
			return
		case <-timer.C:
			subCtx, _ := context.WithTimeout(ctx, time.Minute)
			systemStatus, err := k.api.GetSystemStatus(subCtx)
			if err != nil {
				logger.Debugf("k.api.GetSystemStatus(subCtx) error %v", err)
				select {
				case output <- common.SystemStatusError:
				default:
					logger.Debugf("output <- common.SystemStatusError failed ch len %d", len(output))
				}
			} else {
				if systemStatus.Status == SystemStatusOpen {
					select {
					case output <- common.SystemStatusReady:
					default:
						logger.Debugf("output <- common.SystemStatusReady failed ch len %d", len(output))
					}
				} else {
					select {
					case output <- common.SystemStatusNotReady:
					default:
						logger.Debugf("output <- common.SystemStatusNotReady failed ch len %d", len(output))
					}
				}
			}
			timer.Reset(time.Now().Truncate(pullInterval).Add(pullInterval).Sub(time.Now()))
		}
	}
}

func (k *KucoinUsdtSpot) accountLoop(
	ctx context.Context, output chan []Account,
) {
	pullInterval := k.settings.PullInterval
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			subCtx, _ := context.WithTimeout(ctx, time.Minute)
			account, err := k.api.GetAccounts(subCtx, AccountsParam{
				//Currency: "USDT",
				Type: "trade",
			})
			if err != nil {
				logger.Debugf("k.api.GetAccounts error %v", err)
			} else {
				output <- account
			}
			timer.Reset(time.Now().Truncate(pullInterval).Add(pullInterval).Sub(time.Now()))
		}
	}
}

func (k *KucoinUsdtSpot) watchOrder(
	ctx context.Context,
	symbol string,
	requestCh chan common.OrderRequest,
	responseCh chan common.Order,
	errorCh chan common.OrderError,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-k.Done():
			return
		case req := <-requestCh:
			if req.New != nil {
				if req.New.Symbol != symbol {
					select {
					case errorCh <- common.OrderError{
						New:   req.New,
						Error: errors.New(fmt.Sprintf("bad create request symbol not match %s %s", req.New.Symbol, symbol)),
					}:
					default:
						logger.Debugf("errorCh <- common.OrderError failed, ch len %d", len(errorCh))
					}
					continue
				}
				k.submitOrder(ctx, *req.New, responseCh, errorCh)
			} else if req.Cancel != nil {
				k.cancelOrder(ctx, *req.Cancel, errorCh)
			}
		}
	}
}

func (k *KucoinUsdtSpot) submitOrder(ctx context.Context, param common.NewOrderParam, respCh chan common.Order, errCh chan common.OrderError) {
	newOrderParam := NewOrderParam{}
	newOrderParam.Symbol = param.Symbol
	newOrderParam.Size = Float64(param.Size)
	if param.Side == common.OrderSideBuy {
		newOrderParam.Side = OrderSideBuy
	} else {
		newOrderParam.Side = OrderSideSell
	}
	if param.Type == common.OrderTypeMarket {
		newOrderParam.Type = OrderTypeMarket
	} else {
		newOrderParam.Type = OrderTypeLimit
	}
	switch param.TimeInForce {
	case common.OrderTimeInForceFOK:
		newOrderParam.TimeInForce = OrderTimeInForceIOC
	case common.OrderTimeInForceIOC:
		newOrderParam.TimeInForce = OrderTimeInForceIOC
	case common.OrderTimeInForceGTC:
		newOrderParam.TimeInForce = OrderTimeInForceGTC
	}
	newOrderParam.PostOnly = param.PostOnly
	if param.Price != 0 {
		newOrderParam.Price = Float64(param.Price)
	}
	newOrderParam.ClientOid = param.ClientID
	_, err := k.api.SubmitOrder(ctx, newOrderParam)
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

func (k *KucoinUsdtSpot) cancelOrder(ctx context.Context, param common.CancelOrderParam, errCh chan common.OrderError) {
	if param.Symbol != "" {
		_, err := k.api.CancelAllOrders(ctx, CancelAllOrdersParam{
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

func (k *KucoinUsdtSpot) buyKcs(
	ctx context.Context,
	deltaValue float64,
	price float64,
) {
	size := math.Round(deltaValue/price/StepSizes["KCS-USDT"]) * StepSizes["KCS-USDT"]
	if size*price < MinNotionals["KCS-USDT"] {
		return
	}
	childCtx, _ := context.WithTimeout(ctx, time.Minute)
	order, err := k.api.SubmitOrder(childCtx, NewOrderParam{
		Symbol:    "KCS-USDT",
		Side:      OrderSideBuy,
		Type:      OrderTypeMarket,
		Size:      Float64(size),
		ClientOid: k.GenerateClientID(),
	})
	if err != nil {
		logger.Debugf("KCS k.api.SubmitOrder buy %f kcs error %v", size, err)
		return
	} else {
		logger.Debugf("KCS k.api.SubmitOrder buy %f kcs success, %v", size, order)
	}
}

type KucoinUsdtSpotWithMergedTicker struct {
	KucoinUsdtSpot
}

func (k *KucoinUsdtSpotWithMergedTicker) StreamTicker(ctx context.Context, channels map[string]chan common.Ticker, batchSize int) {
	logger.Debugf("START StreamTicker")
	defer logger.Debugf("STOP StreamTicker")
	defer k.Stop()
	symbols := make([]string, 0)
	for symbol := range channels {
		symbols = append(symbols, symbol)
	}
	proxy := k.settings.Proxy
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
			ws1 := NewTickerWS(ctx, k.api, proxy, channels)
			ws2 := NewDepth5TickerWS(ctx, k.api, proxy, channels)
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
		case <-k.done:
			return
		}
	}
}
