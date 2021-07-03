package binance_usdtspot

import (
	"github.com/geometrybase/hft-micro/common"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetExchange(t *testing.T) {
	var fr common.FundingRate = &FundingRate{}
	var wsOrder common.Order = &OrderUpdateEvent{}
	var order common.Order = &NewOrderResponse{}
	var balance common.Balance = &Balance{}
	var depth5 common.Depth = &Depth5{}
	var depth20 common.Depth = &Depth20{}
	assert.Equal(t, common.BinanceUsdtSpot, fr.GetExchange())
	assert.Equal(t, common.BinanceUsdtSpot, order.GetExchange())
	assert.Equal(t, common.BinanceUsdtSpot, wsOrder.GetExchange())
	assert.Equal(t, common.BinanceUsdtSpot, balance.GetExchange())
	assert.Equal(t, common.BinanceUsdtSpot, depth5.GetExchange())
	assert.Equal(t, common.BinanceUsdtSpot, depth20.GetExchange())
}
