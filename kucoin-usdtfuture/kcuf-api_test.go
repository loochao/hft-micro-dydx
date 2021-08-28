package kucoin_usdtfuture

import (
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/stretchr/testify/assert"
	"os"
	"sort"
	"strconv"
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

func TestAPI_GetLimits(t *testing.T) {
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

	multipliers := make(map[string]float64)
	tickSizes := make(map[string]float64)
	lotSizes := make(map[string]float64)
	maxPrices := make(map[string]float64)
	tickPrecisions := make(map[string]int)
	stepPrecisions := make(map[string]int)
	ss := make([]string, 0)
	maxOrderSizes := make(map[string]float64)
	for _, s := range symbols {
		if s.QuoteCurrency == "USDT" && s.Status == "Open" && s.FairMethod == "FundingRate" {
			multipliers[s.Symbol] = s.Multiplier
			tickSizes[s.Symbol] = s.TickSize
			tickPrecisions[s.Symbol] = common.GetFloatPrecision(s.TickSize)
			lotSizes[s.Symbol] = s.LotSize
			stepPrecisions[s.Symbol] = common.GetFloatPrecision(s.LotSize)
			maxPrices[s.Symbol] = s.MaxPrice
			maxOrderSizes[s.Symbol] = s.MaxOrderQty
			ss = append(ss, s.Symbol)
		}
	}
	sort.Strings(ss)
	str := "var Multipliers = map[string]float64{\n"
	for _, symbol := range ss {
		str += fmt.Sprintf(`  "%s": %s,
`, symbol, strconv.FormatFloat(multipliers[symbol], 'f', -1, 64))
	}
	str += "}\n\n"
	str += "var TickSizes = map[string]float64{\n"
	for _, symbol := range ss {
		str += fmt.Sprintf(`  "%s": %s,
`, symbol, strconv.FormatFloat(tickSizes[symbol], 'f', -1, 64))
	}
	str += "}\n\n"
	str += "var LotSizes = map[string]float64{\n"
	for _, symbol := range ss {
		str += fmt.Sprintf(`  "%s": %s,
`, symbol, strconv.FormatFloat(lotSizes[symbol], 'f', -1, 64))
	}
	str += "}\n\n"
	str += "var MaxPrices = map[string]float64{\n"
	for _, symbol := range ss {
		str += fmt.Sprintf(`  "%s": %s,
`, symbol, strconv.FormatFloat(maxPrices[symbol], 'f', -1, 64))
	}
	str += "}\n\n"
	str += "var MaxOrderSizes = map[string]float64{\n"
	for _, symbol := range ss {
		str += fmt.Sprintf(`  "%s": %s,
`, symbol, strconv.FormatFloat(maxOrderSizes[symbol], 'f', -1, 64))
	}
	str += "}\n\n"
	str += "var TickPrecisions = map[string]int{\n"
	for _, symbol := range ss {
		str += fmt.Sprintf("  \"%s\": %d,\n", symbol, tickPrecisions[symbol])
	}
	str += "}\n\n"
	str += "var StepPrecisions = map[string]int{\n"
	for _, symbol := range ss {
		str += fmt.Sprintf("  \"%s\": %d,\n", symbol, stepPrecisions[symbol])
	}
	str += "}\n\n"
	fmt.Printf(str)
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
	//api, err = NewAPI(
	//	os.Getenv("KCPERP_KEY"),
	//	os.Getenv("KCPERP_SECRET"),
	//	os.Getenv("KCPERP_PASSPHRASE"),
	//	"socks5://127.0.0.1:1080")
	api, err = NewAPI(
		"60a3cb5632b1dc000699fc3a",
		"41892859-d509-4e07-ba68-c2a80f1df056",
		"panda03",
		"socks5://127.0.0.1:1080")
	if err != nil {
		t.Fatal(err)
	}
	oid, _ := common.GenerateShortId()
	res, err := api.SubmitOrder(ctx, NewOrderParam{
		ClientOid:   oid,
		Symbol:      "BTTUSDTM",
		Side:        OrderSideSell,
		TimeInForce: OrderTimeInForceIOC,
		Type:        OrderTypeLimit,
		Price:       common.Float64(0.0048),
		Size:        1,
		Leverage:    3,
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
