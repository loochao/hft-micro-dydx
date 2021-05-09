package ftxperp

import (
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/logger"
	"os"
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
			priceIncrements[future.Name] = future.SizeIncrement
		}
	}
	fmt.Printf("var SizeIncrements = map[string]float64{\n")
	for name, value := range sizeIncrements {
		fmt.Printf("  \"%s\":%f,\n", name, value)
	}
	fmt.Printf("}\n\n")
	fmt.Printf("var PriceIncrements = map[string]float64{\n")
	for name, value := range priceIncrements {
		fmt.Printf("  \"%s\":%f,\n", name, value)
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
