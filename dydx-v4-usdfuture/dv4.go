package dydx_v4_usdfuture

import (
	"context"
	"fmt"
	binance_usdtspot "github.com/geometrybase/hft-micro/binance-usdtspot"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math/rand"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type DydxV4UsdFuture struct {
	done        chan interface{}
	stopped     int32
	mu          sync.Mutex
	api         *V4API
	priceFactor *common.AtomicFloat64
	settings    common.ExchangeSettings
	orderBridge *OrderBridge
	tickSizes   map[string]float64
	stepSizes   map[string]float64
	minSizes    map[string]float64
}

func (dd *DydxV4UsdFuture) GetPriceFactor() float64 {
	return dd.priceFactor.Load()
}

func (dd *DydxV4UsdFuture) GetExchange() common.ExchangeID {
	return DydxV4UsdFutureExchangeID
}

func (dd *DydxV4UsdFuture) IsSpot() bool {
	return false
}

func (dd *DydxV4UsdFuture) Done() chan interface{} {
	return dd.done
}

func (dd *DydxV4UsdFuture) Stop() {
	if atomic.CompareAndSwapInt32(&dd.stopped, 0, 1) {
		close(dd.done)
		logger.Debugf("DydxV4UsdFuture stopped")
	}
}

func (dd *DydxV4UsdFuture) watchPriceFactor(ctx context.Context, settings common.ExchangeSettings) {
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

func (dd *DydxV4UsdFuture) Setup(ctx context.Context, settings common.ExchangeSettings) error {
	dd.stopped = 0
	dd.done = make(chan interface{})
	dd.settings = settings

	// Parse subaccount number
	subaccountNumber := 0
	if settings.AccountNumber != "" {
		n, err := strconv.Atoi(settings.AccountNumber)
		if err == nil {
			subaccountNumber = n
		}
	}

	// Create API client (Indexer REST, no auth needed)
	var err error
	dd.api, err = NewV4API(settings.ApiKey, subaccountNumber, settings.Proxy)
	if err != nil {
		return err
	}

	// Create order bridge (Python helper for Phase 1)
	dd.orderBridge = NewOrderBridge(
		settings.ApiKey,       // dydx address
		settings.ApiPassphrase, // mnemonic
		subaccountNumber,
		settings.Proxy,
	)

	dd.priceFactor = common.ForAtomicFloat64(1.0)
	if settings.PriceFactorPair != "" {
		go dd.watchPriceFactor(ctx, settings)
	}

	// Load market info from indexer
	dd.tickSizes, dd.stepSizes, dd.minSizes, err = LoadMarketInfo(ctx, dd.api)
	if err != nil {
		return fmt.Errorf("LoadMarketInfo error: %v", err)
	}

	// Validate configured symbols
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

func (dd *DydxV4UsdFuture) GetMultiplier(symbol string) (float64, error) {
	return 1.0, nil
}

func (dd *DydxV4UsdFuture) GetMinNotional(symbol string) (float64, error) {
	return 0.0, nil
}

func (dd *DydxV4UsdFuture) GetMinSize(symbol string) (float64, error) {
	if v, ok := dd.minSizes[symbol]; ok {
		return v, nil
	}
	return 0.0, fmt.Errorf(common.MinSizeNotFoundError, symbol)
}

func (dd *DydxV4UsdFuture) GetStepSize(symbol string) (float64, error) {
	if v, ok := dd.stepSizes[symbol]; ok {
		return v, nil
	}
	return 0.0, fmt.Errorf(common.StepSizeNotFoundError, symbol)
}

func (dd *DydxV4UsdFuture) GetTickSize(symbol string) (float64, error) {
	if v, ok := dd.tickSizes[symbol]; ok {
		return v, nil
	}
	return 0.0, fmt.Errorf(common.TickSizeNotFoundError, symbol)
}

func (dd *DydxV4UsdFuture) StreamSystemStatus(ctx context.Context, statusCh chan common.SystemStatus) {
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
			subCtx, cancel := context.WithTimeout(ctx, time.Minute)
			err := dd.api.CheckHealth(subCtx)
			cancel()
			if err != nil {
				logger.Debugf("CheckHealth error %v", err)
				select {
				case statusCh <- common.SystemStatusError:
				default:
				}
			} else {
				select {
				case statusCh <- common.SystemStatusReady:
				default:
				}
			}
			timer.Reset(time.Now().Truncate(pullInterval).Add(pullInterval).Sub(time.Now()))
		}
	}
}

