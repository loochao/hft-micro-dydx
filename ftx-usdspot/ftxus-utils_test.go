package ftx_usdspot

import (
	"encoding/json"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
	"time"
)

func TestParseTicker(t *testing.T) {
	var float64pow10 = []float64{
		1e0, 1e1, 1e2, 1e3, 1e4, 1e5, 1e6, 1e7, 1e8, 1e9,
		1e10, 1e11, 1e12, 1e13, 1e14, 1e15, 1e16, 1e17, 1e18, 1e19,
		1e20, 1e21, 1e22,
	}
	str := []byte(`{"channel": "ticker", "market": "DOGE-PERP", "type": "update", "data": {"bid": 0.278362, "ask": 0.2784135, "bidSize": 107.0, "askSize": 5600.0, "last": 0.2783695, "time": 1624183024.08771}}`)
	ticker := Ticker{}
	err := ParseTicker(str, &ticker)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 0.278362, ticker.Bid)
	assert.Equal(t, 0.2784135, ticker.Ask)
	assert.Equal(t, 107.0, ticker.BidSize)
	assert.Equal(t, 5600.0, ticker.AskSize)
	assert.Equal(t, int64(1624183024087), ticker.Time.UnixNano()/1000000)
	logger.Debugf("%v %d", ticker.Time, int64(1624183024.08771*1000000000))
	logger.Debugf("%f", 1624183024.08771)
	f, err := common.ParseDecimal([]byte("1624183024.08771"))
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%f", f)
	logger.Debugf("%v", time.Unix(0, 1624183024087710000))
	logger.Debugf("%v", time.Unix(0, 1624183024087710001))
	logger.Debugf("%v", time.Unix(0, 1624183024087710002))
	v := uint(0)
	v = 162418302408771
	logger.Debugf("%s", strconv.FormatFloat(float64(v)/100000.0, 'f', -1, 64))
	logger.Debugf("%s", strconv.FormatFloat(float64(v)/float64pow10[5], 'f', -1, 64))
}


var GlobalTicker *Ticker
func BenchmarkParseTicker(b *testing.B) {
	str := []byte(`{"channel": "ticker", "market": "DOGE-PERP", "type": "update", "data": {"bid": 0.278362, "ask": 0.2784135, "bidSize": 107.0, "askSize": 5600.0, "last": 0.2783695, "time": 1624183024.08771}}`)
	ticker := Ticker{}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		 _ = ParseTicker(str, &ticker)
	}
	GlobalTicker = &ticker
}

var GlobalTickerData *TickerData
func BenchmarkParseTickerByStdJson(b *testing.B) {
	str := []byte(`{"channel": "ticker", "market": "DOGE-PERP", "type": "update", "data": {"bid": 0.278362, "ask": 0.2784135, "bidSize": 107.0, "askSize": 5600.0, "last": 0.2783695, "time": 1624183024.08771}}`)
	ticker := TickerData{}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = json.Unmarshal(str, &ticker)
	}
	GlobalTickerData = &ticker
}