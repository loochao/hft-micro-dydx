package main

import (
	"github.com/geometrybase/hft-micro/hbcrossswap"
	"github.com/geometrybase/hft-micro/hbspot"
)

type SwapOrderNewError struct {
	Error  error
	Params hbcrossswap.NewOrderParam
}

type SpotOrderNewError struct {
	Error  error
	Params hbspot.NewOrderParam
}

type Quantile struct {
	Symbol       string
	Top          float64
	Mid          float64
	Bot          float64
	TopBandScale float64
	BotBandScale float64
	MaClose      float64
}


type SpotOrderRequest struct {
	New    *hbspot.NewOrderParam
	Cancel *hbspot.CancelAllParam
}
