package common

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type depthForTest struct {
	Symbol string
	Time   time.Time
	Bids   [5][2]float64
	Asks   [5][2]float64
}

func (dft *depthForTest) GetBids() Bids {
	return dft.Bids[:]
}

func (dft *depthForTest) GetAsks() Asks {
	return dft.Asks[:]
}

func (dft *depthForTest) GetSymbol() string {
	return dft.Symbol
}

func (dft *depthForTest) GetTime() time.Time {
	return dft.Time
}

var depth01 = &depthForTest{
	Asks: [5][2]float64{
		{11, 1},
		{13, 1},
		{15, 1},
		{20, 1},
		{30, 1},
	},
	Bids: [5][2]float64{
		{10, 1},
		{9, 1},
		{8, 1},
		{7, 1},
		{6, 1},
	},
	Symbol: "BTCUSDT",
	Time:   time.Now(),
}

var minFloatDelta = 0.00000001

func TestWalkDepth(t *testing.T) {
	wd := &WalkedDepthBAM{}
	err := WalkDepthWithMultiplier(depth01, 1, 1.0, wd)
	if err != nil {
		t.Fatal(err)
	}
	assert.InDelta(t, depth01.Asks[0][0], wd.AskPrice, minFloatDelta)
	assert.InDelta(t, depth01.Bids[0][0], wd.BidPrice, minFloatDelta)
	err = WalkDepthWithMultiplier(depth01, 0.1, 0.1, wd)
	if err != nil {
		t.Fatal(err)
	}
	assert.InDelta(t, depth01.Asks[0][0], wd.AskPrice, minFloatDelta)
	assert.InDelta(t, depth01.Bids[0][0], wd.BidPrice, minFloatDelta)
	err = WalkDepthWithMultiplier(depth01, 1, 11, wd)
	if err != nil {
		t.Fatal(err)
	}
	assert.InDelta(t, 11.0, wd.AskPrice, minFloatDelta)
	assert.InDelta(t, (10.0*1.0+1.0)/(1.0+1.0/9.0), wd.BidPrice, minFloatDelta)

	err = WalkDepthWithMultiplier(depth01, 0.01, 0.11, wd)
	if err != nil {
		t.Fatal(err)
	}
	assert.InDelta(t, 11, wd.AskPrice, minFloatDelta)
	assert.InDelta(t, (10.0*1.0+1.0)/(1.0+1.0/9.0), wd.BidPrice, minFloatDelta)

	err = WalkDepthWithMultiplier(depth01, 1, 12, wd)
	if err != nil {
		t.Fatal(err)
	}
	assert.InDelta(t, (11.0+1)/(1.0+1.0/13.0), wd.AskPrice, minFloatDelta)
	assert.InDelta(t, (10.0*1.0+2.0)/(1.0+2.0/9.0), wd.BidPrice, minFloatDelta)
	err = WalkDepthWithMultiplier(depth01, 1, 25, wd)
	if err != nil {
		t.Fatal(err)
	}
	assert.InDelta(t, (11.0+13.0+1)/(1.0+1.0+1.0/15.0), wd.AskPrice, minFloatDelta)
	assert.InDelta(t, (10.0+9.0+6.0)/(1.0+1.0+6.0/8.0), wd.BidPrice, minFloatDelta)
}

func BenchmarkWalkDepth(b *testing.B) {
	wd := &WalkedDepthBAM{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = WalkDepthWithMultiplier(depth01, 0.001, 1000000.0, wd)
	}
}
