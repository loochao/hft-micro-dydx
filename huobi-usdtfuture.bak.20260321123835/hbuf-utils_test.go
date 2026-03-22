package huobi_usdtfuture

import (
	"encoding/json"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseDepth20(t *testing.T) {
	bytes := []byte(`{"ch":"market.BTC-USDT.depth.step6","ts":1618410970115,"tick":{"mrid":28158325357,"id":1618410970,"bids":[[63402.5,88],[63402.2,6],[63402.1,42],[63401.4,50],[63400.7,24],[63400,238],[63398.9,1],[63398.6,31],[63398.5,300],[63397.3,39],[63397.1,200],[63397,115],[63396.3,51],[63394.6,200],[63393.2,1000],[63392.5,1],[63392,177],[63391.6,115],[63391.5,115],[63391.4,115]],"asks":[[63402.6,20318],[63402.8,46],[63405,1583],[63405.2,300],[63406.7,108],[63406.8,484],[63406.9,325],[63407,58],[63407.1,1120],[63407.2,16590],[63407.3,1016],[63407.4,797],[63407.5,270],[63407.6,753],[63407.7,1178],[63407.8,521],[63407.9,330],[63408,170],[63408.1,1064],[63408.2,606]],"ts":1618410970112,"version":1618410970,"ch":"market.BTC-USDT.depth.step6"}}`)
	parseD := &Depth20{}
	err := ParseDepth20(bytes, parseD)
	if err != nil {
		t.Fatal(err)
	}
	wsCap := WsDepthCap{}
	err = json.Unmarshal(bytes, &wsCap)
	if err != nil {
		t.Fatal(err)
	}
	jsonD := Depth20{}
	err = json.Unmarshal(wsCap.Tick, &jsonD)
	if err != nil {
		t.Fatal(err)
	}
	for i, b := range parseD.Bids {
		assert.Equal(t, jsonD.Bids[i][0], b[0])
		assert.Equal(t, jsonD.Bids[i][1], b[1])
	}
	for i, a := range parseD.Asks {
		assert.Equal(t, jsonD.Asks[i][0], a[0])
		assert.Equal(t, jsonD.Asks[i][1], a[1])
	}
	assert.Equal(t, jsonD.Symbol, parseD.Symbol)
	assert.Equal(t, jsonD.EventTime, parseD.EventTime)
	assert.Equal(t, jsonD.ID, parseD.ID)
	assert.Equal(t, jsonD.MRID, parseD.MRID)
	assert.Equal(t, jsonD.Version, parseD.Version)
}

func BenchmarkParseDepth50(t *testing.B) {
	bytes := []byte(`{"ch":"market.BTC-USDT.depth.step6","ts":1618410970115,"tick":{"mrid":28158325357,"id":1618410970,"bids":[[63402.5,88],[63402.2,6],[63402.1,42],[63401.4,50],[63400.7,24],[63400,238],[63398.9,1],[63398.6,31],[63398.5,300],[63397.3,39],[63397.1,200],[63397,115],[63396.3,51],[63394.6,200],[63393.2,1000],[63392.5,1],[63392,177],[63391.6,115],[63391.5,115],[63391.4,115]],"asks":[[63402.6,20318],[63402.8,46],[63405,1583],[63405.2,300],[63406.7,108],[63406.8,484],[63406.9,325],[63407,58],[63407.1,1120],[63407.2,16590],[63407.3,1016],[63407.4,797],[63407.5,270],[63407.6,753],[63407.7,1178],[63407.8,521],[63407.9,330],[63408,170],[63408.1,1064],[63408.2,606]],"ts":1618410970112,"version":1618410970,"ch":"market.BTC-USDT.depth.step6"}}`)
	parseD := &Depth20{}
	t.ReportAllocs()
	for i := 0; i < t.N; i++ {
		_ = ParseDepth20(bytes, parseD)
	}
}

func TestParseTicker(t *testing.T) {
	bytes := []byte(`{"ch":"market.1INCH-USDT.bbo","ts":1626480000472,"tick":{"mrid":13218587529,"id":1626480000,"bid":[1.9631,10],"ask":[1.9641,38],"ts":1626480000472,"version":13218587529,"ch":"market.1INCH-USDT.bbo"}}`)
	ticker1 := &Ticker{}
	err := ParseTicker(bytes, ticker1)
	if err != nil {
		t.Fatal(err)
	}
	tickerData := TickerData{}
	err = json.Unmarshal(bytes, &tickerData)
	if err != nil {
		t.Fatal(err)
	}
	ticker2 := tickerData.Tick
	logger.Debugf("%v %v", ticker1.EventTime, ticker2.EventTime)
	assert.Equal(t, ticker2.Symbol, ticker1.Symbol)
	assert.Equal(t, 0.0, ticker2.EventTime.Sub(ticker1.EventTime).Seconds())
	assert.Equal(t, ticker2.Bid[0], ticker1.Bid[0])
	assert.Equal(t, ticker2.Bid[1], ticker1.Bid[1])
	assert.Equal(t, ticker2.Ask[0], ticker1.Ask[0])
	assert.Equal(t, ticker2.Ask[1], ticker1.Ask[1])
}

func BenchmarkParseTicker(t *testing.B) {
	msg := []byte(`{"ch":"market.1INCH-USDT.bbo","ts":1626480000472,"tick":{"mrid":13218587529,"id":1626480000,"bid":[1.9631,10],"ask":[1.9641,38],"ts":1626480000472,"version":13218587529,"ch":"market.1INCH-USDT.bbo"}}`)
	ticker := &Ticker{}
	t.ReportAllocs()
	for i := 0; i < t.N; i++ {
		_ = ParseTicker(msg, ticker)
	}
}