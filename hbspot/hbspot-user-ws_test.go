package hbspot

import (
	"context"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"testing"
)

func TestNewUserWebsocket(t *testing.T) {
	var ctx = context.Background()
	ws := NewUserWebsocket(ctx,
		os.Getenv("HBSWAP_KEY"),
		os.Getenv("HBSWAP_SECRET"),
		[]string{"filusdt"},
		"socks5://127.0.0.1:1082",
	)
	for {
		select {
		case d := <-ws.OrderCh:
			logger.Debugf("%v", d)
		case d := <-ws.BalanceCh:
			logger.Debugf("%v", d)
		}
	}
}
