package dydx_usdfuture

import (
	"context"
	"errors"
	"fmt"
	binance_usdtspot "github.com/geometrybase/hft-micro/binance-usdtspot"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

type DydxUsdFuture struct {
	done     chan interface{}
	stopped  int32
	mu       sync.Mutex
	api      *API
	priceFactor *common.AtomicFloat64
	settings common.ExchangeSettings
}

func (dd *DydxUsdFuture) GetPriceFactor() float64 {
	return dd.priceFactor.Load()
}

func (dd *DydxUsdFuture) StreamSystemStatus(ctx context.Context, statusCh chan common.SystemStatus) {
	dd.mu.Lock()
	pullInterval := dd.settings.PullInterval
	dd.mu.Unlock()
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-dd.done:
			return
		case <-timer.C:
			subCtx, _ := context.WithTimeout(ctx, time.Minute)
			serverTime, err := dd.api.GetServerTime(subCtx)
			if err != nil {
				logger.Debugf("dd.api.GetSystemStatus(subCtx) error %v", err)
				select {
				case statusCh <- common.SystemStatusError:
				default:
					logger.Debugf("statusCh <- common.SystemStatusError failed ch len %d", len(statusCh))
				}
			} else {
				if serverTime.ISO.Sub(time.Now()) < time.Minute {
					select {
					case statusCh <- common.SystemStatusReady:
					default:
						logger.Debugf("statusCh <- common.SystemStatusReady failed ch len %d", len(statusCh))
					}
				} else {
					logger.Debugf("bad remote server time %v, local sever time %v", serverTime.ISO, time.Now())
					select {
					case statusCh <- common.SystemStatusNotReady:
					default:
						logger.Debugf("statusCh <- common.SystemStatusNotReady failed ch len %d", len(statusCh))
					}
				}
			}
			timer.Reset(time.Now().Truncate(pullInterval).Add(pullInterval).Sub(time.Now()))
		}
	}
}

func (dd *DydxUsdFuture) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (dd *DydxUsdFuture) IsSpot() bool {
	return false
}

func (dd *DydxUsdFuture) Done() chan interface{} {
	return dd.done
}

func (dd *DydxUsdFuture) Stop() {
	if atomic.LoadInt32(&dd.stopped) == 0 {
		atomic.StoreInt32(&dd.stopped, 1)
		close(dd.done)
		logger.Debugf("stopped")
	}
}

func (dd *DydxUsdFuture) watchPriceFactor(ctx context.Context, settings common.ExchangeSettings) {
	logger.Debugf("watchPriceFactor for %s", settings.PriceFactorPair)
	defer func() {
		logger.Debugf("stop watchPriceFactor for %s", settings.PriceFactorPair)
	}()
	channels := make(map[string]chan common.Ticker)
	ch := make(chan common.Ticker, 64)
	channels[settings.PriceFactorPair] = ch
	go func(ctx context.Context, proxy string, channels map[string]chan common.Ticker) {
		defer dd.Stop()
		ws1 := binance_usdtspot.NewBookTickerWS(ctx, proxy, channels)
		for {
			select {
			case <-ws1.Done():
				return
			case <-ctx.Done():
				return
			}
		}
	}(ctx, settings.Proxy, channels)
	tm := common.NewTimedMean(time.Second * 5)
	for {
		select {
		case <-dd.done:
			return
		case <-ctx.Done():
			return
		case ticker := <-ch:
			dd.priceFactor.Set(tm.Insert(ticker.GetEventTime(), (ticker.GetAskPrice()+ticker.GetBidPrice())*0.5))
		}
	}
}

func (dd *DydxUsdFuture) Setup(ctx context.Context, settings common.ExchangeSettings) error {
	var err error
	dd.stopped = 0
	dd.done = make(chan interface{})
	dd.settings = settings
	dd.api, err = NewAPI(Credentials{
		ApiKey:        settings.ApiKey,
		ApiSecret:     settings.ApiSecret,
		ApiPassphrase: settings.ApiPassphrase,
		AccountID:     settings.AccountID,
		AccountNumber: settings.AccountNumber,
	}, settings.Proxy)
	if err != nil {
		return err
	}
	dd.priceFactor = common.ForAtomicFloat64(1.0)
	if settings.PriceFactorPair != "" {
		go dd.watchPriceFactor(ctx, settings)
	}
	for _, symbol := range settings.Symbols {
		if _, err = dd.GetStepSize(symbol); err != nil {
			return err
		}
		if _, err = dd.GetTickSize(symbol); err != nil {
			return err
		}
		if _, err = dd.GetMultiplier(symbol); err != nil {
			return err
		}
		if _, err = dd.GetMinSize(symbol); err != nil {
			return err
		}
		if _, err = dd.GetMinNotional(symbol); err != nil {
			return err
		}
	}
	return nil
}

