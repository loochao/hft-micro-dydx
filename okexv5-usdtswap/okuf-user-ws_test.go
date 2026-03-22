package okexv5_usdtswap

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
		os.Getenv("OK_PROXY"),
	)
	for {
		select {
		case d := <-ws.OrdersCh:
			logger.Debugf("O  %v", d)
		case d := <-ws.AccountsCh:
			logger.Debugf("A  %v", d)
		case d := <-ws.PositionsCh:
			logger.Debugf("P %v", d)
		}
	}
}
