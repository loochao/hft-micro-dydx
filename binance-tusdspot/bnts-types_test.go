package binance_tusdspot

import (
	"github.com/geometrybase/hft-micro/common"
	"github.com/stretchr/testify/assert"
	"testing"
)


func TestGetExchange(t *testing.T) {
	var fr common.FundingRate = &FundingRate{}
	var wsOrder common.Order = &OrderUpdateEvent{}
	var balance common.Balance = &Balance{}
	var depth5 common.Depth = &Depth5{}
	var depth20 common.Depth = &Depth20{}
	assert.Equal(t, common.BinanceTusdSpot, fr.GetExchange())
	assert.Equal(t, common.BinanceTusdSpot, wsOrder.GetExchange())
	assert.Equal(t, common.BinanceTusdSpot, balance.GetExchange())
	assert.Equal(t, common.BinanceTusdSpot, depth5.GetExchange())
	assert.Equal(t, common.BinanceTusdSpot, depth20.GetExchange())
}