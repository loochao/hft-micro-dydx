package stream_stats

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"path"
	"sync/atomic"
	"time"
)

type XYSimplifiedTickerStats struct {
	SpreadTD *TimedTDigest `json:"spreadTD,omitempty"`

	XEventTimeDeltaMean  float64 `json:"xEventTimeDeltaMean"`
	YEventTimeDeltaMean  float64 `json:"yEventTimeDeltaMean"`
	XYEventTimeDeltaMean float64 `json:"xyEventTimeDeltaMean"`

	XParseTimeDeltaMean float64 `json:"xParseTimeDeltaMean"`
	YParseTimeDeltaMean float64 `json:"yParseTimeDeltaMean"`

	sampleInterval time.Duration
	saveInterval   time.Duration
	spreadTDPath   string

	timedDeltaK float64

	xEventTimeDelta  time.Duration
	yEventTimeDelta  time.Duration
	xyEventTimeDelta time.Duration

	xParseTimeDelta time.Duration
	yParseTimeDelta time.Duration

	xEventTime  time.Time
	yEventTime  time.Time
	xyEventTime time.Time

	spread float64

	yTicker common.Ticker
	xTicker common.Ticker

	XTickerCh chan common.Ticker `json:"-"`
	YTickerCh chan common.Ticker `json:"-"`

	spreadLongEnterQuantileBot  float64
	spreadLongLeaveQuantileTop  float64
	spreadShortEnterQuantileTop float64
	spreadShortLeaveQuantileBot float64
	baseEnterOffset             float64
	baseLeaveOffset             float64

	Ready bool `json:"-"`

	XParseTimeDeltaMid  time.Duration `json:"-"`
	YParseTimeDeltaMid  time.Duration `json:"-"`
	XYParseTimeDeltaMid time.Duration `json:"-"`

	XEventTimeDeltaMid  time.Duration `json:"-"`
	YEventTimeDeltaMid  time.Duration `json:"-"`
	XYEventTimeDeltaMid time.Duration `json:"-"`
	XEventTimeDeltaTop  time.Duration `json:"-"`
	XEventTimeDeltaBot  time.Duration `json:"-"`
	YEventTimeDeltaTop  time.Duration `json:"-"`
	YEventTimeDeltaBot  time.Duration `json:"-"`
	XYEventTimeDeltaTop time.Duration `json:"-"`
	XYEventTimeDeltaBot time.Duration `json:"-"`

	XMiddlePrice float64 `json:"-"`
	YMiddlePrice float64 `json:"-"`

	SpreadMiddle        float64 `json:"-"`
	SpreadLongEnterBot  float64 `json:"-"`
	SpreadLongLeaveTop  float64 `json:"-"`
	SpreadShortEnterTop float64 `json:"-"`
	SpreadShortLeaveBot float64 `json:"-"`
	SpreadEnterOffset   float64 `json:"-"`
	SpreadLeaveOffset   float64 `json:"-"`

	xTimeDeltaOffsetTop  time.Duration
	xTimeDeltaOffsetBot  time.Duration
	yTimeDeltaOffsetTop  time.Duration
	yTimeDeltaOffsetBot  time.Duration
	xyTimeDeltaOffsetTop time.Duration
	xyTimeDeltaOffsetBot time.Duration

	stopped int32
	done    chan interface{}
}

func (sl *XYSimplifiedTickerStats) Stop() {
	if atomic.CompareAndSwapInt32(&sl.stopped, 0, 1) {
		close(sl.done)
		sl.handleSave()
	}
}

