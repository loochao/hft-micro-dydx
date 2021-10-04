package dxdy_usdtfuture

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"sync/atomic"
	"testing"
	"time"
)
func TestUserWebsocket_Stop(t *testing.T) {
	var a = int32(0)
	if atomic.CompareAndSwapInt32(&a, 0, 1) {
		logger.Debugf("swapped 0 -> %v", a)
	}
	if atomic.CompareAndSwapInt32(&a, 0, 1) {
		logger.Debugf("swapped 1 -> %v", a)
	}
}

func TestNewUserWebsocket(t *testing.T) {
	credentials := common.Credentials{
		Key:    os.Getenv("BN_TEST_KEY"),
		Secret: os.Getenv("BN_TEST_SECRET"),
	}
	proxy := os.Getenv("BN_TEST_PROXY")
	api, err := NewAPI(&credentials, proxy)
	if err != nil {
		t.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), time.Minute*1)
	ws , err := NewUserWebsocket(ctx, api, proxy)
	if err != nil {
		t.Fatal(err)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ws.Done():
		case e := <-ws.BalanceAndPositionUpdateEventCh:
			logger.Debugf("%v", *e)
		case e := <-ws.OrderUpdateEventCh:
			logger.Debugf("%v", *e)
		}
	}
}
