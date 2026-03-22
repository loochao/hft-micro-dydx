package ftx_usdspot

import (
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestAPI_GetMarkets(t *testing.T) {
	api, err := NewAPI(
		os.Getenv("FTX_TEST_KEY"),
		os.Getenv("FTX_TEST_SECRET"),
		os.Getenv("FTX_TEST_SUBACCOUNT"),
		os.Getenv("FTX_TEST_PROXY"),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), time.Minute)
	markets, err := api.GetMarkets(ctx)
	if err != nil {
		t.Fatal(err)
	}
	sizeIncrements := make(map[string]float64)
	priceIncrements := make(map[string]float64)
	minProvideSizes := make(map[string]float64)
	for _, market := range markets {
		if market.Type == "spot" &&
			market.Enabled &&
			market.QuoteCurrency == "USD" &&
			!strings.Contains(market.Name, "BULL") &&
			!strings.Contains(market.Name, "BEAR") &&
			!strings.Contains(market.Name, "HALF") &&
			!strings.Contains(market.Name, "HEDGE") {
			sizeIncrements[market.Name] = market.SizeIncrement
			priceIncrements[market.Name] = market.PriceIncrement
			minProvideSizes[market.Name] = market.MinProvideSize
		}
	}
	fmt.Printf("var SizeIncrements = map[string]float64{\n")
	for name, value := range sizeIncrements {
		fmt.Printf("  \"%s\":%s,\n", name, strconv.FormatFloat(value, 'f', -1, 64))
	}
	fmt.Printf("}\n\n")
	fmt.Printf("var PriceIncrements = map[string]float64{\n")
	for name, value := range priceIncrements {
		fmt.Printf("  \"%s\":%s,\n", name, strconv.FormatFloat(value, 'f', -1, 64))
	}
	fmt.Printf("}\n\n")
	fmt.Printf("var MinProvideSizes = map[string]float64{\n")
	for name, value := range minProvideSizes {
		fmt.Printf("  \"%s\":%s,\n", name, strconv.FormatFloat(value, 'f', -1, 64))
	}
	fmt.Printf("}\n\n")
}

func TestAPI_GetFundingRates(t *testing.T) {
	api, err := NewAPI(
		os.Getenv("FTX_TEST_KEY"),
		os.Getenv("FTX_TEST_SECRET"),
		os.Getenv("FTX_TEST_SUBACCOUNT"),
		os.Getenv("FTX_TEST_PROXY"),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), time.Minute)
	fundingRates, err := api.GetFundingRates(ctx, FundingRateParam{})
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", fundingRates)
}

func TestAPI_ChangeLeverage(t *testing.T) {
	api, err := NewAPI(
		os.Getenv("FTX_TEST_KEY"),
		os.Getenv("FTX_TEST_SECRET"),
		os.Getenv("FTX_TEST_SUBACCOUNT"),
		os.Getenv("FTX_TEST_PROXY"),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), time.Minute)
	_, err = api.ChangeLeverage(ctx, LeverageParam{Leverage: 5})
	if err != nil {
		t.Fatal(err)
	}
}

func TestAPI_GetAccount(t *testing.T) {
	api, err := NewAPI(
		os.Getenv("FTX_TEST_KEY"),
		os.Getenv("FTX_TEST_SECRET"),
		os.Getenv("FTX_TEST_SUBACCOUNT"),
		os.Getenv("FTX_TEST_PROXY"),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), time.Minute)
	account, err := api.GetAccount(ctx)
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", account)
	logger.Debugf("Balance %f", account.GetBalance())
	logger.Debugf("Free %v", account.GetFree())
}

func TestAPI_GetBalances(t *testing.T) {
	api, err := NewAPI(
		os.Getenv("FTX_TEST_KEY"),
		os.Getenv("FTX_TEST_SECRET"),
		os.Getenv("FTX_TEST_SUBACCOUNT"),
		os.Getenv("FTX_TEST_PROXY"),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), time.Minute)
	positions, err := api.GetBalances(ctx)
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", positions)
}

func TestAPI_PlaceOrder(t *testing.T) {
	api, err := NewAPI(
		os.Getenv("FTX_TEST_KEY"),
		os.Getenv("FTX_TEST_SECRET"),
		os.Getenv("FTX_TEST_SUBACCOUNT"),
		os.Getenv("FTX_TEST_PROXY"),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), time.Minute)
	positions, err := api.PlaceOrder(ctx, NewOrderParam{
		Market: "FTT/USD",
		Size: 0.1,
		Type: OrderTypeMarket,
		Side: OrderSideSell,
	})
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", positions)
}
