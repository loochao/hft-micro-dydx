package bybit_usdtfuture

import (
	"encoding/json"
	"github.com/geometrybase/hft-micro/common"
	"time"
)

type PriceFilter struct {
	MinPrice float64 `json:"min_price,string"`
	MaxPrice float64 `json:"max_price,string"`
	TickSize float64 `json:"tick_size,string"`
}

type LotSizeFilter struct {
	MaxTradingQty float64 `json:"max_trading_qty"`
	MinTradingQty float64 `json:"min_trading_qty"`
	QtyStep       float64 `json:"qty_step"`
}

type LeverageFilter struct {
	MinLeverage  float64 `json:"min_leverage"`
	MaxLeverage  float64 `json:"max_leverage"`
	LeverageStep float64 `json:"leverage_step,string"`
}

type Symbol struct {
	Name           string         `json:"name"`
	Alias          string         `json:"alias"`
	Status         string         `json:"status"`
	BaseCurrency   string         `json:"base_currency"`
	QuoteCurrency  string         `json:"quote_currency"`
	PriceScale     float64        `json:"price_scale"`
	TakerFee       float64        `json:"taker_fee,string"`
	MakerFee       float64        `json:"maker_fee,string"`
	LeverageFilter LeverageFilter `json:"leverage_filter"`
	PriceFilter    PriceFilter    `json:"price_filter"`
	LotSizeFilter  LotSizeFilter  `json:"lot_size_filter"`
}

type ResponseCap struct {
	RetCode int64           `json:"ret_code"`
	RetMsg  string          `json:"ret_msg"`
	ExtCode string          `json:"ext_code"`
	ExtInfo string          `json:"ext_info"`
	Result  json.RawMessage `json:"result"`
}

type OrderBookLevel struct {
	Price  float64 `json:"price,string"`
	Symbol string  `json:"symbol"`
	ID     int64   `json:"id,string"`
	Side   string  `json:"side"`
	Size   float64 `json:"size"`
}

type OrderBookData struct {
	OrderBook      []OrderBookLevel `json:"order_book"`
	Update         []OrderBookLevel `json:"update"`
	Insert         []OrderBookLevel `json:"insert"`
	Delete         []OrderBookLevel `json:"delete"`
	TransactTimeE6 int64            `json:"transactTimeE6"`
}

type OrderBookMsg struct {
	Topic       string        `json:"topic"`
	Type        string        `json:"type"`
	Data        OrderBookData `json:"data"`
	CrossSeq    int64         `json:"cross_seq,string"`
	TimestampE6 int64         `json:"timestamp_e6,string"`
}

type OrderBook struct {
	Bids      common.Bids
	Asks      common.Asks
	EventTime time.Time
	ParseTime time.Time
	Symbol    string
}

func (o OrderBook) IsValidate() bool {
	if len(o.Asks) > 25 || len(o.Bids) > 25 {
		return false
	}
	if len(o.Asks) > 0 && len(o.Bids) > 0 && o.Asks[0][0] <= o.Bids[0][0] {
		return false
	}
	for i := 0; i < len(o.Asks)-1; i ++ {
		if o.Asks[i][0] >= o.Asks[i+1][0] {
			return false
		}
	}
	for i := 0; i < len(o.Bids)-1; i ++ {
		if o.Bids[i][0] <= o.Bids[i+1][0] {
			return false
		}
	}
	return true
}

func (o OrderBook) GetTime() time.Time {
	return o.EventTime
}

func (o OrderBook) GetAsks() common.Asks {
	return o.Asks
}

func (o OrderBook) GetBids() common.Bids {
	return o.Bids
}

func (o OrderBook) GetSymbol() string {
	return o.Symbol
}

func (o OrderBook) GetExchange() common.ExchangeID {
	return ExchangeID
}

type SubscribeParam struct {
	Op   string   `json:"op"`
	Args []string `json:"args"`
}
