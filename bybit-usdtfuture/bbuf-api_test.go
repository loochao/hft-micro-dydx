package bybit_usdtfuture

import (
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/logger"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestAPI_GetSymbols(t *testing.T) {
	api, err := NewAPI("", "","https://api.bybitglobal.com", "socks5://127.0.0.1:1083")
	if err != nil {
		t.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), time.Minute)
	markets, err := api.GetSymbols(ctx)
	if err != nil {
		t.Fatal(err)
	}
	tickSizes := make(map[string]float64)
	stepSizes := make(map[string]float64)
	for _, market := range markets {
		if market.Status == "Trading" && market.QuoteCurrency == "USDT" {
			tickSizes[market.Name] = market.PriceFilter.TickSize
			stepSizes[market.Name] = market.LotSizeFilter.QtyStep
		}
	}
	fmt.Printf("var TickSizes = map[string]float64{\n")
	for name, value := range tickSizes {
		fmt.Printf("  \"%s\":%s,\n", name, strconv.FormatFloat(value, 'f', -1, 64))
	}
	fmt.Printf("}\n\n")
	fmt.Printf("var StepSizes = map[string]float64{\n")
	for name, value := range stepSizes {
		fmt.Printf("  \"%s\":%s,\n", name, strconv.FormatFloat(value, 'f', -1, 64))
	}
	fmt.Printf("}\n\n")
}

func TestAPI_GetPositions(t *testing.T) {
	api, err := NewAPI(
		os.Getenv("BYBIT_TEST_KEY"),
		os.Getenv("BYBIT_TEST_SECRET"),
		"https://api.bybitglobal.com",
		os.Getenv("BYBIT_TEST_PROXY"),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), time.Minute)
	positions, err := api.GetPositions(ctx)
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", positions)
}

func TestAPI_SetAutoAddMargin(t *testing.T) {
	api, err := NewAPI(
		os.Getenv("BYBIT_TEST_KEY"),
		os.Getenv("BYBIT_TEST_SECRET"),
		"",
		os.Getenv("BYBIT_TEST_PROXY"),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), time.Minute)
	err = api.SetAutoAddMargin(ctx, SetAutoAddMarginParam{
		Symbol: "BTCUSDT",
		Side: PositionSideBuy,
		AutoAddMargin: true,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestAPI_SwitchIsolated(t *testing.T) {
	api, err := NewAPI(
		os.Getenv("BYBIT_TEST_KEY"),
		os.Getenv("BYBIT_TEST_SECRET"),
		"",
		os.Getenv("BYBIT_TEST_PROXY"),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), time.Minute)
	err = api.SwitchIsolated(ctx, SwitchIsolatedParam{
		Symbol: "BTCUSDT",
		IsIsolated: false,
		BuyLeverage: 10,
		SellLeverage: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
}


func TestAPI_SetLeverage(t *testing.T) {
	api, err := NewAPI(
		os.Getenv("BYBIT_TEST_KEY"),
		os.Getenv("BYBIT_TEST_SECRET"),
		"",
		os.Getenv("BYBIT_TEST_PROXY"),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), time.Minute)
	err = api.SetLeverage(ctx, SetLeverageParam{
		Symbol: "BTCUSDT",
		BuyLeverage: 10,
		SellLeverage: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestAPI_GetPrevFundingRate(t *testing.T) {
	api, err := NewAPI(
		os.Getenv("BYBIT_TEST_KEY"),
		os.Getenv("BYBIT_TEST_SECRET"),
		"",
		os.Getenv("BYBIT_TEST_PROXY"),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), time.Minute)
	fr, err := api.GetPrevFundingRate(ctx, PrevFundingRateParam{
		Symbol: "ADAUSDT",
	})
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", fr)
}

func TestAPI_GetBalances(t *testing.T) {
	api, err := NewAPI(
		os.Getenv("BYBIT_TEST_KEY"),
		os.Getenv("BYBIT_TEST_SECRET"),
		"",
		os.Getenv("BYBIT_TEST_PROXY"),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), time.Minute)
	balance, err := api.GetBalance(ctx, BalanceParam{
		Coin: "USDT",
	})
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", balance)
	logger.Debugf("%f %f", balance.Equity, balance.UnrealisedPnl)
}

func TestAPI_PlaceOrder(t *testing.T) {
	api, err := NewAPI(
		os.Getenv("BYBIT_TEST_KEY"),
		os.Getenv("BYBIT_TEST_SECRET"),
		"https://api.bybitglobal.com",
		os.Getenv("BYBIT_TEST_PROXY"),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), time.Minute)
	ids, err := api.PlaceOrder(ctx, NewOrderParam{
		Symbol: "BTCUSDT",
		Qty: 0.001,
		Price: 31990,
		Side: OrderSideBuy,
		OrderType: OrderTypeLimit,
		TimeInForce: TimeInForceGoodTillCancel,
		ReduceOnly: false,
		CloseOnTrigger: false,
		OrderLinkID: fmt.Sprintf("%d%04d", time.Now().Unix(), rand.Intn(10000)),
	})
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", ids)
}

func TestAPI_CancelAllOrders(t *testing.T) {
	api, err := NewAPI(
		os.Getenv("BYBIT_TEST_KEY"),
		os.Getenv("BYBIT_TEST_SECRET"),
		"",
		os.Getenv("BYBIT_TEST_PROXY"),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), time.Minute)
	ids, err := api.CancelAllOrders(ctx, CancelAllParam{
		Symbol: "BTCUSDT",
	})
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", ids)
}

func TestAPI_GetServerTime(t *testing.T) {
	api, err := NewAPI(
		os.Getenv("BYBIT_TEST_KEY"),
		os.Getenv("BYBIT_TEST_SECRET"),
		"",
		os.Getenv("BYBIT_TEST_PROXY"),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), time.Minute)
	tt, err := api.GetServerTime(ctx)
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", tt)
}
