package coinbase_usdspot

import (
	"context"
	"github.com/geometrybase/hft-micro/logger"
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
	for _, p := range products {
		if p.QuoteCurrency == "USD" {
			tickSizes[p.ID] = p.QuoteIncrement
		}
	}
	logger.Debugf("%v", tickSizes)
}
