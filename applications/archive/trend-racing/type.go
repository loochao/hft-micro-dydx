package main

import "time"

type Signal struct {
	Symbol          string
	TradeVolume     float64
	BookVolume float64
	TradeBookRatio  float64
	BestBidPrice    float64
	BestAskPrice    float64
	LastTradePrice  float64
	Direction       float64
	Time            time.Time
}
