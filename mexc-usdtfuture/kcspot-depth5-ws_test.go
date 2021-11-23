package mexc_usdtfuture

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"testing"
)



func TestNewDepth5WS(t *testing.T) {
	var ctx = context.Background()
	symbols := []string{"BTC_USDT"}
	//symbols := make([]string, 0)
	//for symbol := range TickSizes {
	//	symbols = append(symbols, symbol)
	//}
	channels := make(map[string]chan common.Depth)
	outputCh := make(chan common.Depth, 128)
	for _, symbol := range symbols {
		channels[symbol] = outputCh
	}
	ws := NewDepth5WS(
		ctx,
		"socks5://127.0.0.1:1081",
		channels,
	)
	for {
		select {
		case <- ws.Done():
			return
		case d := <-outputCh:
			logger.Debugf("%s %v", d.GetSymbol(), d.GetEventTime())
		}
	}
}


