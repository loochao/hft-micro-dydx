package main

import (
	"fmt"
	"github.com/geometrybase/hft-micro/bnswap"
	"github.com/geometrybase/hft-micro/huobi-usdtfuture"
	"time"
)

type TakerOrderNewError struct {
	Error  error
	Params bnswap.NewOrderParams
}

type MakerOrderNewError struct {
	Error  error
	Params huobi_usdtfuture.NewOrderParam
}

type MakerOrderRequest struct {
	New    *huobi_usdtfuture.NewOrderParam
	Cancel *huobi_usdtfuture.CancelAllParam
}

type MakerOpenOrder struct {
	*huobi_usdtfuture.NewOrderParam
	ResponseOrderID string
	Symbol          string
}

type MakerTakerDeltaQuantile struct {
	Symbol       string
	TakerSymbol  string
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
	MakerName                = "huobi"
	TakerName                = "binance"
)

type WalkedOrderBook struct {
	Symbol           string
	Type             string
	BidPrice         float64
	BidSize          float64
	AskPrice         float64
	AskSize          float64
	BidVWAP          float64
	AskVWAP          float64
	BidFarPrice      float64
	TakerAskFarPrice float64
	ImpactValue      float64
	ParseTime        time.Time
	EventTime        time.Time
}

func (wo *WalkedOrderBook) ToString() string {
	return fmt.Sprintf(
		"%s %s TAKER BID VWAP %f PRICE %f SIZE %f TAKER ASK VWAP %f PRICE %f SIZE %f",
		wo.Type,
		wo.Symbol,
		wo.BidVWAP,
		wo.BidPrice,
		wo.BidSize,
		wo.AskVWAP,
		wo.AskPrice,
		wo.AskSize,
	)
}

type Spread struct {
	Symbol           string
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
		s.Symbol,
		s.Age,
		s.AgeDiff,
		s.ShortLastEnter,
		s.ShortMedianEnter,
		s.ShortLastExit,
		s.ShortMedianExit,
		s.LastUpdateTime,
	)
}
