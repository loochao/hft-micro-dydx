package backtests

import "math"

type Position struct {
	Size  float64
	Price float64
}

func (p *Position) GetUnrealisedPnl(price float64) float64 {
	return p.Size * (price - p.Price)
}

func (p *Position) Add(size, price float64) float64 {
	if size*p.Size < 0 {
		if math.Abs(size) < math.Abs(p.Size) {
			p.Size += size
			//减仓
			return -size * (price - p.Price)
		} else {
			//换仓
			pnl := p.Size * (price - p.Price)
			p.Size = size + p.Size
			p.Price = price
			return pnl
		}
	} else if p.Size+size != 0 {
		p.Price = (p.Size*p.Price + size*price) / (p.Size + size)
		p.Size += size
	}
	return 0.0
}
