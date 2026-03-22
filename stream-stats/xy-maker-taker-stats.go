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

type XYMakerTakerStats struct {
	params NewXYMakerTakerStatsParams

	timedDeltaK float64

	xEventTimeDelta  time.Duration
	yEventTimeDelta  time.Duration
	xyEventTimeDelta time.Duration

	xParseTimeDelta time.Duration
	yParseTimeDelta time.Duration

	xEventTimeDeltaMean  float64
	yEventTimeDeltaMean  float64
	xyEventTimeDeltaMean float64

	xParseTimeDeltaMean float64
	yParseTimeDeltaMean float64

	xBidVolatility *TimedDelta
	xAskVolatility *TimedDelta

	xBidVolatilityTD *TimedTDigest
	xAskVolatilityTD *TimedTDigest

	yBidSizeTD *TimedTDigest
	yAskSizeTD *TimedTDigest

	bidSpreadTD *TimedTDigest
	askSpreadTD *TimedTDigest

	xBidVolatilityTDPath string
	xAskVolatilityTDPath string

	bidSpreadTDPath string
	askSpreadTDPath string

	yBidSizeTDPath string
	yAskSizeTDPath string

	xEventTime  time.Time
	yEventTime  time.Time
	xyEventTime time.Time

	bidSpread float64
	askSpread float64

	yTicker common.Ticker
	xTicker common.Ticker

	XTickerCh chan common.Ticker
	YTickerCh chan common.Ticker

	Ready bool

	XTimeDeltaBot  time.Duration
	XTimeDeltaMid  time.Duration
	XTimeDeltaTop  time.Duration
	YTimeDeltaBot  time.Duration
	YTimeDeltaMid  time.Duration
	YTimeDeltaTop  time.Duration
	XYTimeDeltaBot time.Duration
	XYTimeDeltaMid time.Duration
	XYTimeDeltaTop time.Duration

	XBidVolatility     float64
	XAskVolatility     float64
	XBidVolatilityNear float64
	XAskVolatilityNear float64
	XBidVolatilityFar  float64
	XAskVolatilityFar  float64

	XMiddlePrice float64
	YMiddlePrice float64

	YBidSize float64
	YAskSize float64

	BidSpreadMiddle float64
	AskSpreadMiddle float64

	AskSpreadEnter    float64
	AskSpreadLeave    float64
	BidSpreadEnter    float64
	BidSpreadLeave    float64
	SpreadEnterOffset float64
	SpreadLeaveOffset float64

	XParseTimeDeltaMid  time.Duration
	YParseTimeDeltaMid  time.Duration
	XYParseTimeDeltaMid time.Duration

	XEventTimeDeltaMid  time.Duration
	YEventTimeDeltaMid  time.Duration
	XYEventTimeDeltaMid time.Duration
	XEventTimeDeltaTop  time.Duration
	XEventTimeDeltaBot  time.Duration
	YEventTimeDeltaTop  time.Duration
	YEventTimeDeltaBot  time.Duration
	XYEventTimeDeltaTop time.Duration
	XYEventTimeDeltaBot time.Duration

	stopped int32
	done    chan interface{}
}

func (sl *XYMakerTakerStats) Stop() {
	if atomic.CompareAndSwapInt32(&sl.stopped, 0, 1) {
		close(sl.done)
		sl.handleSave()
	}
}

