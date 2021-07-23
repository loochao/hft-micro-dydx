package bybit_usdtfuture

import (
	"context"
	"errors"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math/rand"
	"sync/atomic"
	"time"
)

type BybitUsdtFuture struct {
	done     chan interface{}
	stopped  int32
	api      *API
	settings common.ExchangeSettings
}

func (h *BybitUsdtFuture) StreamSystemStatus(ctx context.Context, statusCh chan common.SystemStatus) {
	panic("implement me")
}

func (h *BybitUsdtFuture) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (h *BybitUsdtFuture) IsSpot() bool {
	return false
}

func (h *BybitUsdtFuture) Done() chan interface{} {
	return h.done
}

func (h *BybitUsdtFuture) Stop() {
	if atomic.CompareAndSwapInt32(&h.stopped, 0, 1) {
		close(h.done)
		logger.Debugf("stopped")
	}
}

func (h *BybitUsdtFuture) Setup(ctx context.Context, settings common.ExchangeSettings) error {
	var err error
	h.stopped = 0
	h.done = make(chan interface{})
	h.settings = settings
	h.api, err = NewAPI(settings.ApiKey, settings.ApiSecret, settings.ApiUrl, settings.Proxy)
	if err != nil {
		return err
	}
	for _, symbol := range settings.Symbols {
		if _, err = h.GetStepSize(symbol); err != nil {
			return err
		}
		if _, err = h.GetTickSize(symbol); err != nil {
			return err
		}
		if _, err = h.GetMultiplier(symbol); err != nil {
			return err
		}
		if _, err = h.GetMinSize(symbol); err != nil {
			return err
		}
		if _, err = h.GetMinNotional(symbol); err != nil {
			return err
		}
		if settings.ChangeLeverage {
			err := h.api.SwitchIsolated(ctx, SwitchIsolatedParam{
				Symbol:       symbol,
				IsIsolated:   false,
				BuyLeverage:  int(settings.Leverage),
				SellLeverage: int(settings.Leverage),
			})
			if err != nil {
				logger.Debugf("SwitchIsolated %s %v", symbol, err)
			}
		}
	}
	return nil
}

func (h *BybitUsdtFuture) GetMultiplier(symbol string) (float64, error) {
	return 1.0, nil
}

func (h *BybitUsdtFuture) GetMinNotional(symbol string) (float64, error) {
	return 0.0, nil
}

func (h *BybitUsdtFuture) GetMinSize(symbol string) (float64, error) {
	if v, ok := StepSizes[symbol]; ok {
		return v, nil
	} else {
		return 0.0, fmt.Errorf(common.MinSizeNotFoundError, symbol)
	}
}

func (h *BybitUsdtFuture) GetStepSize(symbol string) (float64, error) {
	if v, ok := StepSizes[symbol]; ok {
		return v, nil
	} else {
		return 0.0, fmt.Errorf(common.StepSizeNotFoundError, symbol)
	}
}

func (h *BybitUsdtFuture) GetTickSize(symbol string) (float64, error) {
	if v, ok := TickSizes[symbol]; ok {
		return v, nil
	} else {
		return 0.0, fmt.Errorf(common.TickSizeNotFoundError, symbol)
	}
}

