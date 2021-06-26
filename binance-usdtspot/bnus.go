package binance_usdtspot

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

type BinanceUsdtSpot struct {
	api      *API
	done     chan interface{}
	stopped  bool
	mu       sync.Mutex
	settings common.ExchangeSettings
}

func (bn *BinanceUsdtSpot) IsSpot() bool {
	return true
}

func (bn *BinanceUsdtSpot) StreamSymbolStatus(ctx context.Context, channels map[string]chan common.SymbolStatusMsg, batchSize int) {
	panic("implement me")
}

func (bn *BinanceUsdtSpot) GenerateClientID() string {
	return fmt.Sprintf("%d%04d", time.Now().Unix(), rand.Intn(10000))
}

func (bn *BinanceUsdtSpot) GetMultiplier(symbol string) (float64, error) {
	return 1.0, nil
}

func (bn *BinanceUsdtSpot) GetMinNotional(symbol string) (float64, error) {
	if value, ok := MinNotionals[symbol]; ok {
		return value, nil
	} else {
		return 0, fmt.Errorf(common.MinNotionalNotFoundError, symbol)
	}
}

func (bn *BinanceUsdtSpot) GetMinSize(symbol string) (float64, error) {
	if value, ok := MinSizes[symbol]; ok {
		return value, nil
	} else {
		return 0, fmt.Errorf(common.MinSizeNotFoundError, symbol)
	}
}

func (bn *BinanceUsdtSpot) GetStepSize(symbol string) (float64, error) {
	if value, ok := StepSizes[symbol]; ok {
		return value, nil
	} else {
		return 0, fmt.Errorf(common.StepSizeNotFoundError, symbol)
	}
}

func (bn *BinanceUsdtSpot) GetTickSize(symbol string) (float64, error) {
	if value, ok := TickSizes[symbol]; ok {
		return value, nil
	} else {
		return 0, fmt.Errorf(common.TickSizeNotFoundError, symbol)
	}
}

