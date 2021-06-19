package okex_usdtspot

import (
	"encoding/json"
	"strconv"
	"time"
)

type CommonCapture struct {
	Event  *string          `json:"event,omitempty"`
	Table  *string          `json:"table,omitempty"`
	Action *string          `json:"action,omitempty"`
	Data   *json.RawMessage `json:"data,omitempty"`
}

type Subscription struct {
	Op   string   `json:"op"`
	Args []string `json:"args"`
}

type LoginEvent struct {
	Event   string `json:"event,omitempty"`
	Success bool   `json:"success,omitempty"`
}

type SubscribeEvent struct {
	Event   string `json:"event,omitempty"`
	Channel string `json:"channel,omitempty"`
}

type ErrorEvent struct {
	Event     string `json:"event,omitempty"`
	Message   string `json:"message,omitempty"`
	ErrorCode int    `json:"errorCode,omitempty"`
}

type Candle struct {
	InstrumentID string   `json:"instrument_id"`
	Candle       []string `json:"candle,omitempty"` // [0]timestamp, [1]open, [2]high, [3]low, [4]close, [5]volume, [6]currencyVolume
}

//{
//    "table":"spot/ticker",
//    "data":[
//        {
//            "instrument_id":"ETH-USDT",
//            "last":"146.24",
//            "last_qty":"0.082483",
//            "best_bid":"146.24",
//            "best_bid_size":"0.006822",
//            "best_ask":"146.25",
//            "best_ask_size":"80.541709",
//            "open_24h":"147.17",
//            "high_24h":"147.48",
//            "low_24h":"143.88",
//            "open_utc0": "34067.1",
//            "open_utc8": "33830.9",
//            "base_volume_24h":"117387.58",
//            "quote_volume_24h":"17159427.21",
//            "timestamp":"2019-12-11T02:31:40.436Z"
//        }
//    ]
//}
type WSTicker struct {
	InstrumentID   string    `json:"instrument_id,omitempty"`
	Last           float64   `json:"last,string,omitempty"`
	LastQty        float64   `json:"last_qty,string,omitempty"`
	BestBid        float64   `json:"best_bid,string,omitempty"`
	BestBidSize    float64   `json:"best_bid_size,string,omitempty"`
	BestAsk        float64   `json:"best_ask,string,omitempty"`
	BestAskSize    float64   `json:"best_ask_size,string,omitempty"`
	Open24h        float64   `json:"open_24h,string,omitempty"`
	High24h        float64   `json:"high_24h,string,omitempty"`
	Low24h         float64   `json:"low_24h,string,omitempty"`
	OpenUTC0       float64   `json:"open_utc0,string,omitempty"`
	OpenUTC8       float64   `json:"open_utc8,string,omitempty"`
	BaseVolume24h  float64   `json:"base_volume_24h,string,omitempty"`
	QuoteVolume24h float64   `json:"quote_volume_24h,string,omitempty"`
	Timestamp      time.Time `json:"timestamp,string,omitempty"`
}

//{
//    "table": "spot/trade",
//    "data": [{
//        "instrument_id": "ETH-USDT",
//        "price": "162.12",
//        "side": "buy",
//        "size": "11.085",
//        "timestamp": "2019-05-06T06:51:24.389Z",
//        "trade_id": "1210447366"
//    }]
//}

type WSTrade struct {
	InstrumentID string    `json:"instrument_id,omitempty"`
	Price        float64   `json:"price,string,omitempty"`
	Side         string    `json:"side,omitempty"`
	Size         float64   `json:"size,string,omitempty"`
	TradeId      int64     `json:"trade_id,string,omitempty"`
	Timestamp    time.Time `json:"timestamp,string,omitempty"`
}



//order_id	String	Order ID
//client_oid	String	Client supplied order ID
//price	String	Price
//size	String	Size of the order in the unit of the base currency
//notional	String	The amount allocated for buying. Returned for market orders
//instrument_id	String	Trading pair
//side	String	Buy or sell
//type	String	limit,market(defaulted as limit)
//timestamp	String	Time of order being updated
//filled_size	String	Quantity of order filled
//filled_notional	String	Amount of order filled
//margin_trading	String	1 spot order. 2 margin order
//order_type	String	0: Normal limit order 1: Post only 2: Fill Or Kill 3: Immediatel Or Cancel
//last_fill_px	String	Latest Filled Price. '0' will be returned if the data is empty
//last_fill_id	String	Trade id. '0' will be returned if the data is empty
//last_fill_qty	String	Latest Filled Volume. '0' will be returned if the data is empty.
//last_fill_time	String	Latest Filled Time. The '1970-01-01T00:00:00.000Z' will be returned if the data is empty.
//state	String	Order Status(-2:Failed,-1:Canceled,0:Open ,1:Partially Filled, 2:Fully Filled,3:Submitting,4:Cancelling,）
//created_at	String	Time of order being created
//fee_currency	String	Transaction fee currency, if it is buy, it is charged in BTC; if it is sell, it is charged in USDT
//fee	String	Order transaction fee. The transaction fees charged by the platform to users are negative, for example: -0.01
//rebate_currency	String	Anti-commission currency， e.g：USDT
//rebate	String	Anti-commission amount, the platform pays rewards (rebates) to users who reach the specified lv transaction level. If there is no rebates, this field is "", which is a positive number, for example: 0.5
//last_request_id	String	request_id for latest order amendment（"" if no order amendment）
//last_amend_result	String	result for latest order amendment，-1： Failed，0： Success，1：Auto Cancel (due to order amendment) 1. If the API user's order cancel_on_fail is set to 0 or web/APP cancels the order, after the modification fails, last_amend_result will return "-1" 2. If the API user's order cancel_on_fail is set to 1, after the modification fails, last_amend_result returns "1"
//event_code	String	event code(the default is 0)
//event_message	String	event message(the default is "")

