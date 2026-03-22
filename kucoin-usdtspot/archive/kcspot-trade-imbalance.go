package archive

import (
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"time"
)

func StreamTimedTradeImbalances(
	ctx context.Context,
	cancel context.CancelFunc,
	proxyAddress string,
	lookback time.Duration,
	channels map[string]chan *common.Signal,
) {
	matchesCh := make(map[string]chan common.Trade)
	for symbol, output := range channels {
		matchesCh[symbol] = make(chan common.Trade, 10000)
		go common.StreamTimedTradeImbalance(
			ctx,
			fmt.Sprintf("kucoin-usdtspot-trade-imbalance-%s", symbol),
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

