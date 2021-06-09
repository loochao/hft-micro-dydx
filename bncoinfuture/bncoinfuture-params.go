package bncoinfuture

import (
	"fmt"
	"net/url"
)

type KlineParams struct {
	Symbol    string
	Interval  string
	Limit     int64
	StartTime int64
	EndTime   int64
}

func (kp *KlineParams) ToUrlValues() url.Values {
	values := url.Values{}
	values.Set("symbol", kp.Symbol)
	values.Set("interval", kp.Interval)
	if kp.Limit > 0 {
		values.Set("limit", fmt.Sprintf("%d", kp.Limit))
	}
	if kp.StartTime > 0 {
		values.Set("startTime", fmt.Sprintf("%d", kp.StartTime))
	}
	if kp.EndTime > 0 {
		values.Set("endTime", fmt.Sprintf("%d", kp.EndTime))
	}
	return values
}


type ChangePositionModeParam struct {
	DualSidePosition bool
}

func (cpmp *ChangePositionModeParam) ToUrlValues() url.Values {
	values := url.Values{}
	if cpmp.DualSidePosition {
		values.Set("dualSidePosition", "true")
	} else {
		values.Set("dualSidePosition", "false")
	}
	return values
}
