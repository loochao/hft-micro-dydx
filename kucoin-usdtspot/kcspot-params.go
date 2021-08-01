package kucoin_usdtspot

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
	OrderTimeInForceGTT = "GTT"
	OrderTimeInForceIOC = "IOC"
	OrderTimeInForceFOK = "FOK"
)

type CandlesParam struct {
	Symbol  string
	StartAt int64
	EndAt   int64
	Type    string
}

func (cp *CandlesParam) ToUrlValues() url.Values {
	values := url.Values{}
	values.Set("symbol", cp.Symbol)
	if cp.StartAt > 0 {
		values.Set("startAt", fmt.Sprintf("%d", cp.StartAt))
	}
	if cp.EndAt > 0 {
		values.Set("endAt", fmt.Sprintf("%d", cp.EndAt))
	}
	values.Set("type", cp.Type)
	return values
}

type SubscribeMsg struct {
	ID             string `json:"id"`
	Type           string `json:"type"`
	Topic          string `json:"topic"`
	PrivateChannel bool   `json:"privateChannel"`
	Response       bool   `json:"response"`
}
type AccountsParam struct {
	Currency string
	Type     string
}

func (cp *AccountsParam) ToUrlValues() url.Values {
	values := url.Values{}
	if cp.Currency != "" {
		values.Set("currency", cp.Currency)
	}
	if cp.Type != "" {
		values.Set("type", cp.Type)
	}
	return values
}

type AccountParam struct {
	ID string
}

func (cp *AccountParam) ToUrlValues() url.Values {
	values := url.Values{}
	values.Set("id", cp.ID)
	return values
}

type NewOrderParam struct {
	ClientOid   string         `json:"clientOid,omitempty"`
	Side        string         `json:"side,omitempty"`
	Symbol      string         `json:"symbol,omitempty"`
	Type        string         `json:"type,omitempty"`
	Remark      string         `json:"remark,omitempty"`
	Price       common.Float64 `json:"price,omitempty"`
	Size        common.Float64 `json:"size,omitempty"`
	TimeInForce string         `json:"timeInForce,omitempty"`
	CancelAfter int            `json:"cancelAfter,omitempty"`
	PostOnly    bool           `json:"postOnly,omitempty"`
	Hidden      bool           `json:"hidden,omitempty"`
}

type CancelAllOrdersParam struct {
	Symbol string
}

func (cp *CancelAllOrdersParam) ToUrlValues() url.Values {
	values := url.Values{}
	values.Set("symbol", cp.Symbol)
	return values
}


type TickerParam struct {
	Symbol string
}

func (tp *TickerParam) ToUrlValues() url.Values {
	values := url.Values{}
	values.Set("symbol", tp.Symbol)
	return values
}