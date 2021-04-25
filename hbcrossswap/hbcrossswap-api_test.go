package hbcrossswap

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"
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
	hb, err := api.GetTimestamp(ctx)
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
		"socks5://127.0.0.1:1082",
	)
	if err != nil {
		t.Fatal(err)
	}
	cs, err := api.GetContracts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cs {
		if c.SupportMarginMode == "all" {
			logger.Debugf("%s: %s", strings.Replace(strings.ToLower(c.Symbol), "-", "", -1), c.Symbol)
		}
	}
}

func TestAPI_GetKlines(t *testing.T) {
	var api *API
	var ctx = context.Background()
	var err error
	api, err = NewAPI(
		os.Getenv("HBSWAP_KEY"),
		os.Getenv("HBSWAP_SECRET"),
		"socks5://127.0.0.1:1082",
	)
	if err != nil {
		t.Fatal(err)
	}
	hb, err := api.GetKlines(ctx, KlinesParam{
		Symbol: "BTC-USDT",
		Period: KlinePeriod60min,
		Size:   1000,
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
		"socks5://127.0.0.1:1082",
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

func TestAPI_GetPositions(t *testing.T) {
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
	frs, err := api.GetPositions(ctx)
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", frs)
}

func TestAPI_GetAccounts(t *testing.T) {
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
	frs, err := api.GetAccounts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", frs)
}

func TestAPI_SubmitOrder(t *testing.T) {
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
	frs, err := api.SubmitOrder(ctx, NewOrderParam{
		Symbol:         "FIL-USDT",
		ClientOrderID:  time.Now().Unix()*10000 + int64(rand.Intn(10000)),
		Price:          common.Float64(173),
		Volume:         1,
		Direction:      OrderDirectionSell,
		Offset:         OrderOffsetOpen,
		LeverRate:      3,
		OrderPriceType: OrderPriceTypeLimit,
	})
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", frs)
}

func TestAPI_CancelAllOrder(t *testing.T) {
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
	order, err := api.SubmitOrder(ctx, NewOrderParam{
		Symbol:         "FIL-USDT",
		ClientOrderID:  time.Now().Unix()*10000 + int64(rand.Intn(10000)),
		Price:          common.Float64(180),
		Volume:         1,
		Direction:      OrderDirectionSell,
		Offset:         OrderOffsetOpen,
		LeverRate:      3,
		OrderPriceType: OrderPriceTypeLimit,
	})
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", order)
	resp, err := api.CancelAllOrders(ctx, CancelAllParam{
		Symbol: "FIL-USDT",
	})
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", resp)

}
