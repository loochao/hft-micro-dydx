package main

import "time"

type Params struct {
	xSymbol        string
	ySymbol        string
	enterOffset    float64
	leaveOffset    float64
	frFactor       float64
	tradeCost      float64
	startValue     float64
	enterStep      float64
	bestSizeFactor float64
	leverage       float64
	outputInterval time.Duration
	enterInterval  time.Duration
}

type Result struct {
	Params     Params
	NetWorth   []float64
	Positions  []float64
	EventTimes []time.Time
	Turnover   float64
}
