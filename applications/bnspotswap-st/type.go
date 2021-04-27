package main

import (
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/bnspot"
)

type TakerOrderNewError struct {
	Error  error
	Params bnswap.NewOrderParams
}

type MakerOrderNewError struct {
	Error  error
	Params bnspot.NewOrderParam
}

type TakerOrderRequest struct {
	New    *bnswap.NewOrderParams
	Cancel *bnswap.CancelAllOrderParams
}

type TakerOpenOrder struct {
	*bnswap.NewOrderParams
	Symbol    string
}

type MakerOrderRequest struct {
	New    *bnspot.NewOrderParam
	Cancel *bnspot.CancelAllOrdersParam
}

type MakerOpenOrder struct {
	*bnspot.NewOrderParam
	ResponseOrderID string
	Symbol          string
}

type MakerTakerDeltaQuantile struct {
	Symbol       string
	TakerSymbol  string
	BSymbol      string
	ShortTop     float64
	ShortBot     float64
	LongTop      float64
	LongBot      float64
	Mid          float64
	TopBandScale float64
	BotBandScale float64
	MaClose      float64
}

const (
	MakerName = "KCPERP"
	TakerName = "BNSWAP"
)
