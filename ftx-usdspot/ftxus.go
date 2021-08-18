package ftx_usdspot

import (
	"context"
	"errors"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"math/rand"
	"time"
)

type FtxUsdSpot struct {
	api      *API
	done     chan interface{}
	stopped  bool
	settings common.ExchangeSettings
}

func (ftx *FtxUsdSpot) StreamSystemStatus(ctx context.Context, statusCh chan common.SystemStatus) {
	panic("implement me")
}

func (ftx *FtxUsdSpot) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (ftx *FtxUsdSpot) WatchBatchOrders(ctx context.Context, requestChannels map[string]chan common.BatchOrderRequest, responseChannels map[string]chan common.Order, errorChannels map[string]chan common.OrderError) {
	panic("implement me")
}

func (ftx *FtxUsdSpot) StartSideLoop() {
	panic("implement me")
}

func (ftx *FtxUsdSpot) GetMultiplier(symbol string) (float64, error) {
	return 1.0, nil
}

func (ftx *FtxUsdSpot) IsSpot() bool {
	return false
}

func (ftx *FtxUsdSpot) GenerateClientID() string {
	return fmt.Sprintf("%d%04d", time.Now().Unix(), rand.Intn(10000))
}

func (ftx *FtxUsdSpot) StreamSymbolStatus(ctx context.Context, channels map[string]chan common.SymbolStatusMsg, batchSize int) {
	panic("implement me")
}

func (ftx *FtxUsdSpot) GetMinNotional(symbol string) (float64, error) {
	return 0.0, nil
}

func (ftx *FtxUsdSpot) GetMinSize(symbol string) (float64, error) {
	if value, ok := SizeIncrements[symbol]; ok {
		return value, nil
	} else {
		return 0, fmt.Errorf(common.MinSizeNotFoundError, symbol)
	}
}

func (ftx *FtxUsdSpot) GetStepSize(symbol string) (float64, error) {
	if value, ok := SizeIncrements[symbol]; ok {
		return value, nil
	} else {
		return 0, fmt.Errorf(common.StepSizeNotFoundError, symbol)
	}
}

func (ftx *FtxUsdSpot) GetTickSize(symbol string) (float64, error) {
	if value, ok := PriceIncrements[symbol]; ok {
		return value, nil
	} else {
		return 0, fmt.Errorf(common.TickSizeNotFoundError, symbol)
	}
}

