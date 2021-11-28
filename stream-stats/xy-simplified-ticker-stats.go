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
	xExchange      common.UsdExchange
	yExchange      common.UsdExchange
	sampleInterval time.Duration
	saveInterval   time.Duration

	xTimeDeltaTM  *TimedMean
	yTimeDeltaTM  *TimedMean
	xyTimeDeltaTM *TimedMean

	spreadTD *TimedTDigest

	xTimeDeltaTDPath  string
	yTimeDeltaTDPath  string
	xyTimeDeltaTDPath string

	spreadTDPath string

	xTimeDelta  time.Duration
	yTimeDelta  time.Duration
	xyTimeDelta time.Duration

	xEventTime  time.Time
	yEventTime  time.Time
	xyEventTime time.Time

	spread float64

	yTicker common.Ticker
	xTicker common.Ticker

	XTickerCh chan common.Ticker
	YTickerCh chan common.Ticker

	spreadLongEnterQuantileBot  float64
	spreadLongLeaveQuantileTop  float64
	spreadShortEnterQuantileTop float64
	spreadShortLeaveQuantileBot float64
	baseEnterOffset             float64
	baseLeaveOffset             float64

	Ready *common.AtomicBool

	XTimeDeltaMean  *common.AtomicDuration
	YTimeDeltaMean  *common.AtomicDuration
	XYTimeDeltaMean *common.AtomicDuration

	XMiddlePrice *common.AtomicFloat64
	YMiddlePrice *common.AtomicFloat64

	SpreadMiddle        *common.AtomicFloat64
	SpreadLongEnterBot  *common.AtomicFloat64
	SpreadLongLeaveTop  *common.AtomicFloat64
	SpreadShortEnterTop *common.AtomicFloat64
	SpreadShortLeaveBot *common.AtomicFloat64
	SpreadEnterOffset   *common.AtomicFloat64
	SpreadLeaveOffset   *common.AtomicFloat64

	stopped int32
	done    chan interface {
	}
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
	for {
		select {
		case <-ctx.Done():
			return
		case <-sl.done:
			return
		case <-sampleTimer.C:
			if sl.xTicker != nil {
				sl.XTimeDeltaMean.Set(
					time.Duration(sl.xTimeDeltaTM.Insert(sl.xEventTime, sl.xTimeDelta.Seconds()) * secondFloat64),
				)
				sl.XMiddlePrice.Set((sl.xTicker.GetBidPrice() + sl.xTicker.GetAskPrice()) * 0.5)
			}
			if sl.yTicker != nil {
				sl.YTimeDeltaMean.Set(
					time.Duration(sl.yTimeDeltaTM.Insert(sl.yEventTime, sl.yTimeDelta.Seconds()) * secondFloat64),
				)
				sl.YMiddlePrice.Set((sl.yTicker.GetBidPrice() + sl.yTicker.GetAskPrice()) * 0.5)

				if sl.xTicker != nil {
					sl.xyTimeDelta = sl.yEventTime.Sub(sl.xEventTime)
					sl.spread = ((sl.yTicker.GetBidPrice()+sl.yTicker.GetAskPrice())*sl.yExchange.GetPriceFactor() - (sl.xTicker.GetBidPrice()+sl.xTicker.GetAskPrice())*sl.xExchange.GetPriceFactor()) / ((sl.xTicker.GetBidPrice() + sl.xTicker.GetAskPrice()) * sl.xExchange.GetPriceFactor())
					if sl.xyTimeDelta > 0 {
						sl.xyEventTime = sl.yEventTime
					} else {
						sl.xyEventTime = sl.xEventTime
					}
					sl.XYTimeDeltaMean.Set(
						time.Duration(sl.xyTimeDeltaTM.Insert(sl.xyEventTime, sl.xyTimeDelta.Seconds()) * secondFloat64),
					)
					err = sl.spreadTD.Insert(sl.xyEventTime, sl.spread)
					if err != nil {
						logger.Debugf("sl.spreadTD.Insert error %v", err)
					}
				}
			}

			sl.xTicker = nil
			sl.yTicker = nil

			if sl.spreadTD.Range() < sl.spreadTD.HalfLookback {
				sl.Ready.Set(true)
			}
			longEnterBot := sl.spreadTD.Quantile(sl.spreadLongEnterQuantileBot)
			longExitTop := sl.spreadTD.Quantile(sl.spreadLongLeaveQuantileTop)
			shortEnterTop := sl.spreadTD.Quantile(sl.spreadShortEnterQuantileTop)
			shortExitBot := sl.spreadTD.Quantile(sl.spreadShortLeaveQuantileBot)
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

			sl.SpreadMiddle.Set(sl.spreadTD.Quantile(0.5))
			sl.SpreadLongEnterBot.Set(longEnterBot)
			sl.SpreadLongLeaveTop.Set(longExitTop)
			sl.SpreadShortEnterTop.Set(shortEnterTop)
			sl.SpreadShortLeaveBot.Set(shortExitBot)
			sl.SpreadEnterOffset.Set(enterOffset)
			sl.SpreadLeaveOffset.Set(exitOffset)

			sampleTimer.Reset(sl.sampleInterval)
			break
		case <-saveTimer.C:
			sl.handleSave()
			saveTimer.Reset(sl.saveInterval)
			break
		case sl.xTicker = <-sl.XTickerCh:
			sl.xEventTime = sl.xTicker.GetEventTime()
			sl.xTimeDelta = sl.xEventTime.Sub(time.Now())
			break
		case sl.yTicker = <-sl.YTickerCh:
			sl.yEventTime = sl.yTicker.GetEventTime()
			sl.yTimeDelta = sl.yEventTime.Sub(time.Now())
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
	tdBytes, err := json.Marshal(td)
	if err != nil {
		return err
	}
	tdFile, err := os.OpenFile(tdPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	} else {
		_, err = tdFile.Write(tdBytes)
		if err != nil {
			return err
		} else {
			err = tdFile.Close()
			if err != nil {
				return err
			}
		}
	}
	select {
	case <-sl.done:
		logger.Debugf("stats save %s", tdPath)
	default:
	}
	return nil
}

func (sl *XYSimplifiedTickerStats) handleSave() {
	err := sl.saveTD(sl.spreadTD, sl.spreadTDPath)
	if err != nil {
		logger.Debugf("sl.saveTD to %s error %v", sl.spreadTDPath, err)
	}
}

//做成struct, 防止函数传太多参数给传错顺序了
type NewXYSimplifiedTickerStatsParams struct {
	XSymbol        string
	YSymbol        string
	XExchange      common.UsdExchange
	YExchange      common.UsdExchange
	RootPath       string
	SampleInterval time.Duration
	SaveInterval   time.Duration

	TimeDeltaLookback time.Duration

	SpreadTDLookback    time.Duration
	SpreadTDSubInterval time.Duration
	SpreadTDCompression uint32

	XTimeDeltaQuantileBot  float64
	XTimeDeltaQuantileTop  float64
	YTimeDeltaQuantileBot  float64
	YTimeDeltaQuantileTop  float64
	XYTimeDeltaQuantileBot float64
	XYTimeDeltaQuantileTop float64
	XLiquidityQuantile     float64
	YLiquidityQuantile     float64
	XOffsetQuantile        float64
	YOffsetQuantile        float64

	SpreadLongEnterQuantileBot  float64
	SpreadLongLeaveQuantileTop  float64
	SpreadShortEnterQuantileTop float64
	SpreadShortLeaveQuantileBot float64
	BaseEnterOffset             float64
	BaseLeaveOffset             float64
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

		xExchange:      params.XExchange,
		yExchange:      params.YExchange,
		sampleInterval: params.SampleInterval,
		saveInterval:   params.SaveInterval,

		XTickerCh: make(chan common.Ticker, 16),
		YTickerCh: make(chan common.Ticker, 16),

		xTimeDeltaQuantileBot:  params.XTimeDeltaQuantileBot,
		xTimeDeltaQuantileTop:  params.XTimeDeltaQuantileTop,
		yTimeDeltaQuantileBot:  params.YTimeDeltaQuantileBot,
		yTimeDeltaQuantileTop:  params.YTimeDeltaQuantileTop,
		xyTimeDeltaQuantileBot: params.XYTimeDeltaQuantileBot,
		xyTimeDeltaQuantileTop: params.XYTimeDeltaQuantileTop,

		xLiquidityQuantile: params.XLiquidityQuantile,
		yLiquidityQuantile: params.YLiquidityQuantile,
		xOffsetQuantile:    params.XOffsetQuantile,
		yOffsetQuantile:    params.YOffsetQuantile,

		spreadLongEnterQuantileBot:  params.SpreadLongEnterQuantileBot,
		spreadLongLeaveQuantileTop:  params.SpreadLongLeaveQuantileTop,
		spreadShortEnterQuantileTop: params.SpreadShortEnterQuantileTop,
		spreadShortLeaveQuantileBot: params.SpreadShortLeaveQuantileBot,
		baseEnterOffset:             params.BaseEnterOffset,
		baseLeaveOffset:             params.BaseLeaveOffset,

		xTimeDeltaTDPath:  path.Join(params.RootPath, fmt.Sprintf("%s-%s.XTD.json", common.SymbolSanitize(params.XSymbol), common.SymbolSanitize(params.YSymbol))),
		yTimeDeltaTDPath:  path.Join(params.RootPath, fmt.Sprintf("%s-%s.YTD.json", common.SymbolSanitize(params.XSymbol), common.SymbolSanitize(params.YSymbol))),
		xyTimeDeltaTDPath: path.Join(params.RootPath, fmt.Sprintf("%s-%s.XYTD.json", common.SymbolSanitize(params.XSymbol), common.SymbolSanitize(params.YSymbol))),
		xBidSizeTDPath:    path.Join(params.RootPath, fmt.Sprintf("%s-%s.XB.json", common.SymbolSanitize(params.XSymbol), common.SymbolSanitize(params.YSymbol))),
		xAskSizeTDPath:    path.Join(params.RootPath, fmt.Sprintf("%s-%s.XA.json", common.SymbolSanitize(params.XSymbol), common.SymbolSanitize(params.YSymbol))),
		yBidSizeTDPath:    path.Join(params.RootPath, fmt.Sprintf("%s-%s.YB.json", common.SymbolSanitize(params.XSymbol), common.SymbolSanitize(params.YSymbol))),
		yAskSizeTDPath:    path.Join(params.RootPath, fmt.Sprintf("%s-%s.YA.json", common.SymbolSanitize(params.XSymbol), common.SymbolSanitize(params.YSymbol))),

		xBidOffsetTDPath: path.Join(params.RootPath, fmt.Sprintf("%s-%s.XBO.json", common.SymbolSanitize(params.XSymbol), common.SymbolSanitize(params.YSymbol))),
		xAskOffsetTDPath: path.Join(params.RootPath, fmt.Sprintf("%s-%s.XAO.json", common.SymbolSanitize(params.XSymbol), common.SymbolSanitize(params.YSymbol))),
		yBidOffsetTDPath: path.Join(params.RootPath, fmt.Sprintf("%s-%s.YBO.json", common.SymbolSanitize(params.XSymbol), common.SymbolSanitize(params.YSymbol))),
		yAskOffsetTDPath: path.Join(params.RootPath, fmt.Sprintf("%s-%s.YAO.json", common.SymbolSanitize(params.XSymbol), common.SymbolSanitize(params.YSymbol))),

		spreadTDPath: path.Join(params.RootPath, fmt.Sprintf("%s-%s.S.json", common.SymbolSanitize(params.XSymbol), common.SymbolSanitize(params.YSymbol))),

		Ready: common.ForAtomicBool(false),

		XTimeDeltaBot:  common.ForAtomicDuration(0),
		XTimeDeltaMid:  common.ForAtomicDuration(0),
		XTimeDeltaTop:  common.ForAtomicDuration(0),
		YTimeDeltaBot:  common.ForAtomicDuration(0),
		YTimeDeltaMid:  common.ForAtomicDuration(0),
		YTimeDeltaTop:  common.ForAtomicDuration(0),
		XYTimeDeltaBot: common.ForAtomicDuration(0),
		XYTimeDeltaMid: common.ForAtomicDuration(0),
		XYTimeDeltaTop: common.ForAtomicDuration(0),

		XBidSize: common.ForAtomicFloat64(0),
		XAskSize: common.ForAtomicFloat64(0),
		YBidSize: common.ForAtomicFloat64(0),
		YAskSize: common.ForAtomicFloat64(0),

		XBidOffset: common.ForAtomicFloat64(common.DefaultBidAskOffset),
		XAskOffset: common.ForAtomicFloat64(common.DefaultBidAskOffset),
		YBidOffset: common.ForAtomicFloat64(common.DefaultBidAskOffset),
		YAskOffset: common.ForAtomicFloat64(common.DefaultBidAskOffset),

		XMiddlePrice: common.ForAtomicFloat64(0),
		YMiddlePrice: common.ForAtomicFloat64(0),

		SpreadMiddle:        common.ForAtomicFloat64(0),
		SpreadLongEnterBot:  common.ForAtomicFloat64(0),
		SpreadLongLeaveTop:  common.ForAtomicFloat64(0),
		SpreadShortEnterTop: common.ForAtomicFloat64(0),
		SpreadShortLeaveBot: common.ForAtomicFloat64(0),
		SpreadEnterOffset:   common.ForAtomicFloat64(0),
		SpreadLeaveOffset:   common.ForAtomicFloat64(0),

		stopped: 0,
		done:    make(chan interface{}),
	}

	sl.xTimeDeltaTD = sl.loadTD(sl.xTimeDeltaTDPath, params.TimeDeltaTDLookback, params.TimeDeltaTDSubInterval, params.TimeDeltaTDCompression)
	logger.Debugf("%10s - %10s X TIME DELTA QUANTILE %.6f - %.6f", params.XSymbol, params.YSymbol, sl.xTimeDeltaTD.Quantile(params.XTimeDeltaQuantileBot), sl.xTimeDeltaTD.Quantile(params.XTimeDeltaQuantileTop))

	sl.yTimeDeltaTD = sl.loadTD(sl.yTimeDeltaTDPath, params.TimeDeltaTDLookback, params.TimeDeltaTDSubInterval, params.TimeDeltaTDCompression)
	logger.Debugf("%10s - %10s Y TIME DELTA QUANTILE %.6f - %.6f", params.XSymbol, params.YSymbol, sl.yTimeDeltaTD.Quantile(params.YTimeDeltaQuantileBot), sl.yTimeDeltaTD.Quantile(params.YTimeDeltaQuantileTop))

	sl.xyTimeDeltaTD = sl.loadTD(sl.xyTimeDeltaTDPath, params.TimeDeltaTDLookback, params.TimeDeltaTDSubInterval, params.TimeDeltaTDCompression)
	logger.Debugf("%10s - %10s XY TIME DELTA QUANTILE %.6f - %.6f", params.XSymbol, params.YSymbol, sl.xyTimeDeltaTD.Quantile(params.XYTimeDeltaQuantileBot), sl.xyTimeDeltaTD.Quantile(params.XYTimeDeltaQuantileTop))

	sl.xBidSizeTD = sl.loadTD(sl.xBidSizeTDPath, params.XLiquidityTDLookback, params.XLiquidityTDSubInterval, params.XLiquidityTDCompression)
	logger.Debugf("%10s - %10s X BID SIZE QUANTILE %.6f", params.XSymbol, params.YSymbol, sl.xBidSizeTD.Quantile(params.XLiquidityQuantile))

	sl.xAskSizeTD = sl.loadTD(sl.xAskSizeTDPath, params.XLiquidityTDLookback, params.XLiquidityTDSubInterval, params.XLiquidityTDCompression)
	logger.Debugf("%10s - %10s X ASK SIZE QUANTILE %.6f", params.XSymbol, params.YSymbol, sl.xAskSizeTD.Quantile(params.XLiquidityQuantile))

	sl.yBidSizeTD = sl.loadTD(sl.yBidSizeTDPath, params.YLiquidityTDLookback, params.YLiquidityTDSubInterval, params.YLiquidityTDCompression)
	logger.Debugf("%10s - %10s Y BID SIZE QUANTILE %.6f", params.XSymbol, params.YSymbol, sl.yBidSizeTD.Quantile(params.YLiquidityQuantile))

	sl.yAskSizeTD = sl.loadTD(sl.yAskSizeTDPath, params.YLiquidityTDLookback, params.YLiquidityTDSubInterval, params.YLiquidityTDCompression)
	logger.Debugf("%10s - %10s Y ASK SIZE QUANTILE %.6f", params.XSymbol, params.YSymbol, sl.yAskSizeTD.Quantile(params.YLiquidityQuantile))

	sl.xBidOffsetTD = sl.loadTD(sl.xBidOffsetTDPath, params.XOffsetTDLookback, params.XOffsetTDSubInterval, params.XOffsetTDCompression)
	logger.Debugf("%10s - %10s X BID OFFSET QUANTILE %.6f", params.XSymbol, params.YSymbol, sl.xBidOffsetTD.Quantile(params.XOffsetQuantile))

	sl.xAskOffsetTD = sl.loadTD(sl.xAskOffsetTDPath, params.XOffsetTDLookback, params.XOffsetTDSubInterval, params.XOffsetTDCompression)
	logger.Debugf("%10s - %10s X ASK OFFSET QUANTILE %.6f", params.XSymbol, params.YSymbol, sl.xAskOffsetTD.Quantile(params.XOffsetQuantile))

	sl.yBidOffsetTD = sl.loadTD(sl.yBidOffsetTDPath, params.YOffsetTDLookback, params.YOffsetTDSubInterval, params.YOffsetTDCompression)
	logger.Debugf("%10s - %10s Y BID OFFSET QUANTILE %.6f", params.XSymbol, params.YSymbol, sl.yBidOffsetTD.Quantile(params.YOffsetQuantile))

	sl.yAskOffsetTD = sl.loadTD(sl.yAskOffsetTDPath, params.YOffsetTDLookback, params.YOffsetTDSubInterval, params.YOffsetTDCompression)
	logger.Debugf("%10s - %10s Y ASK OFFSET QUANTILE %.6f", params.XSymbol, params.YSymbol, sl.yAskOffsetTD.Quantile(params.YOffsetQuantile))

	sl.spreadTD = sl.loadTD(sl.spreadTDPath, params.SpreadTDLookback, params.SpreadTDSubInterval, params.SpreadTDCompression)
	logger.Debugf("%10s - %10s SPREAD QUANTILE MIDDLE %.6f", params.XSymbol, params.YSymbol, sl.spreadTD.Quantile(0.5))

	return sl, nil
}
