package kucoin_usdtspot

import (
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/stretchr/testify/assert"
	"log"
	"os"
	"sort"
	"strconv"
	"testing"
	"time"
)

func TestAPI_GetAccounts(t *testing.T) {
	var api *API
	var ctx = context.Background()
	var err error
	logger.Debugf("KCSPOT_KEY %s", os.Getenv("KCSPOT_KEY"))
	api, err = NewAPI(
		os.Getenv("KCSPOT_KEY"),
		os.Getenv("KCSPOT_SECRET"),
		os.Getenv("KCSPOT_PASSPHRASE"),
		"socks5://127.0.0.1:1081")
	if err != nil {
		log.Fatal(err)
	}
	accounts, err := api.GetAccounts(ctx, AccountsParam{})
	if err != nil {
		logger.Debugf("%v", err)
		t.Fatal(err)
	}
	logger.Debugf("%v", accounts)
}

func TestAPI_GetSymbols(t *testing.T) {
	var api *API
	var ctx = context.Background()
	var err error
	api, err = NewAPI(
		"60731ffcd260170006f6d51f",
		"da7dcb64-a777-432f-8dc2-bc96d6ff4288",
		"bitcoin",
		//os.Getenv("KCSPOT_KEY"),
		//os.Getenv("KCSPOT_SECRET"),
		//os.Getenv("KCSPOT_PASSPHRASE"),
		"socks5://127.0.0.1:1081")
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

func TestAPI_GetLimits(t *testing.T) {
	var api *API
	var ctx = context.Background()
	var err error
	api, err = NewAPI(
		os.Getenv("KCSPOT_KEY"),
		os.Getenv("KCSPOT_SECRET"),
		os.Getenv("KCSPOT_PASSPHRASE"),
		"socks5://127.0.0.1:1083")
	if err != nil {
		log.Fatal(err)
	}
	symbols, err := api.GetSymbols(ctx)
	if err != nil {
		t.Fatal(err)
	}
	usdtSymbols := make([]Symbol, 0)
	for _, s := range symbols {
		if s.QuoteCurrency == "USDT" && s.Market == "USDS" && s.EnableTrading {
			s := s
			usdtSymbols = append(usdtSymbols, s)
		}
	}
	stepSizes := make(map[string]float64)
	tickSizes := make(map[string]float64)
	minSizes := make(map[string]float64)
	maxSizes := make(map[string]float64)
	minNotionals := make(map[string]float64)
	maxNotionals := make(map[string]float64)
	ss := make([]string, 0)
	for _, s := range symbols {
		if s.QuoteCurrency == "USDT" && s.Market == "USDS" && s.EnableTrading {
			stepSizes[s.Symbol] = s.BaseIncrement
			tickSizes[s.Symbol] = s.PriceIncrement
			minSizes[s.Symbol] = s.BaseMinSize
			maxSizes[s.Symbol] = s.BaseMaxSize
			minNotionals[s.Symbol] = s.QuoteMinSize
			maxNotionals[s.Symbol] = s.QuoteMaxSize
			ss = append(ss, s.Symbol)
		}
	}
	sort.Strings(ss)
	str := "var StepSizes = map[string]float64{\n"
	for _, symbol := range ss {
		str += fmt.Sprintf(`  "%s": %s,
`, symbol, strconv.FormatFloat(stepSizes[symbol], 'f', -1, 64))
	}
	str += "}\n\n"
	str += "var TickSizes = map[string]float64{\n"
	for _, symbol := range ss {
		str += fmt.Sprintf(`  "%s": %s,
`, symbol, strconv.FormatFloat(tickSizes[symbol], 'f', -1, 64))
	}
	str += "}\n\n"
	str += "var MinSizes = map[string]float64{\n"
	for _, symbol := range ss {
		str += fmt.Sprintf(`  "%s": %s,
`, symbol, strconv.FormatFloat(minSizes[symbol], 'f', -1, 64))
	}
	str += "}\n\n"
	str += "var MaxSizes = map[string]float64{\n"
	for _, symbol := range ss {
		str += fmt.Sprintf(`  "%s": %s,
`, symbol, strconv.FormatFloat(maxSizes[symbol], 'f', -1, 64))
	}
	str += "}\n\n"
	str += "var MinNotionals = map[string]float64{\n"
	for _, symbol := range ss {
		str += fmt.Sprintf(`  "%s": %s,
`, symbol, strconv.FormatFloat(minNotionals[symbol], 'f', -1, 64))
	}
	str += "}\n\n"
	str += "var MaxNotionals = map[string]float64{\n"
	for _, symbol := range ss {
		str += fmt.Sprintf(`  "%s": %s,
`, symbol, strconv.FormatFloat(maxNotionals[symbol], 'f', -1, 64))
	}
	str += "}\n\n"
	fmt.Printf(str)
}

func TestAPI_GetTicker(t *testing.T) {
	var api *API
	var ctx = context.Background()
	var err error
	api, err = NewAPI(
		os.Getenv("KCSPOT_KEY"),
		os.Getenv("KCSPOT_SECRET"),
		os.Getenv("KCSPOT_PASSPHRASE"),
		"socks5://127.0.0.1:1081")
	if err != nil {
		log.Fatal(err)
	}
	ticker, err := api.GetTicker(ctx, TickerParam{
		Symbol: "KCS-USDT",
	})
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", *ticker)
}

func TestAPI_GetCandles(t *testing.T) {
	var api *API
	var ctx = context.Background()
	var err error
	api, err = NewAPI(
		os.Getenv("KCSPOT_KEY"),
		os.Getenv("KCSPOT_SECRET"),
		os.Getenv("KCSPOT_PASSPHRASE"),
		"socks5://127.0.0.1:1081")
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
	api, err = NewAPI(
		os.Getenv("KCSPOT_KEY"),
		os.Getenv("KCSPOT_SECRET"),
		os.Getenv("KCSPOT_PASSPHRASE"),
		"socks5://127.0.0.1:1081")
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
	api, err = NewAPI(
		os.Getenv("KCSPOT_KEY"),
		os.Getenv("KCSPOT_SECRET"),
		os.Getenv("KCSPOT_PASSPHRASE"),
		"socks5://127.0.0.1:1081")
	if err != nil {
		log.Fatal(err)
	}
	pct, err := api.GetPrivateConnectToken(ctx)
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", pct)
}

func TestAPI_SubmitOrder(t *testing.T) {
	var api *API
	var ctx = context.Background()
	var err error
	api, err = NewAPI(
		os.Getenv("KCSPOT_KEY"),
		os.Getenv("KCSPOT_SECRET"),
		os.Getenv("KCSPOT_PASSPHRASE"),
		"socks5://127.0.0.1:1081")
	if err != nil {
		t.Fatal(err)
	}
	oid, _ := common.GenerateShortId()
	res, err := api.SubmitOrder(ctx, NewOrderParam{
		ClientOid: oid,
		Symbol:    "BNB-USDT",
		Side:      OrderSideBuy,
		Type:      OrderTypeLimit,
		Price:     550,
		Size:      0.001,
	})
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", res)
}

func TestAPI_CancelAllOrders(t *testing.T) {
	var api *API
	var ctx = context.Background()
	var err error
	api, err = NewAPI(
		os.Getenv("KCSPOT_KEY"),
		os.Getenv("KCSPOT_SECRET"),
		os.Getenv("KCSPOT_PASSPHRASE"),
		"socks5://127.0.0.1:1081")
	if err != nil {
		t.Fatal(err)
	}
	res, err := api.CancelAllOrders(ctx, CancelAllOrdersParam{
		Symbol: "BNB-USDT",
	})
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", res)
}
