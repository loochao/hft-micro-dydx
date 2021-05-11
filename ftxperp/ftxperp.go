package ftxperp

import (
	"context"
	"errors"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"sync"
	"time"
)

type Ftxperp struct {
	api      *API
	done     chan interface{}
	stopped  bool
	mu       sync.Mutex
	settings common.ExchangeSettings
}

func (ftx *Ftxperp) GetMinNotional(symbol string) float64 {
	return 0.0
}

func (ftx *Ftxperp) GetMinSize(symbol string) float64 {
	return SizeIncrements[symbol]
}

func (ftx *Ftxperp) GetStepSize(symbol string) float64 {
	return SizeIncrements[symbol]
}

func (ftx *Ftxperp) GetTickSize(symbol string) float64 {
	return PriceIncrements[symbol]
}

func (ftx *Ftxperp) StreamBasic(
	ctx context.Context,
	statusCh chan common.SystemStatus,
	accountCh chan common.Account,
	positionsCh map[string]chan common.Position,
	ordersCh map[string]chan common.Order,
) {
	defer ftx.Stop()

	ftx.mu.Lock()
	userWS := NewUserWS(
		ftx.settings.ApiKey,
		ftx.settings.ApiSecret,
		ftx.settings.Proxy,
	)
	ftx.mu.Unlock()
	go userWS.Start(ctx)
	defer userWS.Stop()
	internalOrders := make(map[int64]Order)

	positionMarkets := make([]string, 0)
	for market := range positionsCh {
		positionMarkets = append(positionMarkets, market)
	}

	internalPositions := make(map[string]Position)
	internalPositionsCh := make(chan []Position, 100)
	go ftx.positionsLoop(ctx, positionMarkets, internalPositionsCh)
	internalAccount := Account{}
	internalAccountCh := make(chan *Account, 100)
	go ftx.accountLoop(ctx, internalAccountCh)

	logSilentTime := time.Now()

	select {
	case statusCh <- common.SystemStatusReady:
	default:
		if time.Now().Sub(logSilentTime) > 0 {
			logger.Debugf("restartCh <- common.SystemStatusReady failed, ch len %d", len(statusCh))
			logSilentTime = time.Now().Add(time.Minute)
		}
	}

	for {
		select {
		case <-ftx.done:
			return
		case <-userWS.Done():
			return
		case <-userWS.RestartCh:
			select {
			case statusCh <- common.SystemStatusRestart:
			default:
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("restartCh <- common.SystemStatusRestart failed, ch len %d", len(statusCh))
					logSilentTime = time.Now().Add(time.Minute)
				}
			}
		case a := <-internalAccountCh:
			if a != nil {
				internalAccount = *a
				select {
				case accountCh <- &internalAccount:
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("accountCh <- &internalAccount failed, ch len %d", len(accountCh))
						logSilentTime = time.Now().Add(time.Minute)
					}
				}
			}
		case ps := <-internalPositionsCh:
			for _, p := range ps {
				p := p
				if oldP, ok := internalPositions[p.Future]; ok {
					if oldP.ParseTime.Sub(p.ParseTime) < 0 {
						internalPositions[p.Future] = p
					}
				}
				if positionCh, ok := positionsCh[p.Future]; ok {
					select {
					case positionCh <- &p:
					default:
						if time.Now().Sub(logSilentTime) > 0 {
							logger.Debugf("positionCh <- &p failed, ch len %d", len(positionCh))
							logSilentTime = time.Now().Add(time.Minute)
						}
					}
				}
			}
		case order := <-userWS.OrderCh:
			internalOrders[order.ID] = order
			if orderCh, ok := ordersCh[order.Future]; ok {
				select {
				case orderCh <- &order:
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("orderCh <- &order failed, ch len %d", len(orderCh))
						logSilentTime = time.Now().Add(time.Minute)
					}
				}

			}
			break
		case fill := <-userWS.FillCh:
			if order, ok := internalOrders[fill.ID]; ok {
				fill.PostOnly = order.PostOnly
				fill.ReduceOnly = order.ReduceOnly
				fill.ClientId = order.ClientId
				fill.Price = order.Price
				fill.Size = order.Size
				fill.OrderType = order.Type
			}
			if pos, ok := internalPositions[fill.Future]; ok {
				size := fill.FilledSize
				if fill.Side != OrderSideBuy {
					size = -fill.FilledSize
				}
				price := fill.FilledPrice
				if pos.NetSize*size < 0 {
					if math.Abs(size) > math.Abs(pos.NetSize) {
						pos.Cost = (pos.NetSize + size) * price
					} else {
						pos.Cost = pos.Cost - size*price
					}
				} else {
					pos.Cost += size * price
					pos.NetSize += size
				}
				internalPositions[fill.Future] = pos
				if positionCh, ok := positionsCh[fill.Future]; ok {
					select {
					case positionCh <- &pos:
					default:
						if time.Now().Sub(logSilentTime) > 0 {
							logger.Debugf("positionCh <- &pos failed, ch len %d", len(positionCh))
							logSilentTime = time.Now().Add(time.Minute)
						}
					}
				}
			}
			if orderCh, ok := ordersCh[fill.Future]; ok {
				select {
				case orderCh <- &fill:
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("orderCh <- &fill failed, ch len %d", len(orderCh))
						logSilentTime = time.Now().Add(time.Minute)
					}
				}
			}
			break
		}
	}
}

