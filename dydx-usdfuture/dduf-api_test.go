package dydx_usdfuture

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestAPI_GetExchangeInfo(t *testing.T) {
	proxy := "socks5://127.0.0.1:1083"

	api, err := NewAPI(Credentials{}, proxy)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	markets, err := api.GetMarkets(ctx)
	if err != nil {
		t.Fatal(err)
	}

	tickSizes := make(map[string]float64)
	stepSizes := make(map[string]float64)
	minSizes := make(map[string]float64)
	tickPrecisions := make(map[string]int)
	stepPrecisions := make(map[string]int)
	for _, market := range markets {
		if market.Type != "PERPETUAL" || market.Status != "ONLINE" {
			continue
		}
		tickSizes[market.Market] = market.TickSize
		tickPrecisions[market.Market] = common.GetFloatPrecision(market.TickSize)
		stepSizes[market.Market] = market.StepSize
		minSizes[market.Market] = market.MinOrderSize
		stepPrecisions[market.Market] = common.GetFloatPrecision(market.StepSize)
	}
	str := "var TickSizes = map[string]float64{\n"
	for symbol, value := range tickSizes {
		str += fmt.Sprintf("  \"%s\": %s,\n", symbol, strconv.FormatFloat(value, 'f', -1, 64))
	}
	str += "}\n\n"
	str += "var StepSizes = map[string]float64{\n"
	for symbol, value := range stepSizes {
		str += fmt.Sprintf("  \"%s\": %s,\n", symbol, strconv.FormatFloat(value, 'f', -1, 64))
	}
	str += "}\n\n"
	str += "var MinSizes = map[string]float64{\n"
	for symbol, value := range minSizes {
		str += fmt.Sprintf("  \"%s\": %s,\n", symbol, strconv.FormatFloat(value, 'f', -1, 64))
	}
	str += "}\n\n"
	str += "var TickPrecisions = map[string]int{\n"
	for symbol, value := range tickPrecisions {
		str += fmt.Sprintf("  \"%s\": %d,\n", symbol, value)
	}
	str += "}\n\n"
	str += "var StepPrecisions = map[string]int{\n"
	for symbol, value := range stepPrecisions {
		str += fmt.Sprintf("  \"%s\": %d,\n", symbol, value)
	}
	str += "}\n\n"
	fmt.Printf("%s", str)
}
func TestAPI_GetAccounts2(t *testing.T) {

	signature := "123"
	secret, err := base64.URLEncoding.DecodeString(os.Getenv("DYDX_TEST_SECRET"))
	if err != nil {
		t.Fatal(err)
	}
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(signature))
	hmacSigned := h.Sum(nil)
	s := base64.URLEncoding.EncodeToString(hmacSigned)
	logger.Debugf("%d", len(s))
	logger.Debugf("%s", s)
}

func TestAPI_GetAccounts(t *testing.T) {
	api, err := NewAPI(Credentials{
		ApiKey:        os.Getenv("DYDX_TEST_KEY"),
		ApiSecret:     os.Getenv("DYDX_TEST_SECRET"),
		ApiPassphrase: os.Getenv("DYDX_TEST_PASSPHRASE"),
		AccountID:     os.Getenv("DYDX_TEST_ACCOUNT_ID"),
	}, os.Getenv("DYDX_TEST_PROXY"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	accounts, err := api.GetAccounts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", accounts)
}

func TestAPI_GetAccount(t *testing.T) {
	api, err := NewAPI(Credentials{
		ApiKey:        os.Getenv("DYDX_TEST_KEY"),
		ApiSecret:     os.Getenv("DYDX_TEST_SECRET"),
		ApiPassphrase: os.Getenv("DYDX_TEST_PASSPHRASE"),
		AccountID:     os.Getenv("DYDX_TEST_ACCOUNT_ID"),
	}, os.Getenv("DYDX_TEST_PROXY"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	account, err := api.GetAccount(ctx)
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", account)
}

func TestAPI_GetOrders(t *testing.T) {
	api, err := NewAPI(Credentials{
		ApiKey:        os.Getenv("DYDX_TEST_KEY"),
		ApiSecret:     os.Getenv("DYDX_TEST_SECRET"),
		ApiPassphrase: os.Getenv("DYDX_TEST_PASSPHRASE"),
		AccountID:     os.Getenv("DYDX_TEST_ACCOUNT_ID"),
	}, os.Getenv("DYDX_TEST_PROXY"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	account, err := api.GetOrders(ctx)
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", account)
}

func TestAPI_CancelOrders(t *testing.T) {
	api, err := NewAPI(Credentials{
		ApiKey:        os.Getenv("DYDX_TEST_KEY"),
		ApiSecret:     os.Getenv("DYDX_TEST_SECRET"),
		ApiPassphrase: os.Getenv("DYDX_TEST_PASSPHRASE"),
		AccountID:     os.Getenv("DYDX_TEST_ACCOUNT_ID"),
	}, os.Getenv("DYDX_TEST_PROXY"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	account, err := api.CancelOrders(ctx, &CancelOrdersParam{
		Market: "BTC-USD",
	})
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", account)
}

func TestAPI_CreateOrders(t *testing.T) {
	err := os.Setenv("DYDX_PYTHON_URL", "http://127.0.0.1:5000/")
	if err != nil {
		t.Fatal(err)
	}
	api, err := NewAPI(Credentials{
		ApiKey:        os.Getenv("DYDX_TEST_KEY"),
		ApiSecret:     os.Getenv("DYDX_TEST_SECRET"),
		ApiPassphrase: os.Getenv("DYDX_TEST_PASSPHRASE"),
		AccountID:     os.Getenv("DYDX_TEST_ACCOUNT_ID"),
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	account, err := api.CreateOrder(ctx, &NewOrderParams{
		PositionID: "119684",
		Market: "BTC-USD",
		Type: OrderTypeLimit,
		Side: OrderSideBuy,
		PostOnly: true,
		LimitFee: 0.001,
		Price: 54000,
		Size: 0.001,
		Expiration: time.Now().UTC().Add(time.Hour).Format(TimeLayout),
	})
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", account)
}
