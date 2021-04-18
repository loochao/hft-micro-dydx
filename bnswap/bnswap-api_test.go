package bnswap

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"testing"
	"time"
)

func TestAPI_GetServerTime(t *testing.T) {
	proxy := "socks5://127.0.0.1:1081"

	api, err := NewAPI(&common.Credentials{}, proxy)
	if err != nil {
		t.Fatal(err)
	}
	tt, err := api.GetServerTime(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	diff := time.Now().UnixNano()/1000000 - tt.ServerTime
	logger.Debugf("DIFF %v", diff)
}
