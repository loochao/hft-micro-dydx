package hbspot

import (
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/logger"
	"math/rand"
	"os"
	"testing"
	"time"
)

func TestAPI_GetSymbols(t *testing.T) {
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
	hb, err := api.GetSymbols(ctx)
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
	bars, err := api.GetKlines(ctx, KlinesParam{
		Symbol: "filusdt",
		Period: KlinePeriod60min,
		Size:   100,
	})
	if err != nil {
		logger.Errorf("%v", err)
		t.Fatal(err)
	}
	logger.Debugf("%v", bars)
	logger.Debugf("%v", bars[len(bars)-1])
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
	accounts, err := api.GetAccounts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", accounts)
}

func TestAPI_GetAccount(t *testing.T) {
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
	accounts, err := api.GetAccounts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	spotAccountID := int64(0)
	for _, a := range accounts {
		if a.Type == "spot" {
			spotAccountID = a.ID
		}
	}
	if spotAccountID > 0 {
		account, err := api.GetAccount(ctx, spotAccountID)
		if err != nil {
			t.Fatal(err)
		}
		logger.Debugf("%v", *account)
	} else {
		t.Fatal("spot account id not found")
	}
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
	accounts, err := api.GetAccounts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	spotAccountID := int64(0)
	for _, a := range accounts {
		if a.Type == "spot" {
			spotAccountID = a.ID
		}
	}
	if spotAccountID == 0 {
		t.Fatal("spot account id not found")
		return
	}
	frs, err := api.SubmitOrder(ctx, NewOrderParam{
		AccountId:     spotAccountID,
		Symbol:        "filusdt",
		ClientOrderID: fmt.Sprintf("%d%04d", time.Now().Unix(), rand.Intn(10000)),
		Price:         "165",
		Amount:        "0.1",
		Type:          OrderTypeBuyLimit,
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
	accounts, err := api.GetAccounts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	spotAccountID := int64(0)
	for _, a := range accounts {
		if a.Type == "spot" {
			spotAccountID = a.ID
		}
	}
	if spotAccountID == 0 {
		t.Fatal("spot account id not found")
		return
	}

	order, err := api.SubmitOrder(ctx, NewOrderParam{
		AccountId:     spotAccountID,
		Symbol:        "filusdt",
		ClientOrderID: fmt.Sprintf("%d%04d", time.Now().Unix(), rand.Intn(10000)),
		Price:         "165",
		Amount:        "0.1",
		Type:          OrderTypeBuyLimit,
	})
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", order)
	resp, err := api.CancelAllOrders(ctx, CancelAllParam{
		AccountId:     spotAccountID,
		Symbol:        "filusdt",
	})
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", resp)

}
