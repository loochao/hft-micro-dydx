package main

import (
	"fmt"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/hbcrossswap"
	"time"
)

type TakerOrderNewError struct {
	Error  error
	Params bnswap.NewOrderParams
}

type HOrderNewError struct {
	Error  error
	Params hbcrossswap.NewOrderParam
}

type HBDeltaQuantile struct {
	HSymbol      string
	BSymbol      string
	ShortTop     float64
	ShortBot     float64
	LongTop      float64
	LongBot      float64
	Mid          float64
	TopBandScale float64
	BotBandScale float64
	MaClose      float64
}

const (
	WalkedOrderBookTypeMaker = "huobi"
	WalkedOrderBookTypeTaker = "binance"
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
	HSymbol          string
	Age              time.Duration
	AgeDiff          time.Duration
	ShortLastEnter   float64
	ShortLastExit    float64
	ShortMedianEnter float64
	ShortMedianExit  float64
	LongLastEnter    float64
	LongLastExit     float64
	LongMedianEnter  float64
	LongMedianExit   float64
	MakerOrderBook   WalkedOrderBook
	TakerOrderBook   WalkedOrderBook
	LastUpdateTime   time.Time
}

func (s *Spread) ToString() string {
	return fmt.Sprintf(
		"SPREAD %s AGE %v AGE DIFF %v LENTER %f MENTER %f LEXIT %f MEXIT %f %v",
		s.HSymbol,
		s.Age,
		s.AgeDiff,
		s.ShortLastEnter,
		s.ShortMedianEnter,
		s.ShortLastExit,
		s.ShortMedianExit,
		s.LastUpdateTime,
	)
}

type MakerOrderRequest struct {
	New    *hbcrossswap.NewOrderParam
	Cancel *hbcrossswap.CancelAllParam
}
