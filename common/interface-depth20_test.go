package common

import (
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
	"time"
)

type TestDepth20 struct {
	Bids      [20][2]float64
	Asks      [20][2]float64
	Symbol    string
	EventTime time.Time
}

func (t TestDepth20) GetBids() [20][2]float64 {
	return t.Bids
}
func (t TestDepth20) GetAsks() [20][2]float64 {
	return t.Asks
}
func (t TestDepth20) GetTime() time.Time {
	return t.EventTime
}
func (t TestDepth20) GetSymbol() string {
	return t.Symbol
}

func TestWalkMakerTakerDepth20(t *testing.T) {
	depth20 := TestDepth20{
		Symbol:    "BTCUSDT",
		EventTime: time.Now(),
	}
	for i := 0; i < 20; i++ {
		depth20.Bids[i][0] = float64(20 - i)
		depth20.Bids[i][1] = 1
		depth20.Asks[i][0] = float64(21 + i)
		depth20.Asks[i][1] = 1
	}

	wd := WalkMakerTakerDepth20(depth20, 21, 21)

	assert.Equal(t, 20./20.+1./19., wd.MakerBidSize)
	assert.Equal(t, math.Floor(21./(20./20.+1./19.)*1000), math.Floor(wd.MakerBid*1000))
	assert.Equal(t, 19., wd.MakerFarBid)

	assert.Equal(t, 20./20.+1./19., wd.TakerBidSize)
	assert.Equal(t, math.Floor(21./(20./20.+1./19.)), math.Floor(wd.TakerBid))
	assert.Equal(t, 19., wd.TakerFarBid)

	assert.Equal(t, 1., wd.MakerAskSize)
	assert.Equal(t, math.Floor(21.*1000), math.Floor(wd.MakerAsk*1000))
	assert.Equal(t, 21., wd.MakerFarAsk)

	assert.Equal(t, 1., wd.TakerAskSize)
	assert.Equal(t, math.Floor(21.*1000), math.Floor(wd.TakerAsk*1000))
	assert.Equal(t, 21., wd.TakerFarAsk)

	wd = WalkMakerTakerDepth20(depth20, 40, 20)
	assert.Equal(t, 20./20.+19./19.+1./18., wd.MakerBidSize)
	assert.Equal(t, math.Floor(40./(20./20.+19./19.+1./18.)*1000), math.Floor(wd.MakerBid*1000))
	assert.Equal(t, 18., wd.MakerFarBid)

	assert.Equal(t, 20./20., wd.TakerBidSize)
	assert.Equal(t, 20., wd.TakerBid)
	assert.Equal(t, 20., wd.TakerFarBid)

	wd = WalkMakerTakerDepth20(depth20, 1., 2000000)
	assert.Equal(t, 1./20.0, wd.MakerBidSize)
	assert.Equal(t, 20., wd.MakerBid)

	assert.Equal(t, 20., wd.TakerBidSize)
	assert.Equal(t, (20.+1.)/2., wd.TakerBid)
	assert.Equal(t, 1., wd.TakerFarBid)
}
