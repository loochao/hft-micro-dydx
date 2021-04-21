package common

import (
	"time"
)

type Depth20 interface {
	GetBids() [20][2]float64
	GetAsks() [20][2]float64
	GetTime() time.Time
	GetSymbol() string
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

	EventTime time.Time
	Symbol    string
}

func WalkMakerTakerDepth20(depth20 Depth20, makerImpact, takerImpact float64) WalkedMakerTakerDepth {

	wd, hasMakerData, hasTakerData := WalkedMakerTakerDepth{
		Symbol:    depth20.GetSymbol(),
		EventTime: depth20.GetTime(),
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
				wd.TakerAskSize += (takerImpact - wd.TakerAskSize) / ask[0]
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
	wd.TakerAsk /= wd.TakerAskSize
	wd.MakerAsk /= wd.MakerAskSize
	return wd
}
