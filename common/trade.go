package common

import (
	"context"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

type Trade interface {
	GetPrice() float64
	GetSize() float64
	GetTime() time.Time
	IsUpTick() bool
	GetSymbol() string
}

type MIR struct {
	Symbol    string
	Time      time.Time
	Value     float64
	LastPrice float64
}

func StreamMIR(
	ctx context.Context,
	Symbol string,
	lookback time.Duration,
	updateInterval time.Duration,
	updateOffset time.Duration,
	minTradeValue float64,
	tradeCh chan Trade,
	mirCh chan MIR,
) {
	var trade Trade
	updateTimer := time.NewTimer(time.Now().Truncate(updateInterval).Add(updateInterval+updateOffset).Sub(time.Now()))
	defer updateTimer.Stop()
	tf := NewTimedFloat64s(lookback)
	for {
		select {
		case <-ctx.Done():
			return
		case <-updateTimer.C:
			if tf.Range() > lookback/4 {
				select {
				case mirCh <- MIR{
					Value:     ComputeMIR(tf.Values()),
					Symbol:    Symbol,
					Time:      time.Now().Truncate(lookback),
					LastPrice: tf.Values()[len(tf.Values())-1],
				}:
				default:
					logger.Debugf("mirCh <- MIR failed, ch len %d", len(mirCh))
				}
			}
			updateTimer.Reset(time.Now().Truncate(updateInterval).Add(updateInterval+updateOffset).Sub(time.Now()))
		case trade = <-tradeCh:
			if trade.GetPrice()*trade.GetSize() > minTradeValue {
				tf.Insert(trade.GetTime(), trade.GetPrice())
			}
		}
	}
}

func StreamTimedTradeImbalance(
	ctx context.Context,
	signalName string,
	lookback time.Duration,
	tradeCh chan Trade,
	signalCh chan *Signal,
) {
	updateImbalanceTimer := time.NewTimer(time.Hour * 999)
	buyVolume := NewTimedSum(lookback)
	sellVolume := NewTimedSum(lookback)
	var lastTrade, newTrade Trade
	logSilentTime := time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		case <-updateImbalanceTimer.C:
			if lastTrade != nil {
				if lastTrade.IsUpTick() {
					buyVolume.Insert(lastTrade.GetTime(), lastTrade.GetSize())
				} else {
					sellVolume.Insert(lastTrade.GetTime(), lastTrade.GetSize())
				}
				if buyVolume.Range() > lookback/2 &&
					sellVolume.Range() > lookback/2 &&
					buyVolume.Sum()+sellVolume.Sum() != 0 {
					select {
					case signalCh <- &Signal{
						Name:   signalName,
						Value:  (buyVolume.Sum() - sellVolume.Sum()) / (buyVolume.Sum() + sellVolume.Sum()),
						Weight: buyVolume.Sum() + sellVolume.Sum(),
						Time:   lastTrade.GetTime(),
					}:
					default:
						if time.Now().Sub(logSilentTime) > 0 {
							logger.Debugf("signalCh <- &Signal failed, ch len %d", len(signalCh))
							logSilentTime = time.Now().Add(time.Minute)
						}
					}
				}
			}
		case newTrade = <-tradeCh:
			if lastTrade == nil || newTrade.GetTime().Sub(lastTrade.GetTime()) >= 0 {
				lastTrade = newTrade
				updateImbalanceTimer.Reset(time.Nanosecond * 300)
			}
		}
	}

}