func (dd *DydxV4UsdFuture) StreamBasic(ctx context.Context, statusCh chan common.SystemStatus, accountCh chan common.Balance, commissionAssetValueCh chan float64, positionChMap map[string]chan common.Position, orderChs map[string]chan common.Order) {
	defer dd.Stop()
	dd.mu.Lock()
	settings := dd.settings
	dd.mu.Unlock()

	subaccountNumber := 0
	if settings.AccountNumber != "" {
		n, err := strconv.Atoi(settings.AccountNumber)
		if err == nil {
			subaccountNumber = n
		}
	}

	accountWS := NewV4AccountWS(
		ctx,
		settings.ApiKey, // dydx address
		subaccountNumber,
		settings.Proxy,
	)

	go dd.systemStatusLoop(ctx, statusCh)
	httpAccountCh := make(chan Subaccount, 128)
	go dd.accountLoop(ctx, httpAccountCh)

	logSilentTime := time.Now()
	restartResetTimer := time.NewTimer(time.Hour * 9999)
	defer restartResetTimer.Stop()
	var account *Subaccount
	positionsMap := make(map[string]V4Position)
	commissionAssetTimer := time.NewTimer(time.Second)
	defer commissionAssetTimer.Stop()

	for {
		select {
		case <-accountWS.Done():
			return
		case <-dd.done:
			return
		case <-commissionAssetTimer.C:
			select {
			case commissionAssetValueCh <- 0.0:
			default:
			}
			commissionAssetTimer.Reset(time.Minute)
		case <-restartResetTimer.C:
			select {
			case statusCh <- common.SystemStatusReady:
				restartResetTimer.Reset(time.Hour * 9999)
			default:
				restartResetTimer.Reset(time.Minute * 3)
			}
		case a := <-httpAccountCh:
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
			for market, pos := range a.OpenPerpetualPositions {
				if ch, ok := positionChMap[market]; ok {
					hasPositions[market] = true
					if oldPos, ok := positionsMap[market]; ok {
						if pos.ParseTime.Sub(oldPos.ParseTime) > 0 {
							positionsMap[market] = pos
							p := pos
							select {
							case ch <- &p:
							default:
							}
						}
					} else {
						positionsMap[market] = pos
						p := pos
						select {
						case ch <- &p:
						default:
						}
					}
				}
			}
			for market, ch := range positionChMap {
				if _, ok := hasPositions[market]; !ok {
					pos := V4Position{
						Market:    market,
						ParseTime: time.Now(),
					}
					select {
					case ch <- &pos:
					default:
					}
				}
			}
		case <-accountWS.RestartCh:
			select {
			case statusCh <- common.SystemStatusRestart:
				restartResetTimer.Reset(time.Minute * 3)
			default:
			}
		case newAccount := <-accountWS.AccountCh:
			if account != nil {
				if newAccount.Equity != "" {
					account.Equity = newAccount.Equity
				}
				if newAccount.FreeCollateral != "" {
					account.FreeCollateral = newAccount.FreeCollateral
				}
				account.ParseTime = time.Now()
				outAccount := *account
				select {
				case accountCh <- &outAccount:
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("accountCh <- account failed, ch len %d", len(accountCh))
						logSilentTime = time.Now().Add(time.Minute)
					}
				}
			} else {
				account = &newAccount
				select {
				case accountCh <- account:
				default:
				}
			}
		case orders := <-accountWS.OrdersCh:
			for _, order := range orders {
				if ch, ok := orderChs[order.Ticker]; ok {
					o := order
					select {
					case ch <- &o:
					default:
						if time.Now().Sub(logSilentTime) > 0 {
							logger.Debugf("ch <- order failed, ch len %d", len(ch))
							logSilentTime = time.Now().Add(time.Minute)
						}
					}
				}
			}
		case wsPositions := <-accountWS.PositionsCh:
			for _, wsPosition := range wsPositions {
				if ch, ok := positionChMap[wsPosition.Market]; ok {
					p := wsPosition
					select {
					case ch <- &p:
					default:
						if time.Now().Sub(logSilentTime) > 0 {
							logger.Debugf("ch <- position failed, ch len %d", len(ch))
							logSilentTime = time.Now().Add(time.Minute)
						}
					}
				}
			}
		}
	}
}

