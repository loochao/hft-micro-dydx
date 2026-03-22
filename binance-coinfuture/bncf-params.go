package binance_coinfuture

import (
	"fmt"
	"net/url"
	"strconv"
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

type NewOrderParams struct {
	Symbol           string
	Side             string
	Type             string
	Quantity         float64
	ReduceOnly       bool
	Price            float64
	NewClientOrderId string
	TimeInForce      string
	NewOrderRespType string
}

func (no *NewOrderParams) ToUrlValues() url.Values {
	values := url.Values{}
	values.Set("symbol", no.Symbol)
	values.Set("side", no.Side)
	values.Set("type", no.Type)
	values.Set("reduceOnly", strconv.FormatBool(no.ReduceOnly))
	if no.Quantity != 0.0 {
		values.Set("quantity", strconv.FormatFloat(no.Quantity, 'f', 8, 64))
	}
	if no.Price != 0.0 && no.Type != OrderTypeMarket {
		values.Set("price", strconv.FormatFloat(no.Price, 'f', 8, 64))
	}
	values.Set("newClientOrderId", no.NewClientOrderId)
	if no.TimeInForce != "" {
		values.Set("timeInForce", no.TimeInForce)
	}
	values.Set("newOrderRespType", no.NewOrderRespType)
	return values
}

func (no NewOrderParams) ToString() string {
	return fmt.Sprintf(
		"Market=%s, Side=%s, Type=%s, ReduceOnly=%v, "+
			"Quantity=%f, Price=%f, NewClientOrderId=%s, "+
			"TimeInForce=%s",
		no.Symbol, no.Side, no.Type, no.ReduceOnly,
		no.Quantity, no.Price, no.NewClientOrderId,
		no.TimeInForce,
	)
}

type CancelAllOrderParams struct {
	Symbol string
}

func (c *CancelAllOrderParams) ToUrlValues() url.Values {
	values := url.Values{}
	values.Set("symbol", c.Symbol)
	return values
}

type CancelOrderParam struct {
	Symbol            string
	OrderId           int64
	OrigClientOrderId string
}

func (c *CancelOrderParam) ToUrlValues() url.Values {
	values := url.Values{}
	values.Set("symbol", c.Symbol)
	if c.OrderId != 0 {
		values.Set("orderId", fmt.Sprintf("%d", c.OrderId))
	}
	if c.OrigClientOrderId != "" {
		values.Set("origClientOrderId", c.OrigClientOrderId)
	}
	return values
}


type LeverageParams struct {
	Symbol   string
	Leverage int64
}

func (ulp *LeverageParams) ToUrlValues() url.Values {
	values := url.Values{}
	values.Set("symbol", ulp.Symbol)
	values.Set("leverage", fmt.Sprintf("%d", ulp.Leverage))
	return values
}

type MarginTypeParams struct {
	Symbol     string
	MarginType string
}

func (ulp *MarginTypeParams) ToUrlValues() url.Values {
	values := url.Values{}
	values.Set("symbol", ulp.Symbol)
	values.Set("marginType", ulp.MarginType)
	return values
}


type ListenKey struct {
	ListenKey string `json:"listenKey"`
}

func (lk *ListenKey) ToUrlValues() url.Values {
	values := url.Values{}
	values.Set("listenKey", lk.ListenKey)
	return values
}