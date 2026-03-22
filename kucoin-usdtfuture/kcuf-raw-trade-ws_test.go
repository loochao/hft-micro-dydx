package kucoin_usdtfuture

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"testing"
)

func TestNewRawTradeWS(t *testing.T) {
	var ctx = context.Background()
	symbols := []string{"XBTUSDTM", "ATOMUSDTM", "WAVESUSDTM"}
	channels := make(map[string]chan *common.RawMessage)
	outputCh := make(chan *common.RawMessage, 128)
	for _, symbol := range symbols {
		channels[symbol] = outputCh
	}
	ws := NewRawTradeWS(
		ctx,
		"socks5://127.0.0.1:1083",
		[]byte{'T'},
		channels,
	)
	for {
		select {
		case <- ws.Done():
			return
		case d := <-outputCh:
			logger.Debugf("%s",d.Data)
		}
	}
}
