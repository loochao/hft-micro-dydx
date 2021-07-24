package bybit_usdtfuture

import (
	"fmt"
	"net/url"
	"strconv"
)

type Param interface {
	ToUrlValues() url.Values
}

type SetAutoAddMarginParam struct {
	Symbol        string
	Side          string
	AutoAddMargin bool
}

func (s *SetAutoAddMarginParam) ToUrlValues() url.Values {
	values := url.Values{}
	values.Set("symbol", s.Symbol)
	values.Set("side", s.Side)
	values.Set("auto_add_margin", strconv.FormatBool(s.AutoAddMargin))
	return values
}

type SwitchIsolatedParam struct {
	Symbol       string
	IsIsolated   bool
	BuyLeverage  int
	SellLeverage int
}

func (s *SwitchIsolatedParam) ToUrlValues() url.Values {
	values := url.Values{}
	values.Set("symbol", s.Symbol)
	values.Set("is_isolated", strconv.FormatBool(s.IsIsolated))
	values.Set("buy_leverage", fmt.Sprintf("%d", s.BuyLeverage))
	values.Set("sell_leverage", fmt.Sprintf("%d", s.SellLeverage))
	return values
}

type SetLeverageParam struct {
	Symbol       string
	BuyLeverage  int
	SellLeverage int
}

func (s *SetLeverageParam) ToUrlValues() url.Values {
	values := url.Values{}
	values.Set("symbol", s.Symbol)
	values.Set("buy_leverage", fmt.Sprintf("%d", s.BuyLeverage))
	values.Set("sell_leverage", fmt.Sprintf("%d", s.SellLeverage))
	return values
}

type PrevFundingRateParam struct {
	Symbol string
}

func (p *PrevFundingRateParam) ToUrlValues() url.Values {
	values := url.Values{}
	values.Set("symbol", p.Symbol)
	return values
}

type BalanceParam struct {
	Coin string
}

func (b *BalanceParam) ToUrlValues() url.Values {
	values := url.Values{}
	if b.Coin != "" {
		values.Set("coin", b.Coin)
	}
	return values
}

type NewOrderParam struct {
	Side           string
	Symbol         string
	OrderType      string
	Qty            float64
	Price          float64
	TimeInForce    string
	ReduceOnly     bool
	CloseOnTrigger bool
	OrderLinkID    string
}

func (o *NewOrderParam) ToUrlValues() url.Values {
	values := url.Values{}
	values.Set("side", o.Side)
	values.Set("symbol", o.Symbol)
	values.Set("order_type", o.OrderType)
	values.Set("qty", fmt.Sprintf("%.8f", o.Qty))
	if o.Price != 0 {
		values.Set("price", fmt.Sprintf("%.8f", o.Qty))
	}
	if o.TimeInForce == "" {
		values.Set("time_in_force", TimeInForceGoodTillCancel)
	} else {
		values.Set("time_in_force", o.TimeInForce)
	}
	if o.ReduceOnly {
		values.Set("reduce_only", "true")
	} else {
		values.Set("reduce_only", "false")
	}
	if o.CloseOnTrigger {
		values.Set("close_on_trigger", "true")
	} else {
		values.Set("close_on_trigger", "false")
	}
	if o.OrderLinkID != "" {
		values.Set("order_link_id", o.OrderLinkID)
	}
	return values
}

type CancelAllParam struct {
	Symbol string
}

func (ca *CancelAllParam) ToUrlValues() url.Values {
	values := url.Values{}
	values.Set("symbol", ca.Symbol)
	return values
}

type WSRequest struct {
	Op   string   `json:"op"`
	Args []string `json:"args,omitempty"`
}
