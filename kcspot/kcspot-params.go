package kcspot

import (
	"fmt"
	"net/url"
	"strconv"
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
	ClientOid   string
	Side        string
	Symbol      string
	Type        string
	Remark      string
	Price       float64
	Size        float64
	TimeInForce string
	CancelAfter int
	PostOnly    bool
}

func (cp *NewOrderParam) ToUrlValues() url.Values {
	values := url.Values{}
	values.Set("clientOid", cp.ClientOid)
	values.Set("side", cp.Side)
	values.Set("symbol", cp.Symbol)
	values.Set("type", cp.Type)
	if cp.Remark != "" {
		values.Set("remark", cp.Remark)
	}
	values.Set("size", strconv.FormatFloat(cp.Size, 'f', 8, 64))
	if cp.Type == OrderTypeLimit {
		values.Set("quantity", strconv.FormatFloat(cp.Price, 'f', 8, 64))
	}
	if cp.TimeInForce != "" {
		values.Set("timeInForce", cp.TimeInForce)
	}
	if cp.TimeInForce == OrderTimeInForceGTT {
		values.Set("cancelAfter", fmt.Sprintf("%d", cp.CancelAfter))
	}
	if cp.PostOnly && cp.TimeInForce != OrderTimeInForceFOK && cp.TimeInForce != OrderTimeInForceIOC {
		values.Set("postOnly", strconv.FormatBool(cp.PostOnly))
	}
	return values
}

type CancelAllOrdersParam struct {
	Symbol      string
}

func (cp *CancelAllOrdersParam) ToUrlValues() url.Values {
	values := url.Values{}
	values.Set("symbol", cp.Symbol)
	return values
}
