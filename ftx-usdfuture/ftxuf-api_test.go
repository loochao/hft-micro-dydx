package ftx_usdfuture

import (
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestAPI_GetFutures(t *testing.T) {
	api, err := NewAPI(os.Getenv("FTX_TEST_KEY"), os.Getenv("FTX_TEST_SECRET"), os.Getenv("FTX_TEST_PROXY"))
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
	for _, future := range futures {
		if future.Type == "perpetual" && future.Enabled{
			sizeIncrements[future.Name] = future.SizeIncrement
			priceIncrements[future.Name] = future.PriceIncrement
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
}

func TestAPI_GetFundingRates(t *testing.T) {
	api, err := NewAPI(os.Getenv("FTX_TEST_KEY"), os.Getenv("FTX_TEST_SECRET"), os.Getenv("FTX_TEST_PROXY"))
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
	api, err := NewAPI(os.Getenv("FTX_TEST_KEY"), os.Getenv("FTX_TEST_SECRET"), os.Getenv("FTX_TEST_PROXY"))
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
	api, err := NewAPI(os.Getenv("FTX_TEST_KEY"), os.Getenv("FTX_TEST_SECRET"), os.Getenv("FTX_TEST_PROXY"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), time.Minute)
	account, err := api.GetAccount(ctx)
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", account)
}


func TestAPI_GetPositions(t *testing.T) {
	api, err := NewAPI(os.Getenv("FTX_TEST_KEY"), os.Getenv("FTX_TEST_SECRET"), os.Getenv("FTX_TEST_PROXY"))
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