func (h *BybitUsdtFuture) StreamBasic(ctx context.Context, statusCh chan common.SystemStatus, accountCh chan common.Balance, commissionAssetValueCh chan float64, positionChMap map[string]chan common.Position, orderChs map[string]chan common.Order) {
	defer h.Stop()
	settings := h.settings
	symbols := h.settings.Symbols[:]
	userWS := NewUserWS(
		ctx,
		h.settings.ApiKey,
		h.settings.ApiSecret,
		settings.Proxy,
	)
	go h.systemStatusLoop(ctx, statusCh)
	httpPositionsCh := make(chan map[string]MergedPosition, 4)
	go h.positionsLoop(ctx, symbols, httpPositionsCh)
	httpAccountCh := make(chan Balance, 128)
	go h.accountLoop(ctx, httpAccountCh)
	logSilentTime := time.Now()
	restartResetTimer := time.NewTimer(time.Hour * 9999)
	defer restartResetTimer.Stop()
	positionsMap := make(map[string]MergedPosition)
	var commissionAssetTimer = time.NewTimer(time.Second)
	for {
		select {
		case <-userWS.Done():
			return
		case <-h.done:
			return
		case <-commissionAssetTimer.C:
			select {
			case commissionAssetValueCh <- 0.0:
			default:
				logger.Debugf("commissionAssetValueCh <- 0.0 failed, ch len %d", len(commissionAssetValueCh))
			}
			commissionAssetTimer.Reset(time.Minute)
			break
		case positions := <-httpPositionsCh:
			for symbol, pos := range positions {
				//logger.Debugf("HTTP POS %v", pos)
				if ch, ok := positionChMap[symbol]; ok {
					if oldPos, ok := positionsMap[symbol]; ok {
						if pos.EventTime.Sub(oldPos.EventTime) > 0 {
							pos := pos
							positionsMap[symbol] = pos
							select {
							case ch <- &pos:
							default:
								logger.Debugf("ch <- &pos failed, ch len %d", len(ch))
							}
						}
					} else {
						pos := pos
						positionsMap[symbol] = pos
						select {
						case ch <- &pos:
						default:
							logger.Debugf("ch <- &pos failed, ch len %d", len(ch))
						}
					}
				}
			}
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
		case accounts := <-httpAccountCh:
			select {
			case accountCh <- &accounts:
			default:
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("accountCh <- &outAccount failed, ch len %d", len(accountCh))
					logSilentTime = time.Now().Add(time.Minute)
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
		case _ = <-userWS.WalletsCh:
			//if len(wallets) > 0 {
			//	wallet := wallets[0]
			//	select {
			//	case accountCh <- &wallet:
			//	default:
			//		if time.Now().Sub(logSilentTime) > 0 {
			//			logger.Debugf("accountCh <- &wallet failed, ch len %d", len(accountCh))
			//			logSilentTime = time.Now().Add(time.Minute)
			//		}
			//	}
			//	break
			//}
			break
		case orders := <-userWS.OrdersCh:
			for _, order := range orders {
				if ch, ok := orderChs[order.Symbol]; ok {
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
			newPositions := make(map[string]MergedPosition, 0)
			for _, nextPos := range wsPositions {
				if newPos, ok := newPositions[nextPos.Symbol]; ok {
					newPos.ParseTime = nextPos.ParseTime
					newPos.EventTime = nextPos.ParseTime
					if nextPos.Side == PositionSideBuy && nextPos.Size != 0 {
						newPos.Size += nextPos.Size
						newPos.Price = nextPos.EntryPrice
					} else if nextPos.Side == PositionSideSell && nextPos.Size != 0 {
						newPos.Size -= nextPos.Size
						newPos.Price = nextPos.EntryPrice
					}
					newPositions[nextPos.Symbol] = newPos
				} else {
					newPos = MergedPosition{
						Symbol:    nextPos.Symbol,
						ParseTime: nextPos.ParseTime,
						EventTime: nextPos.ParseTime,
					}
					if nextPos.Side == PositionSideBuy && nextPos.Size != 0 {
						newPos.Size = nextPos.Size
						newPos.Price = nextPos.EntryPrice
					} else if nextPos.Side == PositionSideSell && nextPos.Size != 0 {
						newPos.Size = -nextPos.Size
						newPos.Price = nextPos.EntryPrice
					}
					newPositions[nextPos.Symbol] = newPos
					logger.Debugf("WS POS %v, %v", nextPos, newPos)
				}
			}
			for symbol, newPos := range newPositions {
				pos := newPos
				if ch, ok := positionChMap[symbol]; ok {
					select {
					case ch <- &pos:
					default:
						logger.Debugf("ch <- &pos failed, ch len %d", len(ch))
					}
				}
			}
			break
		}
	}
}

func (h *BybitUsdtFuture) StreamSymbolStatus(ctx context.Context, channels map[string]chan common.SymbolStatusMsg, batchSize int) {
	panic("implement me")
}

func (h *BybitUsdtFuture) StreamDepth(ctx context.Context, channels map[string]chan common.Depth, batchSize int) {
	logger.Debugf("START StreamDepth")
	defer logger.Debugf("STOP StreamDepth")
	defer h.Stop()
	symbols := make([]string, 0)
	for symbol := range channels {
		symbols = append(symbols, symbol)
	}
	proxy := h.settings.Proxy
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
			defer h.Stop()
			ws := NewOrderBookWS(ctx, proxy, channels)
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
		case <-h.done:
			return
		}
	}
}

func (h *BybitUsdtFuture) StreamTrade(ctx context.Context, channels map[string]chan common.Trade, batchSize int) {
	panic("implement me")
}

func (h *BybitUsdtFuture) StreamTicker(ctx context.Context, channels map[string]chan common.Ticker, batchSize int) {
	logger.Debugf("START StreamTicker")
	defer logger.Debugf("STOP StreamTicker")
	defer h.Stop()
	symbols := make([]string, 0)
	for symbol := range channels {
		symbols = append(symbols, symbol)
	}
	proxy := h.settings.Proxy
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
			defer h.Stop()
			ws := NewOrderBookTickerWS(ctx, proxy, channels)
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
		case <-h.done:
			return
		}
	}
}