func (dd *DydxUsdFuture) GetMultiplier(symbol string) (float64, error) {
	return 1.0, nil
}

func (dd *DydxUsdFuture) GetMinNotional(symbol string) (float64, error) {
	return 0.0, nil
}

func (dd *DydxUsdFuture) GetMinSize(symbol string) (float64, error) {
	if v, ok := MinSizes[symbol]; ok {
		return v, nil
	} else {
		return 0.0, fmt.Errorf(common.MinSizeNotFoundError, symbol)
	}
}

func (dd *DydxUsdFuture) GetStepSize(symbol string) (float64, error) {
	if v, ok := StepSizes[symbol]; ok {
		return v, nil
	} else {
		return 0.0, fmt.Errorf(common.StepSizeNotFoundError, symbol)
	}
}

func (dd *DydxUsdFuture) GetTickSize(symbol string) (float64, error) {
	if v, ok := TickSizes[symbol]; ok {
		return v, nil
	} else {
		return 0.0, fmt.Errorf(common.TickSizeNotFoundError, symbol)
	}
}

func (dd *DydxUsdFuture) StreamBasic(ctx context.Context, statusCh chan common.SystemStatus, accountCh chan common.Balance, commissionAssetValueCh chan float64, positionChMap map[string]chan common.Position, orderChs map[string]chan common.Order) {
	defer dd.Stop()
	dd.mu.Lock()
	settings := dd.settings
	//symbols := dd.settings.Symbols[:]
	dd.mu.Unlock()
	var err error
	dd.api, err = NewAPI(Credentials{
		ApiKey:        settings.ApiKey,
		ApiSecret:     settings.ApiSecret,
		ApiPassphrase: settings.ApiPassphrase,
		AccountID:     settings.AccountID,
		AccountNumber: settings.AccountNumber,
	}, settings.Proxy)
	if err != nil {
		return
	}
	userWS := NewUserWebsocket(
		ctx,
		&Credentials{
			ApiKey:        settings.ApiKey,
			ApiSecret:     settings.ApiSecret,
			ApiPassphrase: settings.ApiPassphrase,
			AccountID:     settings.AccountID,
			AccountNumber: settings.AccountNumber,
		},
		settings.Proxy,
	)
	go dd.systemStatusLoop(ctx, statusCh)
	httpAccountCh := make(chan Account, 128)
	go dd.accountLoop(ctx, httpAccountCh)
	logSilentTime := time.Now()
	restartResetTimer := time.NewTimer(time.Hour * 9999)
	defer restartResetTimer.Stop()
	var account *Account
	positionsMap := make(map[string]Position)
	var commissionAssetTimer = time.NewTimer(time.Second)
	//matchedOrders := make(map[string]Order)
	accountNumber := dd.settings.AccountNumber
	accountNumberWithQuote :=  fmt.Sprintf("\"%s\"", accountNumber)
	for {
		select {
		case <-userWS.Done():
			return
		case <-dd.done:
			return
		case <-commissionAssetTimer.C:
			select {
			case commissionAssetValueCh <- 0.0:
			default:
				logger.Debugf("commissionAssetValueCh <- 0.0 failed, ch len %d", len(commissionAssetValueCh))
			}
			commissionAssetTimer.Reset(time.Minute)
			break
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
			break
		case a := <-httpAccountCh:
			if string(a.AccountNumber) != accountNumberWithQuote &&
				string(a.AccountNumber) != accountNumber {
				continue
			}
			account = &a
			select {
			case accountCh <- account:
			default:
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("accountCh <- account failed, ch len %d", len(accountCh))
					logSilentTime = time.Now().Add(time.Minute)
				}
			}
			hasPositions := make(map[string]bool)
			for market, pos := range a.OpenPositions {
				if ch, ok := positionChMap[market]; ok {
					hasPositions[market] = true
					if oldPos, ok := positionsMap[market]; ok {
						if pos.ParseTime.Sub(oldPos.ParseTime) > 0 {
							pos := pos
							positionsMap[market] = pos
							select {
							case ch <- &pos:
							default:
								logger.Debugf("ch <- &pos failed, ch len %d", len(ch))
							}
						}
					} else {
						pos := pos
						positionsMap[market] = pos
						select {
						case ch <- &pos:
						default:
							logger.Debugf("ch <- &pos failed, ch len %d", len(ch))
						}
					}
				}
			}
			for market, ch := range positionChMap {
				if _, ok := hasPositions[market]; !ok {
					pos := Position{
						Market:    market,
						ParseTime: time.Now(),
					}
					select {
					case ch <- &pos:
					default:
						logger.Debugf("ch <- &pos failed, ch len %d", len(ch))
					}
				}
			}
			break
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
			break
		case newAccount := <-userWS.AccountCh:
			if string(newAccount.AccountNumber) != accountNumberWithQuote &&
				string(newAccount.AccountNumber) != accountNumber {
				continue
			}
			if account != nil {
				if newAccount.QuoteBalance != 0 {
					account.QuoteBalance = newAccount.QuoteBalance
				}
				if newAccount.Equity != 0 {
					account.Equity = newAccount.Equity
				}
				if newAccount.FreeCollateral != 0 {
					account.FreeCollateral = newAccount.FreeCollateral
				}
				outAccount := *account
				select {
				case accountCh <- &outAccount:
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("accountCh <- account failed, ch len %d", len(accountCh))
						logSilentTime = time.Now().Add(time.Minute)
					}
				}
				for market, pos := range newAccount.OpenPositions {
					if ch, ok := positionChMap[market]; ok {
						if oldPos, ok := positionsMap[market]; ok {
							if pos.ParseTime.Sub(oldPos.ParseTime) > 0 {
								pos := pos
								positionsMap[market] = pos
								select {
								case ch <- &pos:
								default:
									logger.Debugf("ch <- &pos failed, ch len %d", len(ch))
								}
							}
						} else {
							pos := pos
							positionsMap[market] = pos
							select {
							case ch <- &pos:
							default:
								logger.Debugf("ch <- &pos failed, ch len %d", len(ch))
							}
						}
					}
				}
			}
			break
		case orders := <-userWS.OrdersCh:
			for _, order := range orders {
				if ch, ok := orderChs[order.Market]; ok {
					order := order
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
		case wsPositions := <-userWS.PositionsCh:
			for _, wsPosition := range wsPositions {
				if ch, ok := positionChMap[wsPosition.Market]; ok {
					position := wsPosition
					select {
					case ch <- &position:
					default:
						if time.Now().Sub(logSilentTime) > 0 {
							logger.Debugf("ch <- &wsPosition failed, ch len %d", len(ch))
							logSilentTime = time.Now().Add(time.Minute)
						}
					}
				}
			}
			break
		}
	}
}

