package main

import (
	"github.com/geometrybase/hft-micro/bnswap"
)

type SwapOrderNewError struct {
	Error  error
	Params bnswap.NewOrderParams
}

