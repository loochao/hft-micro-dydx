package main

import (
	"github.com/geometrybase/hft-micro/bnswap"
)

type TakerOrderNewError struct {
	Error  error
	Params bnswap.NewOrderParams
}

type TakerOrderRequest struct {
	New    *bnswap.NewOrderParams
	Cancel *bnswap.CancelAllOrderParams
}

type TakerOpenOrder struct {
	*bnswap.NewOrderParams
	Symbol string
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
