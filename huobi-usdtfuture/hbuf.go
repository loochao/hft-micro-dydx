package huobi_usdtfuture

import (
	"context"
	"errors"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"math/rand"
	"sync/atomic"
	"time"
)

type HuobiUsdtFuture struct {
	done     chan interface{}
	stopped  int32
	api      *API
	settings common.ExchangeSettings
}

func (k *HuobiUsdtFuture) GetPriceFactor() float64 {
	return 1.0
}

func (h *HuobiUsdtFuture) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (h *HuobiUsdtFuture) IsSpot() bool {
	return false
}

func (h *HuobiUsdtFuture) Done() chan interface{} {
	return h.done
}

func (h *HuobiUsdtFuture) Stop() {
	if atomic.CompareAndSwapInt32(&h.stopped, 0, 1) {
		close(h.done)
		logger.Debugf("stopped")
	}
}

func (h *HuobiUsdtFuture) Setup(ctx context.Context, settings common.ExchangeSettings) error {
	var err error
	h.stopped = 0
	h.done = make(chan interface{})
	h.settings = settings
	h.api, err = NewAPI(settings.ApiKey, settings.ApiSecret, settings.Proxy)
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
		//if settings.ChangeLeverage {
		//	_, err = h.api.ChangeAutoDepositStatus(ctx, AutoDepositStatusParam{
		//		Symbol: symbol,
		//		Status: true,
		//	})
		//	if err != nil {
		//		return err
		//	}
		//}
	}
	return nil
}

func (h *HuobiUsdtFuture) GetMultiplier(symbol string) (float64, error) {
	if v, ok := ContractSizes[symbol]; ok {
		return v, nil
	} else {
		return 0.0, fmt.Errorf(common.MultiplierNotFoundError, symbol)
	}
}

func (h *HuobiUsdtFuture) GetMinNotional(symbol string) (float64, error) {
	return 0.0, nil
}

func (h *HuobiUsdtFuture) GetMinSize(symbol string) (float64, error) {
	return 1.0, nil
}

func (h *HuobiUsdtFuture) GetStepSize(symbol string) (float64, error) {
	return 1.0, nil
}

func (h *HuobiUsdtFuture) GetTickSize(symbol string) (float64, error) {
	if v, ok := PriceTicks[symbol]; ok {
		return v, nil
	} else {
		return 0.0, fmt.Errorf(common.TickSizeNotFoundError, symbol)
	}
}

