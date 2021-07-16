package main

import (
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/huobi-usdtfuture"
)

type OrderNewError struct {
	Error  error
	Params bnswap.NewOrderParams
}

type OrderRequest struct {
	New    *bnswap.NewOrderParams
	Cancel *bnswap.CancelAllOrderParams
}

type MakerOpenOrder struct {
	*huobi_usdtfuture.NewOrderParam
	ResponseOrderID string
	Symbol          string
}

type HighLowQuantile struct {
	Symbol  string
	Top     float64
	Mid     float64
	Bot     float64
	Dir     float64
}

type BidPrice struct {
	Price  float64
	Symbol string
}
