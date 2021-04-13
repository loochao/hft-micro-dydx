package main

import (
	"fmt"
	"github.com/geometrybase/hft-micro/kcperp"
	"github.com/geometrybase/hft-micro/kcspot"
	"time"
)

type PerpOrderNewError struct {
	Error  error
	Params kcperp.NewOrderParam
}

type SpotOrderNewError struct {
	Error  error
	Params kcspot.NewOrderParam
}

type Quantile struct {
	Symbol       string
	FarTop       float64
	Top          float64
	Mid          float64
	Bot          float64
	FarBot       float64
	TopBandScale float64
	BotBandScale float64
	MaClose      float64
}

const (
	WalkedOrderBookTypePerp = "PERP"
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
	MakerBidVWAP     float64
	MakerAskVWAP     float64
	MakerBidFarPrice float64
	MakerAskFarPrice float64
	ImpactValue      float64
	ParseTime        time.Time
	EventTime        time.Time
}

func (wo *WalkedOrderBook) ToString() string {
	return fmt.Sprintf(
		"%s %s TAKER BID VWAP %f MAKER BID VWAP %f PRICE %f SIZE %f TAKER ASK VWAP %f MAKER ASK VWAP %f PRICE %f SIZE %f",
		wo.Type,
		wo.Symbol,
		wo.TakerBidVWAP,
		wo.MakerBidVWAP,
		wo.BidPrice,
		wo.BidSize,
		wo.TakerAskVWAP,
		wo.MakerAskVWAP,
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
	PerpOrderBook  WalkedOrderBook
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

type SpotOrderRequest struct {
	New    *kcspot.NewOrderParam
	Cancel *kcspot.CancelAllOrdersParam
}
