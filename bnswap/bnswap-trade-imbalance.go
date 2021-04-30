package bnswap

import (
"context"
"fmt"
"github.com/geometrybase/hft-micro/common"
"github.com/geometrybase/hft-micro/logger"
"time"
)

func WatchTimedTradeImbalances(
	ctx context.Context,
	cancel context.CancelFunc,
	proxyAddress string,
	lookback time.Duration,
	channels map[string]chan *common.Signal,
) {
	matchesCh := make(map[string]chan *Trade)
	for symbol, output := range channels {
		matchesCh[symbol] = make(chan *Trade, 10000)
		go WatchTimedTradeImbalance(
			ctx,
			symbol,
			lookback,
			matchesCh[symbol],
			output,
		)
	}
	ws := NewTradeRoutedWS(
		ctx,
		proxyAddress,
		matchesCh,
	)
	defer ws.Stop()
	for {
		select {
		case <-ws.Done():
			cancel()
			return
		case <-ctx.Done():
			return
		}
	}

}

func WatchTimedTradeImbalance(
	ctx context.Context,
	symbol string,
	lookback time.Duration,
	inputCh chan *Trade,
	output chan *common.Signal,
) {
	updateImbalanceTimer := time.NewTimer(time.Hour * 999)
	buyVolume := common.NewTimedSum(lookback)
	sellVolume := common.NewTimedSum(lookback)
	var lastTrade, newMatch *Trade
	signalName := fmt.Sprintf("%s-bnswap-timed-trade-imbalance-%v", symbol, lookback)
	logSilentTime := time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		case <-updateImbalanceTimer.C:
			if lastTrade != nil {
				logger.Debugf("%s %v %f", lastTrade.Symbol, !lastTrade.IsTheBuyerTheMarketMaker, lastTrade.Quantity)
				if lastTrade.IsTheBuyerTheMarketMaker  {
					sellVolume.Insert(lastTrade.EventTime, lastTrade.Quantity)
				} else {
					buyVolume.Insert(lastTrade.EventTime, lastTrade.Quantity)
				}
				if buyVolume.Sum() < 0 {
					logger.Debugf("negative buy %v", buyVolume)
				}
				if sellVolume.Sum() < 0 {
					logger.Debugf("negative sell %v", sellVolume)
				}
				if (buyVolume.Sum() - sellVolume.Sum()) / (buyVolume.Sum() + sellVolume.Sum()) > 1 {
					logger.Debugf("bad > 1 buy %v", sellVolume)
					logger.Debugf("bad > 1 sell %v", buyVolume)
				}else if (buyVolume.Sum() - sellVolume.Sum()) / (buyVolume.Sum() + sellVolume.Sum()) < -1 {
					logger.Debugf("bad < -1 buy %v", sellVolume)
					logger.Debugf("bad < -1 sell %v", buyVolume)
				}
				if buyVolume.Range() > lookback/2 &&
					sellVolume.Range() > lookback/2 &&
					buyVolume.Sum()+sellVolume.Sum() != 0 {
					select {
					case output <- &common.Signal{
						Name:  signalName,
						Value: (buyVolume.Sum() - sellVolume.Sum()) / (buyVolume.Sum() + sellVolume.Sum()),
						Time:  lastTrade.EventTime,
					}:
					default:
						if time.Now().Sub(logSilentTime) > 0 {
							logger.Debugf("output <- &common.Signal failed, ch len %d", len(output))
							logSilentTime = time.Now().Add(time.Minute)
						}
					}
				}
			}
		case newMatch = <-inputCh:
			if lastTrade == nil || newMatch.EventTime.Sub(lastTrade.EventTime) > 0{
				lastTrade = newMatch
				updateImbalanceTimer.Reset(time.Nanosecond * 300)
			}
		}
	}

}
