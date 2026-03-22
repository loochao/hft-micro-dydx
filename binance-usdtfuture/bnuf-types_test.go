package binance_usdtfuture

import (
	"github.com/geometrybase/hft-micro/common"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetExchange(t *testing.T) {
	var wsPosition common.Position = &WSPosition{}
	var position common.Position = &Position{}
	var fr common.FundingRate = &PremiumIndex{}
	var wsOrder common.Order = &WSOrder{}
	var order common.Order = &Order{}
	var balance common.Balance = &Asset{}
	var depth5 common.Depth = &Depth5{}
	var depth20 common.Depth = &Depth20{}
	var bookTicker common.Ticker = &BookTicker{}
	assert.Equal(t, common.BinanceUsdtFuture, fr.GetExchange())
	assert.Equal(t, common.BinanceUsdtFuture, wsPosition.GetExchange())
	assert.Equal(t, common.BinanceUsdtFuture, position.GetExchange())
	assert.Equal(t, common.BinanceUsdtFuture, order.GetExchange())
	assert.Equal(t, common.BinanceUsdtFuture, wsOrder.GetExchange())
	assert.Equal(t, common.BinanceUsdtFuture, balance.GetExchange())
	assert.Equal(t, common.BinanceUsdtFuture, depth5.GetExchange())
	assert.Equal(t, common.BinanceUsdtFuture, depth20.GetExchange())
	assert.Equal(t, common.BinanceUsdtFuture, bookTicker.GetExchange())
}