func (h *HuobiUsdtFuture) StreamBasic(ctx context.Context, statusCh chan common.SystemStatus, accountCh chan common.Balance, commissionAssetValueCh chan float64, positionChMap map[string]chan common.Position, orderChs map[string]chan common.Order) {
	defer h.Stop()
	settings := h.settings
	symbols := h.settings.Symbols[:]
	var err error
	h.api, err = NewAPI(settings.ApiKey, settings.ApiSecret, settings.Proxy)
	if err != nil {
		return
	}
	userWS := NewUserWebsocket(
		ctx,
		h.settings.ApiKey,
		h.settings.ApiSecret,
		symbols[:],
		settings.Proxy,
	)
	go h.systemStatusLoop(ctx, statusCh)
	httpPositionsCh := make(chan map[string]MergedPosition, 4)
	go h.positionsLoop(ctx, symbols, httpPositionsCh)
	httpAccountCh := make(chan []Account, 128)
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
			for _, account := range accounts {
				//msg, _ := json.Marshal(account)
				//logger.Debugf("%s", msg)
				if account.MarginAsset == "USDT" {
					outAccount := account
					select {
					case accountCh <- &outAccount:
					default:
						if time.Now().Sub(logSilentTime) > 0 {
							logger.Debugf("accountCh <- &outAccount failed, ch len %d", len(accountCh))
							logSilentTime = time.Now().Add(time.Minute)
						}
					}
					break
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
		case wsAccounts := <-userWS.AccountCh:
			for _, account := range wsAccounts.Accounts {
				//msg, _ := json.Marshal(account)
				//logger.Debugf("%s", msg)
				if account.MarginAsset == "USDT" {
					outAccount := account
					select {
					case accountCh <- &outAccount:
					default:
						if time.Now().Sub(logSilentTime) > 0 {
							logger.Debugf("accountCh <- &outAccount failed, ch len %d", len(accountCh))
							logSilentTime = time.Now().Add(time.Minute)
						}
					}
					break
				}
			}
			break
		case order := <-userWS.OrderCh:
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
			break
		case wsPositions := <-userWS.PositionCh:
			newPositions := make(map[string]MergedPosition, 0)
			for _, nextPos := range wsPositions.Positions {
				if newPos, ok := newPositions[nextPos.Symbol]; ok {
					newPos.ParseTime = nextPos.ParseTime
					newPos.EventTime = wsPositions.Timestamp
					if nextPos.Direction == PositionDirectionBuy {
						newPos.Size += nextPos.Volume
						newPos.Price = nextPos.CostHold
					} else {
						newPos.Size -= nextPos.Volume
						newPos.Price = nextPos.CostHold
					}
					newPositions[nextPos.Symbol] = newPos
					//logger.Debugf("WS POS %v, %v", nextPos, newPos)
				} else {
					newPos = MergedPosition{
						Symbol:    nextPos.Symbol,
						ParseTime: nextPos.ParseTime,
						EventTime: wsPositions.Timestamp,
					}
					if nextPos.Direction == PositionDirectionBuy {
						newPos.Size = nextPos.Volume
						newPos.Price = nextPos.CostHold
					} else {
						newPos.Size = -nextPos.Volume
						newPos.Price = nextPos.CostHold
					}
					newPositions[nextPos.Symbol] = newPos
					//logger.Debugf("WS POS %v, %v", nextPos, newPos)
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

func (h *HuobiUsdtFuture) StreamSymbolStatus(ctx context.Context, channels map[string]chan common.SymbolStatusMsg, batchSize int) {
	panic("implement me")
}

func (h *HuobiUsdtFuture) StreamDepth(ctx context.Context, channels map[string]chan common.Depth, batchSize int) {
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
		case <-h.done:
			return
		}
	}
}

func (h *HuobiUsdtFuture) StreamTrade(ctx context.Context, channels map[string]chan common.Trade, batchSize int) {
	panic("implement me")
}

func (h *HuobiUsdtFuture) StreamTicker(ctx context.Context, channels map[string]chan common.Ticker, batchSize int) {
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
		case <-h.done:
			return
		}
	}
}

func (h *HuobiUsdtFuture) StreamKLine(ctx context.Context, channels map[string]chan []common.KLine, batchSize int, interval, lookback time.Duration) {
	panic("implement me")
}

func (h *HuobiUsdtFuture) StreamFundingRate(ctx context.Context, channels map[string]chan common.FundingRate, batchSize int) {
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
			subCtx, _ := context.WithTimeout(ctx, time.Minute)
			frs, err := h.api.GetFundingRates(subCtx)
			if err != nil {
				logger.Debugf("k.api.GetFundingRates error %v", err)
			} else {
				for _, fr := range frs {
					if ch, ok := channels[fr.Symbol]; ok {
						fr := fr
						select {
						case ch <- &fr:
						default:
							logger.Debugf("ch <- &fr failed %s ch len %d", fr.Symbol, len(ch))
						}
					}
				}
			}
			afterFrTimer.Reset(time.Now().Truncate(time.Hour * 4).Add(time.Hour*4 + time.Second).Sub(time.Now()))
			break
		case <-timer.C:
			subCtx, _ := context.WithTimeout(ctx, time.Minute)
			frs, err := h.api.GetFundingRates(subCtx)
			if err != nil {
				logger.Debugf("k.api.GetFundingRates error %v", err)
			} else {
				for _, fr := range frs {
					if ch, ok := channels[fr.Symbol]; ok {
						fr := fr
						select {
						case ch <- &fr:
						default:
							logger.Debugf("ch <- &fr failed %s ch len %d", fr.Symbol, len(ch))
						}
					}
				}
			}
			timer.Reset(time.Now().Truncate(interval).Add(interval).Sub(time.Now()))
			break
		}
	}
}

