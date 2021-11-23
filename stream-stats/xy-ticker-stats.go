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

type XYTickerStats struct {
	xExchange      common.UsdExchange
	yExchange      common.UsdExchange
	sampleInterval time.Duration
	saveInterval   time.Duration

	xTimeDeltaTD  *TimedTDigest
	yTimeDeltaTD  *TimedTDigest
	xyTimeDeltaTD *TimedTDigest

	xBidSizeTD *TimedTDigest
	xAskSizeTD *TimedTDigest
	yBidSizeTD *TimedTDigest
	yAskSizeTD *TimedTDigest

	xBidOffsetTD *TimedTDigest
	xAskOffsetTD *TimedTDigest
	yBidOffsetTD *TimedTDigest
	yAskOffsetTD *TimedTDigest

	spreadTD *TimedTDigest

	xTimeDeltaTDPath  string
	yTimeDeltaTDPath  string
	xyTimeDeltaTDPath string
	xBidSizeTDPath    string
	xAskSizeTDPath    string
	yBidSizeTDPath    string
	yAskSizeTDPath    string
	xBidOffsetTDPath  string
	xAskOffsetTDPath  string
	yBidOffsetTDPath  string
	yAskOffsetTDPath  string
	spreadTDPath      string

	xTimeDelta  time.Duration
	xEventTime  time.Time
	yTimeDelta  time.Duration
	yEventTime  time.Time
	xyTimeDelta time.Duration
	xyEventTime time.Time

	spread float64

	yTicker common.Ticker
	xTicker common.Ticker

	XTickerCh chan common.Ticker
	YTickerCh chan common.Ticker

	timeDeltaQuantileBot float64
	timeDeltaQuantileTop float64
	xLiquidityQuantile   float64
	yLiquidityQuantile   float64
	xOffsetQuantile      float64
	yOffsetQuantile      float64

	spreadLongEnterQuantileBot  float64
	spreadLongLeaveQuantileTop  float64
	spreadShortEnterQuantileTop float64
	spreadShortLeaveQuantileBot float64
	baseEnterOffset             float64
	baseLeaveOffset             float64

	Ready *common.AtomicBool

	XTimeDeltaBot  *common.AtomicDuration
	XTimeDeltaMid  *common.AtomicDuration
	XTimeDeltaTop  *common.AtomicDuration
	YTimeDeltaBot  *common.AtomicDuration
	YTimeDeltaMid  *common.AtomicDuration
	YTimeDeltaTop  *common.AtomicDuration
	XYTimeDeltaBot *common.AtomicDuration
	XYTimeDeltaMid *common.AtomicDuration
	XYTimeDeltaTop *common.AtomicDuration

	XBidSize *common.AtomicFloat64
	XAskSize *common.AtomicFloat64
	YBidSize *common.AtomicFloat64
	YAskSize *common.AtomicFloat64

	XBidOffset *common.AtomicFloat64
	XAskOffset *common.AtomicFloat64
	YBidOffset *common.AtomicFloat64
	YAskOffset *common.AtomicFloat64

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

func (sl *XYTickerStats) Stop() {
	if atomic.CompareAndSwapInt32(&sl.stopped, 0, 1) {
		close(sl.done)
		sl.handleSave()
	}
}

func (sl *XYTickerStats) Start(ctx context.Context) {
	sampleTimer := time.NewTimer(sl.sampleInterval)
	saveTimer := time.NewTimer(sl.saveInterval)
	defer func() {
		sl.Stop()
		sampleTimer.Stop()
		saveTimer.Stop()
	}()
	hasAllFields := true
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
				err = sl.xTimeDeltaTD.Insert(sl.xEventTime, sl.xTimeDelta.Seconds())
				if err != nil {
					logger.Debugf("sl.xTimeDeltaTD.Insert error %v", err)
				}

				err = sl.xBidSizeTD.Insert(sl.xEventTime, sl.xTicker.GetBidSize())
				if err != nil {
					logger.Debugf("sl.xBidSizeTD.Insert error %v", err)
				}
				err = sl.xAskSizeTD.Insert(sl.xEventTime, sl.xTicker.GetAskSize())
				if err != nil {
					logger.Debugf("sl.xAskSizeTD.Insert error %v", err)
				}

				err = sl.xBidOffsetTD.Insert(sl.xEventTime, sl.xTicker.GetBidOffset())
				if err != nil {
					logger.Debugf("sl.xBidOffsetTD.Insert error %v", err)
				}
				err = sl.xAskOffsetTD.Insert(sl.xEventTime, sl.xTicker.GetAskOffset())
				if err != nil {
					logger.Debugf("sl.xAskOffsetTD.Insert error %v", err)
				}

				sl.XMiddlePrice.Set((sl.xTicker.GetBidPrice() + sl.xTicker.GetAskPrice()) * 0.5)
			}
			if sl.yTicker != nil {
				err = sl.yTimeDeltaTD.Insert(sl.yEventTime, sl.yTimeDelta.Seconds())
				if err != nil {
					logger.Debugf("sl.yTimeDeltaTD.Insert error %v", err)
				}

				err = sl.yBidSizeTD.Insert(sl.yEventTime, sl.yTicker.GetBidSize())
				if err != nil {
					logger.Debugf("sl.yBidSizeTD.Insert error %v", err)
				}
				err = sl.yAskSizeTD.Insert(sl.yEventTime, sl.yTicker.GetAskSize())
				if err != nil {
					logger.Debugf("sl.yAskSizeTD.Insert error %v", err)
				}

				err = sl.yBidOffsetTD.Insert(sl.yEventTime, sl.yTicker.GetBidOffset())
				if err != nil {
					logger.Debugf("sl.yBidOffsetTD.Insert error %v", err)
				}
				err = sl.yAskOffsetTD.Insert(sl.yEventTime, sl.yTicker.GetAskOffset())
				if err != nil {
					logger.Debugf("sl.yAskOffsetTD.Insert error %v", err)
				}

				sl.YMiddlePrice.Set((sl.yTicker.GetBidPrice() + sl.yTicker.GetAskPrice()) * 0.5)

				if sl.xTicker != nil {
					sl.xyTimeDelta = sl.yEventTime.Sub(sl.xEventTime)
					sl.spread = ((sl.yTicker.GetBidPrice()+sl.yTicker.GetAskPrice())*sl.yExchange.GetPriceFactor() - (sl.xTicker.GetBidPrice()+sl.xTicker.GetAskPrice())*sl.xExchange.GetPriceFactor()) / ((sl.xTicker.GetBidPrice() + sl.xTicker.GetAskPrice()) * sl.xExchange.GetPriceFactor())
					if sl.xyTimeDelta > 0 {
						sl.xyEventTime = sl.yEventTime
					} else {
						sl.xyEventTime = sl.xEventTime
					}
					err = sl.xyTimeDeltaTD.Insert(sl.xyEventTime, sl.xyTimeDelta.Seconds())
					if err != nil {
						logger.Debugf("sl.xyTimeDeltaTD.Insert error %v", err)
					}
					err = sl.spreadTD.Insert(sl.xyEventTime, sl.spread)
					if err != nil {
						logger.Debugf("sl.spreadTD.Insert error %v", err)
					}
				}
			}

			sl.xTicker = nil
			sl.yTicker = nil

			hasAllFields = true

			if hasAllFields && sl.xTimeDeltaTD.Range() < sl.xTimeDeltaTD.HalfLookback {
				hasAllFields = false
			}
			sl.XTimeDeltaBot.Set(time.Duration(secondFloat64 * sl.xTimeDeltaTD.Quantile(sl.timeDeltaQuantileBot)))
			sl.XTimeDeltaMid.Set(time.Duration(secondFloat64 * sl.xTimeDeltaTD.Quantile(0.5)))
			sl.XTimeDeltaTop.Set(time.Duration(secondFloat64 * sl.xTimeDeltaTD.Quantile(sl.timeDeltaQuantileTop)))

			if hasAllFields && sl.yTimeDeltaTD.Range() < sl.yTimeDeltaTD.HalfLookback {
				hasAllFields = false
			}
			sl.YTimeDeltaBot.Set(time.Duration(secondFloat64 * sl.yTimeDeltaTD.Quantile(sl.timeDeltaQuantileBot)))
			sl.YTimeDeltaMid.Set(time.Duration(secondFloat64 * sl.yTimeDeltaTD.Quantile(0.5)))
			sl.YTimeDeltaTop.Set(time.Duration(secondFloat64 * sl.yTimeDeltaTD.Quantile(sl.timeDeltaQuantileTop)))

			if hasAllFields && sl.xyTimeDeltaTD.Range() < sl.xyTimeDeltaTD.HalfLookback {
				hasAllFields = false
			}
			sl.XYTimeDeltaBot.Set(time.Duration(secondFloat64 * sl.xyTimeDeltaTD.Quantile(sl.timeDeltaQuantileBot)))
			sl.XYTimeDeltaMid.Set(time.Duration(secondFloat64 * sl.xyTimeDeltaTD.Quantile(0.5)))
			sl.XYTimeDeltaTop.Set(time.Duration(secondFloat64 * sl.xyTimeDeltaTD.Quantile(sl.timeDeltaQuantileTop)))

			if hasAllFields && sl.xBidSizeTD.Range() < sl.xBidSizeTD.HalfLookback {
				hasAllFields = false
			}
			sl.XBidSize.Set(sl.xBidSizeTD.Quantile(sl.xLiquidityQuantile))
			sl.XBidOffset.Set(sl.xBidOffsetTD.Quantile(sl.xOffsetQuantile))

			if hasAllFields && sl.xAskSizeTD.Range() < sl.xAskSizeTD.HalfLookback {
				hasAllFields = false
			}
			sl.XAskSize.Set(sl.xAskSizeTD.Quantile(sl.xLiquidityQuantile))
			sl.XAskOffset.Set(sl.xAskOffsetTD.Quantile(sl.xOffsetQuantile))

			if hasAllFields && sl.yBidSizeTD.Range() < sl.yBidSizeTD.HalfLookback {
				hasAllFields = false
			}
			sl.YBidSize.Set(sl.yBidSizeTD.Quantile(sl.yLiquidityQuantile))
			sl.YBidOffset.Set(sl.yBidOffsetTD.Quantile(sl.yOffsetQuantile))

			if hasAllFields && sl.yAskSizeTD.Range() < sl.yAskSizeTD.HalfLookback {
				hasAllFields = false
			}
			sl.YAskSize.Set(sl.yAskSizeTD.Quantile(sl.yLiquidityQuantile))
			sl.YAskOffset.Set(sl.yAskOffsetTD.Quantile(sl.yOffsetQuantile))

			if hasAllFields && sl.spreadTD.Range() < sl.spreadTD.HalfLookback {
				hasAllFields = false
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

			sl.Ready.Set(hasAllFields)

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

func (sl *XYTickerStats) loadTD(tdPath string, lookback, subInterval time.Duration, compression uint32) *TimedTDigest {
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

func (sl *XYTickerStats) saveTD(td *TimedTDigest, tdPath string) error {
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

func (sl *XYTickerStats) handleSave() {
	tds := []*TimedTDigest{
		sl.xTimeDeltaTD,
		sl.yTimeDeltaTD,
		sl.xyTimeDeltaTD,
		sl.xBidSizeTD,
		sl.xAskSizeTD,
		sl.yBidSizeTD,
		sl.yAskSizeTD,
		sl.xBidOffsetTD,
		sl.xAskOffsetTD,
		sl.yBidOffsetTD,
		sl.yAskOffsetTD,
		sl.spreadTD,
	}
	paths := []string{
		sl.xTimeDeltaTDPath,
		sl.yTimeDeltaTDPath,
		sl.xyTimeDeltaTDPath,
		sl.xBidSizeTDPath,
		sl.xAskSizeTDPath,
		sl.yBidSizeTDPath,
		sl.yAskSizeTDPath,
		sl.xBidOffsetTDPath,
		sl.xAskOffsetTDPath,
		sl.yBidOffsetTDPath,
		sl.yAskOffsetTDPath,
		sl.spreadTDPath,
	}
	for i, td := range tds {
		err := sl.saveTD(td, paths[i])
		if err != nil {
			logger.Debugf("sl.saveTD to %s error %v", paths[i], err)
		}
	}
}

//做成struct, 防止函数传太多参数给传错顺序了
type NewXYTickerStatsParams struct {
	XSymbol        string
	YSymbol        string
	XExchange      common.UsdExchange
	YExchange      common.UsdExchange
	RootPath       string
	SampleInterval time.Duration
	SaveInterval   time.Duration

	TimeDeltaTDLookback    time.Duration
	TimeDeltaTDSubInterval time.Duration
	TimeDeltaTDCompression uint32

	XLiquidityTDLookback    time.Duration
	XLiquidityTDSubInterval time.Duration
	XLiquidityTDCompression uint32

	YLiquidityTDLookback    time.Duration
	YLiquidityTDSubInterval time.Duration
	YLiquidityTDCompression uint32

	XOffsetTDLookback    time.Duration
	XOffsetTDSubInterval time.Duration
	XOffsetTDCompression uint32

	YOffsetTDLookback    time.Duration
	YOffsetTDSubInterval time.Duration
	YOffsetTDCompression uint32

	SpreadTDLookback    time.Duration
	SpreadTDSubInterval time.Duration
	SpreadTDCompression uint32

	TimeDeltaQuantileBot float64
	TimeDeltaQuantileTop float64
	XLiquidityQuantile   float64
	YLiquidityQuantile   float64
	XOffsetQuantile      float64
	YOffsetQuantile      float64

	SpreadLongEnterQuantileBot  float64
	SpreadLongLeaveQuantileTop  float64
	SpreadShortEnterQuantileTop float64
	SpreadShortLeaveQuantileBot float64
	BaseEnterOffset             float64
	BaseLeaveOffset             float64
}

func NewXYTickerStats(params NewXYTickerStatsParams) (*XYTickerStats, error) {

	hasDefault, fields := common.DetectDefaultValues(params, []string{})
	if hasDefault {
		return nil, fmt.Errorf("bad params, has default field for %s", fields)
	}

	if params.RootPath == "" {
		logger.Fatal("need stats root path")
	}

	sl := &XYTickerStats{

		xExchange:      params.XExchange,
		yExchange:      params.YExchange,
		sampleInterval: params.SampleInterval,
		saveInterval:   params.SaveInterval,

		XTickerCh: make(chan common.Ticker, 16),
		YTickerCh: make(chan common.Ticker, 16),

		timeDeltaQuantileBot: params.TimeDeltaQuantileBot,
		timeDeltaQuantileTop: params.TimeDeltaQuantileTop,
		xLiquidityQuantile:   params.XLiquidityQuantile,
		yLiquidityQuantile:   params.YLiquidityQuantile,
		xOffsetQuantile:      params.XOffsetQuantile,
		yOffsetQuantile:      params.YOffsetQuantile,

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
	logger.Debugf("%10s - %10s X TIME DELTA QUANTILE %.6f - %.6f", params.XSymbol, params.YSymbol, sl.xTimeDeltaTD.Quantile(params.TimeDeltaQuantileBot), sl.xTimeDeltaTD.Quantile(params.TimeDeltaQuantileTop))

	sl.yTimeDeltaTD = sl.loadTD(sl.yTimeDeltaTDPath, params.TimeDeltaTDLookback, params.TimeDeltaTDSubInterval, params.TimeDeltaTDCompression)
	logger.Debugf("%10s - %10s Y TIME DELTA QUANTILE %.6f - %.6f", params.XSymbol, params.YSymbol, sl.yTimeDeltaTD.Quantile(params.TimeDeltaQuantileBot), sl.yTimeDeltaTD.Quantile(params.TimeDeltaQuantileTop))

	sl.xyTimeDeltaTD = sl.loadTD(sl.xyTimeDeltaTDPath, params.TimeDeltaTDLookback, params.TimeDeltaTDSubInterval, params.TimeDeltaTDCompression)
	logger.Debugf("%10s - %10s XY TIME DELTA QUANTILE %.6f - %.6f", params.XSymbol, params.YSymbol, sl.xyTimeDeltaTD.Quantile(params.TimeDeltaQuantileBot), sl.xyTimeDeltaTD.Quantile(params.TimeDeltaQuantileTop))

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
