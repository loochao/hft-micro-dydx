package hbswap

import (
	"context"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"testing"
)

func TestAPI_GetHeartbeat(t *testing.T) {
	var api *API
	var ctx = context.Background()
	var err error
	api, err = NewAPI(
		os.Getenv("HBSWAP_KEY"),
		os.Getenv("HBSWAP_SECRET"),
		"socks5://127.0.0.1:1080",
	)
	if err != nil {
		t.Fatal(err)
	}
	hb, err := api.GetHeartbeat(ctx)
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", hb)
}

func TestAPI_GetContracts(t *testing.T) {
	var api *API
	var ctx = context.Background()
	var err error
	api, err = NewAPI(
		os.Getenv("HBSWAP_KEY"),
		os.Getenv("HBSWAP_SECRET"),
		"socks5://127.0.0.1:1080",
	)
	if err != nil {
		t.Fatal(err)
	}
	hb, err := api.GetContracts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", hb)
}

func TestAPI_GetKlines(t *testing.T) {
	var api *API
	var ctx = context.Background()
	var err error
	api, err = NewAPI(
		os.Getenv("HBSWAP_KEY"),
		os.Getenv("HBSWAP_SECRET"),
		"socks5://127.0.0.1:1080",
	)
	if err != nil {
		t.Fatal(err)
	}
	hb, err := api.GetKlines(ctx, KlinesParam{
		ContractCode: "BTC-USDT",
		Period: KlinePeriod60min,
		Size: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", hb)
}

func TestAPI_GetFundingRates(t *testing.T) {
	var api *API
	var ctx = context.Background()
	var err error
	api, err = NewAPI(
		os.Getenv("HBSWAP_KEY"),
		os.Getenv("HBSWAP_SECRET"),
		"socks5://127.0.0.1:1080",
	)
	if err != nil {
		t.Fatal(err)
	}
	frs, err := api.GetFundingRates(ctx)
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", frs)
}
