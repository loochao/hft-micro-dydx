package ftxperp

import (
	"encoding/json"
	"time"
)

type Response struct {
	Success bool            `json:"success"`
	Result  json.RawMessage `json:"result"`
}

//{
//      "ask": 4196,
//      "bid": 4114.25,
//      "change1h": 0,
//      "change24h": 0,
//      "changeBod": 0,
//      "volumeUsd24h": 100000000,
//      "volume": 24390.24,
//      "description": "Bitcoin March 2019 Futures",
//      "enabled": true,
//      "expired": false,
//      "expiry": "2019-03-29T03:00:00+00:00",
//      "index": 3919.58841011,
//      "imfFactor": 0.002,
//      "last": 4196,
//      "lowerBound": 3663.75,
//      "mark": 3854.75,
//      "name": "BTC-0329",
//      "perpetual": false,
//      "positionLimitWeight": 1.0,
//      "postOnly": false,
//      "priceIncrement": 0.25,
//      "sizeIncrement": 0.0001,
//      "underlying": "BTC",
//      "upperBound": 4112.2,
//      "type": "future"
//}

type Future struct {
	Ask                 float64 `json:"ask"`
	Bid                 float64 `json:"bid"`
	Change1h            float64 `json:"change1h"`
	Change24h           float64 `json:"change24h"`
	ChangeBod           float64 `json:"changeBod"`
	VolumeUSD24h        float64 `json:"volumeUsd24h"`
	Volume              float64 `json:"volume"`
	Description         string  `json:"description"`
	Enabled             bool    `json:"enabled"`
	Expired             bool    `json:"expired"`
	Expiry              string  `json:"expiry"`
	Index               float64 `json:"index"`
	ImfFactor           float64 `json:"imfFactor"`
	Last                float64 `json:"last"`
	LowerBound          float64 `json:"lowerBound"`
	Mark                float64 `json:"mark"`
	Name                string  `json:"name"`
	Perpetual           bool    `json:"perpetual"`
	PositionLimitWeight float64 `json:"positionLimitWeight"`
	PostOnly            bool    `json:"postOnly"`
	PriceIncrement      float64 `json:"priceIncrement"`
	SizeIncrement       float64 `json:"sizeIncrement"`
	Underlying          string  `json:"underlying"`
	UpperBound          float64 `json:"upperBound"`
	Type                string  `json:"type"`
}

type FundingRate struct {
	Future string    `json:"future"`
	Rate   float64   `json:"rate"`
	Time   time.Time `json:"-"`
}

func (fr *FundingRate) UnmarshalJSON(data []byte) error {
	type Alias FundingRate
	aux := &struct {
		Time string `json:"time"`
		*Alias
	}{
		Alias: (*Alias)(fr),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	} else {
		fr.Time, err = time.Parse("2006-01-02T15:04:05-07:00", aux.Time)
		if err != nil {
			return err
		}
	}
	return nil
}

type Trade struct {
	ID     int64     `json:"id"`
	Price  float64   `json:"price"`
	Size   float64   `json:"size"`
	Side   string    `json:"side"`
	Time   time.Time `json:"-"`
	Symbol string    `json:"-"`
}

func (trade *Trade) GetSymbol() string  { return trade.Symbol }
func (trade *Trade) GetSize() float64   { return trade.Size }
func (trade *Trade) GetPrice() float64  { return trade.Price }
func (trade *Trade) GetTime() time.Time { return trade.Time }
func (trade *Trade) IsUpTick() bool     { return trade.Side == TradeSideBuy }

func (trade *Trade) UnmarshalJSON(data []byte) error {
	type Alias Trade
	aux := &struct {
		Time string `json:"time"`
		*Alias
	}{
		Alias: (*Alias)(trade),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	} else {
		trade.Time, err = time.Parse("2006-01-02T15:04:05.999999-07:00", aux.Time)
		if err != nil {
			return err
		}
	}
	return nil
}

type WSTrades struct {
	Data []Trade `json:"data"`
}
