package binance_usdtspot

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"testing"
	"time"
)

func TestNewRawBookTickerWS(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), time.Minute*5)
	symbols := []string{
		"BUSDUSDT",
		"TUSDUSDT", "TUSDBUSD",
		"USDPUSDT", "USDPBUSD",
		"USDCUSDT", "USDCBUSD",
		"USDTDAI",
	}
	proxy := "socks5://127.0.0.1:1081"
	channels := make(map[string]chan *common.RawMessage)
	ch := make(chan *common.RawMessage)
	for _, symbol := range symbols {
		channels[symbol] = ch
	}
	ws := NewRawBookTickerWS(ctx, proxy, []byte{'X','T'},channels)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ws.Done():
			return
		case data := <-ch:
			logger.Debugf("%s", data.Data)
			break
		}
	}
}
