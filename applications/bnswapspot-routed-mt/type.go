package main

import (
	"github.com/geometrybase/hft-micro/bnspot"
	"github.com/geometrybase/hft-micro/bnswap"
)

type TakerOrderNewError struct {
	Error  error
	Params bnswap.NewOrderParams
}

type MakerOrderNewError struct {
	Error  error
	Params bnspot.NewOrderParams
}

type Quantile struct {
	Symbol      string
	Mid         float64
	Top         float64
	Bot         float64
	OriginalTop float64
	OriginalBot float64
	MaClose     float64
}


type SpotOrderRequest struct {
	New    *bnspot.NewOrderParams
	Cancel *bnspot.CancelAllOrderParams
}
