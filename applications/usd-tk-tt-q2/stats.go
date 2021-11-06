package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	stream_stats "github.com/geometrybase/hft-micro/stream-stats"
	"os"
	"path"
)

type StatsLoop struct {
	xSymbol       string
	ySymbol       string
	spreadTD      *stream_stats.TimedTDigest
	xTimeDeltaTD  *stream_stats.TimedTDigest
	yTimeDeltaTD  *stream_stats.TimedTDigest
	xyTimeDeltaTD *stream_stats.TimedTDigest
	xBidSizeTD    *stream_stats.TimedTDigest
	xAskSizeTD    *stream_stats.TimedTDigest
	yBidSizeTD    *stream_stats.TimedTDigest
	yAskSizeTD    *stream_stats.TimedTDigest

	spreadTDPath      string
	xTimeDeltaTDPath  string
	yTimeDeltaTDPath  string
	xyTimeDeltaTDPath string
	xBidSizeTDPath    string
	xAskSizeTDPath    string
	yBidSizeTDPath    string
	yAskSizeTDPath    string

	xTimeDelta         *TimeDelta
	yTimeDelta         *TimeDelta
	xyTimeDelta        *TimeDelta
	yLiquidity         *common.Ticker
	spread             *Spread
	XDepthTimeDeltaCh  chan TimeDelta
	YDepthTimeDeltaCh  chan TimeDelta
	XYDepthTimeDeltaCh chan TimeDelta
	YLiquidityCh       chan *common.Ticker
	SpreadCh           chan Spread
}

func (sl *StatsLoop) start(ctx context.Context) {

}

func (sl *StatsLoop) handleSave(ctx context.Context) {
	var tdBytes []byte
	var err error
	var tdFile *os.File

	//xTimeDeltaTD
	tdBytes, err = json.Marshal(sl.xTimeDeltaTD)
	if err != nil {
		logger.Debugf("%s json.Marshal(sl.xTimeDeltaTD) error %v", sl.xSymbol, err)
	} else {
		tdFile, err = os.OpenFile(sl.xTimeDeltaTDPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			logger.Debugf("%s os.OpenFile(sl.xTimeDeltaTDPath... error %v", sl.xSymbol, err)
		} else {
			_, err = tdFile.Write(tdBytes)
			if err != nil {
				logger.Debugf("%s tdFile.Write(tdBytes) error %v", sl.xSymbol, err)
			} else {
				err = tdFile.Close()
				if err != nil {
					logger.Debugf("%s tdFile.Close() error %v", sl.xSymbol, err)
				}
			}
		}
	}

	//yTimeDeltaTD
	tdBytes, err = json.Marshal(sl.yTimeDeltaTD)
	if err != nil {
		logger.Debugf("%s json.Marshal(sl.yTimeDeltaTD) error %v", sl.xSymbol, err)
	} else {
		tdFile, err = os.OpenFile(sl.yTimeDeltaTDPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			logger.Debugf("%s os.OpenFile(sl.yTimeDeltaTDPath... error %v", sl.xSymbol, err)
		} else {
			_, err = tdFile.Write(tdBytes)
			if err != nil {
				logger.Debugf("%s tdFile.Write(tdBytes) error %v", sl.xSymbol, err)
			} else {
				err = tdFile.Close()
				if err != nil {
					logger.Debugf("%s tdFile.Close() error %v", sl.xSymbol, err)
				}
			}
		}
	}

	//xyTimeDeltaTD
	tdBytes, err = json.Marshal(sl.xyTimeDeltaTD)
	if err != nil {
		logger.Debugf("%s json.Marshal(sl.xyTimeDeltaTD) error %v", sl.xSymbol, err)
	} else {
		tdFile, err = os.OpenFile(sl.xyTimeDeltaTDPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			logger.Debugf("%s os.OpenFile(sl.xyTimeDeltaTDPath... error %v", sl.xSymbol, err)
		} else {
			_, err = tdFile.Write(tdBytes)
			if err != nil {
				logger.Debugf("%s tdFile.Write(tdBytes) error %v", sl.xSymbol, err)
			} else {
				err = tdFile.Close()
				if err != nil {
					logger.Debugf("%s tdFile.Close() error %v", sl.xSymbol, err)
				}
			}
		}
	}

	//spreadTD
	tdBytes, err = json.Marshal(sl.spreadTD)
	if err != nil {
		logger.Debugf("%s json.Marshal(sl.spreadTD) error %v", sl.xSymbol, err)
	} else {
		tdFile, err = os.OpenFile(sl.spreadTDPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			logger.Debugf("%s os.OpenFile(sl.spreadTDPath... error %v", sl.xSymbol, err)
		} else {
			_, err = tdFile.Write(tdBytes)
			if err != nil {
				logger.Debugf("%s tdFile.Write(tdBytes) error %v", sl.xSymbol, err)
			} else {
				err = tdFile.Close()
				if err != nil {
					logger.Debugf("%s tdFile.Close() error %v", sl.xSymbol, err)
				}
			}
		}
	}

}

