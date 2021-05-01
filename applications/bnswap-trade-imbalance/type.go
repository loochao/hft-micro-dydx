package main

import (
	"github.com/geometrybase/hft-micro/bnswap"
)

type TakerOrderNewError struct {
	Error  error
	Params bnswap.NewOrderParams
}

type DepthReport struct {
	FilterRatio  float64
	TimeDelta    float64
	TimeDeltaEma float64
	MsgAvgLen    int
	Symbol       string
}

type MergedSignal struct {
	Symbol  string
	Value  float64
	Signals map[string]float64
}
