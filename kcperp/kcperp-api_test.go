package kcperp

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
)

func TestAPI_GetContracts(t *testing.T) {
	var api *API
	var ctx = context.Background()
	var err error
	api, err = NewAPI(
		os.Getenv("KCPERP_KEY"),
		os.Getenv("KCPERP_SECRET"),
		os.Getenv("KCPERP_PASSPHRASE"),
		"socks5://127.0.0.1:1080")
	if err != nil {
		t.Fatal(err)
	}
	symbols, err := api.GetContracts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	usdSymbols := make([]string, 0)
	for _, s := range symbols {
		if s.QuoteCurrency == "USDT" && s.Status == "Open" && s.FairMethod == "FundingRate" {
			logger.Debugf("%s %s-%s:%s", s.FairMethod, s.BaseCurrency, s.QuoteCurrency, s.Symbol)
			usdSymbols = append(usdSymbols, s.Symbol)
		}
	}
	logger.Debugf("%d", len(usdSymbols))
}

func TestAPI_GetKlines(t *testing.T) {
	var api *API
	var ctx = context.Background()
	var err error
	api, err = NewAPI(
		os.Getenv("KCPERP_KEY"),
		os.Getenv("KCPERP_SECRET"),
		os.Getenv("KCPERP_PASSPHRASE"),
		"socks5://127.0.0.1:1081")
	if err != nil {
		t.Fatal(err)
	}
	klines, err := api.GetKlines(ctx, KlinesParam{
		Symbol:      "XBTUSDTM",
		Granularity: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 200, len(klines))
	for i := 0; i < 199; i++ {
		//logger.Debugf("%v", klines[i].Timestamp)
		assert.Equal(t, true, klines[i+1].Timestamp.Sub(klines[i].Timestamp) > 0)
		assert.Equal(t, time.Minute, klines[i+1].Timestamp.Sub(klines[i].Timestamp))
	}
}

func TestAPI_GetPositions(t *testing.T) {
	var api *API
	var ctx = context.Background()
	var err error
	api, err = NewAPI(
		os.Getenv("KCPERP_KEY"),
		os.Getenv("KCPERP_SECRET"),
		os.Getenv("KCPERP_PASSPHRASE"),
		"socks5://127.0.0.1:1081")
	if err != nil {
		t.Fatal(err)
	}
	positions, err := api.GetPositions(ctx)
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", positions)
}

func TestAPI_GetAccountOverView(t *testing.T) {
	var api *API
	var ctx = context.Background()
	var err error
	api, err = NewAPI(
		os.Getenv("KCPERP_KEY"),
		os.Getenv("KCPERP_SECRET"),
		os.Getenv("KCPERP_PASSPHRASE"),
		"socks5://127.0.0.1:1081")
	if err != nil {
		t.Fatal(err)
	}
	accounts, err := api.GetAccountOverView(ctx, AccountParam{Currency: "USDT"})
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", accounts)
}

func TestAPI_CancelAllOrders(t *testing.T) {
	var api *API
	var ctx = context.Background()
	var err error
	api, err = NewAPI(
		os.Getenv("KCPERP_KEY"),
		os.Getenv("KCPERP_SECRET"),
		os.Getenv("KCPERP_PASSPHRASE"),
		"socks5://127.0.0.1:1081")
	if err != nil {
		t.Fatal(err)
	}
	res, err := api.CancelAllOrders(ctx, CancelAllOrdersParam{Symbol: "XBTUSDTM"})
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", res)
}

func TestAPI_SubmitOrder(t *testing.T) {
	var api *API
	var ctx = context.Background()
	var err error
	api, err = NewAPI(
		os.Getenv("KCPERP_KEY"),
		os.Getenv("KCPERP_SECRET"),
		os.Getenv("KCPERP_PASSPHRASE"),
		"socks5://127.0.0.1:1081")
	if err != nil {
		t.Fatal(err)
	}
	oid, _ := common.GenerateShortId()
	res, err := api.SubmitOrder(ctx, NewOrderParam{
		ClientOid: oid,
		Symbol:    "XBTUSDTM",
		Side:      OrderSideSell,
		Type:      OrderTypeLimit,
		Price:     63000,
		Size:      1,
		Leverage:  10,
	})
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", res)
}

func TestAPI_ChangeAutoDepositStatus(t *testing.T) {
	var api *API
	var ctx = context.Background()
	var err error
	api, err = NewAPI(
		os.Getenv("KCPERP_KEY"),
		os.Getenv("KCPERP_SECRET"),
		os.Getenv("KCPERP_PASSPHRASE"),
		"socks5://127.0.0.1:1081")
	if err != nil {
		t.Fatal(err)
	}
	res, err := api.ChangeAutoDepositStatus(ctx, AutoDepositStatusParam{
		Symbol: "XBTUSDTM",
		Status: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", res)
}

func TestAPI_GetSystemStatus(t *testing.T) {
	var api *API
	var ctx = context.Background()
	var err error
	api, err = NewAPI(
		os.Getenv("KCPERP_KEY"),
		os.Getenv("KCPERP_SECRET"),
		os.Getenv("KCPERP_PASSPHRASE"),
		"socks5://127.0.0.1:1081")
	if err != nil {
		t.Fatal(err)
	}
	res, err := api.GetSystemStatus(ctx)
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", res)
}
