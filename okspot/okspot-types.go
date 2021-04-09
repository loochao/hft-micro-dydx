package okspot

import (
	"fmt"
	"time"
)

type Credentials struct {
	Key        string
	Secret     string
	Passphrase string
}

type ErrorCap struct {
	Code         int64  `json:"code,omitempty"`
	Message      string `json:"message,omitempty"`
	ErrorCode    int64  `json:"error_code,omitempty,string"`
	ErrorMessage string `json:"error_message,omitempty"`
	Result       bool   `json:"result,string,omitempty"`
}

//{
//    "frozen":"0",
//    "hold":"0",
//    "id": "",
//    "currency":"BTC",
//    "balance":"0.0049925",
//    "available":"0.0049925",
//    "holds":"0"
//}

type Balance struct {
	Id        string  `json:"id"`
	Currency  string  `json:"currency"`
	Balance   float64 `json:"balance,string"`
	Available float64 `json:"available,string"`
	Hold      float64 `json:"hold,string"`
}

func (b *Balance) ToString() string {
	return fmt.Sprintf(
		"%s Balance=%f, Available=%f, Hold=%f",
		b.Currency,
		b.Balance,
		b.Available,
		b.Hold,
	)
}

//client_oid	String	No	You can customize order IDs to identify your orders. The system supports alphabets + numbers(case-sensitive，e.g:A123、a123), or alphabets (case-sensitive，e.g:Abc、abc) only, between 1-32 characters.
//type	String	No	Supports types limit or market (default: limit). When placing market orders, order_type must be 0 (normal order)
//side	String	Yes	Specify buy or sell
//instrument_id	String	Yes	Trading pair symbol
//order_type	String	No	Specify 0: Normal order (Unfilled and 0 imply normal limit order) 1: Post only 2: Fill or Kill 3: Immediate Or Cancel

type NewOrderParams struct {
	InstrumentId string   `json:"instrument_id,omitempty"`
	ClientOID    string   `json:"client_oid,omitempty"`
	Type         string   `json:"type"`
	Side         string   `json:"side"`
	OrderType    string   `json:"order_type"`
	Price        *float64 `json:"price,string,omitempty"`
	Size         *float64 `json:"size,string,omitempty"`
	Notional     *float64 `json:"notional,string,omitempty"`
}

type NewOrderResponse struct {
	ClientOID    string `json:"client_oid,omitempty"`
	OrderId      string `json:"order_id,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
	ErrorCode    string `json:"error_code,omitempty"`
	Result       bool   `json:"result,omitempty"`
}

//instrument_id	String	Yes	Trading pairs symbol
//start	String	No	Start time in ISO 8601
//end	String	No	End time in ISO 8601
//granularity	String	No	Bar size in seconds, default 60, must be one of [60/180/300/900/1800/3600/7200/14400/21600/43200/86400/604800] or returns error
//limit	String	No	The number of candles returned, the default is 300，and the maximum is 300

type MarketDataParams struct {
	InstrumentId string     `json:"instrument_id"`
	Granularity  int64      `json:"granularity,string"`
	Start        *time.Time `json:"start,string,omitempty"`
	End          *time.Time `json:"end,string,omitempty"`
	Limit        *int64     `json:"limit,string"`
}

type MarketData [6]string

//    {
//        "base_currency":"BTC",
//        "instrument_id":"BTC-USDT",
//        "min_size":"0.001",
//        "quote_currency":"USDT",
//        "size_increment":"0.00000001",
//        "category":"1",
//        "tick_size":"0.1"
//    }

type Instrument struct {
	BaseCurrency  string  `json:"base_currency"`
	InstrumentId  string  `json:"instrument_id"`
	MinSize       float64 `json:"min_size,string"`
	QuoteCurrency string  `json:"quote_currency"`
	SizeIncrement float64 `json:"size_increment,string"`
	Category      int     `json:"category,string"`
	TickSize      float64 `json:"tick_size,string"`
}
