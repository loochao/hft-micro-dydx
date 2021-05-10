package common

import (
	"github.com/geometrybase/hft-micro/logger"
	"github.com/stretchr/testify/assert"
	"sort"
	"testing"
)

func TestAsks(t *testing.T) {
	asks := Asks{
		[2]float64{100, 20},
		[2]float64{500, 500},
		[2]float64{300, 300},
		[2]float64{110, 110},
		[2]float64{1000, 1000},
	}
	sort.Sort(asks)
	logger.Debugf("%f", asks)
	assert.True(t, sort.IsSorted(asks), "asks should be sorted")
	asks = asks.Update([2]float64{100,100})
	assert.Equal(t, 5, len(asks))
	assert.Equal(t, 100.0, asks[0][0])
	assert.Equal(t, 100.0, asks[0][1])
	assert.True(t, sort.IsSorted(asks), "asks should be sorted")
	asks = asks.Update([2]float64{101,101})
	assert.Equal(t, 6, len(asks))
	assert.Equal(t, 100.0, asks[0][0])
	assert.Equal(t, 100.0, asks[0][1])
	assert.Equal(t, 101.0, asks[1][0])
	assert.Equal(t, 101.0, asks[1][1])
	assert.True(t, sort.IsSorted(asks), "asks should be sorted")
	asks = asks.Update([2]float64{101,0})
	assert.Equal(t, 5, len(asks))
	assert.Equal(t, 100.0, asks[0][0])
	assert.Equal(t, 100.0, asks[0][1])
	assert.Equal(t, 110.0, asks[1][0])
	assert.Equal(t, 110.0, asks[1][1])
	assert.True(t, sort.IsSorted(asks), "asks should be sorted")
	asks = asks.Update([2]float64{110,0})
	assert.Equal(t, 4, len(asks))
	assert.Equal(t, 100.0, asks[0][0])
	assert.Equal(t, 100.0, asks[0][1])
	assert.Equal(t, 300.0, asks[1][0])
	assert.Equal(t, 300.0, asks[1][1])
	assert.True(t, sort.IsSorted(asks), "asks should be sorted")
	asks = asks.Update([2]float64{300,0})
	assert.Equal(t, 3, len(asks))
	assert.Equal(t, 100.0, asks[0][0])
	assert.Equal(t, 100.0, asks[0][1])
	assert.Equal(t, 500.0, asks[1][0])
	assert.Equal(t, 500.0, asks[1][1])
	assert.True(t, sort.IsSorted(asks), "asks should be sorted")
	asks = asks.Update([2]float64{500,0})
	assert.Equal(t, 2, len(asks))
	assert.Equal(t, 100.0, asks[0][0])
	assert.Equal(t, 100.0, asks[0][1])
	assert.Equal(t, 1000.0, asks[1][0])
	assert.Equal(t, 1000.0, asks[1][1])
	assert.True(t, sort.IsSorted(asks), "asks should be sorted")
	asks = asks.Update([2]float64{1000,0})
	assert.Equal(t, 1, len(asks))
	assert.Equal(t, 100.0, asks[0][0])
	assert.Equal(t, 100.0, asks[0][1])
	assert.True(t, sort.IsSorted(asks), "asks should be sorted")
	asks = asks.Update([2]float64{100,0})
	assert.Equal(t, 0, len(asks))
	assert.True(t, sort.IsSorted(asks), "asks should be sorted")
}

func TestBids(t *testing.T) {
	bids := Bids{
		[2]float64{100, 20},
		[2]float64{500, 500},
		[2]float64{300, 300},
		[2]float64{110, 110},
		[2]float64{1000, 1000},
	}
	sort.Sort(bids)
	assert.True(t, sort.IsSorted(bids), "bids should be sorted")
	bids = bids.Update([2]float64{100,100})
	assert.Equal(t, 5, len(bids))
	assert.Equal(t, 100.0, bids[4][0])
	assert.Equal(t, 100.0, bids[4][1])
	assert.True(t, sort.IsSorted(bids), "bids should be sorted")
	bids = bids.Update([2]float64{101,101})
	assert.Equal(t, 6, len(bids))
	assert.Equal(t, 100.0, bids[5][0])
	assert.Equal(t, 100.0, bids[5][1])
	assert.Equal(t, 101.0, bids[4][0])
	assert.Equal(t, 101.0, bids[4][1])
	assert.True(t, sort.IsSorted(bids), "bids should be sorted")
	bids = bids.Update([2]float64{101,0})
	assert.Equal(t, 5, len(bids))
	assert.Equal(t, 100.0, bids[4][0])
	assert.Equal(t, 100.0, bids[4][1])
	assert.Equal(t, 110.0, bids[3][0])
	assert.Equal(t, 110.0, bids[3][1])
	assert.True(t, sort.IsSorted(bids), "bids should be sorted")
	bids = bids.Update([2]float64{110,0})
	assert.Equal(t, 4, len(bids))
	assert.Equal(t, 100.0, bids[3][0])
	assert.Equal(t, 100.0, bids[3][1])
	assert.Equal(t, 300.0, bids[2][0])
	assert.Equal(t, 300.0, bids[2][1])
	assert.True(t, sort.IsSorted(bids), "bids should be sorted")
	bids = bids.Update([2]float64{300,0})
	assert.Equal(t, 3, len(bids))
	assert.Equal(t, 100.0, bids[2][0])
	assert.Equal(t, 100.0, bids[2][1])
	assert.Equal(t, 500.0, bids[1][0])
	assert.Equal(t, 500.0, bids[1][1])
	assert.True(t, sort.IsSorted(bids), "bids should be sorted")
	bids = bids.Update([2]float64{500,0})
	assert.Equal(t, 2, len(bids))
	assert.Equal(t, 100.0, bids[1][0])
	assert.Equal(t, 100.0, bids[1][1])
	assert.Equal(t, 1000.0, bids[0][0])
	assert.Equal(t, 1000.0, bids[0][1])
	assert.True(t, sort.IsSorted(bids), "bids should be sorted")
	bids = bids.Update([2]float64{1000,0})
	assert.Equal(t, 1, len(bids))
	assert.Equal(t, 100.0, bids[0][0])
	assert.Equal(t, 100.0, bids[0][1])
	assert.True(t, sort.IsSorted(bids), "bids should be sorted")
	bids = bids.Update([2]float64{100,0})
	assert.Equal(t, 0, len(bids))
	assert.True(t, sort.IsSorted(bids), "bids should be sorted")
}

