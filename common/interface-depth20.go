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
	MaxAgeDiff            time.Duration
	MatchRatio            float64
	TakerDepthFilterRatio float64
	MakerDepthFilterRatio float64
	TakerTimeDeltaEma     float64
	MakerTimeDeltaEma     float64
	TakerMsgAvgLen        int
	MakerMsgAvgLen        int
	MakerSymbol           string
	TakerSymbol           string
}

func (s *SpreadReport) ToString() string {
	return fmt.Sprintf(
		"%s-%s MR %f MDFR %f TDFR %f MTEMA %f TTEMA %f MAL %d TAL %d",
		s.MakerSymbol, s.TakerSymbol,
		s.MatchRatio,
		s.MakerDepthFilterRatio, s.TakerDepthFilterRatio,
		s.MakerTimeDeltaEma, s.TakerTimeDeltaEma,
		s.MakerMsgAvgLen, s.TakerMsgAvgLen,
	)
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

type DepthFilter struct {
	decay1    float64
	decay2    float64
	bias      float64
	timeDelta float64

	TimeDeltaEma float64
	msgLen       int
	FilterCount  int
	TotalCount   int
	Report       DepthReport
}

func (m *DepthFilter) Filter(msg *DepthRawMessage) bool {
	m.TotalCount++
	m.msgLen += len(msg.Depth)
	m.timeDelta = float64(time.Now().Sub(msg.Time) / time.Millisecond)
	//if m.timeDelta > 1000 {
	//	m.timeDelta = 1000
	//}
	//if m.timeDelta < -1000 {
	//	m.timeDelta = -1000
	//}
	m.TimeDeltaEma = m.decay1*m.TimeDeltaEma + m.decay2*m.timeDelta
	if m.timeDelta > m.TimeDeltaEma+m.bias {
		m.FilterCount++
		//logger.Debugf("FILTER ++ %v", m.FilterCount)
		return true
	}
	return false
}

func (m *DepthFilter) GenerateReport() DepthReport {
	if m.TotalCount > 0 {
		//logger.Debugf("TOTAL COUNT %v FILTER COUNT %v", m.TotalCount, m.FilterCount)
		m.Report.FilterRatio = float64(m.FilterCount) / float64(m.TotalCount)
		m.Report.MsgAvgLen = m.msgLen / m.TotalCount
		m.Report.TimeDeltaEma = m.TimeDeltaEma
		m.TotalCount = 0
		m.FilterCount = 0
		m.msgLen = 0
	}
	return m.Report
}

func NewDepthFilter(
	decay, bias float64,
) DepthFilter {
	return DepthFilter{
		decay1:    decay,
		decay2:    1. - decay,
		bias:      bias,
		timeDelta: bias,
		msgLen:    0,

		TimeDeltaEma: bias,
		FilterCount:  0,
		TotalCount:   0,
		Report: DepthReport{
			Decay: decay,
			Bias:  bias,
		},
	}
}

type DepthReport struct {
	FilterRatio  float64
	TimeDeltaEma float64
	MsgAvgLen    int
	Decay        float64
	Bias         float64
}
