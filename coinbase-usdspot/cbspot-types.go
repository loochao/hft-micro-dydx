package coinbase_usdspot

import (
	"encoding/json"
	"github.com/geometrybase/hft-micro/common"
	"time"
)

type DataCap struct {
}

type Channel struct {
	Name       string   `json:"name"`
	ProductIDs []string `json:"product_ids"`
}

type Request struct {
	Type     string    `json:"type"`
	Channels []Channel `json:"channels"`
}

//{
//    "type": "match",
//    "trade_id": 10,
//    "sequence": 50,
//    "maker_order_id": "ac928c66-ca53-498f-9c13-a110027a60e8",
//    "taker_order_id": "132fb6ae-456b-4654-b4e0-d681ac05cea1",
//    "time": "2014-11-07T08:19:27.028459Z",
//    "product_id": "BTC-USD",
//    "size": "5.23512",
//    "price": "400.23",
//    "side": "sell"
//}

type Match struct {
	Type         string    `json:"type"`
	TradeID      int64     `json:"trade_id"`
	Sequence     int64     `json:"sequence"`
	MakerOrderId string    `json:"maker_order_id"`
	TakerOrderId string    `json:"taker_order_id"`
	Time         time.Time `json:"-"`
	ProductId    string    `json:"product_id"`
	Size         float64   `json:"-"`
	Price        float64   `json:"-"`
	Side         string    `json:"side"`
}

func (match *Match) UnmarshalJSON(data []byte) error {
	type Alias Match
	aux := struct {
		Time  string          `json:"time"`
		Size  json.RawMessage `json:"size"`
		Price json.RawMessage `json:"price"`
		*Alias
	}{Alias: (*Alias)(match)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	} else {
		match.Time, err = time.Parse("2006-01-02T15:04:05.999999Z", aux.Time)
		if err != nil {
			return err
		}
		match.Price, err = common.ParseFloat(aux.Price[1 : len(aux.Price)-1])
		if err != nil {
			return err
		}
		match.Size, err = common.ParseFloat(aux.Size[1 : len(aux.Size)-1])
		if err != nil {
			return err
		}
		return nil
	}
}

var MatchMakerSideSell = "sell"
var MatchMakerSideBuy = "buy"

func (match *Match) GetSymbol() string  { return match.ProductId }
func (match *Match) GetSize() float64   { return match.Size }
func (match *Match) GetPrice() float64  { return match.Price }
func (match *Match) GetTime() time.Time { return match.Time }
func (match *Match) IsUpTick() bool     { return match.Side == MatchMakerSideSell }

//{
//    "id": "OMG-EUR",
//    "base_currency": "OMG",
//    "quote_currency": "EUR",
//    "base_min_size": "1",
//    "base_max_size": "500000",
//    "quote_increment": "0.0001",
//    "base_increment": "0.1",
//    "display_name": "OMG/EUR",
//    "min_market_funds": "1",
//    "max_market_funds": "100000",
//    "margin_enabled": false,
//    "fx_stablecoin": false,
//    "post_only": false,
//    "limit_only": false,
//    "cancel_only": false,
//    "trading_disabled": false,
//    "status": "online",
//    "status_message": ""
//}

type Product struct {
	ID              string  `json:"id"`
	BaseCurrency    string  `json:"base_currency"`
	QuoteCurrency   string  `json:"quote_currency"`
	BaseMinSize     float64 `json:"base_min_size,string"`
	BaseMaxSize     float64 `json:"base_max_size,string"`
	QuoteIncrement  float64 `json:"quote_increment,string"`
	BaseIncrement   float64 `json:"base_increment,string"`
	DisplayName     string  `json:"display_name"`
	MinMarketFunds  float64 `json:"min_market_funds,string"`
	MaxMarketFunds  float64 `json:"max_market_funds,string"`
	MarginEnabled   bool    `json:"margin_enabled"`
	FxStableCoin    bool    `json:"fx_stablecoin"`
	PostOnly        bool    `json:"post_only"`
	LimitOnly       bool    `json:"limit_only"`
	CancelOnly      bool    `json:"cancel_only"`
	TradingDisabled bool    `json:"trading_disabled"`
	Status          string  `json:"status"`
	StatusMessage   string  `json:"status_message"`
}
