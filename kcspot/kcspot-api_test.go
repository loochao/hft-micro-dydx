package kcspot

import (
	"context"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/stretchr/testify/assert"
	"log"
	"os"
	"testing"
	"time"
)

func TestAPI_GetAccounts(t *testing.T) {
	var api *API
	var ctx = context.Background()
	var err error
	logger.Debugf("KCSPOT_KEY %s", os.Getenv("KCSPOT_KEY"))
	api, err = NewAPI(NewKcSigner(
		"6073215c5e2b070006fa35fa",
		"29c5571b-eed3-4774-8102-c38c232fe25e",
		"maodayemaodaye",
		//os.Getenv("KCSPOT_KEY"),
		//os.Getenv("KCSPOT_SECRET"),
		//os.Getenv("KCSPOT_PASSPHRASE"),
	), "socks5://127.0.0.1:1081")
	if err != nil {
		log.Fatal(err)
	}
	accounts, err := api.GetAccounts(ctx, AccountsParam{Currency: "USDT"})
	if err != nil {
		logger.Debugf("%v", err)
		t.Fatal(err)
	}
	logger.Debugf("%d", len(accounts))
}

func TestAPI_GetSymbols(t *testing.T) {
	var api *API
	var ctx = context.Background()
	var err error
	api, err = NewAPI(NewKcSigner(
		"60731ffcd260170006f6d51f",
		"da7dcb64-a777-432f-8dc2-bc96d6ff4288",
		"bitcoin",
		//os.Getenv("KCSPOT_KEY"),
		//os.Getenv("KCSPOT_SECRET"),
		//os.Getenv("KCSPOT_PASSPHRASE"),
	), "socks5://127.0.0.1:1081")
	if err != nil {
		log.Fatal(err)
	}
	symbols, err := api.GetSymbols(ctx)
	if err != nil {
		logger.Debugf("%v", err)
		t.Fatal(err)
	}
	usdSymbols := make([]string, 0)
	for _, s := range symbols {
		if s.QuoteCurrency == "USDT" && s.Market == "USDS" && s.EnableTrading {
			logger.Debugf("%s", s.Symbol)
			usdSymbols = append(usdSymbols, s.Symbol)
		}
	}
	logger.Debugf("%d", len(usdSymbols))
}

func TestAPI_GetCandles(t *testing.T) {
	var api *API
	var ctx = context.Background()
	var err error
	api, err = NewAPI(NewKcSigner(
		os.Getenv("KCSPOT_KEY"),
		os.Getenv("KCSPOT_SECRET"),
		os.Getenv("KCSPOT_PASSPHRASE"),
	), "socks5://127.0.0.1:1081")
	if err != nil {
		log.Fatal(err)
	}
	candles, err := api.GetCandles(ctx, CandlesParam{
		Symbol: "BTC-USDT",
		Type:   CandleType1Min,
	})
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, time.Now().Truncate(time.Minute).Add(time.Minute), candles[len(candles)-1].Timestamp)
}

func TestAPI_GetPublicConnectToken(t *testing.T) {
	var api *API
	var ctx = context.Background()
	var err error
	api, err = NewAPI(NewKcSigner(
		os.Getenv("KCSPOT_KEY"),
		os.Getenv("KCSPOT_SECRET"),
		os.Getenv("KCSPOT_PASSPHRASE"),
	), "socks5://127.0.0.1:1081")
	if err != nil {
		log.Fatal(err)
	}
	pct, err := api.GetPublicConnectToken(ctx)
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", pct)
}

func TestAPI_GetPrivateConnectToken(t *testing.T) {
	var api *API
	var ctx = context.Background()
	var err error
	api, err = NewAPI(NewKcSigner(
		os.Getenv("KCSPOT_KEY"),
		os.Getenv("KCSPOT_SECRET"),
		os.Getenv("KCSPOT_PASSPHRASE"),
	), "socks5://127.0.0.1:1081")
	if err != nil {
		log.Fatal(err)
	}
	pct, err := api.GetPrivateConnectToken(ctx)
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", pct)
}