func NewStatsLoop(
	xSymbol, ySymbol string,
	config Config,
) *StatsLoop {

	if config.TDRootPath != "" {
		logger.Fatal("need stats path")
	}

	sl := &StatsLoop{
		xSymbol:            xSymbol,
		ySymbol:            ySymbol,
		XDepthTimeDeltaCh:  make(chan TimeDelta, 16),
		YDepthTimeDeltaCh:  make(chan TimeDelta, 16),
		XYDepthTimeDeltaCh: make(chan TimeDelta, 16),
		spreadTDPath:       path.Join(config.TDRootPath, fmt.Sprintf("%s-%s.S.json", common.SymbolSanitize(xSymbol), common.SymbolSanitize(ySymbol))),
		xTimeDeltaTDPath:   path.Join(config.TDRootPath, fmt.Sprintf("%s-%s.XTD.json", common.SymbolSanitize(xSymbol), common.SymbolSanitize(ySymbol))),
		yTimeDeltaTDPath:   path.Join(config.TDRootPath, fmt.Sprintf("%s-%s.YTD.json", common.SymbolSanitize(xSymbol), common.SymbolSanitize(ySymbol))),
		xyTimeDeltaTDPath:  path.Join(config.TDRootPath, fmt.Sprintf("%s-%s.XYTD.json", common.SymbolSanitize(xSymbol), common.SymbolSanitize(ySymbol))),
		yAskSizeTDPath:     path.Join(config.TDRootPath, fmt.Sprintf("%s-%s.YLL.json", common.SymbolSanitize(xSymbol), common.SymbolSanitize(ySymbol))),
		yBidSizeTDPath:     path.Join(config.TDRootPath, fmt.Sprintf("%s-%s.YSL.json", common.SymbolSanitize(xSymbol), common.SymbolSanitize(ySymbol))),
	}

	sl.spreadTD = stream_stats.NewTimedTDigestWithCompression(config.SpreadTDLookback, config.SpreadTDSubInterval, config.SpreadTDCompression)
	tdBytes, err := os.ReadFile(sl.spreadTDPath)
	if err != nil {
		logger.Debugf("%s os.ReadFile error %v", xSymbol, err)
	} else {
		err = json.Unmarshal(tdBytes, sl.spreadTD)
		if err != nil {
			logger.Debugf("%s json.Unmarshal error %v", xSymbol, err)
		} else {
			sl.spreadTD.Lookback = config.SpreadTDLookback
			sl.spreadTD.SubInterval = config.SpreadTDSubInterval
			sl.spreadTD.Compression = config.SpreadTDCompression
			logger.Debugf("%s - %s SPREAD QUANTILE MIDDLE %f", xSymbol, ySymbol, sl.spreadTD.Quantile(0.5))
		}
	}

	sl.xTimeDeltaTD = stream_stats.NewTimedTDigestWithCompression(config.TimeDeltaTDLookback, config.TimeDeltaTDSubInterval, config.TimeDeltaTDCompression)
	tdBytes, err = os.ReadFile(sl.xTimeDeltaTDPath)
	if err != nil {
		logger.Debugf("%s os.ReadFile error %v", xSymbol, err)
	} else {
		err = json.Unmarshal(tdBytes, sl.xTimeDeltaTD)
		if err != nil {
			logger.Debugf("%s json.Unmarshal error %v", xSymbol, err)
		} else {
			sl.xTimeDeltaTD.Lookback = config.TimeDeltaTDLookback
			sl.xTimeDeltaTD.SubInterval = config.TimeDeltaTDSubInterval
			sl.xTimeDeltaTD.Compression = config.TimeDeltaTDCompression
			logger.Debugf("%s - %s X TIME DELTA QUANTILE %f - %f", xSymbol, ySymbol, sl.xTimeDeltaTD.Quantile(config.TimeDeltaQuantileBot), sl.xTimeDeltaTD.Quantile(config.TimeDeltaQuantileTop))
		}
	}

	sl.yTimeDeltaTD = stream_stats.NewTimedTDigestWithCompression(config.TimeDeltaTDLookback, config.TimeDeltaTDSubInterval, config.TimeDeltaTDCompression)
	tdBytes, err = os.ReadFile(sl.yTimeDeltaTDPath)
	if err != nil {
		logger.Debugf("%s os.ReadFile error %v", xSymbol, err)
	} else {
		err = json.Unmarshal(tdBytes, sl.yTimeDeltaTD)
		if err != nil {
			logger.Debugf("%s json.Unmarshal error %v", xSymbol, err)
		} else {
			sl.yTimeDeltaTD.Lookback = config.TimeDeltaTDLookback
			sl.yTimeDeltaTD.SubInterval = config.TimeDeltaTDSubInterval
			sl.yTimeDeltaTD.Compression = config.TimeDeltaTDCompression
			logger.Debugf("%s - %s Y TIME DELTA QUANTILE %f - %f", xSymbol, ySymbol, sl.yTimeDeltaTD.Quantile(config.TimeDeltaQuantileBot), sl.yTimeDeltaTD.Quantile(config.TimeDeltaQuantileTop))
		}
	}

	sl.xyTimeDeltaTD = stream_stats.NewTimedTDigestWithCompression(config.TimeDeltaTDLookback, config.TimeDeltaTDSubInterval, config.TimeDeltaTDCompression)
	tdBytes, err = os.ReadFile(sl.xyTimeDeltaTDPath)
	if err != nil {
		logger.Debugf("%s os.ReadFile error %v", xSymbol, err)
	} else {
		err = json.Unmarshal(tdBytes, sl.xyTimeDeltaTD)
		if err != nil {
			logger.Debugf("%s json.Unmarshal error %v", xSymbol, err)
		} else {
			sl.xyTimeDeltaTD.Lookback = config.TimeDeltaTDLookback
			sl.xyTimeDeltaTD.SubInterval = config.TimeDeltaTDSubInterval
			sl.xyTimeDeltaTD.Compression = config.TimeDeltaTDCompression
			logger.Debugf("%s - %s XY TIME DELTA QUANTILE %f - %f", xSymbol, ySymbol, sl.yTimeDeltaTD.Quantile(config.TimeDeltaQuantileBot), sl.xyTimeDeltaTD.Quantile(config.TimeDeltaQuantileTop))
		}
	}

	sl.xBidSizeTD = stream_stats.NewTimedTDigestWithCompression(config.XLiquidityTDLookback, config.XLiquidityTDSubInterval, config.XLiquidityTDCompression)
	tdBytes, err = os.ReadFile(sl.xBidSizeTDPath)
	if err != nil {
		logger.Debugf("%s os.ReadFile error %v", xSymbol, err)
	} else {
		err = json.Unmarshal(tdBytes, sl.xBidSizeTD)
		if err != nil {
			logger.Debugf("%s json.Unmarshal error %v", xSymbol, err)
		} else {
			sl.xBidSizeTD.Lookback = config.XLiquidityTDLookback
			sl.xBidSizeTD.SubInterval = config.XLiquidityTDSubInterval
			sl.xBidSizeTD.Compression = config.XLiquidityTDCompression
			logger.Debugf("%s - %s X BID SIZE QUANTILE %f", xSymbol, ySymbol, sl.xBidSizeTD.Quantile(config.XLiquidityQuantile))
		}
	}

	sl.xAskSizeTD = stream_stats.NewTimedTDigestWithCompression(config.XLiquidityTDLookback, config.XLiquidityTDSubInterval, config.XLiquidityTDCompression)
	tdBytes, err = os.ReadFile(sl.xAskSizeTDPath)
	if err != nil {
		logger.Debugf("%s os.ReadFile error %v", xSymbol, err)
	} else {
		err = json.Unmarshal(tdBytes, sl.xAskSizeTD)
		if err != nil {
			logger.Debugf("%s json.Unmarshal error %v", xSymbol, err)
		} else {
			sl.xAskSizeTD.Lookback = config.XLiquidityTDLookback
			sl.xAskSizeTD.SubInterval = config.XLiquidityTDSubInterval
			sl.xAskSizeTD.Compression = config.XLiquidityTDCompression
			logger.Debugf("%s - %s X ASK SIZE QUANTILE %f", xSymbol, ySymbol, sl.xAskSizeTD.Quantile(config.XLiquidityQuantile))
		}
	}

	sl.yBidSizeTD = stream_stats.NewTimedTDigestWithCompression(config.YLiquidityTDLookback, config.YLiquidityTDSubInterval, config.YLiquidityTDCompression)
	tdBytes, err = os.ReadFile(sl.yBidSizeTDPath)
	if err != nil {
		logger.Debugf("%s os.ReadFile error %v", xSymbol, err)
	} else {
		err = json.Unmarshal(tdBytes, sl.yBidSizeTD)
		if err != nil {
			logger.Debugf("%s json.Unmarshal error %v", xSymbol, err)
		} else {
			sl.yBidSizeTD.Lookback = config.YLiquidityTDLookback
			sl.yBidSizeTD.SubInterval = config.YLiquidityTDSubInterval
			sl.yBidSizeTD.Compression = config.YLiquidityTDCompression
			logger.Debugf("%s - %s Y BID SIZE QUANTILE %f", xSymbol, ySymbol, sl.yBidSizeTD.Quantile(config.YLiquidityQuantile))
		}
	}

	sl.yAskSizeTD = stream_stats.NewTimedTDigestWithCompression(config.YLiquidityTDLookback, config.YLiquidityTDSubInterval, config.YLiquidityTDCompression)
	tdBytes, err = os.ReadFile(sl.yAskSizeTDPath)
	if err != nil {
		logger.Debugf("%s os.ReadFile error %v", xSymbol, err)
	} else {
		err = json.Unmarshal(tdBytes, sl.yAskSizeTD)
		if err != nil {
			logger.Debugf("%s json.Unmarshal error %v", xSymbol, err)
		} else {
			sl.yAskSizeTD.Lookback = config.YLiquidityTDLookback
			sl.yAskSizeTD.SubInterval = config.YLiquidityTDSubInterval
			sl.yAskSizeTD.Compression = config.YLiquidityTDCompression
			logger.Debugf("%s - %s Y ASK SIZE QUANTILE %f", xSymbol, ySymbol, sl.yAskSizeTD.Quantile(config.YLiquidityQuantile))
		}
	}
	return sl
}