func (dd *DydxUsdFuture) StreamSymbolStatus(ctx context.Context, channels map[string]chan common.SymbolStatusMsg, batchSize int) {
	checkInterval := time.Second * 5
	startTime := time.Now()
	updateTimes := make(map[string]time.Time)
	for symbol := range channels {
		updateTimes[symbol] = startTime
		startTime = startTime.Add(checkInterval)
	}
	loopTimer := time.NewTimer(time.Second)
	//dd.mu.Lock()
	//leverage := int(dd.settings.Leverage)
	//dd.mu.Unlock()
	for {
		select {
		case <-ctx.Done():
			return
		case <-dd.done:
			return
		case <-loopTimer.C:
			for symbol, ch := range channels {
				if time.Now().Sub(updateTimes[symbol]) > 0 {
					status := common.SymbolStatusReady
					//ticker, err := dd.api.GetTicker(ctx, TickerParam{
					//	Symbol: symbol,
					//})
					//if err != nil {
					//	logger.Debugf("%s dd.api.GetTicker error %v", symbol, err)
					//	status = common.SymbolStatusNotReady
					//} else {
					//	size := LotSizes[symbol]
					//	price := ticker.BestAskPrice * 1.05
					//	price = math.Ceil(price/TickSizes[symbol]) * TickSizes[symbol]
					//	_, err := dd.api.SubmitOrder(ctx, NewOrderParam{
					//		Symbol:      symbol,
					//		Side:        OrderSideSell,
					//		TimeInForce: OrderTimeInForceIOC,
					//		Price:       common.Float64(price),
					//		Size:        int64(size),
					//		Leverage:    leverage,
					//	})
					//	if err != nil {
					//		logger.Debugf("dd.api.SubmitOrder error %v", err)
					//		status = common.SymbolStatusNotReady
					//	}
					//}
					select {
					case ch <- status:
					default:
						logger.Debugf("%s ch <- status failed, ch len %d", symbol, len(ch))
					}
					if time.Now().Sub(startTime) > 0 {
						startTime = time.Now().Add(checkInterval)
					} else {
						startTime = startTime.Add(checkInterval)
					}
					updateTimes[symbol] = startTime.Add(checkInterval)
				}
			}
			loopTimer.Reset(time.Second)
		}
	}
}

