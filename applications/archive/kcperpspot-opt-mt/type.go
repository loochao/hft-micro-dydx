package main

import (
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/kucoin-usdtfuture"
	"github.com/geometrybase/hft-micro/kucoin-usdtspot"
	"strings"
)

type PerpOrderNewError struct {
	Error  error
	Params kucoin_usdtfuture.NewOrderParam
}

type SpotOrderNewError struct {
	Error  error
	Params kucoin_usdtspot.NewOrderParam
}

type SpotOrderRequest struct {
	New    *kucoin_usdtspot.NewOrderParam
	Cancel *kucoin_usdtspot.CancelAllOrdersParam
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
		Top:     offsets[8],
		NearTop: offsets[6],
		NearBot: offsets[3],
		Bot:     offsets[1],
		FarBot:  offsets[0],
	}, nil
}
