package okspot

import (
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"strconv"
	"time"
)

var (
	TradeSideSell = "sell"
	TradeSideBuy  = "buy"
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

type NewOrderParam struct {
	Symbol    string   `json:"instrument_id,omitempty"`
	ClientOID string   `json:"client_oid,omitempty"`
	Type      string   `json:"type"`
	Side      string   `json:"side"`
	OrderType string   `json:"order_type"`
	Price     *float64 `json:"price,string,omitempty"`
	Size      *float64 `json:"size,string,omitempty"`
	Notional  *float64 `json:"notional,string,omitempty"`
}

type OrderResponse struct {
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

type Depth5 struct {
	Symbol    string        `json:"-"`
	Bids      [5][2]float64 `json:"-"`
	Asks      [5][2]float64 `json:"_"`
	ParseTime time.Time     `json:"-"`
	EventTime time.Time     `json:"-"`
}

func (depth *Depth5) GetBids() [5][2]float64 { return depth.Bids }
func (depth *Depth5) GetAsks() [5][2]float64 { return depth.Asks }
func (depth *Depth5) GetSymbol() string      { return depth.Symbol }
func (depth *Depth5) GetTime() time.Time     { return depth.EventTime }
func (depth *Depth5) UnmarshalJSON(data []byte) error {
	type Alias Depth5
	aux := struct {
		Data []struct {
			Bids      [5][3]string `json:"bids"`
			Asks      [5][3]string `json:"asks"`
			EventTime string       `json:"timestamp"`
			Symbol    string       `json:"instrument_id"`
		} `json:"data"`
		Table string `json:"table"`
	}{}
	var err error
	if err = json.Unmarshal(data, &aux); err != nil {
		logger.Debugf("ERR %v", err)
		return err
	}
	if len(aux.Data) != 1 {
		return fmt.Errorf("bad deth5 format %s", data)
	}
	depth.Bids = [5][2]float64{}
	depth.Asks = [5][2]float64{}
	for i, d := range aux.Data[0].Bids {
		price, _ := strconv.ParseFloat(d[0], 64)
		size, _ := strconv.ParseFloat(d[1], 64)
		depth.Bids[i][0] = price
		depth.Bids[i][1] = size
	}
	for i, d := range aux.Data[0].Asks {
		price, _ := strconv.ParseFloat(d[0], 64)
		size, _ := strconv.ParseFloat(d[1], 64)
		depth.Asks[i][0] = price
		depth.Asks[i][1] = size
	}
	depth.Symbol = aux.Data[0].Symbol
	depth.EventTime, err = time.Parse(okspotTimeLayout, aux.Data[0].EventTime)
	depth.ParseTime = time.Now()
	return nil
}

type CancelOrderParam struct {
	Symbol    string `json:"instrument_id"`
	ClientOid string `json:"client_oid,omitempty"`
	OrderId   string `json:"order_id,omitempty"`
}

type CancelBatchOrders struct {
	Symbol string `json:"instrument_id"`
}

//{
//        "title": "Spot System Optimization",
//        "href": "",
//        "product_type": "1",
//        "status": "2",
//        "maint_type": "upgrade",
//        "sche_desc": "",
//        "start_time": "2020-04-10T04:30:00.000Z",
//        "end_time": "2020-04-10T04:40:00.000Z"
//    }

type Status struct {
	Title        string `json:"title"`
	Href         string `json:"href"`
	ProductType  string `json:"product_type"`
	Status       string `json:"status"`
	MaintType    string `json:"maint_type"`
	ScheduleDesc string `json:"sche_desc"`
	StartTime    string `json:"start_time"`
	EndTime      string `json:"end_time"`
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

type WSTrades struct {
	Table string  `json:"table"`
	Data  []Trade `json:"data"`
}

type Trade struct {
	Symbol    string    `json:"instrument_id"`
	Price     float64   `json:"-"`
	Side      string    `json:"side"`
	Size      float64   `json:"-"`
	TradeId   int64     `json:"-"`
	EventTime time.Time `json:"-"`
}

func (trade *Trade) UnmarshalJSON(data []byte) error {
	type Alias Trade
	aux := struct {
		Price     json.RawMessage `json:"price"`
		Size      json.RawMessage `json:"size"`
		TradeId   json.RawMessage `json:"trade_id"`
		Timestamp string          `json:"timestamp"`
		*Alias
	}{
		Alias: (*Alias)(trade),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	} else {
		trade.EventTime, err = time.Parse(okspotTimeLayout, aux.Timestamp)
		if err != nil {
			return err
		}
		trade.TradeId, err = common.ParseInt(aux.TradeId[1 : len(aux.TradeId)-1])
		if err != nil {
			return err
		}
		trade.Price, err = common.ParseFloat(aux.Price[1 : len(aux.Price)-1])
		if err != nil {
			return err
		}
		trade.Size, err = common.ParseFloat(aux.Size[1 : len(aux.Size)-1])
		if err != nil {
			return err
		}
		return nil
	}
}

func (trade *Trade) GetSymbol() string  { return trade.Symbol }
func (trade *Trade) GetSize() float64   { return trade.Size }
func (trade *Trade) GetPrice() float64  { return trade.Price }
func (trade *Trade) GetTime() time.Time { return trade.EventTime }
func (trade *Trade) IsBuy() bool { return trade.Side == TradeSideBuy }