func (h *BybitUsdtFuture) StreamKLine(ctx context.Context, channels map[string]chan []common.KLine, batchSize int, interval, lookback time.Duration) {
	panic("implement me")
}

func (h *BybitUsdtFuture) StreamFundingRate(ctx context.Context, channels map[string]chan common.FundingRate, batchSize int) {
	interval := h.settings.PullInterval * 2
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	afterFrTimer := time.NewTimer(time.Now().Truncate(time.Hour * 4).Add(time.Hour*4 + time.Second).Sub(time.Now()))
	defer afterFrTimer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-h.done:
			return
		case <-afterFrTimer.C:
			for symbol, ch := range channels {
				subCtx, _ := context.WithTimeout(ctx, time.Minute)
				fr, err := h.api.GetPrevFundingRate(subCtx, PrevFundingRateParam{
					Symbol: symbol,
				})
				if err != nil {
					logger.Debugf("h.api.GetPrevFundingRate error %v", err)
				} else {
					select {
					case ch <- fr:
					default:
						logger.Debugf("ch <- &fr failed %s ch len %d", fr.Symbol, len(ch))
					}
				}
			}
			afterFrTimer.Reset(time.Now().Truncate(time.Hour * 4).Add(time.Hour*4 + time.Second).Sub(time.Now()))
			break
		case <-timer.C:
			for symbol, ch := range channels {
				subCtx, _ := context.WithTimeout(ctx, time.Minute)
				fr, err := h.api.GetPrevFundingRate(subCtx, PrevFundingRateParam{
					Symbol: symbol,
				})
				if err != nil {
					logger.Debugf("h.api.GetPrevFundingRate error %v", err)
				} else {
					select {
					case ch <- fr:
					default:
						logger.Debugf("ch <- &fr failed %s ch len %d", fr.Symbol, len(ch))
					}
				}
			}
			timer.Reset(time.Now().Truncate(interval).Add(interval).Sub(time.Now()))
			break
		}
	}
}

func (h *BybitUsdtFuture) WatchOrders(ctx context.Context, requestChannels map[string]chan common.OrderRequest, responseChannels map[string]chan common.Order, errorChannels map[string]chan common.OrderError) {
	defer h.Stop()
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
		go h.watchOrder(ctx, symbol, reqCh, respCh, errCh)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-h.done:
			return
		}
	}
}

func (h *BybitUsdtFuture) GenerateClientID() string {
	return fmt.Sprintf("%d", time.Now().Unix()*10000+int64(rand.Intn(10000)))
}

func (h *BybitUsdtFuture) systemStatusLoop(
	ctx context.Context, output chan common.SystemStatus,
) {
	pullInterval := h.settings.PullInterval
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-h.done:
			return
		case <-timer.C:
			subCtx, _ := context.WithTimeout(ctx, time.Second*3)
			heartBeat, err := h.api.GetServerTime(subCtx)
			if err != nil {
				logger.Debugf("h.api.GetHeartBeat(subCtx) error %v", err)
				select {
				case output <- common.SystemStatusError:
				default:
					logger.Debugf("output <- common.SystemStatusError failed ch len %d", len(output))
				}
			} else {
				if time.Now().Sub(*heartBeat) < time.Minute &&
					time.Now().Sub(*heartBeat) > -time.Minute {
					select {
					case output <- common.SystemStatusReady:
					default:
						logger.Debugf("output <- common.SystemStatusReady failed ch len %d", len(output))
					}
				} else {
					select {
					case output <- common.SystemStatusNotReady:
					default:
						logger.Debugf("output <- common.SystemStatusReady failed ch len %d", len(output))
					}
				}
			}
			timer.Reset(time.Now().Truncate(pullInterval).Add(pullInterval).Sub(time.Now()))
		}
	}
}