func (dd *DydxV4UsdFuture) StreamSymbolStatus(ctx context.Context, channels map[string]chan common.SymbolStatusMsg, batchSize int) {
	checkInterval := time.Second * 5
	startTime := time.Now()
	updateTimes := make(map[string]time.Time)
	for symbol := range channels {
		updateTimes[symbol] = startTime
		startTime = startTime.Add(checkInterval)
	}
	loopTimer := time.NewTimer(time.Second)
	defer loopTimer.Stop()
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
					select {
					case ch <- status:
					default:
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

func (dd *DydxV4UsdFuture) StreamDepth(ctx context.Context, channels map[string]chan common.Depth, batchSize int) {
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
			ws := NewV4DepthWS(ctx, proxy, channels)
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

func (dd *DydxV4UsdFuture) StreamTrade(ctx context.Context, channels map[string]chan common.Trade, batchSize int) {
	logger.Debugf("START StreamTrade")
	defer logger.Debugf("STOP StreamTrade")
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
		subChannels := make(map[string]chan common.Trade)
		for _, symbol := range symbols[start:end] {
			subChannels[symbol] = channels[symbol]
		}
		go func(ctx context.Context, proxy string, channels map[string]chan common.Trade) {
			defer dd.Stop()
			ws := NewV4TradeWS(ctx, proxy, channels)
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

func (dd *DydxV4UsdFuture) StreamTicker(ctx context.Context, channels map[string]chan common.Ticker, batchSize int) {
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
			ws := NewV4TickerWS(ctx, proxy, channels)
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

func (dd *DydxV4UsdFuture) StreamKLine(ctx context.Context, channels map[string]chan []common.KLine, batchSize int, interval, lookback time.Duration) {
	// dYdX v4 indexer does not have a native KLine/candle WS channel
	// For now, this is a no-op that keeps the goroutine alive
	logger.Debugf("StreamKLine not natively supported on dYdX v4, running no-op loop")
	for {
		select {
		case <-ctx.Done():
			return
		case <-dd.done:
			return
		}
	}
}

func (dd *DydxV4UsdFuture) StreamFundingRate(ctx context.Context, channels map[string]chan common.FundingRate, batchSize int) {
	dd.mu.Lock()
	interval := time.Minute
	dd.mu.Unlock()
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	// dYdX v4 funding interval is every hour
	afterFrTimer := time.NewTimer(time.Now().Truncate(time.Hour).Add(time.Hour + time.Second).Sub(time.Now()))
	defer afterFrTimer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-dd.done:
			return
		case <-afterFrTimer.C:
			dd.fetchAndPushFundingRates(ctx, channels)
			afterFrTimer.Reset(time.Now().Truncate(time.Hour).Add(time.Hour + time.Second).Sub(time.Now()))
		case <-timer.C:
			dd.fetchAndPushFundingRates(ctx, channels)
			timer.Reset(time.Now().Truncate(interval).Add(interval).Sub(time.Now()))
		}
	}
}

func (dd *DydxV4UsdFuture) fetchAndPushFundingRates(ctx context.Context, channels map[string]chan common.FundingRate) {
	subCtx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	markets, err := dd.api.GetPerpetualMarkets(subCtx)
	if err != nil {
		logger.Debugf("GetPerpetualMarkets error %v", err)
		return
	}
	for symbol, ch := range channels {
		if m, ok := markets[symbol]; ok {
			fr := &V4FundingRate{
				Symbol:          symbol,
				FundingRate:     m.NextFundingRateFloat(),
				NextFundingTime: time.Now().Truncate(time.Hour).Add(time.Hour),
			}
			select {
			case ch <- fr:
			default:
				logger.Debugf("ch <- fr failed, %s ch len %d", symbol, len(ch))
			}
		}
	}
}

func (dd *DydxV4UsdFuture) WatchOrders(ctx context.Context, requestChannels map[string]chan common.OrderRequest, responseChannels map[string]chan common.Order, errorChannels map[string]chan common.OrderError) {
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

func (dd *DydxV4UsdFuture) WatchBatchOrders(ctx context.Context, requestChannels map[string]chan common.BatchOrderRequest, responseChannels map[string]chan common.Order, errorChannels map[string]chan common.OrderError) {
	// Phase 1: process batch orders sequentially
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
		go dd.watchBatchOrder(ctx, symbol, reqCh, respCh, errCh)
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

func (dd *DydxV4UsdFuture) watchBatchOrder(
	ctx context.Context,
	symbol string,
	requestCh chan common.BatchOrderRequest,
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
			if req.Cancel != nil {
				dd.cancelOrder(ctx, *req.Cancel, errorCh)
			}
			for _, newOrder := range req.New {
				newOrder := newOrder
				dd.submitOrder(ctx, newOrder, responseCh, errorCh)
			}
		}
	}
}

func (dd *DydxV4UsdFuture) GenerateClientID() string {
	return fmt.Sprintf("%d%04d", time.Now().Unix(), rand.Intn(10000))
}

func (dd *DydxV4UsdFuture) StartSideLoop() {
	// No additional side loop needed for Phase 1
	logger.Debugf("DydxV4UsdFuture StartSideLoop (no-op)")
}

func (dd *DydxV4UsdFuture) systemStatusLoop(ctx context.Context, output chan common.SystemStatus) {
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
			}
			timer.Reset(time.Now().Truncate(pullInterval).Add(pullInterval).Sub(time.Now()))
		}
	}
}

func (dd *DydxV4UsdFuture) accountLoop(ctx context.Context, output chan Subaccount) {
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
			subCtx, cancel := context.WithTimeout(ctx, time.Minute)
			account, err := dd.api.GetSubaccount(subCtx)
			cancel()
			if err != nil {
				logger.Debugf("GetSubaccount error %v", err)
			} else {
				output <- *account
			}
			timer.Reset(time.Now().Truncate(pullInterval).Add(pullInterval).Sub(time.Now()))
		}
	}
}

// --- Variants for merged ticker / walked depth ---

type DydxV4UsdFutureWithMergedTicker struct {
	DydxV4UsdFuture
}

func (k *DydxV4UsdFutureWithMergedTicker) StreamTicker(ctx context.Context, channels map[string]chan common.Ticker, batchSize int) {
	logger.Debugf("START StreamTicker (MergedTicker)")
	defer logger.Debugf("STOP StreamTicker (MergedTicker)")
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
			ws := NewV4TickerWS(ctx, proxy, channels)
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
