package bswap

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"testing"
)

func TestAPI_GetPools(t *testing.T) {
	api, err := NewAPI(&common.Credentials{
		Key: os.Getenv("BN_TEST_KEY"),
		Secret: os.Getenv("BN_TEST_SECRET"),
	}, "socks5://127.0.0.1:1080")
	if err != nil {
		t.Fatal(err)
	}
	pools, err := api.GetPools(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", pools)
}

func TestAPI_GetLiquidity(t *testing.T) {
	api, err := NewAPI(&common.Credentials{
		Key: os.Getenv("BN_TEST_KEY"),
		Secret: os.Getenv("BN_TEST_SECRET"),
	}, "socks5://127.0.0.1:1080")
	if err != nil {
		t.Fatal(err)
	}
	pools, err := api.GetLiquidity(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", pools)
}