func (ftx *Ftxperp) StreamTrade(ctx context.Context, channels map[string]chan common.Trade, batchSize int) {
	logger.Debugf("START StreamTrade")
	defer logger.Debugf("STOP StreamTrade")
	defer ftx.Stop()

	markets := make([]string, 0)
	for market := range channels {
		markets = append(markets, market)
	}

	ftx.mu.Lock()
	proxy := ftx.settings.Proxy
	ftx.mu.Unlock()

	for start := 0; start < len(markets); start += batchSize {
		end := start + batchSize
		if end > len(markets) {
			end = len(markets)
		}
		subChannels := make(map[string]chan common.Trade)
		for _, market := range markets[start:end] {
			subChannels[market] = channels[market]
		}
		go func(ctx context.Context, proxy string, channels map[string]chan common.Trade) {
			ws := NewTradeWS(ctx, proxy, channels)
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
		case <-ftx.done:
			return
		}
	}
}

func (ftx *Ftxperp) StreamTicker(ctx context.Context, channels map[string]chan common.Ticker, batchSize int) {
	panic("implement me")
}

func (ftx *Ftxperp) StreamKLine(ctx context.Context, channels map[string]chan []common.KLine, batchSize int, interval, lookback time.Duration) {
	panic("implement me")
}

func (ftx *Ftxperp) StreamFundingRate(ctx context.Context, channels map[string]chan common.FundingRate, batchSize int) {
	panic("implement me")
}

func (ftx *Ftxperp) WatchOrders(
	ctx context.Context,
	requestChannels map[string]chan common.OrderRequest,
	responseChannels map[string]chan common.Order,
	errorChannels map[string]chan common.OrderError,
) {
	defer ftx.Stop()
	for market, reqCh := range requestChannels {
		tickSize, ok := PriceIncrements[market]
		if !ok {
			logger.Debugf("miss price increment for %s, exit", market)
			return
		}
		stepSize, ok := SizeIncrements[market]
		if !ok {
			logger.Debugf("miss size increment for %s, exit", market)
			return
		}
		respCh, ok := responseChannels[market]
		if !ok {
			logger.Debugf("miss response ch for %s, exit", market)
			return
		}
		errCh, ok := errorChannels[market]
		if !ok {
			logger.Debugf("miss error ch for %s, exit", market)
			return
		}
		go ftx.watchOrder(ctx, market, tickSize, stepSize, reqCh, respCh, errCh)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ftx.done:
			return
		}
	}
}

