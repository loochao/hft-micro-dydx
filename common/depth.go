package common

import (
	"fmt"
	"time"
)

type WalkedDepthBAM struct {
	BidPrice float64
	AskPrice float64
	MidPrice float64
	Symbol   string
	Time     time.Time
}

func WalkDepthWithMultiplier(depth Depth, multiplier float64, impact float64, output *WalkedDepthBAM) error {

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

func WalkCoinDepthWithMultiplier(depth Depth, multiplier float64, impact float64, output *WalkedDepthBAM) error {

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
			coinValue += value/price
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
			coinValue += value/price
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
