package kucoin_usdtfuture

import (
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"net/url"
)

const (
	AccountTypeMain     = "main"
	AccountTypeTrade    = "trade"
	AccountTypeMargin   = "margin"
	AccountTypePool     = "pool"
	OrderSideBuy        = "buy"
	OrderSideSell       = "sell"
	OrderTypeLimit      = "limit"
	OrderTypeMarket     = "market"
	OrderTimeInForceGTC = "GTC"
	OrderTimeInForceIOC = "IOC"
	ExchangeID = common.KucoinUsdtFuture
)

type KlinesParam struct {
	Symbol      string
	From        int64
	To          int64
	Granularity int
}

func (cp *KlinesParam) ToUrlValues() url.Values {
	values := url.Values{}
	values.Set("symbol", cp.Symbol)
	values.Set("granularity", fmt.Sprintf("%d", cp.Granularity))
	if cp.From > 0 {
		values.Set("from", fmt.Sprintf("%d", cp.From))
	}
	if cp.To > 0 {
		values.Set("to", fmt.Sprintf("%d", cp.To))
	}
	return values
}

type SubscribeMsg struct {
	ID             string `json:"id"`
	Type           string `json:"type"`
	Topic          string `json:"topic"`
	PrivateChannel bool   `json:"privateChannel"`
	Response       bool   `json:"response"`
}
type AccountParam struct {
	Currency string
	Type     string
}

func (cp *AccountParam) ToUrlValues() url.Values {
	values := url.Values{}
	if cp.Currency != "" {
		values.Set("currency", cp.Currency)
	}
	if cp.Type != "" {
		values.Set("type", cp.Type)
	}
	return values
}

type AccountOverviewParam struct {
	Currency string
}

func (cp *AccountOverviewParam) ToUrlValues() url.Values {
	values := url.Values{}
	if cp.Currency != "" {
		values.Set("currency", cp.Currency)
	}
	return values
}

type NewOrderParam struct {
	ClientOid   string         `json:"clientOid,omitempty"`
	Side        string         `json:"side,omitempty"`
	Symbol      string         `json:"symbol,omitempty"`
	Type        string         `json:"type,omitempty"`
	Leverage    int            `json:"leverage,omitempty"`
	Remark      string         `json:"remark,omitempty"`
	Price       common.Float64 `json:"price,omitempty"`
	Size        int64          `json:"size,omitempty"`
	ReduceOnly  bool           `json:"reduceOnly,omitempty"`
	CloseOrder  bool           `json:"closeOrder,omitempty"`
	PostOnly    bool           `json:"postOnly,omitempty"`
	TimeInForce string         `json:"timeInForce,omitempty"`
}

type CancelAllOrdersParam struct {
	Symbol string
}

func (cp *CancelAllOrdersParam) ToUrlValues() url.Values {
	values := url.Values{}
	values.Set("symbol", cp.Symbol)
	return values
}

type AutoDepositStatusParam struct {
	Symbol string `json:"symbol"`
	Status bool   `json:"status"`
}

type TickerParam struct {
	Symbol string
}

func (t *TickerParam) ToUrlValues() url.Values {
	values := url.Values{}
	if t.Symbol != "" {
		values.Set("symbol", t.Symbol)
	}
	return values
}
