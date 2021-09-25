package main

import "time"

type Params struct {
	XSymbol        string
	YSymbol        string
	EnterOffset    float64
	LeaveOffset    float64
	FrFactor       float64
	TradeCost      float64
	StartValue     float64
	EnterStep      float64
	BestSizeFactor float64
	Leverage       float64
	MaxFundingRate float64
	OutputInterval time.Duration
	enterInterval  time.Duration
}

type Result struct {
	Params       Params
	NetWorth     []float64
	Positions    []float64
	MidPrices    []float64
	Costs        []float64
	FundingRates []float64
	EventTimes   []time.Time
	Turnover     float64
}
