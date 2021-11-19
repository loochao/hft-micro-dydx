package common

import (
	"fmt"
	"time"
)

type WalkedDepthBMA struct {
	BidPrice float64
	AskPrice float64
	MidPrice float64
	Symbol   string
	Time     time.Time
}

type WalkedDepthBBMAA struct {
	BidPrice     float64
	BestBidPrice float64
	MidPrice     float64
	BestAskPrice float64
	AskPrice     float64
	MircoPrice   float64
	Symbol       string
	Time         time.Time
}

type WalkedDepth struct {
	Symbol    string
	ExchangeID ExchangeID
	MidPrice  float64
	BidPrice  float64
	AskPrice  float64
	BidSize   float64
	AskSize   float64
	BidOffset float64
	AskOffset float64
	Time      time.Time
}

func (w WalkedDepth) GetSymbol() string {
	return w.Symbol
}

func (w WalkedDepth) GetTime() time.Time {
	return w.Time
}

func (w WalkedDepth) GetBidPrice() float64 {
	return w.BidPrice
}

func (w WalkedDepth) GetAskPrice() float64 {
	return w.AskPrice
}

func (w WalkedDepth) GetBidSize() float64 {
	return w.BidSize
}

func (w WalkedDepth) GetAskSize() float64 {
	return w.AskSize
}

func (w WalkedDepth) GetBidOffset() float64 {
	return w.BidOffset
}

func (w WalkedDepth) GetAskOffset() float64 {
	return w.AskOffset
}

func (w WalkedDepth) GetExchange() ExchangeID {
	return w.ExchangeID
}

func WalkDepth(depth Depth, multiplier float64, impact float64, output *WalkedDepth) error {
	output.Time = depth.GetTime()
	output.Symbol = depth.GetSymbol()
	output.AskPrice = 0.0
	output.BidPrice = 0.0
	output.ExchangeID = depth.GetExchange()

	totalBidSize := 0.0
	var value float64
	var price float64
	var size float64
	bids := depth.GetBids()
	for i := 0; i < len(bids); i++ {
		price = bids[i][0]
		size = bids[i][1] * multiplier
		value = price * size
		if output.BidPrice+value >= impact {
			totalBidSize += (impact - output.BidPrice) / price
			output.BidPrice = impact
			break
		} else {
			totalBidSize += size
			output.BidPrice += value
		}
	}
	if totalBidSize == 0 {
		return fmt.Errorf("bad depth bids %v", depth.GetBids())
	}
	output.BidPrice /= totalBidSize
	output.BidSize = totalBidSize / multiplier

	totalAskSize := 0.0
	asks := depth.GetAsks()
	for i := 0; i < len(asks); i++ {
		price = asks[i][0]
		size = asks[i][1] * multiplier
		value = price * size
		if output.AskPrice+value >= impact {
			totalAskSize += (impact - output.AskPrice) / price
			output.AskPrice = impact
			break
		} else {
			totalAskSize += size
			output.AskPrice += value
		}
	}
	if totalAskSize == 0 {
		return fmt.Errorf("bad depth asks %v", depth.GetAsks())
	}
	output.AskPrice /= totalAskSize
	output.AskSize = totalAskSize / multiplier
	output.MidPrice = (output.BidPrice + output.AskPrice) * 0.5
	output.BidOffset = (output.MidPrice - output.BidPrice) / output.MidPrice
	output.AskOffset = (output.AskPrice - output.MidPrice) / output.MidPrice
	return nil
}

