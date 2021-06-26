package binance_coinfuture

import (
	"context"
	"errors"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math/rand"
	"sync"
	"time"
)

type Exchange struct {
	api      *API
	done     chan interface{}
	stopped  bool
	mu       sync.Mutex
	settings common.ExchangeSettings
}

func (bn *Exchange) IsSpot() bool {
	return false
}

func (bn *Exchange) StreamSymbolStatus(ctx context.Context, channels map[string]chan common.SymbolStatusMsg, batchSize int) {
	panic("implement me")
}

func (bn *Exchange) GenerateClientID() string {
	return fmt.Sprintf("%d%04d", time.Now().Unix(), rand.Intn(10000))
}

func (bn *Exchange) GetMinNotional(symbol string) (float64, error) {
	return 0.0, nil

}

func (bn *Exchange) GetMultiplier(symbol string) (float64, error) {
	if value, ok := ContractSizes[symbol]; ok {
		return value, nil
	} else {
		return 0, fmt.Errorf(common.ContractSizeNotFoundError, symbol)
	}
}

func (bn *Exchange) GetMinSize(symbol string) (float64, error) {
	if value, ok := MinSizes[symbol]; ok {
		return value, nil
	} else {
		return 0, fmt.Errorf(common.MinSizeNotFoundError, symbol)
	}
}

func (bn *Exchange) GetStepSize(symbol string) (float64, error) {
	if value, ok := StepSizes[symbol]; ok {
		return value, nil
	} else {
		return 0, fmt.Errorf(common.StepSizeNotFoundError, symbol)
	}
}

func (bn *Exchange) GetTickSize(symbol string) (float64, error) {
	if value, ok := TickSizes[symbol]; ok {
		return value, nil
	} else {
		return 0, fmt.Errorf(common.TickSizeNotFoundError, symbol)
	}
}

