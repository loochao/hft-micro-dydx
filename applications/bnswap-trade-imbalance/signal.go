package main

import (
	"context"
	"github.com/geometrybase/hft-micro/bnspot"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/cbspot"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/gtspot"
	"github.com/geometrybase/hft-micro/hbspot"
	"github.com/geometrybase/hft-micro/kcspot"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/geometrybase/hft-micro/okspot"
	"strings"
	"time"
)

func StreamMergedSignals(
	ctx context.Context,
	cancel context.CancelFunc,
	proxy string,
	exchanges map[string]string,
	symbolSignalMap map[string][]string,
	lookback time.Duration,
	signalTimeToLive time.Duration,
	updateInterval time.Duration,
	outputCh chan MergedSignal,
) {
	signalCh := make(chan *common.Signal, 10000)
	for exchange, symbolsStr := range exchanges {
		symbols := strings.Split(symbolsStr, ",")
		channels := make(map[string]chan *common.Signal)
		for _, symbol := range symbols {
			channels[symbol] = signalCh
		}
		switch exchange {
		case "bnswap":
			go bnswap.StreamTimedTradeImbalances(ctx, cancel, proxy, lookback, channels)
		case "bnspot":
			go bnspot.StreamTimedTradeImbalances(ctx, cancel, proxy, lookback, channels)
		case "cbspot":
			go cbspot.StreamTimedTradeImbalances(ctx, cancel, proxy, lookback, channels)
		case "hbspot":
			go hbspot.StreamTimedTradeImbalances(ctx, cancel, proxy, lookback, channels)
		case "kcspot":
			go kcspot.StreamTimedTradeImbalances(ctx, cancel, proxy, lookback, channels)
		case "gtspot":
			go gtspot.StreamTimedTradeImbalances(ctx, cancel, proxy, lookback, channels)
		case "okspot":
			go okspot.StreamTimedTradeImbalances(ctx, cancel, proxy, lookback, channels)
		default:
			logger.Debugf("unknown trade imbalance exchange %s", exchange)
		}
	}
	updateTimer := time.NewTimer(lookback + updateInterval)
	maps := make(map[string]*common.Signal)
	for {
		select {
		case <-ctx.Done():
			return
		case s := <-signalCh:
			maps[s.Name] = s
		case <-updateTimer.C:
			for symbol, signals := range symbolSignalMap {
				dir := 0.0
				values := make(map[string]float64)
				weight := 0.0
				for _, name := range signals {
					if s, ok := maps[name]; ok {
						if time.Now().Sub(s.Time) < signalTimeToLive {
							dir += s.Value*s.Weight
							values[name] = s.Value
							weight += s.Weight
						}
					} else {
						if time.Now().Truncate(time.Second*5).Add(updateInterval).Sub(time.Now()) > 0 {
							logger.Debugf("%s not found", name)
						}
					}
				}
				select {
				case outputCh <- MergedSignal{
					Symbol: symbol,
					Value:    dir/ weight,
					Signals: values,
				}:
					if time.Now().Truncate(time.Second*5).Add(updateInterval).Sub(time.Now()) > 0 {
						logger.Debugf("%s %f", symbol, dir/weight)
					}
				default:
					logger.Debugf("outputCh <- MergedSignal failed, ch len %d", len(outputCh))
				}
			}
			updateTimer.Reset(updateInterval)
		}
	}
}

