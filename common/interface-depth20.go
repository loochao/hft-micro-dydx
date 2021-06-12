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

func (d DepthRawMessage) GetTime() time.Time {
	return d.Time
}

type TradeRaw struct {
	Data   []byte
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

	BestBidPrice float64
	BestAskPrice float64

	MidPrice   float64
	MircoPrice float64

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
	MakerDir         float64
	TakerDir         float64
	Time             time.Time
}

type SpreadReport struct {
	AdjustedAgeDiff       time.Duration
	MatchRatio            float64
	TakerDepthFilterRatio float64
	MakerDepthFilterRatio float64
	TakerTimeDeltaEma     float64
	MakerTimeDeltaEma     float64
	MakerTimeDelta        float64
	TakerTimeDelta        float64
	TakerMidPrice         float64
	MakerMidPrice         float64
	MakerSymbol           string
	TakerSymbol           string
	MakerExpireRatio      float64
	TakerExpireRatio      float64
}

func (s *SpreadReport) ToString() string {
	return fmt.Sprintf(
		"%s-%s MR %f MDFR %f TDFR %f MTD %f TTD %f MTEMA %f TTEMA %f MAD %v",
		s.MakerSymbol, s.TakerSymbol,
		s.MatchRatio,
		s.MakerDepthFilterRatio, s.TakerDepthFilterRatio,
		s.MakerTimeDelta, s.TakerTimeDelta,
		s.MakerTimeDeltaEma, s.TakerTimeDeltaEma,
		s.AdjustedAgeDiff,
	)
}

func WalkMakerTakerDepth(depth Depth, makerImpact, takerImpact float64) (*WalkedMakerTakerDepth, error) {

	wd, hasMakerData, hasTakerData := &WalkedMakerTakerDepth{
		Symbol:       depth.GetSymbol(),
		Time:         depth.GetTime(),
		TakerAsk:     0,
		TakerBid:     0,
		MakerAsk:     0,
		MakerBid:     0,
		TakerBidSize: 0,
		TakerAskSize: 0,
		MakerAskSize: 0,
		MakerBidSize: 0,
	}, false, false

	bids := depth.GetBids()
	bidLen := len(bids)
	if bidLen > 20 {
		bidLen = 20
	}
	for i := 0; i < bidLen; i++ {
		bid := bids[i]
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
		return nil, fmt.Errorf("bad depth bids %v", depth.GetBids())
	}
	wd.TakerBid /= wd.TakerBidSize
	wd.MakerBid /= wd.MakerBidSize

	hasMakerData = false
	hasTakerData = false
	asks := depth.GetAsks()
	askLen := len(asks)
	if askLen > 20 {
		askLen = 20
	}
	for i := 0; i < askLen; i++ {
		ask := asks[i]
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
		return nil, fmt.Errorf("bad depth ask %v", depth.GetAsks())
	}
	wd.TakerAsk /= wd.TakerAskSize
	wd.MakerAsk /= wd.MakerAskSize
	wd.MidPrice = (depth.GetBids()[0][0] + depth.GetAsks()[0][0]) * 0.5
	wd.BestBidPrice = depth.GetBids()[0][0]
	wd.BestAskPrice = depth.GetAsks()[0][0]
	return wd, nil
}

func WalkMakerTakerDepth5(depth5 Depth, makerImpact, takerImpact float64) (*WalkedMakerTakerDepth, error) {

	wd, hasMakerData, hasTakerData := &WalkedMakerTakerDepth{
		Symbol:       depth5.GetSymbol(),
		Time:         depth5.GetTime(),
		TakerAsk:     0,
		TakerBid:     0,
		MakerAsk:     0,
		MakerBid:     0,
		TakerBidSize: 0,
		TakerAskSize: 0,
		MakerAskSize: 0,
		MakerBidSize: 0,
	}, false, false

	for _, bid := range depth5.GetBids() {
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
		return nil, fmt.Errorf("bad depth bids %v", depth5.GetBids())
	}
	wd.TakerBid /= wd.TakerBidSize
	wd.MakerBid /= wd.MakerBidSize

	hasMakerData = false
	hasTakerData = false
	for _, ask := range depth5.GetAsks() {
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
		return nil, fmt.Errorf("bad depth ask %v", depth5.GetAsks())
	}
	wd.TakerAsk /= wd.TakerAskSize
	wd.MakerAsk /= wd.MakerAskSize
	wd.BestBidPrice = depth5.GetBids()[0][0]
	wd.BestAskPrice = depth5.GetAsks()[0][0]
	wd.MidPrice = (depth5.GetBids()[0][0] + depth5.GetAsks()[0][0]) * 0.5
	wd.MircoPrice = (depth5.GetBids()[0][0]*depth5.GetAsks()[0][1] +
		depth5.GetAsks()[0][0]*depth5.GetBids()[0][1]) / (depth5.GetAsks()[0][1] + depth5.GetBids()[0][1])
	return wd, nil
}