func WalkDepthBBMAA(depth Depth, multiplier float64, impact float64, output *WalkedDepthBBMAA) error {

	output.Time = depth.GetTime()
	output.Symbol = depth.GetSymbol()
	output.AskPrice = 0.0
	output.BidPrice = 0.0
	output.MidPrice = 0.0

	totalBidSize := 0.0
	var value float64
	var price float64
	var size float64
	bids := depth.GetBids()
	output.BestBidPrice = bids[0][0]
	for i := 0; i < len(bids); i++ {
		price = bids[i][0]
		size = bids[i][1] * multiplier
		value = price * size
		if output.BidPrice+value >= impact {
			totalBidSize += (impact - output.BidPrice) / price
			output.BidPrice = impact
			break
		} else {
			totalBidSize += size
			output.BidPrice += value
		}
	}
	if totalBidSize == 0 {
		return fmt.Errorf("bad depth bids %v", depth.GetBids())
	}
	output.BidPrice /= totalBidSize

	totalAskSize := 0.0
	asks := depth.GetAsks()
	output.BestAskPrice = asks[0][0]
	for i := 0; i < len(asks); i++ {
		price = asks[i][0]
		size = asks[i][1] * multiplier
		value = price * size
		if output.AskPrice+value >= impact {
			totalAskSize += (impact - output.AskPrice) / price
			output.AskPrice = impact
			break
		} else {
			totalAskSize += size
			output.AskPrice += value
		}
	}
	if totalAskSize == 0 {
		return fmt.Errorf("bad depth asks %v", depth.GetAsks())
	}
	output.AskPrice /= totalAskSize
	output.MidPrice = (depth.GetBids()[0][0] + depth.GetAsks()[0][0]) * 0.5
	output.MircoPrice = (output.BidPrice + output.AskPrice) * 0.5
	return nil
}

func WalkDepthBMA(depth Depth, multiplier float64, impact float64, output *WalkedDepthBMA) error {

	output.Time = depth.GetTime()
	output.Symbol = depth.GetSymbol()
	output.AskPrice = 0.0
	output.BidPrice = 0.0
	output.MidPrice = 0.0

	totalSize := 0.0
	var value float64
	var price float64
	var size float64
	bids := depth.GetBids()
	for i := 0; i < len(bids); i++ {
		price = bids[i][0]
		size = bids[i][1] * multiplier
		value = price * size
		if output.BidPrice+value >= impact {
			totalSize += (impact - output.BidPrice) / price
			output.BidPrice = impact
			break
		} else {
			totalSize += size
			output.BidPrice += value
		}
	}
	if totalSize == 0 {
		return fmt.Errorf("bad depth bids %v", depth.GetBids())
	}
	output.BidPrice /= totalSize

	totalSize = 0.0
	asks := depth.GetAsks()
	for i := 0; i < len(asks); i++ {
		price = asks[i][0]
		size = asks[i][1] * multiplier
		value = price * size
		if output.AskPrice+value >= impact {
			totalSize += (impact - output.AskPrice) / price
			output.AskPrice = impact
			break
		} else {
			totalSize += size
			output.AskPrice += value
		}
	}
	if totalSize == 0 {
		return fmt.Errorf("bad depth asks %v", depth.GetAsks())
	}
	output.AskPrice /= totalSize
	output.MidPrice = (depth.GetBids()[0][0] + depth.GetAsks()[0][0]) * 0.5
	return nil
}

func WalkCoinDepthWithMultiplier(depth Depth, multiplier float64, impact float64, output *WalkedDepthBMA) error {

	output.Time = depth.GetTime()
	output.Symbol = depth.GetSymbol()
	output.AskPrice = 0.0
	output.BidPrice = 0.0
	output.MidPrice = 0.0

	coinValue := 0.0
	var value float64
	var price float64
	bids := depth.GetBids()
	for i := 0; i < len(bids); i++ {
		price = bids[i][0]
		value = bids[i][1] * multiplier
		if output.BidPrice+value >= impact {
			coinValue += (impact - output.BidPrice) / price
			output.BidPrice = impact
			break
		} else {
			coinValue += value / price
			output.BidPrice += value
		}
	}
	if coinValue == 0 {
		return fmt.Errorf("bad depth bids %v", depth.GetBids())
	}
	output.BidPrice /= coinValue

	coinValue = 0.0
	asks := depth.GetAsks()
	for i := 0; i < len(asks); i++ {
		price = asks[i][0]
		value = asks[i][1] * multiplier
		if output.AskPrice+value >= impact {
			coinValue += (impact - output.AskPrice) / price
			output.AskPrice = impact
			break
		} else {
			coinValue += value / price
			output.AskPrice += value
		}
	}
	if coinValue == 0 {
		return fmt.Errorf("bad depth asks %v", depth.GetAsks())
	}
	output.AskPrice /= coinValue
	output.MidPrice = (depth.GetBids()[0][0] + depth.GetAsks()[0][0]) * 0.5
	return nil
}