func (bn *BinanceUsdtSpot) StreamBasic(ctx context.Context, statusCh chan common.SystemStatus, usdtAccountCh chan common.Balance, commissionAssetValueCh chan float64, positionChMap map[string]chan common.Position, orderChMap map[string]chan common.Order) {
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
	internalAccountCh := make(chan Account, 4)
	go bn.watchAccount(ctx, internalAccountCh)
	go bn.watchSystemStatus(ctx, statusCh)
	logSilentTime := time.Now()
	usdtAccount := Balance{
		Asset:     "USDT",
		Free:      0,
		Locked:    0,
		EventTime: time.Time{},
		ParseTime: time.Time{},
	}
	bnbPriceCh := make(chan float64, 4)
	go bn.watchBnbPrice(ctx, bnbPriceCh)
	var bnbBalance *float64

	restartToReadyTimer := time.NewTimer(time.Hour * 9999)
	defer restartToReadyTimer.Stop()
	var rebalancedBnbSilentTime = time.Now()
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
		case price := <-bnbPriceCh:
			if bn.settings.AutoAddCommissionDiscountAsset {
				if bnbBalance != nil {
					select {
					case commissionAssetValueCh <- *bnbBalance * price:
					default:
						logger.Debugf("commissionAssetValueCh <- *bnbBalance * price failed ch len %d", len(commissionAssetValueCh))
					}
					if bn.settings.MinimalCommissionDiscountAssetValue*0.5 > *bnbBalance*price &&
						time.Now().Sub(rebalancedBnbSilentTime) > 0 {
						deltaValue := bn.settings.MinimalCommissionDiscountAssetValue - *bnbBalance*price
						if deltaValue > MinNotionals["BNBUSDT"] {
							go bn.buyBnb(ctx, deltaValue, price)
							rebalancedBnbSilentTime = time.Now().Add(time.Hour)
						}
					}
				}
			} else {
				select {
				case commissionAssetValueCh <- 0.0:
				default:
					logger.Debugf("commissionAssetValueCh <- *bnbAsset.MarginBalance*price failed ch len %d", len(commissionAssetValueCh))
				}
			}
		case account := <-userWS.AccountUpdateEventCh:
			for _, wsBalance := range account.Balances {
				if wsBalance.Asset == "USDT" {
					if wsBalance.EventTime.Sub(usdtAccount.EventTime) > 0 {
						usdtAccount.Free = wsBalance.FreeAmount
						usdtAccount.Locked = wsBalance.LockedAmount
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
				symbol := wsBalance.Asset + "USDT"
				if ch, ok := positionChMap[symbol]; ok {
					lastBalance, ok := balancesMap[wsBalance.Asset]
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
			for _, balance := range account.Balances {
				balance := balance
				if balance.Asset == "USDT" {
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
				} else if balance.Asset == "BNB" {
					bnbBalance = &balance.Free
					continue
				}
				symbol := balance.Asset + "USDT"
				lastBalance, ok := balancesMap[balance.Asset]
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
						Asset:     strings.Replace(symbol, "USDT", "", -1),
						EventTime: account.EventTime,
						ParseTime: account.ParseTime,
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

func (bn *BinanceUsdtSpot) buyBnb(
	ctx context.Context,
	deltaValue float64,
	price float64,
) {
	size := math.Round(deltaValue/price/StepSizes["BNBUSDT"]) * StepSizes["BNBUSDT"]
	if size*price < MinNotionals["BNBUSDT"] {
		return
	}
	childCtx, _ := context.WithTimeout(ctx, time.Minute)
	_, _, err := bn.api.SubmitOrder(childCtx, NewOrderParams{
		Symbol:           "BNBUSDT",
		Side:             OrderSideBuy,
		Type:             OrderTypeMarket,
		Quantity:         size,
		NewClientOrderID: fmt.Sprintf("%d%04d", time.Now().Unix(), rand.Intn(10000)),
	})
	if err != nil {
		logger.Debugf("bn.api.SubmitOrder buy %f bnb error %v", size, err)
		return
	} else {
		logger.Debugf("BNB bn.usApi.SubmitOrder buy %f bnb success, wait 3 minutes", size)
	}
}

func (bn *BinanceUsdtSpot) StreamDepth(ctx context.Context, channels map[string]chan common.Depth, batchSize int) {
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

func (bn *BinanceUsdtSpot) StreamTrade(ctx context.Context, channels map[string]chan common.Trade, batchSize int) {
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

func (bn *BinanceUsdtSpot) StreamTicker(ctx context.Context, channels map[string]chan common.Ticker, batchSize int) {
	panic("implement me")
}

func (bn *BinanceUsdtSpot) StreamKLine(ctx context.Context, channels map[string]chan []common.KLine, batchSize int, interval, lookback time.Duration) {
	panic("implement me")
}

func (bn *BinanceUsdtSpot) StreamFundingRate(ctx context.Context, channels map[string]chan common.FundingRate, batchSize int) {
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

func (bn *BinanceUsdtSpot) WatchOrders(ctx context.Context, requestChannels map[string]chan common.OrderRequest, responseChannels map[string]chan common.Order, errorChannels map[string]chan common.OrderError) {
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

func (bn *BinanceUsdtSpot) Setup(ctx context.Context, settings common.ExchangeSettings) (err error) {
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
	}
	symbol := "BNBUSDT"
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
	return
}

func (bn *BinanceUsdtSpot) Stop() {
	bn.mu.Lock()
	defer bn.mu.Unlock()
	if !bn.stopped {
		bn.stopped = true
		close(bn.done)
		logger.Debugf("stopped")
	}
}

func (bn *BinanceUsdtSpot) Done() chan interface{} {
	return bn.done
}

func (bn *BinanceUsdtSpot) watchSystemStatus(
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

func (bn *BinanceUsdtSpot) watchBnbPrice(
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
			ticker, err := bn.api.GetTicker(ctx, TickerParam{Symbol: "BNBUSDT"})
			if err != nil {
				logger.Debugf("spotApi.GetTicker BNBUSDT error %v", err)
				pullTimer.Reset(time.Minute)
			} else {
				select {
				case priceCh <- ticker.Price:
				default:
					logger.Debugf("priceCh <- ticker.Price failed ch len %d", len(priceCh))
				}
				pullTimer.Reset(time.Minute * 5)
			}
		}
	}
}

func (bn *BinanceUsdtSpot) watchAccount(
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

func (bn *BinanceUsdtSpot) watchOrder(
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

func (bn *BinanceUsdtSpot) submitOrder(ctx context.Context, param common.NewOrderParam, tickSize, stepSize float64, respCh chan common.Order, errCh chan common.OrderError) {
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
	if !param.PostOnly {
		switch param.TimeInForce {
		case common.OrderTimeInForceIOC:
			newOrderParam.TimeInForce = OrderTimeInForceIOC
		case common.OrderTimeInForceGTC:
			newOrderParam.TimeInForce = OrderTimeInForceGTC
		case common.OrderTimeInForceFOK:
			newOrderParam.TimeInForce = OrderTimeInForceFOK
		}
	}
	if param.Price != 0 {
		newOrderParam.Price = math.Round(param.Price/tickSize) * tickSize
	}
	newOrderParam.NewClientOrderID = param.ClientID
	//logger.Debugf("%s SubmitOrder", newOrderParam.Symbol)
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

func (bn *BinanceUsdtSpot) cancelOrder(ctx context.Context, param common.CancelOrderParam, errCh chan common.OrderError) {
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

type BinanceUsdtSpotWithDepth5 struct {
	BinanceUsdtSpot
}

func (b BinanceUsdtSpotWithDepth5) WatchBatchOrders(ctx context.Context, requestChannels map[string]chan common.BatchOrderRequest, responseChannels map[string]chan common.Order, errorChannels map[string]chan common.OrderError) {
	panic("implement me")
}

func (b BinanceUsdtSpotWithDepth5) StartSideLoop() {
	panic("implement me")
}

type BinanceUsdtSpotWithDepth20 struct {
	BinanceUsdtSpot
}

func (bn *BinanceUsdtSpotWithDepth20) WatchBatchOrders(ctx context.Context, requestChannels map[string]chan common.BatchOrderRequest, responseChannels map[string]chan common.Order, errorChannels map[string]chan common.OrderError) {
	panic("implement me")
}

func (bn *BinanceUsdtSpotWithDepth20) StartSideLoop() {
	panic("implement me")
}

func (bn *BinanceUsdtSpotWithDepth20) StreamDepth(ctx context.Context, channels map[string]chan common.Depth, batchSize int) {
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
			defer bn.Stop()
			ws := NewDepth20WS(ctx, proxy, channels)
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
