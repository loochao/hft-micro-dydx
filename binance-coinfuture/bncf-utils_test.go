package binance_coinfuture

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var GlobalDepth20 *Depth20
var GlobalDepth5 *Depth5
var depth20Msg = []byte(`{"stream":"bnbusd_perp@depth20@100ms","data":{"e":"depthUpdate","E":1623294719683,"T":1623294719651,"s":"BNBUSD_PERP","ps":"BNBUSD","U":137362346007,"u":137362346725,"pu":137362345003,"b":[["373.743","691"],["373.700","11209"],["373.643","300"],["373.642","500"],["373.629","180"],["373.628","200"],["373.621","4"],["373.611","7"],["373.610","3"],["373.608","750"],["373.603","1000"],["373.601","300"],["373.598","1"],["373.587","400"],["373.580","375"],["373.563","207"],["373.561","2176"],["373.556","300"],["373.536","1125"],["373.535","4"]],"a":[["373.744","843"],["373.763","210"],["373.764","200"],["373.765","200"],["373.781","1"],["373.797","115"],["373.812","300"],["373.823","34"],["373.825","310"],["373.826","410"],["373.844","180"],["373.845","455"],["373.846","750"],["373.857","21"],["373.858","500"],["373.866","50"],["373.878","619"],["373.879","1125"],["373.886","293"],["373.907","400"]]}}`)
var depth5Msg = []byte(`{"stream":"bnbusd_perp@depth5@100ms","data":{"e":"depthUpdate","E":1623297648173,"T":1623297648166,"s":"BNBUSD_PERP","ps":"BNBUSD","U":137388060548,"u":137388063414,"pu":137388059926,"b":[["369.073","1564"],["369.034","6"],["369.033","115"],["369.031","34"],["369.017","400"]],"a":[["369.074","375"],["369.137","79"],["369.138","115"],["369.141","34"],["369.145","246"]]}}`)

func TestParseDepth20(t *testing.T) {
	depth20 := &Depth20{}
	err := ParseDepth20(depth20Msg, depth20)
	if err != nil {
		t.Fatal(err)
	}
	wsCap := WSCap{}
	err = json.Unmarshal(depth20Msg, &wsCap)
	if err != nil {
		t.Fatal(err)
	}
	jsonDepth20 := Depth20{}
	err = json.Unmarshal(wsCap.Data, &jsonDepth20)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, jsonDepth20.Symbol, depth20.Symbol)
	assert.Equal(t, jsonDepth20.LastUpdateId, depth20.LastUpdateId)
	assert.Equal(t, time.Duration(0), depth20.EventTime.Sub(jsonDepth20.EventTime))
	for i := range jsonDepth20.Bids {
		assert.Equal(t, jsonDepth20.Bids[i][0], depth20.Bids[i][0])
		assert.Equal(t, jsonDepth20.Bids[i][1], depth20.Bids[i][1])
	}
	for i := range jsonDepth20.Asks {
		assert.Equal(t, jsonDepth20.Asks[i][0], depth20.Asks[i][0])
		assert.Equal(t, jsonDepth20.Asks[i][1], depth20.Asks[i][1])
	}
}

func BenchmarkParseDepth20(b *testing.B) {
	depth20 := &Depth20{}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		 _ = ParseDepth20(depth20Msg, depth20)
	}
	GlobalDepth20 = depth20
}

func BenchmarkParseDepth20ByStdJson(b *testing.B) {
	wsCap := WSCap{}
	depth20 := Depth20{}
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		_ = json.Unmarshal(depth20Msg, &wsCap)
		_ = json.Unmarshal(wsCap.Data, &depth20)
	}
	GlobalDepth20 = &depth20
}

func TestParseDepth5(t *testing.T) {
	depth5 := &Depth5{}
	err := ParseDepth5(depth5Msg, depth5)
	if err != nil {
		t.Fatal(err)
	}
	wsCap := WSCap{}
	err = json.Unmarshal(depth5Msg, &wsCap)
	if err != nil {
		t.Fatal(err)
	}
	jsonDepth5 := Depth5{}
	err = json.Unmarshal(wsCap.Data, &jsonDepth5)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, jsonDepth5.Symbol, depth5.Symbol)
	assert.Equal(t, jsonDepth5.LastUpdateId, depth5.LastUpdateId)
	assert.Equal(t, time.Duration(0), depth5.EventTime.Sub(jsonDepth5.EventTime))
	for i := range jsonDepth5.Bids {
		assert.Equal(t, jsonDepth5.Bids[i][0], depth5.Bids[i][0])
		assert.Equal(t, jsonDepth5.Bids[i][1], depth5.Bids[i][1])
	}
	for i := range jsonDepth5.Asks {
		assert.Equal(t, jsonDepth5.Asks[i][0], depth5.Asks[i][0])
		assert.Equal(t, jsonDepth5.Asks[i][1], depth5.Asks[i][1])
	}
}

func BenchmarkParseDepth5(b *testing.B) {
	depth5 := &Depth5{}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		 _ = ParseDepth5(depth5Msg, depth5)
	}
	GlobalDepth5 = depth5
}

func BenchmarkParseDepth5ByStdJson(b *testing.B) {
	wsCap := WSCap{}
	depth5 := Depth5{}
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		_ = json.Unmarshal(depth5Msg, &wsCap)
		_ = json.Unmarshal(wsCap.Data, &depth5)
	}
	GlobalDepth5 = &depth5
}
