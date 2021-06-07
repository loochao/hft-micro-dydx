package bncoinfuture

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"testing"
	"time"
)

func TestNewKline1MRoutedWebsocket(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), time.Minute*5)
	symbols := []string{"BTCUSDT", "LINKUSDT", "WAVESUSDT"}
	proxy := "socks5://127.0.0.1:1081"

	channels := make(map[string]chan common.KLine)
	for _, symbol := range symbols[:1] {
		channels[symbol] = make(chan common.KLine, 100)
	}

	ws := NewKline1MRoutedWebsocket(ctx, proxy, channels)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ws.Done():
			return
		case msg := <-channels[symbols[0]]:
			logger.Debugf("%s %v %f", msg.Symbol, msg.Timestamp, msg.Close)
		//case _ = <-channels[symbols[1]]:
		//	//logger.Debugf("%s %v %f", msg.Market, msg.Timestamp, msg.Close)
		//case _ = <-channels[symbols[2]]:
		//	//logger.Debugf("%s %v %f", msg.Market, msg.Timestamp, msg.Close)
		}
	}
}
