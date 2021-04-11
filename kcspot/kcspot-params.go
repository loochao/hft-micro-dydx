package kcspot

import (
	"fmt"
	"net/url"
)

const (
	AccountTypeMain   = "main"
	AccountTypeTrade  = "trade"
	AccountTypeMargin = "margin"
	AccountTypePool   = "pool"
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
	values.Set("currency", cp.Currency)
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
