package hbcrossswap

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
	KlinePeriod1mon  = "1mon"

	OrderDirectionBuy  = "buy"
	OrderDirectionSell = "sell"
	OrderOffsetOpen    = "open"
	OrderOffsetClose   = "close"

	OrderPriceTypeLimit           = "limit"
	OrderPriceTypeOpponent        = "opponent"
	OrderPriceTypePostOnly        = "post_only"
	OrderPriceTypeOptimal5        = "optimal_5"
	OrderPriceTypeOptimal10       = "optimal_10"
	OrderPriceTypeOptimal20       = "optimal_20"
	OrderPriceTypeIOC             = "ioc"
	OrderPriceTypeFOK             = "fok"
	OrderPriceTypeOpponentIOC     = "opponent_ioc"
	OrderPriceTypeOptimal5IOC     = "optimal_5_ioc"
	OrderPriceTypeOptimal10IOC    = "optimal_10_ioc"
	OrderPriceTypeOptimal20IOC    = "optimal_20_ioc"
	OrderPriceTypeOpponentFOK     = "opponent_fok"
	OrderPriceTypeFOKOptimal5FOK  = "optimal_5_fok"
	OrderPriceTypeFOKOptimal10FOK = "optimal_5_fok"
	OrderPriceTypeFOKOptimal20FOK = "optimal_5_fok"

	OrderStatusSubmit                              = 1  //1. Placing orders to order book;
	OrderStatusSubmitting                          = 2  // 2 Placing orders to order book;
	OrderStatusSubmitted                           = 3  // 3. Placed to order book
	OrderStatusPartiallyFilled                     = 4  // 4. Partially filled;
	OrderStatusPartiallyFilledButCancelledByClient = 5  // 5 partially filled but cancelled by client;
	OrderStatusFilled                              = 6  // 6. Fully filled;
	OrderStatusCancelled                           = 7  // 7. Cancelled;
	OrderStatusCancelling                          = 11 // 11Cancelling
)

var KlinePeriodDuration = map[string]time.Duration{
	KlinePeriod1min:  time.Minute,
	KlinePeriod5min:  time.Minute * 5,
	KlinePeriod15min: time.Minute * 15,
	KlinePeriod30min: time.Minute * 30,
	KlinePeriod60min: time.Minute * 60,
	KlinePeriod4hour: time.Hour * 4,
	KlinePeriod1day:  time.Hour * 24,
	KlinePeriod1mon:  time.Hour * 24 * 30,
}

type KlinesParam struct {
	Symbol string
	Period string
	Size   int
	From   int64
	To     int64
}

func (p *KlinesParam) ToUrlValues() url.Values {
	values := url.Values{}
	values.Set("contract_code", p.Symbol)
	values.Set("period", p.Period)
	if p.Size > 0 {
		values.Set("size", fmt.Sprintf("%d", p.Size))
	}
	if p.From > 0 {
		values.Set("from", fmt.Sprintf("%d", p.From))
	}
	if p.To > 0 {
		values.Set("to", fmt.Sprintf("%d", p.To))
	}
	return values
}

type SubParam struct {
	Sub string `json:"sub"`
	ID  string `json:"id"`
}

type NewOrderParam struct {
	Symbol         string         `json:"contract_code"`
	ClientOrderID  string         `json:"client_order_id"`
	Price          common.Float64 `json:"price"`
	Volume         int64          `json:"volume"`
	Direction      string         `json:"direction"`
	Offset         string         `json:"offset"`
	LeverRate      int            `json:"lever_rate"`
	OrderPriceType string         `json:"order_price_type"`
}

type CancelAllParam struct {
	Symbol    string `json:"contract_code,omitempty"`
	Direction string `json:"direction,omitempty"`
	Offset    string `json:"offset,omitempty"`
}

type AuthenticationParam struct {
	Op               string `json:"op"`
	Type             string `json:"type"`
	AccessKeyId      string `json:"AccessKeyId"`
	SignatureMethod  string `json:"SignatureMethod"`
	SignatureVersion string `json:"SignatureVersion"`
	Timestamp        string `json:"Timestamp"`
	Signature        string `json:"Signature"`
}

type AccountSubParam struct {
	Op    string `json:"op"`
	CID   string `json:"cid,omitempty"`
	Topic string `json:"topic"`
}
