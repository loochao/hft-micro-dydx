package bswap

import (
	"encoding/json"
	"strconv"
	"time"
)

type Pool struct {
	ID     int       `json:"poolId"`
	Name   string    `json:"poolName"`
	Assets [2]string `json:"assets"`
}

type Share struct {
	ShareAmount     float64            `json:"shareAmount,string"`
	SharePercentage float64            `json:"sharePercentage,string"`
	Asset           map[string]float64 `json:"-"`
}

func (s *Share) UnmarshalJSON(data []byte) error {
	type Alias Share
	aux := &struct {
		Asset map[string]string `json:"asset"`
		*Alias
	}{
		Alias: (*Alias)(s),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	} else {
		s.Asset = make(map[string]float64)
		for symbol, value := range aux.Asset {
			s.Asset[symbol], err = strconv.ParseFloat(value, 64)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

type Liquidity struct {
	Id        int                `json:"poolId"`
	Name      string             `json:"poolName"`
	Liquidity map[string]float64 `json:"-"`
	Share     Share              `json:"share"`
	EventTime time.Time          `json:"-"`
	ParseTime time.Time          `json:"-"`
}

func (l *Liquidity) UnmarshalJSON(data []byte) error {
	type Alias Liquidity
	aux := &struct {
		UpdateTime int64             `json:"updateTime"`
		Liquidity  map[string]string `json:"liquidity"`
		*Alias
	}{
		Alias: (*Alias)(l),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	} else {
		l.EventTime = time.Unix(0, aux.UpdateTime*1000000)
		l.ParseTime = time.Now()
		l.Liquidity = make(map[string]float64)
		for symbol, value := range aux.Liquidity {
			l.Liquidity[symbol], err = strconv.ParseFloat(value, 64)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

type QuoteParam struct {
	QuoteAsset string  `json:"quoteAsset"`
	BaseAsset  string  `json:"baseAsset"`
	QuoteQty   float64 `json:"quoteQty"`
}

type Quote struct {
	QuoteAsset string  `json:"quoteAsset"`
	BaseAsset  string  `json:"baseAsset"`
	QuoteQty   float64 `json:"quoteQty"`
	BaseQty    float64 `json:"baseQty"`

	"quoteAsset": "USDT",
	"baseAsset": "BUSD",
	"quoteQty": 300000,
	"baseQty": 299975,
	"price": 1.00008334,
	"slippage": 0.00007245,
	"fee": 120
}
