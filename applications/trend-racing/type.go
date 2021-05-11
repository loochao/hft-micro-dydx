package main

import "time"

type Signal struct {
	Symbol         string
	BuyVolume      float64
	SellVolume     float64
	BidVolume      float64
	AskVolume      float64
	BestBidPrice   float64
	BestAskPrice   float64
	LastTradePrice float64
	Direction      float64
	Time           time.Time
}