func (dd *DydxUsdFuture) StreamDepth(ctx context.Context, channels map[string]chan common.Depth, batchSize int) {
	logger.Debugf("START StreamDepth")
	defer logger.Debugf("STOP StreamDepth")
	defer dd.Stop()
	symbols := make([]string, 0)
	for symbol := range channels {
		symbols = append(symbols, symbol)
	}
	dd.mu.Lock()
	proxy := dd.settings.Proxy
	dd.mu.Unlock()
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
			defer dd.Stop()
			ws := NewDepthWS(ctx, proxy, channels)
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
		case <-dd.done:
			return
		}
	}
}

func (dd *DydxUsdFuture) StreamTrade(ctx context.Context, channels map[string]chan common.Trade, batchSize int) {
	panic("implement me")
}

func (dd *DydxUsdFuture) StreamTicker(ctx context.Context, channels map[string]chan common.Ticker, batchSize int) {
	logger.Debugf("START StreamTicker")
	defer logger.Debugf("STOP StreamTicker")
	defer dd.Stop()
	symbols := make([]string, 0)
	for symbol := range channels {
		symbols = append(symbols, symbol)
	}
	dd.mu.Lock()
	proxy := dd.settings.Proxy
	dd.mu.Unlock()
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
			defer dd.Stop()
			ws := NewTickerWS(ctx, proxy, channels)
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
		case <-dd.done:
			return
		}
	}
}

func (dd *DydxUsdFuture) StreamKLine(ctx context.Context, channels map[string]chan []common.KLine, batchSize int, interval, lookback time.Duration) {
	panic("implement me")
}

func (dd *DydxUsdFuture) StreamFundingRate(ctx context.Context, channels map[string]chan common.FundingRate, batchSize int) {
	dd.mu.Lock()
	interval := time.Minute
	dd.mu.Unlock()
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	afterFrTimer := time.NewTimer(time.Now().Truncate(time.Hour * 4).Add(time.Hour*4 + time.Second).Sub(time.Now()))
	defer afterFrTimer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-dd.done:
			return
		case <-afterFrTimer.C:
			subCtx, _ := context.WithTimeout(ctx, time.Minute)
			markets, err := dd.api.GetMarkets(subCtx)
			if err != nil {
				logger.Debugf("dd.api.GetMarkets error %v", err)
			} else {
				for symbol, ch := range channels {
					if fr, ok := markets[symbol]; ok {
						select {
						case ch <- &fr:
						default:
							logger.Debugf("ch <- fr failed, %s ch len %d", symbol, len(ch))
						}
					}
				}
			}
			afterFrTimer.Reset(time.Now().Truncate(time.Hour * 4).Add(time.Hour*4 + time.Second).Sub(time.Now()))
			break
		case <-timer.C:
			subCtx, _ := context.WithTimeout(ctx, time.Minute)
			markets, err := dd.api.GetMarkets(subCtx)
			if err != nil {
				logger.Debugf("dd.api.GetMarkets error %v", err)
			} else {
				for symbol, ch := range channels {
					if fr, ok := markets[symbol]; ok {
						select {
						case ch <- &fr:
						default:
							logger.Debugf("ch <- fr failed, %s ch len %d", symbol, len(ch))
						}
					}
				}
			}
			timer.Reset(time.Now().Truncate(interval).Add(interval).Sub(time.Now()))
			break
		}
	}
}

func (dd *DydxUsdFuture) WatchOrders(ctx context.Context, requestChannels map[string]chan common.OrderRequest, responseChannels map[string]chan common.Order, errorChannels map[string]chan common.OrderError) {
	defer dd.Stop()
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
		go dd.watchOrder(ctx, symbol, reqCh, respCh, errCh)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-dd.done:
			return
		}
	}
}

func (dd *DydxUsdFuture) GenerateClientID() string {
	return fmt.Sprintf("%d%04d", time.Now().Unix(), rand.Intn(10000))
}