func (bn *Exchange) StreamBasic(
	ctx context.Context,
	statusCh chan common.SystemStatus,
	balanceChMap map[string]chan common.Balance,
	commissionAssetValueCh chan float64,
	positionChMap map[string]chan common.Position, orderChMap map[string]chan common.Order) {
	defer bn.Stop()
	bn.mu.Lock()
	proxy := bn.settings.Proxy
	bn.mu.Unlock()
	userWS, err := NewUserWebsocket(ctx, bn.api, proxy)
	if err != nil {
		logger.Debugf("NewUserWebsocket(ctx,  bn.api, proxy) error %v", err)
		return
	}
	balancesMap := make(map[string]WSBalance)
	positionSymbols := make([]string, 0)
	for symbol := range positionChMap {
		positionSymbols = append(positionSymbols, symbol)
	}
	//internalAccountCh := make(chan Account, 10)
	//go bn.watchAccount(ctx, internalAccountCh)
	go bn.watchSystemStatus(ctx, statusCh)

	logSilentTime := time.Now()
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
		case bp := <-userWS.BalanceAndPositionUpdateEventCh:
			for _, nextPos := range bp.Account.Positions {
				if nextPos.PositionSide != "BOTH" {
					continue
				}
				if outputCh, ok := positionChMap[nextPos.Symbol]; ok {
					nextPos := nextPos
					select {
					case outputCh <- &nextPos:
					default:
						if time.Now().Sub(logSilentTime) > 0 {
							logger.Debugf("outputCh <- &nextPos failed, ch len %d", len(outputCh))
							logSilentTime = time.Now().Add(time.Minute)
						}
					}
				}
			}
			for _, balanceUpdate := range bp.Account.Balances {
				balanceCh, okCh := balanceChMap[balanceUpdate.Asset]
				balance, okB := balancesMap[balanceUpdate.Asset]
				if okCh && okB && balance.EventTime.Sub(balanceUpdate.EventTime) < 0 {
					if balanceUpdate.CrossWalletBalance != nil {
						balance.CrossWalletBalance = *balanceUpdate.CrossWalletBalance
						balance.EventTime = balanceUpdate.EventTime
						balance.ParseTime = balanceUpdate.ParseTime
						balancesMap[balanceUpdate.Asset] = balance
						select {
						case balanceCh <- &balance:
						default:
							if time.Now().Sub(logSilentTime) > 0 {
								logger.Debugf("balanceCh <- &asset failed, ch len %d", len(balanceCh))
								logSilentTime = time.Now().Add(time.Minute)
							}
						}
					}
				}
			}
			break
		case wsOrder := <-userWS.OrderUpdateEventCh:
			if ch, ok := orderChMap[wsOrder.Order.Symbol]; ok {
				select {
				case ch <- &wsOrder.Order:
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("ch <- &wsOrder.Order failed, ch len %d", len(ch))
						logSilentTime = time.Now().Add(time.Minute)
					}
				}
			}
			break
		case wsPositions := <-userWS.PositionsCh:
			for _, nextPos := range wsPositions {
				if nextPos.PositionSide != "BOTH" {
					continue
				}
				if outputCh, ok := positionChMap[nextPos.Symbol]; ok {
					nextPos := nextPos
					select {
					case outputCh <- &nextPos:
					default:
						if time.Now().Sub(logSilentTime) > 0 {
							logger.Debugf("outputCh <- &nextPos failed, ch len %d", len(outputCh))
							logSilentTime = time.Now().Add(time.Minute)
						}
					}
				}
			}
			break
		case wsBalances := <-userWS.BalancesCh:
			for _, nextBalance := range wsBalances {
				balanceCh, okCh := balanceChMap[nextBalance.Asset]
				balance, okB := balancesMap[nextBalance.Asset]
				if (okCh && okB && balance.EventTime.Sub(nextBalance.EventTime) < 0) || (okCh && !okB) {
					nextBalance := nextBalance
					balancesMap[nextBalance.Asset] = nextBalance
					select {
					case balanceCh <- &nextBalance:
					default:
						if time.Now().Sub(logSilentTime) > 0 {
							logger.Debugf("balanceCh <- &nextBalance failed, ch len %d", len(balanceCh))
							logSilentTime = time.Now().Add(time.Minute)
						}
					}
				}
			}

			//case account := <-internalAccountCh:
			//	for _, asset := range account.Assets {
			//		balanceCh, okCh := balanceChMap[asset.Asset]
			//		balance, okB := balancesMap[asset.Asset]
			//		if okCh && okB && balance.EventTime.Sub(asset.EventTime) < 0 {
			//			balance.EventTime = asset.EventTime
			//			balance.Balance = asset.MarginBalance
			//			balance.AvailableBalance = asset.AvailableBalance
			//			balance.CrossWalletBalance = asset.CrossWalletBalance
			//			balance.MaxWithdrawAmount = asset.MaxWithdrawAmount
			//			select {
			//			case balanceCh <- &balance:
			//			default:
			//				if time.Now().Sub(logSilentTime) > 0 {
			//					logger.Debugf("balanceCh <- &balance failed, ch len %d", len(balanceCh))
			//					logSilentTime = time.Now().Add(time.Minute)
			//				}
			//			}
			//		}
			//	}
			//	hasPositions := make(map[string]bool)
			//	for _, nextPos := range account.Positions {
			//		//logger.Debugf("%s %v", nextPos.Symbol, nextPos.EventTime)
			//		if nextPos.PositionSide != "BOTH" {
			//			continue
			//		}
			//		if outputCh, ok := positionChMap[nextPos.Symbol]; ok {
			//			hasPositions[nextPos.Symbol] = true
			//			nextPos := nextPos
			//			select {
			//			case outputCh <- &nextPos:
			//			default:
			//				if time.Now().Sub(logSilentTime) > 0 {
			//					logger.Debugf("outputCh <- &nextPos failed, ch len %d", len(outputCh))
			//					logSilentTime = time.Now().Add(time.Minute)
			//				}
			//			}
			//		}
			//	}
			//	for symbol, ch := range positionChMap {
			//		if _, ok := hasPositions[symbol]; !ok {
			//			select {
			//			case ch <- &HttpPosition{
			//				Symbol:       symbol,
			//				PositionSide: "BOTH",
			//				EventTime:    account.EventTime,
			//				ParseTime:    account.ParseTime,
			//			}:
			//			default:
			//				if time.Now().Sub(logSilentTime) > 0 {
			//					logger.Debugf("ch <- &Position failed, %s ch len %d", symbol, len(ch))
			//					logSilentTime = time.Now().Add(time.Minute)
			//				}
			//			}
			//		}
			//	}
			//	break
		}
	}

}

