package stream_stats

import (
	"encoding/json"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
	"time"
)

func TestNewXYSimplifiedTickerStats2(t *testing.T) {
	s, err := NewXYSimplifiedTickerStats2(NewXYSimplifiedTickerStats2Params{
		XSymbol:                     "BTCUSDT",
		YSymbol:                     "BTCUSDT",
		SampleInterval:              time.Second,
		TimeDeltaLookback:           time.Hour * 2,
		SpreadTDLookback:            time.Hour * 240,
		SpreadTDSubInterval:         time.Minute * 15,
		SpreadTDCompression:         10,
		SpreadLongEnterQuantileBot:  0.005,
		SpreadLongLeaveQuantileTop:  0.8,
		SpreadShortEnterQuantileTop: 0.995,
		SpreadShortLeaveQuantileBot: 0.2,
		BaseEnterOffset:             0.01,
		BaseLeaveOffset:             -0.01,
		XTimeDeltaOffsetTop:         time.Millisecond * 20,
		XTimeDeltaOffsetBot:         -time.Millisecond * 20,
		YTimeDeltaOffsetTop:         time.Millisecond * 20,
		YTimeDeltaOffsetBot:         -time.Millisecond * 20,
		XYTimeDeltaOffsetTop:        time.Millisecond * 100,
		XYTimeDeltaOffsetBot:        -time.Millisecond * 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	contents, err := ioutil.ReadFile("./jsons/xy-simplified-ticker-stats2.json")
	if err != nil {
		t.Fatal(err)
	}
	err = json.Unmarshal(contents, s)
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", s.SpreadTD)
	assert.NotNil(t, s.XTickerCh)
	assert.NotNil(t, s.YTickerCh)
	assert.NotNil(t, s.done)
	assert.NotNil(t, s.SpreadTD)
	assert.Equal(t, time.Hour*240, s.SpreadTD.Lookback)
	assert.Equal(t, time.Hour*120, s.SpreadTD.HalfLookback)
	assert.Equal(t, 0.1, s.XEventTimeDeltaMean)
}