func (ftx *Ftxperp) Setup(ctx context.Context, settings common.ExchangeSettings) error {
	var err error
	if settings.HttpPullInterval == 0 {
		settings.HttpPullInterval = time.Minute
	}
	ftx.settings = settings
	ftx.done = make(chan interface{})
	ftx.stopped = false
	ftx.api, err = NewAPI(settings.ApiKey, settings.ApiSecret, settings.Proxy)
	if err != nil {
		return err
	}
	ftx.mu = sync.Mutex{}
	if settings.ChangeLeverage {
		_, err = ftx.api.ChangeLeverage(ctx, LeverageParam{
			Leverage: int(settings.Leverage),
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (ftx *Ftxperp) watchOrder(
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
		case <-ftx.Done():
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
				ftx.submitOrder(ctx, *req.New, tickSize, stepSize, responseCh, errorCh)
			} else if req.Cancel != nil {
				ftx.cancelOrder(ctx, *req.Cancel, errorCh)
			}
		}
	}
}

func (ftx *Ftxperp) Stop() {
	ftx.mu.Lock()
	if !ftx.stopped {
		ftx.stopped = true
		close(ftx.done)
		logger.Debugf("stopped")
	}
	ftx.mu.Unlock()
}

func (ftx *Ftxperp) Done() chan interface{} {
	return ftx.done
}

func (ftx *Ftxperp) positionsLoop(ctx context.Context, markets []string, positionsCh chan []Position) {
	ftx.mu.Lock()
	pullInterval := ftx.settings.HttpPullInterval
	ftx.mu.Unlock()
	pullTimer := time.NewTimer(pullInterval)
	defer pullTimer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ftx.done:
			return
		case <-pullTimer.C:
			positions, err := ftx.api.GetPositions(ctx)
			if err != nil {
				logger.Debugf("ftx.api.GetPositions(ctx) error %v", err)
			} else {
				hasPositions := map[string]bool{}
				outPositions := make([]Position, 0)
				for _, position := range positions {
					hasPositions[position.Future] = true
					position := position
					outPositions = append(outPositions, position)
				}
				for _, market := range markets {
					if _, ok := hasPositions[market]; !ok {
						outPositions = append(outPositions, Position{
							Future:    market,
							ParseTime: time.Now(),
						})
					}
				}
				select {
				case positionsCh <- outPositions:
				default:
					logger.Debugf("positionsCh <- positions failed, ch len %d", len(positionsCh))
				}
			}
			pullTimer.Reset(pullInterval)
			break
		}
	}
}

func (ftx *Ftxperp) accountLoop(ctx context.Context, accountCh chan *Account) {
	ftx.mu.Lock()
	pullInterval := ftx.settings.HttpPullInterval
	ftx.mu.Unlock()
	pullTimer := time.NewTimer(pullInterval)
	defer pullTimer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ftx.done:
			return
		case <-pullTimer.C:
			account, err := ftx.api.GetAccount(ctx)
			if err != nil {
				logger.Debugf("ftx.api.GetAccount(ctx) error %v", err)
			} else {
				select {
				case accountCh <- account:
				default:
					logger.Debugf("accountCh <- account failed, ch len %d", len(accountCh))
				}
			}
			pullTimer.Reset(pullInterval)
			break
		}
	}
}

func (ftx *Ftxperp) submitOrder(ctx context.Context, param common.NewOrderParam, tickSize, stepSize float64, respCh chan common.Order, errCh chan common.OrderError) {
	newOrderParam := NewOrderParam{}
	newOrderParam.Market = param.Symbol
	newOrderParam.Size = math.Round(param.Size/stepSize) * stepSize
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
		newOrderParam.Ioc = true
	}
	newOrderParam.PostOnly = param.PostOnly
	newOrderParam.ReduceOnly = param.ReduceOnly
	if param.Price != 0 {
		newOrderParam.Price = math.Round(param.Price/tickSize) * tickSize
	}
	newOrderParam.ClientID = param.ClientID
	order, err := ftx.api.PlaceOrder(ctx, newOrderParam)
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

func (ftx *Ftxperp) cancelOrder(ctx context.Context, param common.CancelOrderParam, errCh chan common.OrderError) {
	if param.ClientID != "" {
		_, err := ftx.api.CancelOrderByClientID(ctx, param.ClientID)
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
		_, err := ftx.api.CancelAllOrders(ctx, CancelAllParam{
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

func (ftx *Ftxperp) StreamDepth(ctx context.Context, channels map[string]chan common.Depth, batchSize int) {
	logger.Debugf("START StreamDepth")
	defer logger.Debugf("STOP StreamDepth")
	defer ftx.Stop()

	markets := make([]string, 0)
	for market := range channels {
		markets = append(markets, market)
	}

	ftx.mu.Lock()
	proxy := ftx.settings.Proxy
	ftx.mu.Unlock()

	for start := 0; start < len(markets); start += batchSize {
		end := start + batchSize
		if end > len(markets) {
			end = len(markets)
		}
		subChannels := make(map[string]chan common.Depth)
		for _, market := range markets[start:end] {
			subChannels[market] = channels[market]
		}
		go func(ctx context.Context, proxy string, channels map[string]chan common.Depth) {
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
		case <-ftx.done:
			return
		}
	}
}
