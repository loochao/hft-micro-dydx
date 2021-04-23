package common

import (
	"fmt"
	"time"
)

type Depth20 interface {
	GetBids() [20][2]float64
	GetAsks() [20][2]float64
	GetTime() time.Time
	GetSymbol() string
}

type Depth5 interface {
	GetBids() [5][2]float64
	GetAsks() [5][2]float64
	GetTime() time.Time
	GetSymbol() string
}

type DepthRawMessage struct {
	Depth  []byte
	Symbol string
	Time   time.Time
}

type WalkedMakerTakerDepth struct {
	MakerFarAsk float64
	MakerAsk    float64
	MakerBid    float64
	MakerFarBid float64

	TakerFarAsk float64
	TakerAsk    float64
	TakerBid    float64
	TakerFarBid float64

	MakerBidSize float64
	MakerAskSize float64
	TakerBidSize float64
	TakerAskSize float64

	Time   time.Time
	Symbol string
}

type MakerTakerSpread struct {
	MakerSymbol      string
	TakerSymbol      string
	Age              time.Duration
	AgeDiff          time.Duration
	ShortLastEnter   float64
	ShortLastLeave   float64
	ShortMedianEnter float64
	ShortMedianLeave float64
	LongLastEnter    float64
	LongLastLeave    float64
	LongMedianEnter  float64
	LongMedianLeave  float64
	MakerDepth       WalkedMakerTakerDepth
	TakerDepth       WalkedMakerTakerDepth
	Time             time.Time
}

type SpreadReport struct {
	MaxAge      time.Duration
	MaxAgeDiff  time.Duration
	MatchRatio  float64
	MakerSymbol string
	TakerSymbol string
}

type DepthReport struct {
	Exchange     string
	DropRatio    float64
	EmaTimeDelta float64
	AvgLen       int
	Decay        float64
	Bias         float64
}

func WalkMakerTakerDepth20(depth20 Depth20, makerImpact, takerImpact float64) (*WalkedMakerTakerDepth, error) {

	wd, hasMakerData, hasTakerData := &WalkedMakerTakerDepth{
		Symbol:       depth20.GetSymbol(),
		Time:         depth20.GetTime(),
		TakerAsk:     0,
		TakerBid:     0,
		MakerAsk:     0,
		MakerBid:     0,
		TakerBidSize: 0,
		TakerAskSize: 0,
		MakerAskSize: 0,
		MakerBidSize: 0,
	}, false, false

	for _, bid := range depth20.GetBids() {
		value := bid[0] * bid[1]
		if !hasMakerData {
			wd.MakerFarBid = bid[0]
			if wd.MakerBid+value >= makerImpact {
				wd.MakerBidSize += (makerImpact - wd.MakerBid) / bid[0]
				wd.MakerBid = makerImpact
				hasMakerData = true
			} else {
				wd.MakerBidSize += bid[1]
				wd.MakerBid += value
			}
		}
		if !hasTakerData {
			wd.TakerFarBid = bid[0]
			if wd.TakerBid+value >= takerImpact {
				wd.TakerBidSize += (takerImpact - wd.TakerBid) / bid[0]
				wd.TakerBid = takerImpact
				hasTakerData = true
			} else {
				wd.TakerBidSize += bid[1]
				wd.TakerBid += value
			}
		}
		if hasMakerData && hasTakerData {
			break
		}
	}
	if wd.TakerBidSize == 0 || wd.MakerBidSize == 0 {
		return nil, fmt.Errorf("bad depth bids %v", depth20.GetBids())
	}
	wd.TakerBid /= wd.TakerBidSize
	wd.MakerBid /= wd.MakerBidSize

	hasMakerData = false
	hasTakerData = false
	for _, ask := range depth20.GetAsks() {
		value := ask[0] * ask[1]
		if !hasMakerData {
			wd.MakerFarAsk = ask[0]
			if wd.MakerAsk+value >= makerImpact {
				wd.MakerAskSize += (makerImpact - wd.MakerAsk) / ask[0]
				wd.MakerAsk = makerImpact
				hasMakerData = true
			} else {
				wd.MakerAskSize += ask[1]
				wd.MakerAsk += value
			}
		}
		if !hasTakerData {
			wd.TakerFarAsk = ask[0]
			if wd.TakerAsk+value >= takerImpact {
				wd.TakerAskSize += (takerImpact - wd.TakerAsk) / ask[0]
				wd.TakerAsk = takerImpact
				hasTakerData = true
			} else {
				wd.TakerAskSize += ask[1]
				wd.TakerAsk += value
			}
		}
		if hasMakerData && hasTakerData {
			break
		}
	}
	if wd.TakerAskSize == 0 || wd.MakerAskSize == 0 {
		return nil, fmt.Errorf("bad depth ask %v", depth20.GetAsks())
	}
	wd.TakerAsk /= wd.TakerAskSize
	wd.MakerAsk /= wd.MakerAskSize
	return wd, nil
}

