package common

import "fmt"

func WalkCoinPerpetualDepthForMakerAndTaker(depth Depth, contractSize float64, makerImpact, takerImpact float64) (*WalkedMakerTakerDepth, error) {

	wd, hasMakerData, hasTakerData := &WalkedMakerTakerDepth{
		Symbol:       depth.GetSymbol(),
		Time:         depth.GetEventTime(),
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
		if !hasMakerData {
			wd.MakerFarBid = bid[0]
			if wd.MakerBid+bid[1] >= makerImpact {
				wd.MakerBidSize += (makerImpact - wd.MakerBid) / bid[0]
				wd.MakerBid = makerImpact
				hasMakerData = true
			} else {
				wd.MakerBidSize += bid[1] / bid[0]
				wd.MakerBid += bid[1]
			}
		}
		if !hasTakerData {
			wd.TakerFarBid = bid[0]
			if wd.TakerBid+bid[1] >= takerImpact {
				wd.TakerBidSize += (takerImpact - wd.TakerBid) / bid[0]
				wd.TakerBid = takerImpact
				hasTakerData = true
			} else {
				wd.TakerBidSize += bid[1] / bid[0]
				wd.TakerBid += bid[1]
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
		if !hasMakerData {
			wd.MakerFarAsk = ask[0]
			if wd.MakerAsk+ask[1] >= makerImpact {
				wd.MakerAskSize += (makerImpact - wd.MakerAsk) / ask[0]
				wd.MakerAsk = makerImpact
				hasMakerData = true
			} else {
				wd.MakerAskSize += ask[1]/ask[0]
				wd.MakerAsk += ask[1]
			}
		}
		if !hasTakerData {
			wd.TakerFarAsk = ask[0]
			if wd.TakerAsk+ask[1] >= takerImpact {
				wd.TakerAskSize += (takerImpact - wd.TakerAsk) / ask[0]
				wd.TakerAsk = takerImpact
				hasTakerData = true
			} else {
				wd.TakerAskSize += ask[1]/ask[0]
				wd.TakerAsk += ask[1]
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

func WalkCoinPerpetualDepth(depth Depth, contractSize float64, makerImpact, takerImpact float64) (*WalkedMakerTakerDepth, error) {

	wd, hasMakerData, hasTakerData := &WalkedMakerTakerDepth{
		Symbol:       depth.GetSymbol(),
		Time:         depth.GetEventTime(),
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
		if !hasMakerData {
			wd.MakerFarBid = bid[0]
			if wd.MakerBid+bid[1] >= makerImpact {
				wd.MakerBidSize += (makerImpact - wd.MakerBid) / bid[0]
				wd.MakerBid = makerImpact
				hasMakerData = true
			} else {
				wd.MakerBidSize += bid[1] / bid[0]
				wd.MakerBid += bid[1]
			}
		}
		if !hasTakerData {
			wd.TakerFarBid = bid[0]
			if wd.TakerBid+bid[1] >= takerImpact {
				wd.TakerBidSize += (takerImpact - wd.TakerBid) / bid[0]
				wd.TakerBid = takerImpact
				hasTakerData = true
			} else {
				wd.TakerBidSize += bid[1] / bid[0]
				wd.TakerBid += bid[1]
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
		if !hasMakerData {
			wd.MakerFarAsk = ask[0]
			if wd.MakerAsk+ask[1] >= makerImpact {
				wd.MakerAskSize += (makerImpact - wd.MakerAsk) / ask[0]
				wd.MakerAsk = makerImpact
				hasMakerData = true
			} else {
				wd.MakerAskSize += ask[1]/ask[0]
				wd.MakerAsk += ask[1]
			}
		}
		if !hasTakerData {
			wd.TakerFarAsk = ask[0]
			if wd.TakerAsk+ask[1] >= takerImpact {
				wd.TakerAskSize += (takerImpact - wd.TakerAsk) / ask[0]
				wd.TakerAsk = takerImpact
				hasTakerData = true
			} else {
				wd.TakerAskSize += ask[1]/ask[0]
				wd.TakerAsk += ask[1]
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