func (dd *DydxUsdFuture) systemStatusLoop(
	ctx context.Context, output chan common.SystemStatus,
) {
	dd.mu.Lock()
	pullInterval := dd.settings.PullInterval
	dd.mu.Unlock()
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-dd.done:
			return
		case <-timer.C:
			select {
			case output <- common.SystemStatusReady:
			default:
				logger.Debugf("output <- common.SystemStatusError failed ch len %d", len(output))
			}
			timer.Reset(time.Now().Truncate(pullInterval).Add(pullInterval).Sub(time.Now()))
		}
	}
}

func (dd *DydxUsdFuture) accountLoop(
	ctx context.Context, output chan Account,
) {
	dd.mu.Lock()
	pullInterval := dd.settings.PullInterval
	dd.mu.Unlock()
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			subCtx, _ := context.WithTimeout(ctx, time.Minute)
			account, err := dd.api.GetAccount(subCtx)
			if err != nil {
				logger.Debugf("dd.api.GetAccountOverView error %v", err)
			} else {
				output <- *account
			}
			timer.Reset(time.Now().Truncate(pullInterval).Add(pullInterval).Sub(time.Now()))
		}
	}
}

func (dd *DydxUsdFuture) watchOrder(
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
		case <-dd.Done():
			return
		case req := <-requestCh:
			if req.New != nil {
				//logger.Debugf("req := <-requestCh New %s", req.New.Symbol)
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
				dd.submitOrder(ctx, *req.New, responseCh, errorCh)
			} else if req.Cancel != nil {
				dd.cancelOrder(ctx, *req.Cancel, errorCh)
			}
		}
	}
}

func (dd *DydxUsdFuture) submitOrder(ctx context.Context, param common.NewOrderParam, respCh chan common.Order, errCh chan common.OrderError) {
	newOrderParam := NewOrderParams{}
	newOrderParam.Market = param.Symbol
	newOrderParam.Size = param.Size
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
	if param.TimeInForce == common.OrderTimeInForceFOK {
		newOrderParam.TimeInForce = OrderTimeInForceFOK
	} else if param.TimeInForce == common.OrderTimeInForceIOC {
		newOrderParam.TimeInForce = OrderTimeInForceIOC
	} else {
		newOrderParam.TimeInForce = OrderTimeInForceGTT
	}
	newOrderParam.PostOnly = param.PostOnly
	if param.Price != 0 {
		newOrderParam.Price = param.Price
	}
	newOrderParam.ClientId = param.ClientID
	if param.CancelAfter != 0 {
		newOrderParam.Expiration = time.Now().UTC().Add(param.CancelAfter).Format(TimeLayout)
		newOrderParam.TimeInForce = OrderTimeInForceGTT
	}else{
		newOrderParam.Expiration = time.Now().UTC().Add(time.Hour * 24).Format(TimeLayout)
	}
	newOrderParam.LimitFee = 0.0015
	dd.mu.Lock()
	newOrderParam.PositionID = dd.settings.PositionID
	dd.mu.Unlock()
	_, err := dd.api.CreateOrder(ctx, &newOrderParam)
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

func (dd *DydxUsdFuture) cancelOrder(ctx context.Context, param common.CancelOrderParam, errCh chan common.OrderError) {
	if param.Symbol != "" {
		_, err := dd.api.CancelOrders(ctx, &CancelOrdersParam{
			Market: param.Symbol,
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

func (dd *DydxUsdFuture) WatchBatchOrders(ctx context.Context, requestChannels map[string]chan common.BatchOrderRequest, responseChannels map[string]chan common.Order, errorChannels map[string]chan common.OrderError) {
	panic("implement me")
}

func (dd *DydxUsdFuture) StartSideLoop() {
	panic("implement me")
}

type KucoinUsdtFutureWithDepth5 struct {
	DydxUsdFuture
}

type KucoinUsdtFutureWithMergedTicker struct {
	DydxUsdFuture
}

func (k *KucoinUsdtFutureWithMergedTicker) StreamTicker(ctx context.Context, channels map[string]chan common.Ticker, batchSize int) {
	logger.Debugf("START StreamTicker")
	defer logger.Debugf("STOP StreamTicker")
	defer k.Stop()
	symbols := make([]string, 0)
	for symbol := range channels {
		symbols = append(symbols, symbol)
	}
	k.mu.Lock()
	proxy := k.settings.Proxy
	k.mu.Unlock()
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
			ws1 := NewTickerWS(ctx, proxy, channels)
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
		case <-k.done:
			return
		}
	}
}