//      {
//            "client_oid":"",
//            "filled_notional":"0",
//            "filled_size":"0",
//            "instrument_id":"ETC-USDT",
//            "last_fill_px":"0",
//            "last_fill_qty":"0",
//            "last_fill_time":"1970-01-01T00:00:00.000Z",
//            "margin_trading":"1",
//            "notional":"",
//            "order_id":"3576398568830976",
//            "order_type":"0",
//            "price":"5.826",
//            "side":"buy",
//            "size":"0.1",
//            "state":"0",
//            "status":"open",
//            "timestamp":"2019-09-24T06:45:11.394Z",
//            "type":"limit",
//            "created_at":"2019-09-24T06:45:11.394Z"
//        }

type WSOrder struct {
	ClientOID       string    `json:"client_oid,omitempty"`
	OrderId         string    `json:"order_id,omitempty"`
	Price           float64   `json:"-"`
	Size            float64   `json:"-"`
	Notional        float64   `json:"-"`
	Symbol    string    `json:"instrument_id,omitempty"`
	Side            string    `json:"side,omitempty"`
	Type            string    `json:"type,omitempty"`
	Timestamp       time.Time `json:"timestamp,string,omitempty"`
	FilledSize      float64   `json:"filled_size,string,omitempty"`
	FilledNotional  float64   `json:"filled_notional,string,omitempty"`
	MarginTrading   string    `json:"margin_trading,omitempty"`
	OrderType       string    `json:"order_type,omitempty"`
	LastFillPrice   float64   `json:"last_fill_px,string,omitempty"`
	LastFillId      string    `json:"last_fill_id,omitempty"`
	LastFillQty     float64   `json:"last_fill_qty,string,omitempty"`
	LastFillTime    time.Time `json:"last_fill_time,string,omitempty"`
	State           string    `json:"state,omitempty"`
	CreatedAt       time.Time `json:"created_at,string,omitempty"`
	Fee             float64   `json:"-"`
	FeeCurrency     string    `json:"fee_currency,omitempty"`
	Rebate          float64   `json:"-"`
	RebateCurrency  string    `json:"rebate_currency,omitempty"`
	LastRequestId   string    `json:"last_request_id,omitempty"`
	LastAmendResult string    `json:"last_amend_result,omitempty"`
	EventCode       string    `json:"event_code,omitempty"`
	EventMessage    string    `json:"event_message,omitempty"`
}

func (order *WSOrder) UnmarshalJSON(data []byte) error {
	type Alias WSOrder
	aux := &struct {
		Price    *string `json:"price,omitempty"`
		Size     *string `json:"size,omitempty"`
		Notional *string `json:"notional,omitempty"`
		Rebate   *string `json:"rebate,omitempty"`
		Fee      *string `json:"fee,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(order),
	}
	var err error
	if err = json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.Price != nil && *aux.Price != "" {
		order.Price, err = strconv.ParseFloat(*aux.Price, 64)
		if err != nil {
			return err
		}
	}
	if aux.Size != nil && *aux.Size != "" {
		order.Size, err = strconv.ParseFloat(*aux.Size, 64)
		if err != nil {
			return err
		}
	}
	if aux.Notional != nil && *aux.Notional != "" {
		order.Notional, err = strconv.ParseFloat(*aux.Notional, 64)
		if err != nil {
			return err
		}
	}
	if aux.Rebate != nil && *aux.Rebate != "" {
		order.Rebate, err = strconv.ParseFloat(*aux.Rebate, 64)
		if err != nil {
			return err
		}
	}
	if aux.Fee != nil && *aux.Fee != "" {
		order.Fee, err = strconv.ParseFloat(*aux.Fee, 64)
		if err != nil {
			return err
		}
	}
	return nil
}