func (sl *XYMakerTakerStats) Start(ctx context.Context) {
	sampleTimer := time.NewTimer(sl.params.SampleInterval)
	saveTimer := time.NewTimer(sl.params.SaveInterval)
	defer func() {
		sl.Stop()
		sampleTimer.Stop()
		saveTimer.Stop()
	}()
	var err error
	const secondFloat64 = float64(time.Second)
	for {
		select {
		case <-ctx.Done():
			return
		case <-sl.done:
			return
		case <-sampleTimer.C:
			if sl.xTicker != nil {

				// 0 is initial value
				if sl.xEventTimeDeltaMean == 0 {
					//logger.Debugf("X EVENT TIME DELTA %f %f", sl.xEventTimeDelta, sl.timedDeltaK)
					sl.xEventTimeDeltaMean = sl.xEventTimeDelta.Seconds()
				} else {
					sl.xEventTimeDeltaMean = (sl.xEventTimeDelta.Seconds()-sl.xEventTimeDeltaMean)*sl.timedDeltaK + sl.xEventTimeDeltaMean
				}
				if sl.xParseTimeDeltaMean == 0 {
					sl.xParseTimeDeltaMean = sl.xParseTimeDelta.Seconds()
				} else {
					sl.xParseTimeDeltaMean = (sl.xParseTimeDelta.Seconds()-sl.xParseTimeDeltaMean)*sl.timedDeltaK + sl.xParseTimeDeltaMean
				}
				sl.XMiddlePrice = (sl.xTicker.GetBidPrice() + sl.xTicker.GetAskPrice()) * 0.5

				if sl.xBidVolatility.Delta() < 0 {
					err = sl.xBidVolatilityTD.Insert(sl.xEventTime, -sl.xBidVolatility.Delta()/sl.xTicker.GetBidPrice())
					if err != nil {
						logger.Debugf("sl.xBidVolatilityTD.Insert error %v", err)
					}
				}

				if sl.xAskVolatility.Delta() > 0 {
					err = sl.xAskVolatilityTD.Insert(sl.xEventTime, sl.xAskVolatility.Delta()/sl.xTicker.GetAskPrice())
					if err != nil {
						logger.Debugf("sl.xAskVolatilityTD.Insert error %v", err)
					}
				}

			}
			if sl.yTicker != nil {

				if sl.yEventTimeDeltaMean == 0 {
					//logger.Debugf("Y EVENT TIME DELTA %f %f",sl.yEventTimeDelta, sl.timedDeltaK)
					sl.yEventTimeDeltaMean = sl.yEventTimeDelta.Seconds()
				} else {
					sl.yEventTimeDeltaMean = (sl.yEventTimeDelta.Seconds()-sl.yEventTimeDeltaMean)*sl.timedDeltaK + sl.yEventTimeDeltaMean
				}
				if sl.yParseTimeDeltaMean == 0 {
					sl.yParseTimeDeltaMean = sl.yParseTimeDelta.Seconds()
				} else {
					sl.yParseTimeDeltaMean = (sl.yParseTimeDelta.Seconds()-sl.yParseTimeDeltaMean)*sl.timedDeltaK + sl.yParseTimeDeltaMean
				}
				sl.YMiddlePrice = (sl.yTicker.GetBidPrice() + sl.yTicker.GetAskPrice()) * 0.5

				_ = sl.yBidSizeTD.Insert(sl.yEventTime, sl.yTicker.GetBidSize()*sl.yTicker.GetBidPrice()*sl.params.YMultiplier)
				_ = sl.yAskSizeTD.Insert(sl.yEventTime, sl.yTicker.GetAskSize()*sl.yTicker.GetAskPrice()*sl.params.YMultiplier)

				if sl.xTicker != nil {
					sl.xyEventTimeDelta = sl.yEventTime.Sub(sl.xEventTime)
					sl.bidSpread = (sl.yTicker.GetBidPrice() - sl.xTicker.GetBidPrice()) / sl.xTicker.GetBidPrice()
					sl.askSpread = (sl.yTicker.GetAskPrice() - sl.xTicker.GetAskPrice()) / sl.xTicker.GetAskPrice()
					if sl.xyEventTimeDelta > 0 {
						sl.xyEventTime = sl.yEventTime
					} else {
						sl.xyEventTime = sl.xEventTime
					}
					sl.xyEventTimeDeltaMean = (sl.xyEventTimeDelta.Seconds()-sl.xyEventTimeDeltaMean)*sl.timedDeltaK + sl.xyEventTimeDeltaMean
					if err != nil {
						logger.Debugf("sl.xyTimeDeltaTD.Insert error %v", err)
					}
					err = sl.bidSpreadTD.Insert(sl.xyEventTime, sl.bidSpread)
					if err != nil {
						logger.Debugf("sl.bidSpreadTD.Insert error %v", err)
					}
					err = sl.askSpreadTD.Insert(sl.xyEventTime, sl.askSpread)
					if err != nil {
						logger.Debugf("sl.askSpreadTD.Insert error %v", err)
					}
				}
			}

			sl.xTicker = nil
			sl.yTicker = nil

			if sl.bidSpreadTD.Range() > sl.bidSpreadTD.HalfLookback &&
				sl.askSpreadTD.Range() > sl.askSpreadTD.HalfLookback {
				sl.Ready = true
			}

			sl.XBidVolatility = sl.xBidVolatilityTD.Quantile(sl.params.XVolatilityQuantile)
			sl.XBidVolatilityNear = sl.xBidVolatilityTD.Quantile(sl.params.XVolatilityQuantileNear)
			sl.XBidVolatilityFar = sl.xBidVolatilityTD.Quantile(sl.params.XVolatilityQuantileFar)

			sl.XAskVolatility = sl.xAskVolatilityTD.Quantile(sl.params.XVolatilityQuantile)
			sl.XAskVolatilityNear = sl.xAskVolatilityTD.Quantile(sl.params.XVolatilityQuantileNear)
			sl.XAskVolatilityFar = sl.xAskVolatilityTD.Quantile(sl.params.XVolatilityQuantileFar)

			sl.XParseTimeDeltaMid = time.Duration(sl.xParseTimeDeltaMean * secondFloat64)
			sl.YParseTimeDeltaMid = time.Duration(sl.yParseTimeDeltaMean * secondFloat64)

			sl.XEventTimeDeltaMid = time.Duration(sl.xEventTimeDeltaMean * secondFloat64)
			sl.XEventTimeDeltaBot = sl.XEventTimeDeltaMid + sl.params.XTimeDeltaOffsetBot
			sl.XEventTimeDeltaTop = sl.XEventTimeDeltaMid + sl.params.XTimeDeltaOffsetTop

			sl.YEventTimeDeltaMid = time.Duration(sl.yEventTimeDeltaMean * secondFloat64)
			sl.YEventTimeDeltaBot = sl.YEventTimeDeltaMid + sl.params.YTimeDeltaOffsetBot
			sl.YEventTimeDeltaTop = sl.YEventTimeDeltaMid + sl.params.YTimeDeltaOffsetTop

			sl.XYEventTimeDeltaMid = time.Duration(sl.xyEventTimeDeltaMean * secondFloat64)
			sl.XYEventTimeDeltaBot = sl.XYEventTimeDeltaMid + sl.params.XYTimeDeltaOffsetBot
			sl.XYEventTimeDeltaTop = sl.XYEventTimeDeltaMid + sl.params.XYTimeDeltaOffsetTop

			sl.AskSpreadEnter = sl.bidSpreadTD.Quantile(sl.params.AskSpreadQuantileEnter)
			sl.AskSpreadLeave = sl.bidSpreadTD.Quantile(sl.params.AskSpreadQuantileLeave)
			sl.BidSpreadEnter = sl.bidSpreadTD.Quantile(sl.params.BidSpreadQuantileEnter)
			sl.BidSpreadLeave = sl.bidSpreadTD.Quantile(sl.params.BidSpreadQuantileLeave)
			sl.SpreadEnterOffset = (sl.BidSpreadEnter - sl.AskSpreadEnter) * 0.5
			sl.SpreadLeaveOffset = (sl.BidSpreadLeave - sl.AskSpreadLeave) * 0.5
			if sl.SpreadEnterOffset < sl.params.BaseEnterOffset {
				sl.SpreadEnterOffset = sl.params.BaseEnterOffset
			}
			if sl.SpreadLeaveOffset < sl.SpreadEnterOffset*0.25 {
				sl.SpreadLeaveOffset = sl.SpreadEnterOffset * 0.25
			}
			if sl.SpreadLeaveOffset < sl.params.BaseLeaveOffset {
				sl.SpreadLeaveOffset = sl.params.BaseLeaveOffset
			}

			sl.BidSpreadMiddle = sl.bidSpreadTD.Quantile(0.5)
			sl.AskSpreadMiddle = sl.askSpreadTD.Quantile(0.5)

			sampleTimer.Reset(sl.params.SampleInterval)
			break
		case <-saveTimer.C:
			sl.handleSave()
			saveTimer.Reset(sl.params.SaveInterval)
			break
		case sl.xTicker = <-sl.XTickerCh:
			sl.xEventTime = sl.xTicker.GetEventTime()
			sl.xParseTimeDelta = sl.xTicker.GetParseTime().Sub(time.Now())
			sl.xEventTimeDelta = sl.xEventTime.Sub(time.Now())

			sl.xBidVolatility.Insert(sl.xEventTime, sl.xTicker.GetBidPrice())
			sl.xAskVolatility.Insert(sl.xEventTime, sl.xTicker.GetAskPrice())
			break
		case sl.yTicker = <-sl.YTickerCh:
			sl.yEventTime = sl.yTicker.GetEventTime()
			sl.yParseTimeDelta = sl.yTicker.GetParseTime().Sub(time.Now())
			sl.yEventTimeDelta = sl.yEventTime.Sub(time.Now())
			break
		}
	}
}

