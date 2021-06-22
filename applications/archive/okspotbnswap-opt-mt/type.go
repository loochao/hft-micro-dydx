package main

import (
	"fmt"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/okex-usdtspot"
	"strings"
)

type TakerOrderNewError struct {
	Error  error
	Params bnswap.NewOrderParams
}

type MakerOrderNewError struct {
	Error  error
	Params okex_usdtspot.NewOrderParam
}

type MakerOrderRequest struct {
	New    *okex_usdtspot.NewOrderParam
	Cancel *okex_usdtspot.CancelOrderParam
}

type MakerOpenOrder struct {
	*okex_usdtspot.NewOrderParam
	ResponseOrderID string
	Symbol          string
}

type Offset struct {
	FarTop  float64
	Top     float64
	NearTop float64
	NearBot float64
	Bot     float64
	FarBot  float64
}

func (o Offset) ToString() string {
	return fmt.Sprintf(
		"FarBot %f NearBot %f NearTop %f FarTop %f", o.FarBot, o.NearBot, o.NearTop, o.FarTop)
}

func NewOffset(msg string) (Offset, error) {
	splits := strings.Split(msg, ",")
	if len(splits) != 10 {
		return Offset{}, fmt.Errorf("bad offsets %s", msg)
	}
	offsets := [10]float64{}
	var err error
	for i, s := range splits {
		offsets[i], err = common.ParseFloat([]byte(s))
		if err != nil {
			return Offset{}, err
		}
	}
	return Offset{
		FarTop:  offsets[9],
		Top:     offsets[7],
		NearTop: offsets[5],
		NearBot: offsets[4],
		Bot:     offsets[2],
		FarBot:  offsets[0],
	}, nil
}
