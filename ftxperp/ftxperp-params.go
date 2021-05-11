package ftxperp

import (
	"fmt"
	"net/url"
)

type FundingRateParam struct {
	StartTime int64  `json:"start_time"`
	EndTime   int64  `json:"end_time"`
	Future    string `json:"future"`
}

func (frp *FundingRateParam) ToUrlValues() url.Values {
	urlValues := url.Values{}
	if frp.StartTime > 0 {
		urlValues.Set("start_time", fmt.Sprintf("%d", frp.StartTime))
	}
	if frp.StartTime > 0 {
		urlValues.Set("end_time", fmt.Sprintf("%d", frp.EndTime))
	}
	if frp.Future != "" {
		urlValues.Set("future", frp.Future)
	}
	return urlValues
}

type SubscribeParam struct {
	Operation string `json:"op,omitempty"`
	Market    string `json:"market,omitempty"`
	Channel   string `json:"channel,omitempty"`
}

type LeverageParam struct {
	Leverage int `json:"leverage"`
}

type NewOrderParam struct {
	Market     string  `json:"market,omitempty"`
	Side       string  `json:"side,omitempty"`
	Price      float64 `json:"price,omitempty"`
	Type       string  `json:"type,omitempty"`
	Size       float64 `json:"size,omitempty"`
	ReduceOnly bool    `json:"reduceOnly,omitempty"`
	Ioc        bool    `json:"ioc,omitempty"`
	PostOnly   bool    `json:"postOnly,omitempty"`
	ClientID   string  `json:"clientId,omitempty"`
}

type CancelAllParam struct {
	Market                string `json:"market,omitempty"`
	ConditionalOrdersOnly bool   `json:"conditionalOrdersOnly,omitempty"`
	LimitOrdersOnly       string `json:"limitOrdersOnly,omitempty"`
}

type LoginParam struct {
	Args struct {
		Key  string `json:"key"`
		Sign string `json:"sign"`
		Time int64  `json:"time"`
	} `json:"args"`
	Op string `json:"op"`
}