func (ftx *FtxUsdSpot) StreamBasic(
	ctx context.Context,
	statusCh chan common.SystemStatus,
	accountCh chan common.Balance,
	commissionAssetValueCh chan float64,
	positionsCh map[string]chan common.Position,
	ordersCh map[string]chan common.Order,
) {
	defer ftx.Stop()

	userWS := NewUserWS(
		ftx.settings.ApiKey,
		ftx.settings.ApiSecret,
		ftx.settings.Proxy,
	)
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
	internalAccountCh := make(chan *Account, 100)
	go ftx.accountLoop(ctx, internalAccountCh)

	logSilentTime := time.Now()

	orderCleanTimer := time.NewTimer(time.Hour)
	defer orderCleanTimer.Stop()

	select {
	case statusCh <- common.SystemStatusReady:
	default:
		if time.Now().Sub(logSilentTime) > 0 {
			logger.Debugf("restartCh <- common.SystemStatusReady failed, ch len %d", len(statusCh))
			logSilentTime = time.Now().Add(time.Minute)
		}
	}

	restartToReadyTimer := time.NewTimer(time.Hour * 9999)
	defer restartToReadyTimer.Stop()

	commissionAssetValueTimer := time.NewTimer(time.Second)
	defer commissionAssetValueTimer.Stop()

	for {
		select {
		case <-ftx.done:
			return
		case <-userWS.Done():
			return
		case <-restartToReadyTimer.C:
			select {
			case statusCh <- common.SystemStatusReady:
			default:
				logger.Debugf("statusCh <- common.SystemStatusRestart failed ch len %d", len(statusCh))
			}
			restartToReadyTimer = time.NewTimer(time.Hour * 9999)
			break
		case <-orderCleanTimer.C:
			for id, order := range internalOrders {
				if time.Now().Sub(order.CreatedAt) > time.Hour {
					delete(internalOrders, id)
				}
			}
			orderCleanTimer.Reset(time.Hour)
		case <-commissionAssetValueTimer.C:
			select {
			case commissionAssetValueCh <- 0.0:
			default:
				logger.Debugf("commissionAssetValueCh <- 0.0 failed ch len %d", len(commissionAssetValueCh))
			}
			commissionAssetValueTimer.Reset(time.Minute)
			break
		case <-userWS.RestartCh:
			select {
			case statusCh <- common.SystemStatusRestart:
				restartToReadyTimer.Reset(time.Minute * 3)
			default:
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("restartCh <- common.SystemStatusRestart failed, ch len %d", len(statusCh))
					logSilentTime = time.Now().Add(time.Minute)
				}
			}
		case a := <-internalAccountCh:
			if a != nil {
				outputAccount := *a
				select {
				case accountCh <- &outputAccount:
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
				if oldP, ok := internalPositions[p.Market]; ok {
					if oldP.ParseTime.Sub(p.ParseTime) < 0 {
						internalPositions[p.Market] = p
					}
				} else {
					internalPositions[p.Market] = p
				}
				if positionCh, ok := positionsCh[p.Market]; ok {
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
			if order.Status == OrderStatusNew {
				//logger.Debugf("SAVE INTERNAL ORDER %d %v", order.ID, order)
				internalOrders[order.ID] = order
			}
			if order.Status == OrderStatusClosed &&
				order.FilledSize != 0 {
				if pos, ok := internalPositions[order.Market]; ok {
					if pos.ParseTime.Sub(order.ParseTime) > time.Second {
						continue
					}
					size := order.FilledSize
					if order.Side != OrderSideBuy {
						size = -order.FilledSize
					}
					price := order.AvgFillPrice
					if pos.NetSize*size < 0 {
						if math.Abs(size) > math.Abs(pos.NetSize) {
							pos.Cost = (pos.NetSize + size) * price
							pos.NetSize = pos.NetSize + size
						} else {
							pos.Cost = pos.Cost - size*price
							pos.NetSize = pos.NetSize + size
						}
					} else {
						pos.Cost += size * price
						pos.NetSize += size
					}
					pos.ParseTime = order.ParseTime
					//logger.Debugf("UPDATE POSITION BY FILL %v", pos)
					internalPositions[order.Market] = pos
					if positionCh, ok := positionsCh[order.Market]; ok {
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

			}
			if orderCh, ok := ordersCh[order.Market]; ok {
				select {
				case orderCh <- &order:
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("orderCh <- &order failed, ch len %d", len(orderCh))
						logSilentTime = time.Now().Add(time.Minute)
					}
				}
			} else {
				logger.Debugf("ORDER FROM OTHER PLACE %v", order)
			}
			break
		case fill := <-userWS.FillCh:
			//logger.Debugf("FILL %d %v INTERNAL ORDER %v INTERNAL POSITION %v", fill.OrderId, fill, internalOrders[fill.OrderId], internalPositions[fill.Market])
			if order, ok := internalOrders[fill.OrderId]; ok {
				fill.PostOnly = order.PostOnly
				fill.ReduceOnly = order.ReduceOnly
				fill.ClientId = order.ClientId
				fill.Price = order.Price
				fill.Size = order.Size
				fill.OrderType = order.Type
			}
			if orderCh, ok := ordersCh[fill.Market]; ok {
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

func (ftx *FtxUsdSpot) StreamTrade(ctx context.Context, channels map[string]chan common.Trade, batchSize int) {
	logger.Debugf("START StreamTrade")
	defer logger.Debugf("STOP StreamTrade")
	defer ftx.Stop()

	markets := make([]string, 0)
	for market := range channels {
		markets = append(markets, market)
	}

	proxy := ftx.settings.Proxy

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

func (ftx *FtxUsdSpot) StreamTicker(ctx context.Context, channels map[string]chan common.Ticker, batchSize int) {
	logger.Debugf("START StreamTicker")
	defer logger.Debugf("STOP StreamTicker")
	defer ftx.Stop()

	markets := make([]string, 0)
	for market := range channels {
		markets = append(markets, market)
	}

	proxy := ftx.settings.Proxy

	for start := 0; start < len(markets); start += batchSize {
		end := start + batchSize
		if end > len(markets) {
			end = len(markets)
		}
		subChannels := make(map[string]chan common.Ticker)
		for _, market := range markets[start:end] {
			subChannels[market] = channels[market]
		}
		go func(ctx context.Context, proxy string, channels map[string]chan common.Ticker) {
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
		case <-ftx.done:
			return
		}
	}
}

func (ftx *FtxUsdSpot) StreamKLine(ctx context.Context, channels map[string]chan []common.KLine, batchSize int, interval, lookback time.Duration) {
	panic("implement me")
}

func (ftx *FtxUsdSpot) StreamFundingRate(ctx context.Context, channels map[string]chan common.FundingRate, batchSize int) {
	pullInterval := ftx.settings.PullInterval + time.Duration(len(channels))*time.Second
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	pullTimes := make(map[string]time.Time)
	timeOffset := time.Second
	for market := range channels {
		timeOffset += time.Second
		pullTimes[market] = time.Now().Add(timeOffset)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			for market, pullTime := range pullTimes {
				if time.Now().Sub(pullTime) < 0 {
					continue
				}
				subCtx, cancel := context.WithTimeout(ctx, time.Minute)
				fs, err := ftx.api.GetFutureStats(subCtx, market)
				if err != nil {
					logger.Debugf("ftx.api.GetFutureStats(subCtx, %s) error %v", market, err)
					pullTimes[market] = time.Now().Add(time.Second)
				} else {
					if ch, ok := channels[fs.Future]; ok {
						select {
						case ch <- fs:
							pullTimes[market] = time.Now().Add(pullInterval)
						default:
							logger.Debugf("ch <- fs failed ch len %d", len(ch))
							pullTimes[market] = time.Now().Add(time.Second)
						}
					}
				}
				cancel()
			}
			timer.Reset(time.Second)
		}
	}
}

func (ftx *FtxUsdSpot) WatchOrders(
	ctx context.Context,
	requestChannels map[string]chan common.OrderRequest,
	responseChannels map[string]chan common.Order,
	errorChannels map[string]chan common.OrderError,
) {
	defer ftx.Stop()
	for market, reqCh := range requestChannels {
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
		go ftx.watchOrder(ctx, market, reqCh, respCh, errCh)
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

func (ftx *FtxUsdSpot) Setup(ctx context.Context, settings common.ExchangeSettings) error {
	var err error
	if settings.PullInterval == 0 {
		settings.PullInterval = time.Minute
	}
	ftx.settings = settings
	ftx.done = make(chan interface{})
	ftx.stopped = false
	ftx.api, err = NewAPI(settings.ApiKey, settings.ApiSecret, settings.Proxy)
	if err != nil {
		return err
	}
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

func (ftx *FtxUsdSpot) watchOrder(
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
				ftx.submitOrder(ctx, *req.New, responseCh, errorCh)
			} else if req.Cancel != nil {
				ftx.cancelOrder(ctx, *req.Cancel, errorCh)
			}
		}
	}
}

func (ftx *FtxUsdSpot) Stop() {
	if !ftx.stopped {
		ftx.stopped = true
		close(ftx.done)
		logger.Debugf("stopped")
	}
}

func (ftx *FtxUsdSpot) Done() chan interface{} {
	return ftx.done
}

func (ftx *FtxUsdSpot) positionsLoop(ctx context.Context, markets []string, positionsCh chan []Position) {
	pullInterval := ftx.settings.PullInterval
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
					hasPositions[position.Market] = true
					position := position
					outPositions = append(outPositions, position)
				}
				for _, market := range markets {
					if _, ok := hasPositions[market]; !ok {
						outPositions = append(outPositions, Position{
							Market:    market,
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
			pullTimer.Reset(time.Now().Truncate(pullInterval).Add(pullInterval).Sub(time.Now()))
			break
		}
	}
}

func (ftx *FtxUsdSpot) accountLoop(ctx context.Context, accountCh chan *Account) {
	pullInterval := ftx.settings.PullInterval
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
			pullTimer.Reset(time.Now().Truncate(pullInterval).Add(pullInterval).Sub(time.Now()))
			break
		}
	}
}

func (ftx *FtxUsdSpot) submitOrder(ctx context.Context, param common.NewOrderParam, respCh chan common.Order, errCh chan common.OrderError) {
	newOrderParam := NewOrderParam{}
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
	if param.TimeInForce == common.OrderTimeInForceIOC ||
		param.TimeInForce == common.OrderTimeInForceFOK {
		newOrderParam.Ioc = true
	}
	newOrderParam.PostOnly = param.PostOnly
	newOrderParam.ReduceOnly = param.ReduceOnly
	if param.Price != 0 {
		newOrderParam.Price = &param.Price
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

func (ftx *FtxUsdSpot) cancelOrder(ctx context.Context, param common.CancelOrderParam, errCh chan common.OrderError) {
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

func (ftx *FtxUsdSpot) StreamDepth(ctx context.Context, channels map[string]chan common.Depth, batchSize int) {
	logger.Debugf("START StreamDepth")
	defer logger.Debugf("STOP StreamDepth")
	defer ftx.Stop()

	markets := make([]string, 0)
	for market := range channels {
		markets = append(markets, market)
	}

	proxy := ftx.settings.Proxy

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