func (sl *XYSimplifiedTickerStats) Start(ctx context.Context) {
	sampleTimer := time.NewTimer(sl.sampleInterval)
	saveTimer := time.NewTimer(sl.saveInterval)
	defer func() {
		sl.Stop()
		sampleTimer.Stop()
		saveTimer.Stop()
	}()
	var err error
	const secondFloat64 = float64(time.Second)
	var xReady, yReady bool
	for {
		select {
		case <-ctx.Done():
			return
		case <-sl.done:
			return
		case <-sampleTimer.C:

			xReady = sl.xTicker != nil && sl.xTicker.GetBidPrice() > 0 && sl.xTicker.GetAskPrice() > 0
			yReady = sl.yTicker != nil && sl.yTicker.GetBidPrice() > 0 && sl.yTicker.GetAskPrice() > 0

			if xReady {
				// 0 is initial value
				if sl.XEventTimeDeltaMean == 0 {
					//logger.Debugf("X EVENT TIME DELTA %f %f", sl.xEventTimeDelta, sl.timedDeltaK)
					sl.XEventTimeDeltaMean = sl.xEventTimeDelta.Seconds()
				} else {
					sl.XEventTimeDeltaMean = (sl.xEventTimeDelta.Seconds()-sl.XEventTimeDeltaMean)*sl.timedDeltaK + sl.XEventTimeDeltaMean
				}
				if sl.XParseTimeDeltaMean == 0 {
					sl.XParseTimeDeltaMean = sl.xParseTimeDelta.Seconds()
				} else {
					sl.XParseTimeDeltaMean = (sl.xParseTimeDelta.Seconds()-sl.XParseTimeDeltaMean)*sl.timedDeltaK + sl.XParseTimeDeltaMean
				}
				sl.XMiddlePrice = (sl.xTicker.GetBidPrice() + sl.xTicker.GetAskPrice()) * 0.5
			}
			if yReady {
				if sl.YEventTimeDeltaMean == 0 {
					//logger.Debugf("Y EVENT TIME DELTA %f %f",sl.yEventTimeDelta, sl.timedDeltaK)
					sl.YEventTimeDeltaMean = sl.yEventTimeDelta.Seconds()
				} else {
					sl.YEventTimeDeltaMean = (sl.yEventTimeDelta.Seconds()-sl.YEventTimeDeltaMean)*sl.timedDeltaK + sl.YEventTimeDeltaMean
				}
				if sl.YParseTimeDeltaMean == 0 {
					sl.YParseTimeDeltaMean = sl.yParseTimeDelta.Seconds()
				} else {
					sl.YParseTimeDeltaMean = (sl.yParseTimeDelta.Seconds()-sl.YParseTimeDeltaMean)*sl.timedDeltaK + sl.YParseTimeDeltaMean
				}
				sl.YMiddlePrice = (sl.yTicker.GetBidPrice() + sl.yTicker.GetAskPrice()) * 0.5

				if xReady {
					sl.xyEventTimeDelta = sl.yEventTime.Sub(sl.xEventTime)
					sl.spread = ((sl.yTicker.GetBidPrice() + sl.yTicker.GetAskPrice()) - (sl.xTicker.GetBidPrice() + sl.xTicker.GetAskPrice())) / (sl.xTicker.GetBidPrice() + sl.xTicker.GetAskPrice())
					if sl.xyEventTimeDelta > 0 {
						sl.xyEventTime = sl.yEventTime
					} else {
						sl.xyEventTime = sl.xEventTime
					}
					sl.XYEventTimeDeltaMean = (sl.xyEventTimeDelta.Seconds()-sl.XYEventTimeDeltaMean)*sl.timedDeltaK + sl.XYEventTimeDeltaMean
					err = sl.SpreadTD.Insert(sl.xyEventTime, sl.spread)
					if err != nil {
						logger.Debugf("sl.spreadTD.Insert error %v", err)
					}
				}
			}

			sl.xTicker = nil
			sl.yTicker = nil

			if sl.SpreadTD.Range() > sl.SpreadTD.HalfLookback {
				sl.Ready = true
			}
			longEnterBot := sl.SpreadTD.Quantile(sl.spreadLongEnterQuantileBot)
			longExitTop := sl.SpreadTD.Quantile(sl.spreadLongLeaveQuantileTop)
			shortEnterTop := sl.SpreadTD.Quantile(sl.spreadShortEnterQuantileTop)
			shortExitBot := sl.SpreadTD.Quantile(sl.spreadShortLeaveQuantileBot)
			enterOffset := shortEnterTop - longEnterBot
			exitOffset := longExitTop - shortExitBot
			if enterOffset < sl.baseEnterOffset {
				enterOffset = sl.baseEnterOffset
			}
			if exitOffset < enterOffset*0.25 {
				exitOffset = enterOffset * 0.25
			}
			if exitOffset < sl.baseLeaveOffset {
				exitOffset = sl.baseLeaveOffset
			}

			sl.XParseTimeDeltaMid = time.Duration(sl.XParseTimeDeltaMean * secondFloat64)
			sl.YParseTimeDeltaMid = time.Duration(sl.YParseTimeDeltaMean * secondFloat64)

			sl.XEventTimeDeltaMid = time.Duration(sl.XEventTimeDeltaMean * secondFloat64)
			sl.XEventTimeDeltaBot = sl.XEventTimeDeltaMid + sl.xTimeDeltaOffsetBot
			sl.XEventTimeDeltaTop = sl.XEventTimeDeltaMid + sl.xTimeDeltaOffsetTop

			sl.YEventTimeDeltaMid = time.Duration(sl.YEventTimeDeltaMean * secondFloat64)
			sl.YEventTimeDeltaBot = sl.YEventTimeDeltaMid + sl.yTimeDeltaOffsetBot
			sl.YEventTimeDeltaTop = sl.YEventTimeDeltaMid + sl.yTimeDeltaOffsetTop

			sl.XYEventTimeDeltaMid = time.Duration(sl.XYEventTimeDeltaMean * secondFloat64)
			sl.XYEventTimeDeltaBot = sl.XYEventTimeDeltaMid + sl.xyTimeDeltaOffsetBot
			sl.XYEventTimeDeltaTop = sl.XYEventTimeDeltaMid + sl.xyTimeDeltaOffsetTop

			sl.SpreadMiddle = sl.SpreadTD.Quantile(0.5)
			sl.SpreadLongEnterBot = longEnterBot
			sl.SpreadLongLeaveTop = longExitTop
			sl.SpreadShortEnterTop = shortEnterTop
			sl.SpreadShortLeaveBot = shortExitBot
			sl.SpreadEnterOffset = enterOffset
			sl.SpreadLeaveOffset = exitOffset

			sampleTimer.Reset(sl.sampleInterval)
			break
		case <-saveTimer.C:
			sl.handleSave()
			saveTimer.Reset(sl.saveInterval)
			break
		case sl.xTicker = <-sl.XTickerCh:
			sl.xEventTime = sl.xTicker.GetEventTime()
			sl.xParseTimeDelta = sl.xTicker.GetParseTime().Sub(time.Now())
			sl.xEventTimeDelta = sl.xEventTime.Sub(time.Now())
			break
		case sl.yTicker = <-sl.YTickerCh:
			sl.yEventTime = sl.yTicker.GetEventTime()
			sl.yParseTimeDelta = sl.yTicker.GetParseTime().Sub(time.Now())
			sl.yEventTimeDelta = sl.yEventTime.Sub(time.Now())
			break
		}
	}
}

