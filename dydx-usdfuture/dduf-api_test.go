package dydx_usdfuture

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/geometrybase/hft-micro/starkex"
	"math"
	"math/big"
	"math/rand"
	"os"
	"sort"
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
	maxPosSizes := make(map[string]float64)
	tickPrecisions := make(map[string]int)
	stepPrecisions := make(map[string]int)
	maxPosValues := make(map[string]float64)

	symbols := make([]string, 0)
	for _, market := range markets {
		if market.Type != "PERPETUAL" || market.Status != "ONLINE" {
			continue
		}
		symbols = append(symbols, market.Market)
		tickSizes[market.Market] = market.TickSize
		tickPrecisions[market.Market] = common.GetFloatPrecision(market.TickSize)
		stepSizes[market.Market] = market.StepSize
		minSizes[market.Market] = market.MinOrderSize
		maxPosSizes[market.Market] = market.MaxPositionSize
		stepPrecisions[market.Market] = common.GetFloatPrecision(market.StepSize)
		maxPosValues[market.Market] = math.Floor(market.MaxPositionSize*market.IndexPrice/10000) * 10000
	}
	str := "var TickSizes = map[string]float64{\n"
	sort.Strings(symbols)
	for _, symbol := range symbols {
		value := tickSizes[symbol]
		str += fmt.Sprintf("  \"%s\": %s,\n", symbol, strconv.FormatFloat(value, 'f', -1, 64))
	}
	str += "}\n\n"
	str += "var StepSizes = map[string]float64{\n"
	for _, symbol := range symbols {
		value := stepSizes[symbol]
		str += fmt.Sprintf("  \"%s\": %s,\n", symbol, strconv.FormatFloat(value, 'f', -1, 64))
	}
	str += "}\n\n"
	str += "var MinSizes = map[string]float64{\n"
	for _, symbol := range symbols {
		value := minSizes[symbol]
		str += fmt.Sprintf("  \"%s\": %s,\n", symbol, strconv.FormatFloat(value, 'f', -1, 64))
	}
	str += "}\n\n"
	str += "var TickPrecisions = map[string]int{\n"
	for _, symbol := range symbols {
		value := tickPrecisions[symbol]
		str += fmt.Sprintf("  \"%s\": %d,\n", symbol, value)
	}
	str += "}\n\n"
	str += "var StepPrecisions = map[string]int{\n"
	for _, symbol := range symbols {
		value := stepPrecisions[symbol]
		str += fmt.Sprintf("  \"%s\": %d,\n", symbol, value)
	}
	str += "}\n\n"
	str += "var MaxPosSizes = map[string]float64{\n"
	for _, symbol := range symbols {
		value := maxPosSizes[symbol]
		str += fmt.Sprintf("  \"%s\": %s,\n", symbol, strconv.FormatFloat(value, 'f', -1, 64))
	}
	str += "}\n\n"
	str += "var MaxPosValues = map[string]float64{\n"
	for _, symbol := range symbols {
		value := maxPosValues[symbol]
		str += fmt.Sprintf("  \"%s\": %s,\n", symbol, strconv.FormatFloat(value, 'f', -1, 64))
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

func TestAPI_CreateOrder(t *testing.T) {
	api, err := NewAPI(Credentials{
		ApiKey:          os.Getenv("DYDX_TEST_KEY"),
		ApiSecret:       os.Getenv("DYDX_TEST_SECRET"),
		ApiPassphrase:   os.Getenv("DYDX_TEST_PASSPHRASE"),
		AccountID:       os.Getenv("DYDX_TEST_ACCOUNT_ID"),
		StarkPrivateKey: os.Getenv("DYDX_TEST_STARK_PRIVATE_KEY"),
	}, "socks5://127.0.0.1:1081")
	if err != nil {
		t.Fatal(err)
	}
	starkPrivateKey, _ := new(big.Int).SetString(os.Getenv("DYDX_TEST_STARK_PRIVATE_KEY"), 16)
	nop := NewOrderParams{
		PositionID:             119684,
		Market:                 "BTC-USD",
		Type:                   OrderTypeLimit,
		Side:                   OrderSideBuy,
		PostOnly:               true,
		TimeInForce:            TIME_IN_FORCE_GTT,
		LimitFee:               0.001,
		Price:                  45000,
		Size:                   0.001,
		ClientId:  fmt.Sprintf("%d%04d", time.Now().Unix(), rand.Intn(10000)),
		Expiration:             time.Now().UTC().Add(time.Hour).Format(TimeLayout),
		ExpirationEpochSeconds: time.Now().UTC().Add(time.Hour).Unix(),
	}
	err = nop.SetSignature(starkex.NETWORK_ID_MAINNET, starkPrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	o, err := api.CreateOrder(ctx, &nop)
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", o)
}

func TestAPI_CreateOrderByPython(t *testing.T) {
	err := os.Setenv("DYDX_PYTHON_URL", "http://127.0.0.1:5000/")
	if err != nil {
		t.Fatal(err)
	}
	api, err := NewAPI(Credentials{
		ApiKey:          os.Getenv("DYDX_TEST_KEY"),
		ApiSecret:       os.Getenv("DYDX_TEST_SECRET"),
		ApiPassphrase:   os.Getenv("DYDX_TEST_PASSPHRASE"),
		AccountID:       os.Getenv("DYDX_TEST_ACCOUNT_ID"),
		StarkPrivateKey: os.Getenv("DYDX_TEST_STARK_PRIVATE_KEY"),
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	account, err := api.CreateOrderByPython(ctx, &NewOrderParams{
		PositionID: 119684,
		Market:     "BTC-USD",
		Type:       OrderTypeLimit,
		Side:       OrderSideBuy,
		PostOnly:   true,
		LimitFee:   0.001,
		Price:      54000,
		Size:       0.001,
		Expiration: time.Now().UTC().Add(time.Hour).Format(TimeLayout),
	})
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", account)
}

func TestAPI_GetUsers(t *testing.T) {
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
	account, err := api.GetUsers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", account)
}
