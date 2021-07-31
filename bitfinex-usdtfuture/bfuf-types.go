package bitfinex_usdtfuture

import (
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"net/http"
	"time"
)

type Response struct {
	Response *http.Response
	Body     []byte
}

type ErrorResponse struct {
	Response *Response
	Message  string `json:"message"`
	Code     int    `json:"code"`
}

func (r *ErrorResponse) Error() string {
	return fmt.Sprintf("%v %v: %d %v (%d)",
		r.Response.Response.Request.Method,
		r.Response.Response.Request.URL,
		r.Response.Response.StatusCode,
		r.Message,
		r.Code,
	)
}

type Pair struct {
	Symbol        string
	MinOrderSize  float64
	MaxOrderSize  float64
	initialMargin float64
	minMargin     float64
}

type WSRequest struct {
	Event   string `json:"event"`
	Channel string `json:"channel"`
	Symbol  string `json:"symbol"`
}

type WSRequestBook struct {
	Event   string `json:"event"`
	Channel string `json:"channel"`
	Symbol  string `json:"symbol"`
	Prec    string `json:"prec"`
	Freq    string `json:"freq"`
	Len     string `json:"len"`
}

//{"event":"subscribed","channel":"ticker","chanId":639602,"symbol":"tBTCF0:USTF0","pair":"BTCF0:USTF0"}
type SubscribeEvent struct {
	Event   string `json:"event"`
	Channel string `json:"channel"`
	ChanId  int64  `json:"chanId"`
	Symbol  string `json:"symbol"`
	Pair    string `json:"pair"`
}

type Ticker struct {
	Symbol    string
	Bid       float64
	BidSize   float64
	Ask       float64
	AskSize   float64
	ParseTime time.Time
}

func (t *Ticker) GetSymbol() string {
	return t.Symbol
}

func (t *Ticker) GetTime() time.Time {
	return t.ParseTime
}

func (t *Ticker) GetBidPrice() float64 {
	return t.Bid
}

func (t *Ticker) GetAskPrice() float64 {
	return t.Ask
}

func (t *Ticker) GetBidSize() float64 {
	return t.BidSize
}

func (t *Ticker) GetAskSize() float64 {
	return t.AskSize
}

func (t *Ticker) GetExchange() common.ExchangeID {
	return ExchangeID
}