type TimedData interface {
	GetTime() time.Time
}

type TimeFilter struct {
	min       float64
	max       float64
	decay1    float64
	decay2    float64
	bias      float64
	TimeDelta float64

	TimeDeltaEma float64
	FilterCount  int
	TotalCount   int
	Report       TimeReport
}

func (m *TimeFilter) Filter(msg TimedData) bool {
	m.TotalCount++
	m.TimeDelta = float64(time.Now().Sub(msg.GetTime()) / time.Millisecond)
	if m.TimeDelta > m.max {
		m.TimeDelta = m.max
	}
	if m.TimeDelta < m.min {
		m.TimeDelta = m.min
	}
	//before := m.TimeDeltaEma
	m.TimeDeltaEma = m.decay1*m.TimeDeltaEma + m.decay2*m.TimeDelta
	//logger.Debugf("%f before %f after %f", m.TimeDelta,before, m.TimeDeltaEma)
	if m.TimeDelta > m.TimeDeltaEma+m.bias {
		m.FilterCount++
		return true
	}
	return false
}

func (m *TimeFilter) GenerateReport() TimeReport {
	if m.TotalCount > 0 {
		m.Report.FilterRatio = float64(m.FilterCount) / float64(m.TotalCount)
		m.Report.TimeDeltaEma = m.TimeDeltaEma
		m.TotalCount = 0
		m.FilterCount = 0
	}
	return m.Report
}

func NewDepthFilter(
	decay, bias, min, max float64,
) TimeFilter {
	//logger.Debugf("min %f max %f", min, max)
	return TimeFilter{
		decay1:       decay,
		decay2:       1. - decay,
		bias:         bias,
		TimeDelta:    bias,
		min:          min,
		max:          max,
		TimeDeltaEma: bias,
		FilterCount:  0,
		TotalCount:   0,
		Report: TimeReport{
			Decay: decay,
			Bias:  bias,
		},
	}
}

type TimeReport struct {
	FilterRatio  float64
	TimeDeltaEma float64
	Decay        float64
	Bias         float64
}

type ShortSpread struct {
	MakerSymbol string
	TakerSymbol string
	Age         time.Duration
	AgeDiff     time.Duration
	LastEnter   float64
	LastLeave   float64
	MedianEnter float64
	MedianLeave float64
	MakerDepth  WalkedTakerDepth
	TakerDepth  WalkedTakerDepth
	Time        time.Time
}

type WalkedTakerDepth struct {
	TakerFarAsk float64
	TakerAsk    float64
	TakerBid    float64
	TakerFarBid float64

	BestBidPrice float64
	BestAskPrice float64

	MidPrice float64

	TakerBidSize float64
	TakerAskSize float64

	Time   time.Time
	Symbol string
}

func WalkTakerDepth5(depth20 Depth5, takerImpact float64) (*WalkedTakerDepth, error) {

	wd := &WalkedTakerDepth{
		Symbol:       depth20.GetSymbol(),
		Time:         depth20.GetTime(),
		TakerAsk:     0,
		TakerBid:     0,
		TakerBidSize: 0,
		TakerAskSize: 0,
	}

	for _, bid := range depth20.GetBids() {
		value := bid[0] * bid[1]

		wd.TakerFarBid = bid[0]
		if wd.TakerBid+value >= takerImpact {
			wd.TakerBidSize += (takerImpact - wd.TakerBid) / bid[0]
			wd.TakerBid = takerImpact
			break
		} else {
			wd.TakerBidSize += bid[1]
			wd.TakerBid += value
		}
	}
	if wd.TakerBidSize == 0 {
		return nil, fmt.Errorf("bad depth bids %v", depth20.GetBids())
	}
	wd.TakerBid /= wd.TakerBidSize
	for _, ask := range depth20.GetAsks() {
		value := ask[0] * ask[1]
		wd.TakerFarAsk = ask[0]
		if wd.TakerAsk+value >= takerImpact {
			wd.TakerAskSize += (takerImpact - wd.TakerAsk) / ask[0]
			wd.TakerAsk = takerImpact
			break
		} else {
			wd.TakerAskSize += ask[1]
			wd.TakerAsk += value
		}

	}
	if wd.TakerAskSize == 0 {
		return nil, fmt.Errorf("bad depth ask %v", depth20.GetAsks())
	}
	wd.TakerAsk /= wd.TakerAskSize
	wd.MidPrice = (depth20.GetBids()[0][0] + depth20.GetAsks()[0][0]) * 0.5
	wd.BestBidPrice = depth20.GetBids()[0][0]
	wd.BestAskPrice = depth20.GetAsks()[0][0]
	return wd, nil
}
