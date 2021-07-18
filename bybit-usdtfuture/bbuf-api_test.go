package bybit_usdtfuture

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"
)

func TestAPI_GetSymbols(t *testing.T) {
	api, err := NewAPI("", "", "socks5://127.0.0.1:1083")
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
