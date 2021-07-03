package binance_coinfuture

import (
	"github.com/geometrybase/hft-micro/common"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetExchange(t *testing.T) {
	var fr common.FundingRate = &PremiumIndex{}
	var wsOrder common.Order = &WSOrder{}
	var order common.Order = &Order{}
	var wsBalance common.Balance = &WSBalance{}
	var wsPosition common.Position = &WSPositionUpdate{}
	var wsPosition2 common.Position = &WSPosition{}
	var depth5 common.Depth = &Depth5{}
	var depth20 common.Depth = &Depth20{}
	assert.Equal(t, common.BinanceCoinFuture, fr.GetExchange())
	assert.Equal(t, common.BinanceCoinFuture, order.GetExchange())
	assert.Equal(t, common.BinanceCoinFuture, wsOrder.GetExchange())
	assert.Equal(t, common.BinanceCoinFuture, wsBalance.GetExchange())
	assert.Equal(t, common.BinanceCoinFuture, wsPosition.GetExchange())
	assert.Equal(t, common.BinanceCoinFuture, wsPosition2.GetExchange())
	assert.Equal(t, common.BinanceCoinFuture, depth5.GetExchange())
	assert.Equal(t, common.BinanceCoinFuture, depth20.GetExchange())
}

