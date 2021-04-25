package okspot

import (
	"context"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"testing"
)

func TestNewUserWebsocket(t *testing.T) {
	var ctx = context.Background()
	ws := NewUserWebsocket(ctx,
		os.Getenv("OK_KEY"),
		os.Getenv("OK_SECRET"),
		os.Getenv("OK_PASSPHRASE"),
		[]string{"BTC-USDT", "WAVES-USDT", "ETH-USDT"},
		"socks5://127.0.0.1:1081",
	)
	for {
		select {
		case d := <-ws.OrdersCh:
			logger.Debugf("%v", d)
		case d := <-ws.BalancesCh:
			logger.Debugf("%v", d)
		}
	}
}
