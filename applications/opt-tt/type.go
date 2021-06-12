package main

import (
	"github.com/geometrybase/hft-micro/common"
	"time"
)

type XYSpread struct {
	XSymbol          string
	YSymbol          string
	Age              time.Duration
	AgeDiff          time.Duration
	ShortLastEnter   float64
	ShortLastLeave   float64
	ShortMedianEnter float64
	ShortMedianLeave float64
	LongLastEnter    float64
	LongLastLeave    float64
	LongMedianEnter  float64
	LongMedianLeave  float64
	XDepth           common.WalkedMakerTakerDepth
	YDepth           common.WalkedMakerTakerDepth
	Time             time.Time
}

type SpreadReport struct {
	AgeDiff           time.Duration
	AdjustedAgeDiff   time.Duration
	MatchRatio        float64
	XDepthFilterRatio float64
	YDepthFilterRatio float64
	XTimeDeltaEma     float64
	YTimeDeltaEma     float64
	XTimeDelta        float64
	YTimeDelta        float64
	XMidPrice         float64
	YMidPrice         float64
	XSymbol           string
	YSymbol           string
	XExpireRatio      float64
	YExpireRatio      float64
	XTimestamp        int64
	YTimestamp        int64
}
