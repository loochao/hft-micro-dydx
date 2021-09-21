package main

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"time"
)

type Params struct {
	xSymbol        string
	ySymbol        string
	enterQScale    float64
	leaveQScale    float64
	outputInterval time.Duration
}

type Result struct {
	NetWorth   []float32
	XPositions []float32
	YPositions []float32
}

type Strategy struct {
	Params   Params
	InputCh  chan *common.MatchedSpread
	ResultCh chan Result
}

func (st *Strategy) Run(ctx context.Context) {
	result := Result{}
	defer func() {
		select {
		case st.ResultCh <- result:
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return
		}
	}
}

func NewStrategy(params Params) *Strategy {
	if params.outputInterval == 0 {
		params.outputInterval = time.Minute
	}
	return &Strategy{
		Params:   params,
		InputCh:  make(chan *common.MatchedSpread, 128),
		ResultCh: make(chan Result, 4),
	}
}
