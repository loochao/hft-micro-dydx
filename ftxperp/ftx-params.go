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
	Operation string `json:"op"`
	Market    string `json:"market"`
	Channel   string `json:"channel"`
}
