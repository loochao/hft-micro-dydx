package binance_busdspot

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"testing"
	"time"
)

func TestNewUserWebsocket(t *testing.T) {
	credentials := common.Credentials{
		Key:    os.Getenv("BN_KEY"),
		Secret: os.Getenv("BN_SECRET"),
	}
	proxy := os.Getenv("BN_PROXY")
	api, err := NewAPI(&credentials, proxy)
	if err != nil {
		t.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), time.Minute*1)
	ws, err := NewUserWebsocket(ctx, api,  proxy)
	if err != nil {
		t.Fatal(err)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ws.Done():
		case e := <-ws.AccountUpdateEventCh:
			logger.Debugf("%v", *e)
		case e := <-ws.OrderUpdateEventCh:
			logger.Debugf("%v", *e)
		}
	}
}