func (sl *XYMakerTakerStats) loadTD(tdPath string, lookback, subInterval time.Duration, compression uint32) *TimedTDigest {
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

func (sl *XYMakerTakerStats) saveTD(td *TimedTDigest, tdPath string) (err error) {
	defer func() {
		select {
		case <-sl.done:
			//如果已经退出，打一条log
			logger.Debugf("stats save %s", tdPath)
		default:
		}
	}()
	var tdBytes []byte
	var tdFile *os.File
	tdBytes, err = json.Marshal(td)
	if err != nil {
		return err
	}
	tdFile, err = os.OpenFile(tdPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
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

func (sl *XYMakerTakerStats) handleSave() {
	tds := []*TimedTDigest{
		sl.xBidVolatilityTD,
		sl.xAskVolatilityTD,
		sl.bidSpreadTD,
		sl.askSpreadTD,
	}
	paths := []string{
		sl.xBidVolatilityTDPath,
		sl.xAskVolatilityTDPath,
		sl.bidSpreadTDPath,
		sl.askSpreadTDPath,
	}
	for i, td := range tds {
		err := sl.saveTD(td, paths[i])
		if err != nil {
			logger.Debugf("sl.saveTD to %s error %v", paths[i], err)
		}
	}
}

//做成struct, 防止函数传太多参数给传错顺序了
type NewXYMakerTakerStatsParams struct {
	XSymbol string
	YSymbol string

	RootPath           string
	SampleInterval     time.Duration
	SaveInterval       time.Duration
	VolatilityInterval time.Duration

	TimeDeltaLookback time.Duration

	YLiquidityTDLookback    time.Duration
	YLiquidityTDSubInterval time.Duration
	YLiquidityTDCompression uint32
	YLiquidityQuantile      float64
	YMultiplier             float64

	XVolatilityTDLookback    time.Duration
	XVolatilityTDSubInterval time.Duration
	XVolatilityTDCompression uint32

	SpreadTDLookback    time.Duration
	SpreadTDSubInterval time.Duration
	SpreadTDCompression uint32

	XTimeDeltaOffsetTop  time.Duration
	XTimeDeltaOffsetBot  time.Duration
	YTimeDeltaOffsetTop  time.Duration
	YTimeDeltaOffsetBot  time.Duration
	XYTimeDeltaOffsetTop time.Duration
	XYTimeDeltaOffsetBot time.Duration

	XVolatilityQuantileNear float64
	XVolatilityQuantileFar  float64
	XVolatilityQuantile     float64

	AskSpreadQuantileEnter float64
	AskSpreadQuantileLeave float64
	BidSpreadQuantileEnter float64
	BidSpreadQuantileLeave float64
	BaseEnterOffset        float64
	BaseLeaveOffset        float64
}

func NewXYMakerTakerStats(params NewXYMakerTakerStatsParams) (*XYMakerTakerStats, error) {

	hasDefault, fields := common.DetectDefaultValues(params, []string{})
	if hasDefault {
		return nil, fmt.Errorf("bad params, has default filed for %s", fields)
	}

	if params.RootPath == "" {
		logger.Fatal("need stats root path")
	}

	sl := &XYMakerTakerStats{

		params: params,

		timedDeltaK: 2.0 / float64(params.TimeDeltaLookback/params.SampleInterval+1),

		XTickerCh: make(chan common.Ticker, 16),
		YTickerCh: make(chan common.Ticker, 16),

		xBidVolatility: NewTimedDelta(params.VolatilityInterval),
		xAskVolatility: NewTimedDelta(params.VolatilityInterval),

		xBidVolatilityTDPath: path.Join(params.RootPath, fmt.Sprintf("%s-%s.XBV.json", common.SymbolSanitize(params.XSymbol), common.SymbolSanitize(params.YSymbol))),
		xAskVolatilityTDPath: path.Join(params.RootPath, fmt.Sprintf("%s-%s.XAV.json", common.SymbolSanitize(params.XSymbol), common.SymbolSanitize(params.YSymbol))),

		bidSpreadTDPath: path.Join(params.RootPath, fmt.Sprintf("%s-%s.BS.json", common.SymbolSanitize(params.XSymbol), common.SymbolSanitize(params.YSymbol))),
		askSpreadTDPath: path.Join(params.RootPath, fmt.Sprintf("%s-%s.AS.json", common.SymbolSanitize(params.XSymbol), common.SymbolSanitize(params.YSymbol))),

		yBidSizeTDPath: path.Join(params.RootPath, fmt.Sprintf("%s-%s.YBS.json", common.SymbolSanitize(params.XSymbol), common.SymbolSanitize(params.YSymbol))),
		yAskSizeTDPath: path.Join(params.RootPath, fmt.Sprintf("%s-%s.YAS.json", common.SymbolSanitize(params.XSymbol), common.SymbolSanitize(params.YSymbol))),

		Ready: false,

		stopped: 0,
		done:    make(chan interface{}),
	}

	sl.xBidVolatilityTD = sl.loadTD(sl.xBidVolatilityTDPath, params.XVolatilityTDLookback, params.XVolatilityTDSubInterval, params.XVolatilityTDCompression)
	logger.Debugf("%10s - %10s X BID VOLATILITY QUANTILE %.6f", params.XSymbol, params.YSymbol, sl.xBidVolatilityTD.Quantile(params.XVolatilityQuantile))

	sl.xAskVolatilityTD = sl.loadTD(sl.xAskVolatilityTDPath, params.XVolatilityTDLookback, params.XVolatilityTDSubInterval, params.XVolatilityTDCompression)
	logger.Debugf("%10s - %10s X ASK VOLATILITY QUANTILE %.6f", params.XSymbol, params.YSymbol, sl.xAskVolatilityTD.Quantile(params.XVolatilityQuantile))

	sl.bidSpreadTD = sl.loadTD(sl.bidSpreadTDPath, params.SpreadTDLookback, params.SpreadTDSubInterval, params.SpreadTDCompression)
	logger.Debugf("%10s - %10s BID SPREAD QUANTILE MIDDLE %.6f", params.XSymbol, params.YSymbol, sl.bidSpreadTD.Quantile(0.5))

	sl.askSpreadTD = sl.loadTD(sl.askSpreadTDPath, params.SpreadTDLookback, params.SpreadTDSubInterval, params.SpreadTDCompression)
	logger.Debugf("%10s - %10s ASK SPREAD QUANTILE MIDDLE %.6f", params.XSymbol, params.YSymbol, sl.askSpreadTD.Quantile(0.5))

	sl.yBidSizeTD = sl.loadTD(sl.yBidSizeTDPath, params.YLiquidityTDLookback, params.YLiquidityTDSubInterval, params.YLiquidityTDCompression)
	logger.Debugf("%10s - %10s BID SIZE QUANTILE MIDDLE %.6f", params.XSymbol, params.YSymbol, sl.yBidSizeTD.Quantile(params.YLiquidityQuantile))

	sl.yAskSizeTD = sl.loadTD(sl.yAskSizeTDPath, params.YLiquidityTDLookback, params.YLiquidityTDSubInterval, params.YLiquidityTDCompression)
	logger.Debugf("%10s - %10s ASK SIZE QUANTILE MIDDLE %.6f", params.XSymbol, params.YSymbol, sl.yAskSizeTD.Quantile(params.YLiquidityQuantile))

	return sl, nil
}
