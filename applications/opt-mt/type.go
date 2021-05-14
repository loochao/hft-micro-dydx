package main

import (
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"strings"
)

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

type Delta struct {
	LongBot  float64
	LongTop  float64
	ShortBot float64
	ShortTop float64
}

func NewDelta(msg string) (Delta, error) {
	splits := strings.Split(msg, ",")
	if len(splits) != 8 {
		return Delta{}, fmt.Errorf("bad delta %s", msg)
	}
	deltas := [8]float64{}
	var err error
	for i, s := range splits {
		deltas[i], err = common.ParseFloat([]byte(s))
		if err != nil {
			return Delta{}, err
		}
	}
	return Delta{
		LongBot: deltas[0],
		LongTop: deltas[7],
		ShortBot: deltas[1],
		ShortTop: deltas[8],
	}, nil
}
