package common

import "time"

type XYSpread struct {
	ShortLastEnter   float64
	ShortLastLeave   float64
	ShortMedianEnter float64
	ShortMedianLeave float64
	LongLastEnter    float64
	LongLastLeave    float64
	LongMedianEnter  float64
	LongMedianLeave  float64
	EventTime        time.Time
	ParseTime        time.Time
}

type XYSpreadReport struct {
	MatchRatio         float64
	XTickerFilterRatio float64
	YTickerFilterRatio float64
	XTimeDeltaEma      float64
	YTimeDeltaEma      float64
	XTimeDelta         float64
	YTimeDelta         float64
	XMidPrice          float64
	YMidPrice          float64
	XSymbol            string
	YSymbol            string
	XExpireRatio       float64
	YExpireRatio       float64
}
