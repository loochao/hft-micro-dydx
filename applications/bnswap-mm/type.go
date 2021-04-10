package main

import (
	"fmt"
	"github.com/geometrybase/hft-micro/bnspot"
	"github.com/geometrybase/hft-micro/bnswap"
	"time"
)

type SwapOrderNewError struct {
	Error  error
	Params bnswap.NewOrderParams
}

type SpotOrderNewError struct {
	Error  error
	Params bnspot.NewOrderParams
}

type Quantile struct {
	Symbol       string
	Mid          float64
	Top          float64
	Bot          float64
	FarBot       float64
	FarTop       float64
	TopBandScale float64
	BotBandScale float64
	MaClose      float64
}

const (
	WalkedOrderBookTypeSwap = "SWAP"
	WalkedOrderBookTypeSpot = "Spot"
)

type WalkedOrderBook struct {
	Symbol           string
	BidPrice         float64
	BidSize          float64
	AskPrice         float64
	AskSize          float64
	CloseBidVWAP     float64
	CloseAskVWAP     float64
	CloseBidFarPrice float64
	CloseAskFarPrice float64
	OpenBidVWAP      float64
	OpenAskVWAP      float64
	OpenBidFarPrice  float64
	OpenAskFarPrice  float64
	ArrivalTime      time.Time
	EventTime        time.Time
}

func (wo *WalkedOrderBook) ToString() string {
	return fmt.Sprintf(
		"%s CLOSE BID VWAP %f OPEN BID VWAP %f PRICE %f SIZE %f CLOSE ASK VWAP %f OPEN ASK VWAP %f PRICE %f SIZE %f",
		wo.Symbol,
		wo.CloseBidVWAP,
		wo.OpenBidVWAP,
		wo.BidPrice,
		wo.BidSize,
		wo.CloseAskVWAP,
		wo.OpenAskVWAP,
		wo.AskPrice,
		wo.AskSize,
	)
}

type Spread struct {
	Symbol      string
	LastLong    float64
	LastShort   float64
	MedianLong  float64
	MedianShort float64
	OrderBook   WalkedOrderBook
	EventTime   time.Time
}

func (s *Spread) ToString() string {
	return fmt.Sprintf(
		"SPREAD %s LAST LONG %f MEDIAN LONG %f LAST SHORT %f MEDIAN SHORT %f %v",
		s.Symbol,
		s.LastLong,
		s.MedianLong,
		s.LastShort,
		s.MedianShort,
		s.EventTime,
	)
}

type SwapOrderRequest struct {
	New    *bnswap.NewOrderParams
	Cancel *bnswap.CancelAllOrderParams
}
