package hbspot

import (
	"fmt"
	"github.com/geometrybase/hft-micro/common"
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
	KlinePeriod1week = "1week"
	KlinePeriod1mon  = "1mon"
	KlinePeriod1year = "1year"

	OrderTypeBuyMarket        = "buy-market"
	OrderTypeSellMarket       = "sell-market"
	OrderTypeBuyLimit         = "buy-limit"
	OrderTypeSellLimit        = "sell-limit"
	OrderTypeBuyIOC           = "buy-ioc"
	OrderTypeSellIOC          = "sell-ioc"
	OrderTypeBuyLimitMaker    = "buy-limit-maker"
	OrderTypeSellLimitMaker   = "sell-limit-maker"
	OrderTypeBuyStopLimit     = "buy-stop-limit"
	OrderTypeSellStopLimit    = "sell-stop-limit"
	OrderTypeBuyLimitFOK      = "buy-limit-fok"
	OrderTypeSellLimitFOK     = "sell-limit-fok"
	OrderTypeBuyStopLimitFOK  = "buy-stop-limit-fok"
	OrderTypeSellStopLimitFOK = "sell-stop-limit-fok"

	OrderSideBuy  = "buy"
	OrderSideSell = "sell"

	OrderStatusRejected  = "rejected"
	OrderStatusCanceled  = "canceled"
	OrderStatusSubmitted = "submitted"
	OrderStatusFilled    = "filled"

	OrderEventTypeTrigger      = "trigger"
	OrderEventTypeDeletion     = "deletion"
	OrderEventTypeCreation     = "creation"
	OrderEventTypeTrade        = "trade"
	OrderEventTypeCancellation = "cancellation"
)

var KlinePeriodDuration = map[string]time.Duration{
	KlinePeriod1min:  time.Minute,
	KlinePeriod5min:  time.Minute * 5,
	KlinePeriod15min: time.Minute * 15,
	KlinePeriod30min: time.Minute * 30,
	KlinePeriod60min: time.Minute * 60,
	KlinePeriod4hour: time.Hour * 4,
	KlinePeriod1day:  time.Hour * 24,
	KlinePeriod1week: time.Hour * 24 * 7,
	KlinePeriod1mon:  time.Hour * 24 * 30,
	KlinePeriod1year: time.Hour * 24 * 365,
}

type KlinesParam struct {
	Symbol string
	Period string
	Size   int
}

func (p *KlinesParam) ToUrlValues() url.Values {
	values := url.Values{}
	values.Set("symbol", p.Symbol)
	values.Set("period", p.Period)
	if p.Size > 0 {
		values.Set("size", fmt.Sprintf("%d", p.Size))
	}
	return values
}

type SubParam struct {
	Sub string `json:"sub"`
	ID  string `json:"id"`
}

type NewOrderParam struct {
	AccountId       int64              `json:"account-id"`
	Symbol          string             `json:"symbol"`
	Type            string             `json:"type"`
	Amount          common.StringFloat `json:"amount,omitempty"`
	Price           common.StringFloat `json:"price,omitempty"`
	Source          string             `json:"source,omitempty"`
	ClientOrderID   string             `json:"client-order-id"`
	StopPrice       common.StringFloat `json:"stopPrice,omitempty"`
	Operator        string             `json:"operator,omitempty"`
	OriginPrice     float64            `json:"-"`
	OriginAmount    float64            `json:"-"`
	OriginStopPrice float64            `json:"-"`
}

type CancelAllParam struct {
	AccountId int64  `json:"accountId"`
	Symbol    string `json:"symbol"`
	Types     string `json:"types,omitempty"`
	Side      string `json:"side,omitempty"`
	Size      int    `json:"size,omitempty"`
}

type AuthenticationParam struct {
	Action string               `json:"action"`
	Ch     string               `json:"ch"`
	Params AuthenticationParams `json:"params"`
}

type AuthenticationParams struct {
	AuthType         string `json:"authType"`
	AccessKey        string `json:"accessKey"`
	SignatureMethod  string `json:"signatureMethod"`
	SignatureVersion string `json:"signatureVersion"`
	Timestamp        string `json:"timestamp"`
	Signature        string `json:"signature"`
}

type AccountSubParam struct {
	Action string `json:"action"`
	Ch     string `json:"ch"`
}
