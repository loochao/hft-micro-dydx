package bswap

import (
	"encoding/json"
	"net/url"
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

func (q QuoteParam) ToUrlValues() url.Values {
	urlValues := url.Values{}
	urlValues.Set("quoteAsset", q.QuoteAsset)
	urlValues.Set("baseAsset", q.BaseAsset)
	urlValues.Set("quoteQty", strconv.FormatFloat(q.QuoteQty, 'f', -1, 64))
	return urlValues
}

type Quote struct {
	QuoteAsset string  `json:"quoteAsset"`
	BaseAsset  string  `json:"baseAsset"`
	QuoteQty   float64 `json:"quoteQty,string"`
	BaseQty    float64 `json:"baseQty,string"`
	Price      float64 `json:"price,string"`
	Slippage   float64 `json:"slippage,string"`
	Fee        float64 `json:"fee,string"`
}

type SwapParam struct {
	SwapAsset string  `json:"quoteAsset"`
	BaseAsset string  `json:"baseAsset"`
	SwapQty   float64 `json:"quoteQty"`
}

func (q SwapParam) ToUrlValues() url.Values {
	urlValues := url.Values{}
	urlValues.Set("quoteAsset", q.SwapAsset)
	urlValues.Set("baseAsset", q.BaseAsset)
	urlValues.Set("quoteQty", strconv.FormatFloat(q.SwapQty, 'f', -1, 64))
	return urlValues
}

type Depth struct {
	Symbol       string
	BuyPrice     float64
	SellPrice    float64
	BuyFee       float64
	SellFee      float64
	BuySlippage  float64
	SellSlippage float64
	Time         time.Time
}
