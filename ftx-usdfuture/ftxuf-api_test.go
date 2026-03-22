package ftx_usdfuture

import (
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestAPI_GetFutures(t *testing.T) {
	api, err := NewAPI(os.Getenv("FTX_TEST_KEY"), os.Getenv("FTX_TEST_SECRET"), os.Getenv("FTX_TEST_PROXY"), "")
	if err != nil {
		t.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), time.Minute)
	futures, err := api.GetFutures(ctx)
	if err != nil {
		t.Fatal(err)
	}
	sizeIncrements := make(map[string]float64)
	priceIncrements := make(map[string]float64)
	tickPrecisions := make(map[string]int)
	stepPrecisions := make(map[string]int)
	for _, future := range futures {
		if future.Type == "perpetual" && future.Enabled{
			sizeIncrements[future.Name] = future.SizeIncrement
			stepPrecisions[future.Name] = common.GetFloatPrecision(future.SizeIncrement)
			priceIncrements[future.Name] = future.PriceIncrement
			tickPrecisions[future.Name] = common.GetFloatPrecision(future.PriceIncrement)
		}
	}
	fmt.Printf("var SizeIncrements = map[string]float64{\n")
	for name, value := range sizeIncrements {
		fmt.Printf("  \"%s\":%s,\n", name,  strconv.FormatFloat(value, 'f', -1, 64))
	}
	fmt.Printf("}\n\n")
	fmt.Printf("var PriceIncrements = map[string]float64{\n")
	for name, value := range priceIncrements {
		fmt.Printf("  \"%s\":%s,\n", name, strconv.FormatFloat(value, 'f', -1, 64))
	}
	fmt.Printf("}\n\n")
	fmt.Printf("var TickPrecisions = map[string]float64{\n")
	for name, value := range tickPrecisions {
		fmt.Printf("  \"%s\":%d,\n", name, value)
	}
	fmt.Printf("}\n\n")
	fmt.Printf("var StepPrecisions = map[string]float64{\n")
	for name, value := range stepPrecisions {
		fmt.Printf("  \"%s\":%d,\n", name, value)
	}
	fmt.Printf("}\n\n")
}

func TestAPI_GetFundingRates(t *testing.T) {
	api, err := NewAPI(os.Getenv("FTX_TEST_KEY"), os.Getenv("FTX_TEST_SECRET"), os.Getenv("FTX_TEST_PROXY"), "")
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
	api, err := NewAPI(os.Getenv("FTX_TEST_KEY"), os.Getenv("FTX_TEST_SECRET"), os.Getenv("FTX_TEST_PROXY"), "")
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

	//api, err := NewAPI(os.Getenv("FTX_TEST_KEY"), os.Getenv("FTX_TEST_SECRET"), os.Getenv("FTX_TEST_PROXY"), "")
	api, err := NewAPI(
		"4Rd-hge6CBCZNKRPb71oYlQeBaM4Osbrqf_hDeEp",
		"sAXPBuuNE47gE5AyNjRRoznp57Or9s7liogIOaYN",
		"ff01", "socks5://127.0.0.1:1084")
	if err != nil {
		t.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), time.Minute)
	account, err := api.GetAccount(ctx)
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", account)
	logger.Debugf("TA %f CU %f CF %f C %f", account.TotalAccountValue, account.CollateralUsed, account.FreeCollateral, account.Collateral)
}


func TestAPI_GetPositions(t *testing.T) {
	api, err := NewAPI(os.Getenv("FTX_TEST_KEY"), os.Getenv("FTX_TEST_SECRET"), os.Getenv("FTX_TEST_PROXY"), "")
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
