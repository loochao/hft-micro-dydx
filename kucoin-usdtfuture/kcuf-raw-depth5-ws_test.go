package kucoin_usdtfuture

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"testing"
)

func TestNewRawDepth5WS(t *testing.T) {
	var ctx = context.Background()
	symbols := []string{"XBTUSDTM", "ATOMUSDTM", "WAVESUSDTM"}
	channels := make(map[string]chan *common.RawMessage)
	outputCh := make(chan *common.RawMessage, 128)
	for _, symbol := range symbols {
		channels[symbol] = outputCh
	}
	ws := NewRawDepth5WS(
		ctx,
		"socks5://127.0.0.1:1083",
		[]byte{'Y','T'},
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