func (bn *Exchange) StreamDepth(ctx context.Context, channels map[string]chan common.Depth, batchSize int) {
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

func (bn *Exchange) StreamTrade(ctx context.Context, channels map[string]chan common.Trade, batchSize int) {
	panic("implement me")
}

func (bn *Exchange) StreamTicker(ctx context.Context, channels map[string]chan common.Ticker, batchSize int) {
	panic("implement me")
}

func (bn *Exchange) StreamKLine(ctx context.Context, channels map[string]chan []common.KLine, batchSize int, interval, lookback time.Duration) {
	panic("implement me")
}

func (bn *Exchange) StreamFundingRate(ctx context.Context, channels map[string]chan common.FundingRate, batchSize int) {
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
			subCtx, cancel := context.WithTimeout(ctx, time.Minute)
			indexes, err := bn.api.GetPremiumIndex(subCtx)
			if err != nil {
				logger.Debugf("WatchPositionsFromHttp GetPositions error %v", err)
			} else {
				for _, fr := range indexes {
					if ch, ok := channels[fr.Symbol]; ok {
						fr := fr
						select {
						case ch <- &fr:
						default:
						}
					}
				}
			}
			cancel()
			timer.Reset(time.Now().Truncate(pullInterval).Add(pullInterval).Sub(time.Now()))
		}
	}
}

func (bn *Exchange) WatchOrders(ctx context.Context, requestChannels map[string]chan common.OrderRequest, responseChannels map[string]chan common.Order, errorChannels map[string]chan common.OrderError) {
	defer bn.Stop()
	for symbol, reqCh := range requestChannels {
		//logger.Debugf("%v", responseChannels)
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
		go bn.watchOrder(ctx, symbol, reqCh, respCh, errCh)
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

func (bn *Exchange) Setup(ctx context.Context, settings common.ExchangeSettings) (err error) {
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
		if _, ok := MultiplierUps[symbol]; !ok {
			return fmt.Errorf("multiplier up not found for %s", symbol)
		}
		if _, ok := MultiplierDowns[symbol]; !ok {
			return fmt.Errorf("multiplier down not found for %s", symbol)
		}
		if settings.ChangeLeverage {
			res, err := bn.api.ChangeLeverage(ctx, LeverageParams{
				Symbol:   symbol,
				Leverage: int64(settings.Leverage),
			})
			if err != nil {
				logger.Debugf("UPDATE LEVERAGE FOR %s ERROR %v", symbol, err)
			} else {
				logger.Debugf("UPDATE LEVERAGE FOR %s RESPONSE %v", symbol, res)
			}
		}
		if settings.ChangeMarginType {
			res, err := bn.api.ChangeMarginType(ctx, MarginTypeParams{
				Symbol:     symbol,
				MarginType: settings.MarginType,
			})
			if err != nil {
				logger.Debugf("UPDATE MARGIN TYPE FOR %s ERROR %v", symbol, err)
			} else {
				logger.Debugf("UPDATE MARGIN TYPE FOR %s RESPONSE %v", symbol, res)
			}
		}
		time.Sleep(time.Second)
	}
	return
}

func (bn *Exchange) Stop() {
	bn.mu.Lock()
	if !bn.stopped {
		bn.stopped = true
		close(bn.done)
		logger.Debugf("stopped")
	}
	bn.mu.Unlock()
}

func (bn *Exchange) Done() chan interface{} {
	return bn.done
}

func (bn *Exchange) watchSystemStatus(
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

func (bn *Exchange) watchPositions(
	ctx context.Context, symbols []string, output chan []HttpPosition,
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
			positions, err := bn.api.GetPositions(subCtx)
			if err != nil {
				logger.Debugf("bn.api.GetPositions(subCtx) error %v", err)
			} else {
				//有一种情况是有的合约的仓位是拉不到的, 拉不到的都是空仓
				positionBySymbols := make(map[string]HttpPosition)
				for _, symbol := range symbols {
					positionBySymbols[symbol] = HttpPosition{
						Symbol:       symbol,
						PositionSide: "BOTH",
					}
				}
				for _, position := range positions {
					position := position
					position.ParseTime = time.Now()
					positionBySymbols[position.Symbol] = position
				}
				outPositions := make([]HttpPosition, len(symbols))
				for i, symbol := range symbols {
					outPositions[i] = positionBySymbols[symbol]
				}
				output <- outPositions
			}
			timer.Reset(time.Now().Truncate(updateInterval).Add(updateInterval).Sub(time.Now()))
		}
	}
}

