package hbswap

import (
	"fmt"
	"net/url"
	"time"
)

const (
	KlinePeriod1min  = "1min"
	KlinePeriod5min  = "5min"
	KlinePeriod15min = "15min"
	KlinePeriod30min = "30min"
	KlinePeriod60min = "60min"
	KlinePeriod4hour = "4hour"
	KlinePeriod1day  = "1day"
	KlinePeriod1mon  = "1mon"
)

var KlinePeriodDuration = map[string]time.Duration{
	KlinePeriod1min:  time.Minute,
	KlinePeriod5min:  time.Minute * 5,
	KlinePeriod15min: time.Minute * 15,
	KlinePeriod30min: time.Minute * 30,
	KlinePeriod60min: time.Minute * 60,
	KlinePeriod4hour: time.Hour * 4,
	KlinePeriod1day:  time.Hour * 24,
	KlinePeriod1mon:  time.Hour * 24 * 30,
}

type KlinesParam struct {
	ContractCode string
	Period       string
	Size         int
	From         int64
	To           int64
}

func (p *KlinesParam) ToUrlValues() url.Values {
	values := url.Values{}
	values.Set("contract_code", p.ContractCode)
	values.Set("period", p.Period)
	if p.Size > 0 {
		values.Set("size", fmt.Sprintf("%d", p.Size))
	}
	if p.From > 0 {
		values.Set("from", fmt.Sprintf("%d", p.From))
	}
	if p.To > 0 {
		values.Set("to", fmt.Sprintf("%d", p.To))
	}
	return values
}

type SubParam struct {
	Sub string `json:"sub"`
	ID  string `json:"id"`
}