func (h *HuobiUsdtFuture) WatchOrders(ctx context.Context, requestChannels map[string]chan common.OrderRequest, responseChannels map[string]chan common.Order, errorChannels map[string]chan common.OrderError) {
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

func (h *HuobiUsdtFuture) GenerateClientID() string {
	return fmt.Sprintf("%d", time.Now().Unix()*10000+int64(rand.Intn(10000)))
}

func (h *HuobiUsdtFuture) systemStatusLoop(
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
			heartBeat, err := h.api.GetHeartBeat(subCtx)
			if err != nil {
				logger.Debugf("h.api.GetHeartBeat(subCtx) error %v", err)
				select {
				case output <- common.SystemStatusError:
				default:
					logger.Debugf("output <- common.SystemStatusError failed ch len %d", len(output))
				}
			} else {
				//logger.Debugf("heartBeat %v", heartBeat)
				if heartBeat.LinearSwapHeartbeat == 1 {
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

func (h *HuobiUsdtFuture) accountLoop(
	ctx context.Context, output chan []Account,
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
			accounts, err := h.api.GetAccounts(subCtx)
			if err != nil {
				logger.Debugf("h.api.GetAccounts error %v", err)
			} else {
				select {
				case output <- accounts:
				default:
					logger.Debugf("output <- account failed ch len %d", len(output))
				}
				break
			}
		}
		timer.Reset(time.Now().Truncate(pullInterval).Add(pullInterval).Sub(time.Now()))
	}
}

func (h *HuobiUsdtFuture) positionsLoop(
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
					if position.Direction == PositionDirectionBuy {
						mP := positionBySymbols[position.Symbol]
						mP.Size += position.Volume
						mP.Price = position.CostHold
						positionBySymbols[position.Symbol] = mP
					} else if position.Direction == PositionDirectionSell {
						mP := positionBySymbols[position.Symbol]
						mP.Size -= position.Volume
						mP.Price = position.CostHold
						positionBySymbols[position.Symbol] = mP
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

func (h *HuobiUsdtFuture) watchOrder(
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

func (h *HuobiUsdtFuture) submitOrder(ctx context.Context, param common.NewOrderParam, respCh chan common.Order, errCh chan common.OrderError) {
	newOrderParam := NewOrderParam{}
	var err error
	newOrderParam.ClientOrderID, err = common.ParseInt([]byte(param.ClientID))
	if err != nil {
		select {
		case errCh <- common.OrderError{
			New:   &param,
			Error: err,
		}:
		default:
			logger.Debugf("errCh <- common.OrderError failed, ch len %d", len(errCh))
		}
		return
	}
	newOrderParam.Symbol = param.Symbol
	newOrderParam.Volume = int64(math.Round(param.Size))
	if param.Side == common.OrderSideBuy {
		newOrderParam.Direction = OrderDirectionBuy
	} else {
		newOrderParam.Direction = OrderDirectionSell
	}
	if param.Type == common.OrderTypeMarket {
		newOrderParam.OrderPriceType = OrderPriceTypeOpponent
	} else {
		newOrderParam.OrderPriceType = OrderPriceTypeLimit
	}
	if param.TimeInForce == common.OrderTimeInForceIOC {
		newOrderParam.OrderPriceType = OrderPriceTypeIOC
	}
	if param.TimeInForce == common.OrderTimeInForceFOK {
		newOrderParam.OrderPriceType = OrderPriceTypeFOK
	}
	if param.PostOnly {
		newOrderParam.OrderPriceType = OrderPriceTypePostOnly
	}
	newOrderParam.LeverRate = int(h.settings.Leverage)
	if param.Price != 0 {
		newOrderParam.Price = common.Float64(param.Price)
	}
	if param.ReduceOnly {
		newOrderParam.Offset = OrderOffsetClose
	} else {
		newOrderParam.Offset = OrderOffsetOpen
	}
	_, err = h.api.SubmitOrder(ctx, newOrderParam)
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

func (h *HuobiUsdtFuture) cancelOrder(ctx context.Context, param common.CancelOrderParam, errCh chan common.OrderError) {
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

func (h *HuobiUsdtFuture) WatchBatchOrders(ctx context.Context, requestChannels map[string]chan common.BatchOrderRequest, responseChannels map[string]chan common.Order, errorChannels map[string]chan common.OrderError) {
	panic("implement me")
}

func (h *HuobiUsdtFuture) StartSideLoop() {
	panic("implement me")
}

type HuobiUsdtFutureWithDepth5 struct {
	HuobiUsdtFuture
}

type HuobiUsdtFutureWithMergedTicker struct {
	HuobiUsdtFuture
}


func (k *HuobiUsdtFutureWithMergedTicker) StreamSystemStatus(ctx context.Context, statusCh chan common.SystemStatus) {
	panic("implement me")
}

func (k *HuobiUsdtFutureWithMergedTicker) StreamTicker(ctx context.Context, channels map[string]chan common.Ticker, batchSize int) {
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
			ws1 := NewDepth20TickerWS(ctx, proxy, channels)
			ws2 := NewTickerWS(ctx, proxy, channels)
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
