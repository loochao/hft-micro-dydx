package main

import (
	"github.com/geometrybase/hft-micro/bnswap"
)

type TakerOrderNewError struct {
	Error  error
	Params bnswap.NewOrderParams
}