func (sl *XYSimplifiedTickerStats) loadTD(tdPath string, lookback, subInterval time.Duration, compression uint32) *TimedTDigest {
	td := NewTimedTDigestWithCompression(lookback, subInterval, compression)
	tdBytes, err := os.ReadFile(tdPath)
	if err != nil {
		logger.Debugf("os.ReadFile %s error %v", tdPath, err)
	} else {
		err = json.Unmarshal(tdBytes, td)
		if err != nil {
			logger.Debugf("json.Unmarshal %s error %v", tdPath, err)
		} else {
			td.Lookback = lookback
			td.HalfLookback = lookback / 2
			td.SubInterval = subInterval
			td.Compression = compression
		}
	}
	return td
}

func (sl *XYSimplifiedTickerStats) saveTD(td *TimedTDigest, tdPath string) error {
	select {
	case <-sl.done:
		logger.Debugf("stats save %s", tdPath)
	default:
	}
	tdBytes, err := json.Marshal(td)
	if err != nil {
		return err
	}
	tdFile, err := os.OpenFile(tdPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	_, err = tdFile.Write(tdBytes)
	if err != nil {
		return err
	}
	err = tdFile.Close()
	if err != nil {
		return err
	}
	return nil
}

func (sl *XYSimplifiedTickerStats) handleSave() {
	err := sl.saveTD(sl.SpreadTD, sl.spreadTDPath)
	if err != nil {
		logger.Debugf("sl.saveTD to %s error %v", sl.spreadTDPath, err)
	}
}

//做成struct, 防止函数传太多参数给传错顺序了
type NewXYSimplifiedTickerStatsParams struct {
	XSymbol                     string
	YSymbol                     string
	RootPath                    string
	SampleInterval              time.Duration
	SaveInterval                time.Duration
	TimeDeltaLookback           time.Duration
	SpreadTDLookback            time.Duration
	SpreadTDSubInterval         time.Duration
	SpreadTDCompression         uint32
	SpreadLongEnterQuantileBot  float64
	SpreadLongLeaveQuantileTop  float64
	SpreadShortEnterQuantileTop float64
	SpreadShortLeaveQuantileBot float64
	BaseEnterOffset             float64
	BaseLeaveOffset             float64
	XTimeDeltaOffsetTop         time.Duration
	XTimeDeltaOffsetBot         time.Duration
	YTimeDeltaOffsetTop         time.Duration
	YTimeDeltaOffsetBot         time.Duration
	XYTimeDeltaOffsetTop        time.Duration
	XYTimeDeltaOffsetBot        time.Duration
}

func NewXYSimplifiedTickerStats(params NewXYSimplifiedTickerStatsParams) (*XYSimplifiedTickerStats, error) {

	hasDefault, fields := common.DetectDefaultValues(params, []string{})
	if hasDefault {
		return nil, fmt.Errorf("bad params, has default field for %s", fields)
	}

	if params.RootPath == "" {
		logger.Fatal("need stats root path")
	}

	sl := &XYSimplifiedTickerStats{

		sampleInterval: params.SampleInterval,
		saveInterval:   params.SaveInterval,

		timedDeltaK: 2.0 / float64(params.TimeDeltaLookback/params.SampleInterval+1),

		XTickerCh: make(chan common.Ticker, common.ChannelSizeHighLoad),
		YTickerCh: make(chan common.Ticker, common.ChannelSizeHighLoad),

		spreadLongEnterQuantileBot:  params.SpreadLongEnterQuantileBot,
		spreadLongLeaveQuantileTop:  params.SpreadLongLeaveQuantileTop,
		spreadShortEnterQuantileTop: params.SpreadShortEnterQuantileTop,
		spreadShortLeaveQuantileBot: params.SpreadShortLeaveQuantileBot,
		baseEnterOffset:             params.BaseEnterOffset,
		baseLeaveOffset:             params.BaseLeaveOffset,

		xTimeDeltaOffsetTop:  params.XTimeDeltaOffsetTop,
		xTimeDeltaOffsetBot:  params.XTimeDeltaOffsetBot,
		yTimeDeltaOffsetTop:  params.YTimeDeltaOffsetTop,
		yTimeDeltaOffsetBot:  params.YTimeDeltaOffsetBot,
		xyTimeDeltaOffsetTop: params.XYTimeDeltaOffsetTop,
		xyTimeDeltaOffsetBot: params.XYTimeDeltaOffsetBot,

		spreadTDPath: path.Join(params.RootPath, fmt.Sprintf("%s-%s.S.json", common.SymbolSanitize(params.XSymbol), common.SymbolSanitize(params.YSymbol))),

		stopped: 0,
		done:    make(chan interface{}),
	}

	logger.Debugf("%10s TIME DELTA K %f", params.XSymbol, sl.timedDeltaK)

	sl.SpreadTD = sl.loadTD(sl.spreadTDPath, params.SpreadTDLookback, params.SpreadTDSubInterval, params.SpreadTDCompression)
	logger.Debugf("%10s - %10s SPREAD QUANTILE MIDDLE %.6f", params.XSymbol, params.YSymbol, sl.SpreadTD.Quantile(0.5))

	return sl, nil
}
