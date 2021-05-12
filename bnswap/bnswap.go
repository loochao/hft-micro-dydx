package bnswap

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"sync"
	"time"
)

type Bnswap struct {
	api      *API
	done     chan interface{}
	stopped  bool
	mu       sync.Mutex
	settings common.ExchangeSettings
}

func (bn *Bnswap) GetMinNotional(symbol string) float64 {
	panic("implement me")
}

func (bn *Bnswap) GetMinSize(symbol string) float64 {
	panic("implement me")
}

func (bn *Bnswap) GetStepSize(symbol string) float64 {
	panic("implement me")
}

func (bn *Bnswap) GetTickSize(symbol string) float64 {
	panic("implement me")
}

func (bn *Bnswap) StreamBasic(ctx context.Context, statusCh chan common.SystemStatus, accountCh chan common.Account, positionCh map[string]chan common.Position, orderCh map[string]chan common.Order) {
	panic("implement me")
}

func (bn *Bnswap) StreamDepth(ctx context.Context, channels map[string]chan common.Depth, batchSize int) {
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

func (bn *Bnswap) StreamTrade(ctx context.Context, channels map[string]chan common.Trade, batchSize int) {
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

func (bn *Bnswap) StreamTicker(ctx context.Context, channels map[string]chan common.Ticker, batchSize int) {
	panic("implement me")
}

func (bn *Bnswap) StreamKLine(ctx context.Context, channels map[string]chan []common.KLine, batchSize int, interval, lookback time.Duration) {
	panic("implement me")
}

func (bn *Bnswap) StreamFundingRate(ctx context.Context, channels map[string]chan common.FundingRate, batchSize int) {
	panic("implement me")
}

func (bn *Bnswap) WatchOrders(ctx context.Context, requestChannels map[string]chan common.OrderRequest, responseChannels map[string]chan common.Order, errorChannels map[string]chan common.OrderError) {
	panic("implement me")
}

func (bn *Bnswap) Setup(ctx context.Context, settings common.ExchangeSettings) (err error) {
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
		if settings.ChangeLeverage {
			res, err := bn.api.UpdateLeverage(ctx, UpdateLeverageParams{
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
			res, err := bn.api.UpdateMarginType(ctx, UpdateMarginTypeParams{
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

func (bn *Bnswap) Stop() {
	bn.mu.Lock()
	if !bn.stopped {
		bn.stopped = true
		close(bn.done)
		logger.Debugf("stopped")
	}
	bn.mu.Unlock()
}

func (bn *Bnswap) Done() chan interface{} {
	return bn.done
}