func WalkMakerTakerDepth5(depth20 Depth5, makerImpact, takerImpact float64) (*WalkedMakerTakerDepth, error) {

	wd, hasMakerData, hasTakerData := &WalkedMakerTakerDepth{
		Symbol:       depth20.GetSymbol(),
		Time:         depth20.GetTime(),
		TakerAsk:     0,
		TakerBid:     0,
		MakerAsk:     0,
		MakerBid:     0,
		TakerBidSize: 0,
		TakerAskSize: 0,
		MakerAskSize: 0,
		MakerBidSize: 0,
	}, false, false

	for _, bid := range depth20.GetBids() {
		value := bid[0] * bid[1]
		if !hasMakerData {
			wd.MakerFarBid = bid[0]
			if wd.MakerBid+value >= makerImpact {
				wd.MakerBidSize += (makerImpact - wd.MakerBid) / bid[0]
				wd.MakerBid = makerImpact
				hasMakerData = true
			} else {
				wd.MakerBidSize += bid[1]
				wd.MakerBid += value
			}
		}
		if !hasTakerData {
			wd.TakerFarBid = bid[0]
			if wd.TakerBid+value >= takerImpact {
				wd.TakerBidSize += (takerImpact - wd.TakerBid) / bid[0]
				wd.TakerBid = takerImpact
				hasTakerData = true
			} else {
				wd.TakerBidSize += bid[1]
				wd.TakerBid += value
			}
		}
		if hasMakerData && hasTakerData {
			break
		}
	}
	if wd.TakerBidSize == 0 || wd.MakerBidSize == 0 {
		return nil, fmt.Errorf("bad depth bids %v", depth20.GetBids())
	}
	wd.TakerBid /= wd.TakerBidSize
	wd.MakerBid /= wd.MakerBidSize

	hasMakerData = false
	hasTakerData = false
	for _, ask := range depth20.GetAsks() {
		value := ask[0] * ask[1]
		if !hasMakerData {
			wd.MakerFarAsk = ask[0]
			if wd.MakerAsk+value >= makerImpact {
				wd.MakerAskSize += (makerImpact - wd.MakerAsk) / ask[0]
				wd.MakerAsk = makerImpact
				hasMakerData = true
			} else {
				wd.MakerAskSize += ask[1]
				wd.MakerAsk += value
			}
		}
		if !hasTakerData {
			wd.TakerFarAsk = ask[0]
			if wd.TakerAsk+value >= takerImpact {
				wd.TakerAskSize += (takerImpact - wd.TakerAsk) / ask[0]
				wd.TakerAsk = takerImpact
				hasTakerData = true
			} else {
				wd.TakerAskSize += ask[1]
				wd.TakerAsk += value
			}
		}
		if hasMakerData && hasTakerData {
			break
		}
	}
	if wd.TakerAskSize == 0 || wd.MakerAskSize == 0 {
		return nil, fmt.Errorf("bad depth ask %v", depth20.GetAsks())
	}
	wd.TakerAsk /= wd.TakerAskSize
	wd.MakerAsk /= wd.MakerAskSize
	return wd, nil
}
