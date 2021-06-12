package kucoin_usdtfuture

import (
	"context"
	"errors"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"math/rand"
	"sync"
	"time"
)

type Kcperp struct {
	done     chan interface{}
	stopped  bool
	mu       sync.Mutex
	api      *API
	settings common.ExchangeSettings
}

func (k *Kcperp) IsSpot() bool {
	return false
}

func (k *Kcperp) Done() chan interface{} {
	return k.done
}

func (k *Kcperp) Stop() {
	k.mu.Lock()
	defer k.mu.Unlock()
	if !k.stopped {
		close(k.done)
		logger.Debugf("stopped")
	}
}

func (k *Kcperp) Setup(ctx context.Context, settings common.ExchangeSettings) error {
	var err error
	k.stopped = false
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
		if settings.ChangeLeverage {
			_, err = k.api.ChangeAutoDepositStatus(ctx, AutoDepositStatusParam{
				Symbol: symbol,
				Status: true,
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (k *Kcperp) GetMinNotional(symbol string) (float64, error) {
	return 0.0, nil
}

func (k *Kcperp) GetMinSize(symbol string) (float64, error) {
	if v, ok := Multipliers[symbol]; ok {
		return v, nil
	} else {
		return 0.0, fmt.Errorf(common.MinNotionalNotFoundError, symbol)
	}
}

func (k *Kcperp) GetStepSize(symbol string) (float64, error) {
	if v, ok := Multipliers[symbol]; ok {
		return v, nil
	} else {
		return 0.0, fmt.Errorf(common.StepSizeNotFoundError, symbol)
	}
}

func (k *Kcperp) GetTickSize(symbol string) (float64, error) {
	if v, ok := TickSizes[symbol]; ok {
		return v, nil
	} else {
		return 0.0, fmt.Errorf(common.TickSizeNotFoundError, symbol)
	}
}

func (k *Kcperp) StreamBasic(ctx context.Context, statusCh chan common.SystemStatus, accountChMap map[string]chan common.Balance, positionChMap map[string]chan common.Position, orderChs map[string]chan common.Order) {
	defer k.Stop()
	k.mu.Lock()
	settings := k.settings
	symbols := k.settings.Symbols[:]
	k.mu.Unlock()
	var err error
	k.api, err = NewAPI(settings.ApiKey, settings.ApiSecret, settings.ApiPassphrase, settings.Proxy)
	if err != nil {
		return
	}
	userWS := NewUserWebsocket(
		k.api,
		symbols[:],
		settings.Proxy,
	)
	go userWS.Start(ctx)
	go k.systemStatusLoop(ctx, statusCh)
	httpPositionsCh := make(chan map[string]Position)
	go k.positionsLoop(ctx, symbols, httpPositionsCh)
	httpAccountCh := make(chan Account, 100)
	go k.accountLoop(ctx, httpAccountCh)
	logSilentTime := time.Now()
	restartResetTimer := time.NewTimer(time.Hour * 9999)
	defer restartResetTimer.Stop()
	var account *Account
	positionsMap := make(map[string]Position)
	for {
		select {
		case <-userWS.Done():
			return
		case <-k.done:
			return
		case positionsMap = <-httpPositionsCh:
			for symbol, pos := range positionsMap {
				if ch, ok := positionChMap[symbol]; ok {
					//logger.Debugf("%v", pos)
					pos := pos
					select {
					case ch <- &pos:
					default:
						logger.Debugf("ch <- &pos failed, ch len %d", len(ch))
					}
				}
			}
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
		case a := <-httpAccountCh:
			account = &a
			if accountCh, ok := accountChMap["USDT"]; ok {
				select {
				case accountCh <- account:
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("accountCh <- account failed, ch len %d", len(accountCh))
						logSilentTime = time.Now().Add(time.Minute)
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
		case balance := <-userWS.BalanceCh:
			if account != nil {
				if account.EventTime.Sub(balance.EventTime) > 0 {
					continue
				}
				account.EventTime = balance.EventTime
				if balance.AvailableBalance != nil {
					account.AvailableBalance = *balance.AvailableBalance
				}
				if balance.HoldBalance != nil {
					account.FrozenFunds = *balance.HoldBalance
				}
				if balance.OrderMargin != nil {
					account.OrderMargin = *balance.OrderMargin
				}
				outputAccount := *account
				if accountCh, ok := accountChMap["USDT"]; ok {
					select {
					case accountCh <- &outputAccount:
					default:
						if time.Now().Sub(logSilentTime) > 0 {
							logger.Debugf("ch <- &order failed, ch len %d", len(accountCh))
							logSilentTime = time.Now().Add(time.Minute)
						}
					}
				}
			}
		case order := <-userWS.OrderCh:
			//DEBUG 2021/05/22 01:21:54.774559 kucoin-usdtfuture.go:606: 	k.api.SubmitOrder {"clientOid":"16216465147940","side":"buy","symbol":"BNBUSDTM","type":"market","leverage":3,"size":24,"reduceOnly":true}
			//DEBUG 2021/05/22 01:21:54.774573 kucoin-usdtfuture-api.go:77: 	{"clientOid":"16216465147940","side":"buy","symbol":"BNBUSDTM","type":"market","leverage":3,"size":24,"reduceOnly":true}
			//DEBUG 2021/05/22 01:21:54.824218 kucoin-usdtfuture.go:608: 	k.api.SubmitOrder &{60a85cb28833a40006067120} <nil>
			//DEBUG 2021/05/22 01:21:54.833944 kucoin-usdtfuture-user-ws.go:170: 	KCPERP WS ORDER {"symbol":"BNBUSDTM","orderType":"market","side":"buy","canceledSize":"0","orderId":"60a85cb28833a40006067120","liquidity":"taker","type":"match","orderTime":1621646514806444042,"size":"24","filledSize":"1","matchPrice":"325.7","matchSize":"1","tradeId":"60a85cb2b87b911178425c71","remainSize":"23","clientOid":"16216465147940","status":"match","ts":1621646514814245799}
			//DEBUG 2021/05/22 01:21:54.834155 main.go:326: 	x order filled BNBUSDTM FILLED size 1.000000 price 325.700000
			//DEBUG 2021/05/22 01:21:54.839608 kucoin-usdtfuture-user-ws.go:170: 	KCPERP WS ORDER {"symbol":"BNBUSDTM","orderType":"market","side":"buy","canceledSize":"0","orderId":"60a85cb28833a40006067120","type":"filled","orderTime":1621646514806444042,"size":"24","filledSize":"24","price":"","remainSize":"0","clientOid":"16216465147940","status":"done","ts":1621646514814245799}
			//DEBUG 2021/05/22 01:21:54.839661 kucoin-usdtfuture-user-ws.go:170: 	KCPERP WS ORDER {"symbol":"BNBUSDTM","orderType":"market","side":"buy","canceledSize":"0","orderId":"60a85cb28833a40006067120","liquidity":"taker","type":"match","orderTime":1621646514806444042,"size":"24","filledSize":"24","matchPrice":"325.94","matchSize":"21","tradeId":"60a85cb2b87b911178425c73","remainSize":"0","clientOid":"16216465147940","status":"match","ts":1621646514814245799}
			//DEBUG 2021/05/22 01:21:54.839676 kucoin-usdtfuture-user-ws.go:170: 	KCPERP WS ORDER {"symbol":"BNBUSDTM","orderType":"market","side":"buy","canceledSize":"0","orderId":"60a85cb28833a40006067120","liquidity":"taker","type":"match","orderTime":1621646514806444042,"size":"24","filledSize":"3","matchPrice":"325.73","matchSize":"2","tradeId":"60a85cb2b87b911178425c72","remainSize":"21","clientOid":"16216465147940","status":"match","ts":1621646514814245799}
			//滤掉没有价格的事件
			if order.EventType == "filled" || order.EventType == "match" {
				if order.FilledSize == 0 || order.MatchPrice == 0 {
					continue
				}
			}
			if ch, ok := orderChs[order.Symbol]; ok {
				select {
				case ch <- order:
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("ch <- &order failed, ch len %d", len(ch))
						logSilentTime = time.Now().Add(time.Minute)
					}
				}
			}
		case wsPosition := <-userWS.PositionCh:
			if position, ok := positionsMap[wsPosition.Symbol]; ok {
				if position.EventTime.Sub(wsPosition.EventTime) > 0 {
					continue
				}
				position.EventTime = wsPosition.EventTime
				if wsPosition.AvgEntryPrice != nil {
					position.AvgEntryPrice = *wsPosition.AvgEntryPrice
				}
				if wsPosition.UnrealisedPnl != nil {
					position.UnrealisedPnl = *wsPosition.UnrealisedPnl
				}
				if wsPosition.CurrentQty != nil {
					position.CurrentQty = *wsPosition.CurrentQty
				}
				if wsPosition.UnrealisedPnlPcnt != nil {
					position.UnrealisedPnlPcnt = *wsPosition.UnrealisedPnlPcnt
				}
				if wsPosition.UnrealisedRoePcnt != nil {
					position.UnrealisedRoePcnt = *wsPosition.UnrealisedRoePcnt
				}
				positionsMap[wsPosition.Symbol] = position
				if ch, ok := positionChMap[position.Symbol]; ok {
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
		}
	}
}

func (k *Kcperp) StreamSymbolStatus(ctx context.Context, channels map[string]chan common.SymbolStatusMsg, batchSize int) {
	checkInterval := time.Second * 5
	startTime := time.Now()
	updateTimes := make(map[string]time.Time)
	for symbol := range channels {
		updateTimes[symbol] = startTime
		startTime = startTime.Add(checkInterval)
	}
	loopTimer := time.NewTimer(time.Second)
	k.mu.Lock()
	leverage := int(k.settings.Leverage)
	k.mu.Unlock()
	for {
		select {
		case <-ctx.Done():
			return
		case <-k.done:
			return
		case <-loopTimer.C:
			for symbol, ch := range channels {
				if time.Now().Sub(updateTimes[symbol]) > 0 {
					status := common.SymbolStatusReady
					ticker, err := k.api.GetTicker(ctx, TickerParam{
						Symbol: symbol,
					})
					if err != nil {
						logger.Debugf("%s k.api.GetTicker error %v", symbol, err)
						status = common.SymbolStatusNotReady
					} else {
						size := LotSizes[symbol]
						price := ticker.BestAskPrice * 1.05
						price = math.Ceil(price/TickSizes[symbol]) * TickSizes[symbol]
						_, err := k.api.SubmitOrder(ctx, NewOrderParam{
							Symbol:      symbol,
							Side:        OrderSideSell,
							TimeInForce: OrderTimeInForceIOC,
							Price:       common.Float64(price),
							Size:        int64(size),
							Leverage:    leverage,
						})
						if err != nil {
							logger.Debugf("k.api.SubmitOrder error %v", err)
							status = common.SymbolStatusNotReady
						}
					}
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

func (k *Kcperp) StreamDepth(ctx context.Context, channels map[string]chan common.Depth, batchSize int) {
	logger.Debugf("START StreamDepth")
	defer logger.Debugf("STOP StreamDepth")
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
		subChannels := make(map[string]chan common.Depth)
		for _, symbol := range symbols[start:end] {
			subChannels[symbol] = channels[symbol]
		}
		go func(ctx context.Context, proxy string, channels map[string]chan common.Depth) {
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

func (k *Kcperp) StreamTrade(ctx context.Context, channels map[string]chan common.Trade, batchSize int) {
	panic("implement me")
}

func (k *Kcperp) StreamTicker(ctx context.Context, channels map[string]chan common.Ticker, batchSize int) {
	panic("implement me")
}

func (k *Kcperp) StreamKLine(ctx context.Context, channels map[string]chan []common.KLine, batchSize int, interval, lookback time.Duration) {
	panic("implement me")
}

func (k *Kcperp) StreamFundingRate(ctx context.Context, channels map[string]chan common.FundingRate, batchSize int) {
	k.mu.Lock()
	interval := k.settings.PullInterval
	k.mu.Unlock()
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
				subCtx, _ := context.WithTimeout(ctx, time.Minute)
				fr, err := k.api.GetCurrentFundingRate(subCtx, symbol)
				if err != nil {
					logger.Debugf("api.GetCurrentFundingRate error %v", err)
				} else {
					select {
					case ch <- fr:
					default:
						logger.Debugf("ch <- fr failed, %s ch len %d", symbol, len(ch))
					}
				}
				select {
				case <-ctx.Done():
					return
				case <-k.done:
					return
				case <-time.After(time.Second):
				}
			}
			timer.Reset(time.Now().Truncate(interval).Add(interval).Sub(time.Now()))
		}
	}
}

func (k *Kcperp) WatchOrders(ctx context.Context, requestChannels map[string]chan common.OrderRequest, responseChannels map[string]chan common.Order, errorChannels map[string]chan common.OrderError) {
	defer k.Stop()
	for symbol, reqCh := range requestChannels {
		tickSize, ok := TickSizes[symbol]
		if !ok {
			logger.Debugf("miss tick size for %s, exit", symbol)
			return
		}
		multiplier, ok := Multipliers[symbol]
		if !ok {
			logger.Debugf("miss multiplier for %s, exit", symbol)
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
		go k.watchOrder(ctx, symbol, tickSize, multiplier, reqCh, respCh, errCh)
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

func (k *Kcperp) GenerateClientID() string {
	return fmt.Sprintf("%d%04d", time.Now().Unix(), rand.Intn(10000))
}

func (k *Kcperp) systemStatusLoop(
	ctx context.Context, output chan common.SystemStatus,
) {
	k.mu.Lock()
	pullInterval := k.settings.PullInterval
	k.mu.Unlock()
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

func (k *Kcperp) accountLoop(
	ctx context.Context, output chan Account,
) {
	k.mu.Lock()
	pullInterval := k.settings.PullInterval
	k.mu.Unlock()
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			subCtx, _ := context.WithTimeout(ctx, time.Minute)
			account, err := k.api.GetAccountOverView(subCtx, AccountParam{
				Currency: "USDT",
			})
			if err != nil {
				logger.Debugf("k.api.GetAccountOverView error %v", err)
			} else {
				output <- *account
			}
			timer.Reset(time.Now().Truncate(pullInterval).Add(pullInterval).Sub(time.Now()))
		}
	}
}

func (k *Kcperp) positionsLoop(
	ctx context.Context,
	symbols []string,
	outputCh chan map[string]Position,
) {
	k.mu.Lock()
	pullInterval := k.settings.PullInterval
	k.mu.Unlock()
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			subCtx, _ := context.WithTimeout(ctx, time.Minute)
			positions, err := k.api.GetPositions(subCtx)
			if err != nil {
				logger.Debugf("api.GetPositions error %v", err)
			} else {
				//有一种情况是有的合约的仓位是拉不到的, 拉不到的都是空仓
				positionBySymbols := make(map[string]Position)
				for _, symbol := range symbols {
					positionBySymbols[symbol] = Position{
						Symbol:    symbol,
						ParseTime: time.Now(),
						EventTime: time.Now(),
					}
				}
				for _, position := range positions {
					position := position
					positionBySymbols[position.Symbol] = position
				}
				select {
				case outputCh <- positionBySymbols:
				default:
					logger.Debugf("outputCh <- positionBySymbols failed, ch len %d", len(outputCh))
				}
			}
			timer.Reset(time.Now().Truncate(pullInterval).Add(pullInterval).Sub(time.Now()))
		}
	}
}

func (k *Kcperp) watchOrder(
	ctx context.Context,
	symbol string,
	tickSize, stepSize float64,
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
				k.submitOrder(ctx, *req.New, tickSize, stepSize, responseCh, errorCh)
			} else if req.Cancel != nil {
				k.cancelOrder(ctx, *req.Cancel, errorCh)
			}
		}
	}
}

func (k *Kcperp) submitOrder(ctx context.Context, param common.NewOrderParam, tickSize, multiplier float64, respCh chan common.Order, errCh chan common.OrderError) {
	newOrderParam := NewOrderParam{}
	newOrderParam.Symbol = param.Symbol
	newOrderParam.Size = int64(math.Round(param.Size / multiplier))
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
	if param.TimeInForce == common.OrderTimeInForceIOC {
		newOrderParam.TimeInForce = OrderTimeInForceIOC
	}
	newOrderParam.PostOnly = param.PostOnly
	newOrderParam.ReduceOnly = param.ReduceOnly
	if param.Price != 0 {
		newOrderParam.Price = common.Float64(math.Round(param.Price/tickSize) * tickSize)
	}
	newOrderParam.ClientOid = param.ClientID
	k.mu.Lock()
	newOrderParam.Leverage = int(k.settings.Leverage)
	k.mu.Unlock()
	//str, _ := json.Marshal(newOrderParam)
	//logger.Debugf("k.api.SubmitOrder %s", str)
	_, err := k.api.SubmitOrder(ctx, newOrderParam)
	//logger.Debugf("k.api.SubmitOrder %v %v", resp, err)
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

func (k *Kcperp) cancelOrder(ctx context.Context, param common.CancelOrderParam, errCh chan common.OrderError) {
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
