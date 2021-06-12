package main

import (
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/kucoin-usdtfuture"
)

type TakerOrderNewError struct {
	Error  error
	Params bnswap.NewOrderParams
}

type MakerOrderNewError struct {
	Error  error
	Params kucoin_usdtfuture.NewOrderParam
}

type MakerOrderRequest struct {
	New    *kucoin_usdtfuture.NewOrderParam
	Cancel *kucoin_usdtfuture.CancelAllOrdersParam
}

type MakerOpenOrder struct {
	*kucoin_usdtfuture.NewOrderParam
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
	MakerName                = "KCPERP"
	TakerName                = "BNSWAP"
)