func (h *BybitUsdtFuture) accountLoop(
	ctx context.Context, output chan Balance,
) {
	pullInterval := h.settings.PullInterval
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			subCtx, _ := context.WithTimeout(ctx, time.Minute)
			account, err := h.api.GetBalance(subCtx, BalanceParam{
				Coin: "USDT",
			})
			if err != nil {
				logger.Debugf("h.api.GetBalance error %v", err)
			} else {
				select {
				case output <- *account:
				default:
					logger.Debugf("output <- account failed ch len %d", len(output))
				}
				break
			}
		}
		timer.Reset(time.Now().Truncate(pullInterval).Add(pullInterval).Sub(time.Now()))
	}
}

func (h *BybitUsdtFuture) positionsLoop(
	ctx context.Context,
	symbols []string,
	outputCh chan map[string]MergedPosition,
) {
	pullInterval := h.settings.PullInterval
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			subCtx, _ := context.WithTimeout(ctx, time.Minute)
			positions, err := h.api.GetPositions(subCtx)
			if err != nil {
				logger.Debugf("api.GetPositions error %v", err)
			} else {
				//有一种情况是有的合约的仓位是拉不到的, 拉不到的都是空仓
				positionBySymbols := make(map[string]MergedPosition)
				for _, symbol := range symbols {
					positionBySymbols[symbol] = MergedPosition{
						Symbol:    symbol,
						Price:     0,
						Size:      0,
						EventTime: time.Now(),
						ParseTime: time.Now(),
					}
				}
				//假定只有一个方向的仓位
				for _, position := range positions {
					if mP, ok := positionBySymbols[position.Data.Symbol]; ok {
						if position.Data.Side == PositionSideBuy && position.Data.Size != 0{
							mP.Size += position.Data.Size
							mP.Price = position.Data.EntryPrice
						} else if position.Data.Side == PositionSideSell && position.Data.Size != 0{
							mP.Size -= position.Data.Size
							mP.Price = position.Data.EntryPrice
						}
						positionBySymbols[mP.Symbol] = mP
					}
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

func (h *BybitUsdtFuture) watchOrder(
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
		case <-h.Done():
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
				h.submitOrder(ctx, *req.New, responseCh, errorCh)
			} else if req.Cancel != nil {
				h.cancelOrder(ctx, *req.Cancel, errorCh)
			}
		}
	}
}

func (h *BybitUsdtFuture) submitOrder(ctx context.Context, param common.NewOrderParam, respCh chan common.Order, errCh chan common.OrderError) {
	newOrderParam := NewOrderParam{}
	var err error
	newOrderParam.OrderLinkID = param.ClientID
	newOrderParam.Symbol = param.Symbol
	newOrderParam.Qty = param.Size
	newOrderParam.ReduceOnly = param.ReduceOnly
	if param.Side == common.OrderSideBuy {
		newOrderParam.Side = OrderSideBuy
	} else {
		newOrderParam.Side = OrderSideSell
	}
	if param.Type == common.OrderTypeMarket {
		newOrderParam.OrderType = OrderTypeMarket
	} else {
		newOrderParam.OrderType = OrderTypeMarket
	}
	switch param.TimeInForce {
	case common.OrderTimeInForceFOK:
		newOrderParam.TimeInForce = TimeInForceFillOrKill
	case common.OrderTimeInForceIOC:
		newOrderParam.TimeInForce = TimeInForceImmediateOrCancel
	case common.OrderTimeInForceGTC:
		newOrderParam.TimeInForce = TimeInForceGoodTillCancel
	}
	if param.PostOnly {
		newOrderParam.TimeInForce = TimeInForcePostOnly
	}
	if param.Price != 0 {
		newOrderParam.Price = param.Price
	}
	order, err := h.api.PlaceOrder(ctx, newOrderParam)
	logger.Debugf("%v", order)
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

func (h *BybitUsdtFuture) cancelOrder(ctx context.Context, param common.CancelOrderParam, errCh chan common.OrderError) {
	if param.Symbol != "" {
		_, err := h.api.CancelAllOrders(ctx, CancelAllParam{
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

func (h *BybitUsdtFuture) WatchBatchOrders(ctx context.Context, requestChannels map[string]chan common.BatchOrderRequest, responseChannels map[string]chan common.Order, errorChannels map[string]chan common.OrderError) {
	panic("implement me")
}

func (h *BybitUsdtFuture) StartSideLoop() {
	panic("implement me")
}