func (bn *Exchange) watchAccount(
	ctx context.Context,
	outputAccount chan Account,
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
			account, err := bn.api.GetAccount(subCtx)
			if err != nil {
				logger.Debugf("bn.api.GetAccount(subCtx) error %v", err)
			} else {
				select {
				case outputAccount <- *account:
				default:
					logger.Debugf("outputAccount <- *account failed, ch len %d", len(outputAccount))
				}
			}
			timer.Reset(time.Now().Truncate(updateInterval).Add(updateInterval).Sub(time.Now()))
		}
	}
}

func (bn *Exchange) watchOrder(
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

func (bn *Exchange) submitOrder(ctx context.Context, param common.NewOrderParam, respCh chan common.Order, errCh chan common.OrderError) {
	newOrderParam := NewOrderParams{}
	newOrderParam.Symbol = param.Symbol
	newOrderParam.Quantity = param.Size
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
	case common.OrderTimeInForceIOC:
		newOrderParam.TimeInForce = OrderTimeInForceIOC
	case common.OrderTimeInForceGTC:
		newOrderParam.TimeInForce = OrderTimeInForceGTC
	case common.OrderTimeInForceFOK:
		newOrderParam.TimeInForce = OrderTimeInForceFOK
	}
	if param.PostOnly {
		newOrderParam.TimeInForce = OrderTimeInForceGTX
	}
	newOrderParam.ReduceOnly = param.ReduceOnly
	if param.Price != 0 {
		newOrderParam.Price = param.Price
	}
	newOrderParam.NewClientOrderId = param.ClientID
	order, err := bn.api.SubmitOrder(ctx, newOrderParam)
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

func (bn *Exchange) cancelOrder(ctx context.Context, param common.CancelOrderParam, errCh chan common.OrderError) {

	if param.ClientID != "" && param.Symbol != "" {
		cancelOrderParam := CancelOrderParam{
			Symbol:            param.Symbol,
			OrigClientOrderId: param.ClientID,
		}
		_, err := bn.api.CancelOrder(ctx, cancelOrderParam)
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
	} else if param.Symbol != "" {
		_, err := bn.api.CancelAllOpenOrders(ctx, CancelAllOrderParams{
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

type ExchangeWidthDepth5 struct {
	Exchange
}

type ExchangeWidthDepth20 struct {
	Exchange
}

func (bn *ExchangeWidthDepth20) StreamDepth(ctx context.Context, channels map[string]chan common.Depth, batchSize int) {
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
