package bybit_usdtfuture

import (
	"context"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"testing"
	"time"
)

func TestNewUserWebsocket(t *testing.T) {
	var ctx = context.Background()
	ws := NewUserWS(ctx,
		os.Getenv("BYBIT_TEST_KEY"),
		os.Getenv("BYBIT_TEST_SECRET"),
		os.Getenv("BYBIT_TEST_PROXY"),
	)
	for {
		select {
		case d := <-ws.OrdersCh:
			logger.Debugf("%v", d)
		case d := <-ws.WalletsCh:
			logger.Debugf("%v", d)
		case d := <-ws.PositionsCh:
			logger.Debugf("%v", d)
		//case d := <-ws.ExecutionsCh:
		//	logger.Debugf("%v", d)
		case <-ws.RestartCh:
			logger.Debugf("restart")
		}
	}
}

func TestUserWS_Done(t *testing.T) {
	a := make(chan interface{}, 100)
	go func() {
		for {
			a <- nil
		}
	}()
	time.Sleep(time.Second)
	select {
	case <-a:
		logger.Debugf("a")
	default:
		logger.Debugf("default")
	}
}
