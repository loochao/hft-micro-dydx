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

