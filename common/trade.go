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
	IsBuy() bool
	GetSymbol() string
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
				if lastTrade.IsBuy()  {
					buyVolume.Insert(lastTrade.GetTime(), lastTrade.GetSize())
				} else {
					sellVolume.Insert(lastTrade.GetTime(), lastTrade.GetSize())
				}
				if buyVolume.Range() > lookback/2 &&
					sellVolume.Range() > lookback/2 &&
					buyVolume.Sum()+sellVolume.Sum() != 0 {
					select {
					case signalCh <- &Signal{
						Name:  signalName,
						Value: (buyVolume.Sum() - sellVolume.Sum()) / (buyVolume.Sum() + sellVolume.Sum()),
						Weight: buyVolume.Sum()+sellVolume.Sum(),
						Time:  lastTrade.GetTime(),
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
