package kucoin_usdtfuture

import (
	"github.com/geometrybase/hft-micro/common"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetExchange(t *testing.T) {
	var position common.Position = &Position{}
	var fr common.FundingRate = &CurrentFundingRate{}
	var wsOrder common.Order = &WSOrder{}
	var balance common.Balance = &Account{}
	var depth5 common.Depth = &Depth5{}
	assert.Equal(t, common.KucoinUsdtFuture, fr.GetExchange())
	assert.Equal(t, common.KucoinUsdtFuture, wsOrder.GetExchange())
	assert.Equal(t, common.KucoinUsdtFuture, position.GetExchange())
	assert.Equal(t, common.KucoinUsdtFuture, depth5.GetExchange())
	assert.Equal(t, common.KucoinUsdtFuture, balance.GetExchange())
}
