package binance_usdcspot

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
	assert.Equal(t, common.BinanceUsdcSpot, fr.GetExchange())
	assert.Equal(t, common.BinanceUsdcSpot, order.GetExchange())
	assert.Equal(t, common.BinanceUsdcSpot, wsOrder.GetExchange())
	assert.Equal(t, common.BinanceUsdcSpot, balance.GetExchange())
	assert.Equal(t, common.BinanceUsdcSpot, depth5.GetExchange())
	assert.Equal(t, common.BinanceUsdcSpot, depth20.GetExchange())
}
