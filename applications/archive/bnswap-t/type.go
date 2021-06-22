package main

import (
	"github.com/geometrybase/hft-micro/bnswap"
	"time"
)

type SwapOrderNewError struct {
	Error  error
	Params bnswap.NewOrderParams
}

type Signal struct {
	Symbol    string
	EventTime time.Time
	Value     float64
	Buy       int
	Sell      int
}
