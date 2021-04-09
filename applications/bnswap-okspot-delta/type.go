package main

import (
	"github.com/geometrybase/hft/okspot"
	"github.com/geometrybase/hft/bnswap"
)

type SwapOrderNewError struct {
	Error  error
	Params bnswap.NewOrderParams
}

type SpotOrderNewError struct {
	Error  error
	Params okspot.NewOrderParams
}

type Quantile struct {
	Symbol       string
	Mid          float64
	Top          float64
	Bot          float64
	TopBandScale float64
	BotBandScale float64
	MaClose      float64
}
