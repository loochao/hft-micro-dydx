package common

import "fmt"

func WalkCoinPerpetualDepth(depth20 Depth, makerImpact, takerImpact float64) (*WalkedMakerTakerDepth, error) {

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

	bids := depth20.GetBids()
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
		return nil, fmt.Errorf("bad depth bids %v", depth20.GetBids())
	}
	wd.TakerBid /= wd.TakerBidSize
	wd.MakerBid /= wd.MakerBidSize

	hasMakerData = false
	hasTakerData = false
	asks := depth20.GetAsks()
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
		return nil, fmt.Errorf("bad depth ask %v", depth20.GetAsks())
	}
	wd.TakerAsk /= wd.TakerAskSize
	wd.MakerAsk /= wd.MakerAskSize
	wd.MidPrice = (depth20.GetBids()[0][0] + depth20.GetAsks()[0][0]) * 0.5
	wd.BestBidPrice = depth20.GetBids()[0][0]
	wd.BestAskPrice = depth20.GetAsks()[0][0]
	return wd, nil
}
