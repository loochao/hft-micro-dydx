package dydx_usdfuture

import (
	"context"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"testing"
)

func TestNewUserWebsocket(t *testing.T) {
	var ctx = context.Background()
	userWS := NewUserWebsocket(
		ctx,
		&Credentials{
			ApiKey:        os.Getenv("DYDX_TEST_KEY"),
			ApiSecret:     os.Getenv("DYDX_TEST_SECRET"),
			ApiPassphrase: os.Getenv("DYDX_TEST_PASSPHRASE"),
			AccountID:     os.Getenv("DYDX_TEST_ACCOUNT_ID"),
			AccountNumber: "0",
		},
		os.Getenv("DYDX_TEST_PROXY"),
	)
	for {
		select {
		case a := <-userWS.AccountCh:
			logger.Debugf("%v", a)
		case ps := <-userWS.PositionsCh:
			logger.Debugf("%v", ps)
		case os := <-userWS.OrdersCh:
			logger.Debugf("%v", os)
		}
	}
}
