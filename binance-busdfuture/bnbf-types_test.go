package binance_busdfuture

import (
	"github.com/geometrybase/hft-micro/common"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetExchange(t *testing.T) {
	var fr common.FundingRate = &PremiumIndex{}
	var order common.Order = &Order{}
	var wsOrder common.Order = &WSOrder{}
	var position common.Position = &Position{}
	var wsPosition common.Position = &WSPosition{}
	var balance common.Balance = &Asset{}
	var depth5 common.Depth = &Depth5{}
	var depth20 common.Depth = &Depth20{}
	assert.Equal(t, ExchangeID, fr.GetExchange())
	assert.Equal(t, ExchangeID, order.GetExchange())
	assert.Equal(t, ExchangeID, wsOrder.GetExchange())
	assert.Equal(t, ExchangeID, position.GetExchange())
	assert.Equal(t, ExchangeID, wsPosition.GetExchange())
	assert.Equal(t, ExchangeID, balance.GetExchange())
	assert.Equal(t, ExchangeID, depth5.GetExchange())
	assert.Equal(t, ExchangeID, depth20.GetExchange())
}
