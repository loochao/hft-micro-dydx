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
	Type             string
	BidPrice         float64
	BidSize          float64
	AskPrice         float64
	AskSize          float64
	TakerBidVWAP     float64
	TakerAskVWAP     float64
	TakerBidFarPrice float64
	TakerAskFarPrice float64
	ImpactValue      float64
	ArrivalTime      time.Time
	EventTime        time.Time
}

func (wo *WalkedOrderBook) ToString() string {
	return fmt.Sprintf(
		"%s %s TAKER BID VWAP %f PRICE %f SIZE %f TAKER ASK VWAP %f PRICE %f SIZE %f",
		wo.Type,
		wo.Symbol,
		wo.TakerBidVWAP,
		wo.BidPrice,
		wo.BidSize,
		wo.TakerAskVWAP,
		wo.AskPrice,
		wo.AskSize,
	)
}

type Spread struct {
	Symbol         string
	Age            time.Duration
	AgeDiff        time.Duration
	LastEnter      float64
	LastExit       float64
	MedianEnter    float64
	MedianExit     float64
	SwapOrderBook  WalkedOrderBook
	SpotOrderBook  WalkedOrderBook
	LastUpdateTime time.Time
}

func (s *Spread) ToString() string {
	return fmt.Sprintf(
		"SPREAD %s AGE %v AGE DIFF %v LENTER %f MENTER %f LEXIT %f MEXIT %f %v",
		s.Symbol,
		s.Age,
		s.AgeDiff,
		s.LastEnter,
		s.MedianEnter,
		s.LastExit,
		s.MedianExit,
		s.LastUpdateTime,
	)
}

