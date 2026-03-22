package gate_usdtfuture

import (
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/logger"
	"strconv"
	"testing"
)

func TestAPI_GetProducts(t *testing.T) {
	api, err := NewAPI("socks5://127.0.0.1:1083")
	if err != nil {
		t.Fatal(err)
	}
	products, err := api.GetProducts(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	tickSizes := make(map[string]float64)
	stepSizes := make(map[string]float64)
	minSizes := make(map[string]float64)
	for _, p := range products {
		tickSizes[p.Name] = p.OrderPriceRound
		stepSizes[p.Name] = p.OrderSizeMin
		minSizes[p.Name] = p.OrderSizeMin
	}
	logger.Debugf("%v", tickSizes)
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
	fmt.Printf("%s", str)
}
