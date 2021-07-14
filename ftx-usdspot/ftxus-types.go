package ftx_usdspot

import (
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"hash/crc32"
	"math"
	"strconv"
	"time"
)

type Response struct {
	Success bool            `json:"success"`
	Result  json.RawMessage `json:"result"`
	Error   string          `json:"error"`
}

//    {
//      "name": "1INCH/USD",
//      "enabled": true,
//      "postOnly": false,
//      "priceIncrement": 0.0001,
//      "sizeIncrement": 1.0,
//      "minProvideSize": 1.0,
//      "last": 2.2855,
//      "bid": 2.2802,
//      "ask": 2.2831,
//      "price": 2.2831,
//      "type": "spot",
//      "baseCurrency": "1INCH",
//      "quoteCurrency": "USD",
//      "underlying": null,
//      "restricted": false,
//      "highLeverageFeeExempt": true,
//      "change1h": 0.001535357080189507,
//      "change24h": -0.06545231273024969,
//      "changeBod": -0.01709144136387119,
//      "quoteVolume24h": 112464.7293,
//      "volumeUsd24h": 112464.7293
//    }

type Market struct {
	Type                  string  `json:"type"`
	Name                  string  `json:"name"`
	Underlying            string  `json:"underlying"`
	BaseCurrency          string  `json:"baseCurrency"`
	QuoteCurrency         string  `json:"quoteCurrency"`
	Enabled               bool    `json:"enabled"`
	Ask                   float64 `json:"ask"`
	Bid                   float64 `json:"bid"`
	Last                  float64 `json:"last"`
	PostOnly              bool    `json:"postOnly"`
	PriceIncrement        float64 `json:"priceIncrement"`
	SizeIncrement         float64 `json:"sizeIncrement"`
	Restricted            bool  `json:"restricted"`
	Price                 float64 `json:"price"`
	HighLeverageFeeExempt bool    `json:"highLeverageFeeExempt"`
	Change24h             float64 `json:"change24h"`
	ChangeBod             float64 `json:"changeBod"`
	QuoteVolume24h        float64 `json:"quoteVolume24h"`
	VolumeUsd24h          float64 `json:"volumeUsd24h"`
}

type FundingRate struct {
	Future string    `json:"future"`
	Rate   float64   `json:"rate"`
	Time   time.Time `json:"-"`
}

func (fr *FundingRate) UnmarshalJSON(data []byte) error {
	type Alias FundingRate
	aux := &struct {
		Time string `json:"time"`
		*Alias
	}{
		Alias: (*Alias)(fr),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	} else {
		fr.Time, err = time.Parse(TimeLayout, aux.Time)
		if err != nil {
			return err
		}
	}
	return nil
}

type Trade struct {
	ID     int64     `json:"id"`
	Price  float64   `json:"price"`
	Size   float64   `json:"size"`
	Side   string    `json:"side"`
	Time   time.Time `json:"-"`
	Symbol string    `json:"-"`
}

func (trade *Trade) GetSymbol() string  { return trade.Symbol }
func (trade *Trade) GetSize() float64   { return trade.Size }
func (trade *Trade) GetPrice() float64  { return trade.Price }
func (trade *Trade) GetTime() time.Time { return trade.Time }
func (trade *Trade) IsUpTick() bool     { return trade.Side == TradeSideBuy }

func (trade *Trade) UnmarshalJSON(data []byte) error {
	type Alias Trade
	aux := &struct {
		Time string `json:"time"`
		*Alias
	}{
		Alias: (*Alias)(trade),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	} else {
		trade.Time, err = time.Parse(TimeLayout, aux.Time)
		if err != nil {
			return err
		}
	}
	return nil
}

type TradesData struct {
	Data []Trade `json:"data"`
}

type OrderBook struct {
	Action   string      `json:"action"`
	Time     time.Time   `json:"-"`
	Bids     common.Bids `json:"bids"`
	Asks     common.Asks `json:"asks"`
	Checksum uint32      `json:"checksum"`
	Market   string      `json:"-"`
}

func (orderBook *OrderBook) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (orderBook *OrderBook) UnmarshalJSON(data []byte) error {
	type Alias OrderBook
	aux := &struct {
		Time float64 `json:"time"`
		*Alias
	}{
		Alias: (*Alias)(orderBook),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	} else {
		orderBook.Time = time.Unix(0, int64(aux.Time*1000000000))
	}
	return nil
}
func (orderBook *OrderBook) GetSymbol() string {
	return orderBook.Market
}
func (orderBook *OrderBook) GetTime() time.Time {
	return orderBook.Time
}
func (orderBook *OrderBook) GetAsks() common.Asks {
	return orderBook.Asks
}
func (orderBook *OrderBook) GetBids() common.Bids {
	return orderBook.Bids
}
func (orderBook *OrderBook) FormatFloat(value float64) string {
	if math.Floor(value) == value {
		return strconv.FormatFloat(value, 'f', -1, 64) + ".0"
	} else if value == 0.000005 {
		return "5e-06"
	} else if value == 0.000001 {
		return "1e-06"
	} else if value == 0.00005 {
		return "5e-05"
	} else if value == 0.00001 {
		return "1e-05"
	} else {
		return strconv.FormatFloat(value, 'f', -1, 64)
	}
}

func (orderBook *OrderBook) CompareCheckSum() bool {
	return orderBook.GetCheckSum() == orderBook.Checksum
}

func (orderBook *OrderBook) GetCheckSum() uint32 {
	bidLen := len(orderBook.Bids)
	askLen := len(orderBook.Asks)
	if bidLen > 100 {
		bidLen = 100
	}
	if askLen > 100 {
		askLen = 100
	}
	str := ""
	for i := 0; i < bidLen && i < askLen; i++ {
		str += fmt.Sprintf(
			"%s:%s:%s:%s:",
			orderBook.FormatFloat(orderBook.Bids[i][0]),
			orderBook.FormatFloat(orderBook.Bids[i][1]),
			orderBook.FormatFloat(orderBook.Asks[i][0]),
			orderBook.FormatFloat(orderBook.Asks[i][1]),
		)
	}
	if bidLen > askLen {
		for i := askLen; i < bidLen; i++ {
			str += fmt.Sprintf(
				"%s:%s:",
				orderBook.FormatFloat(orderBook.Bids[i][0]),
				orderBook.FormatFloat(orderBook.Bids[i][1]),
			)
		}
	} else if askLen > bidLen {
		for i := bidLen; i < askLen; i++ {
			str += fmt.Sprintf(
				"%s:%s:",
				orderBook.FormatFloat(orderBook.Asks[i][0]),
				orderBook.FormatFloat(orderBook.Asks[i][1]),
			)
		}
	}
	if len(str) > 0 {
		str = str[:len(str)-1]
	}
	return crc32.ChecksumIEEE([]byte(str))
}

type OrderBookData struct {
	Data    OrderBook `json:"data"`
	Market  string    `json:"market"`
	Channel string    `json:"channel"`
	Type    string    `json:"type"`
}

//    {
//      "future": "DOGE-PERP",
//      "size": 184.0,
//      "side": "sell",
//      "netSize": -184.0,
//      "longOrderSize": 0.0,
//      "shortOrderSize": 0.0,
//      "cost": -97.672628,
//      "entryPrice": 0.5308295,
//      "unrealizedPnl": 0.0,
//      "realizedPnl": -0.02857,
//      "initialMarginRequirement": 0.33333333,
//      "maintenanceMarginRequirement": 0.03,
//      "openSize": 184.0,
//      "collateralUsed": 32.55754234109124,
//      "estimatedLiquidationPrice": 3.47169418045141
//    }

type Position struct {
	Market                       string    `json:"future"`
	Size                         float64   `json:"size"`
	Side                         string    `json:"side"`
	NetSize                      float64   `json:"netSize"`
	LongOrderSize                float64   `json:"longOrderSize"`
	ShortOrderSize               float64   `json:"shortOrderSize"`
	Cost                         float64   `json:"cost"`
	EntryPrice                   float64   `json:"entryPrice"`
	UnrealizedPnl                float64   `json:"unrealizedPnl"`
	RealizedPnl                  float64   `json:"realizedPnl"`
	InitialMarginRequirement     float64   `json:"initialMarginRequirement"`
	MaintenanceMarginRequirement float64   `json:"maintenanceMarginRequirement"`
	OpenSize                     float64   `json:"openSize"`
	CollateralUsed               float64   `json:"collateralUsed"`
	EstimatedLiquidationPrice    float64   `json:"estimatedLiquidationPrice"`
	ParseTime                    time.Time `json:"-"`
	EventTime                    time.Time `json:"-"`
}

func (position *Position) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (position *Position) GetEventTime() time.Time {
	return position.EventTime
}

func (position *Position) GetParseTime() time.Time {
	return position.ParseTime
}

func (position *Position) UnmarshalJSON(data []byte) error {
	type Alias Position
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(position),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	} else {
		position.ParseTime = time.Now()
		position.EventTime = time.Now()
	}
	return nil
}
func (position *Position) GetSymbol() string {
	return position.Market
}
func (position *Position) GetSize() float64 {
	return position.NetSize
}
func (position *Position) GetPrice() float64 {
	if position.NetSize == 0 {
		return 0.0
	} else {
		return (position.Cost - position.RealizedPnl) / position.NetSize
	}
}
func (position *Position) GetTime() time.Time {
	return position.ParseTime
}

type PositionsHttpResponse struct {
	Success bool       `json:"success"`
	Result  []Position `json:"result"`
}

type Account struct {
	BackstopProvider             bool       `json:"backstopProvider"`
	Collateral                   float64    `json:"collateral"`
	FreeCollateral               float64    `json:"freeCollateral"`
	CollateralUsed               float64    `json:"collateralUsed"`
	Leverage                     float64    `json:"leverage"`
	Liquidating                  bool       `json:"liquidating"`
	MaintenanceMarginRequirement float64    `json:"maintenanceMarginRequirement"`
	MakerFee                     float64    `json:"makerFee"`
	MarginFraction               float64    `json:"marginFraction"`
	OpenMarginFraction           float64    `json:"openMarginFraction"`
	TakerFee                     float64    `json:"takerFee"`
	TotalAccountValue            float64    `json:"totalAccountValue"`
	TotalPositionSize            float64    `json:"totalPositionSize"`
	Username                     string     `json:"username"`
	Positions                    []Position `json:"positions"`
	ParseTime                    time.Time  `json:"-"`
}

func (account *Account) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (account *Account) UnmarshalJSON(data []byte) error {
	type Alias Account
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(account),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	} else {
		account.ParseTime = time.Now()
	}
	return nil
}
func (account *Account) GetTime() time.Time {
	return account.ParseTime
}
func (account *Account) GetCurrency() string {
	return "USDT"
}
func (account *Account) GetBalance() float64 {
	return account.TotalAccountValue
}
func (account *Account) GetFree() float64 {
	return account.FreeCollateral
}
func (account *Account) GetUsed() float64 {
	return account.CollateralUsed
}

type AccountHttpResponse struct {
	Success bool    `json:"success"`
	Result  Account `json:"result"`
}

//{
//  "channel": "orders",
//  "type": "update",
//  "data": {
//    "id": 48026351503,
//    "clientId": "16209851438511",
//    "market": "DOGE-PERP",
//    "type": "limit",
//    "side": "sell",
//    "price": 0.528221,
//    "size": 46.0,
//    "status": "closed",
//    "filledSize": 0.0,
//    "remainingSize": 0.0,
//    "reduceOnly": false,
//    "liquidation": false,
//    "avgFillPrice": null,
//    "postOnly": true,
//    "ioc": false,
//    "createdAt": "2021-05-14T09:39:03.121411+00:00"
//  }
//}

type Order struct {
	ID            int64     `json:"id"`
	ClientId      string    `json:"clientId"`
	Market        string    `json:"market"`
	Type          string    `json:"type"`
	Side          string    `json:"side"`
	Price         float64   `json:"price"`
	Size          float64   `json:"size"`
	Status        string    `json:"status"`
	FilledSize    float64   `json:"filledSize"`
	RemainingSize float64   `json:"remainingSize"`
	ReduceOnly    bool      `json:"reduceOnly"`
	Liquidation   bool      `json:"liquidation"`
	AvgFillPrice  float64   `json:"avgFillPrice"`
	PostOnly      bool      `json:"postOnly"`
	Ioc           bool      `json:"ioc"`
	CreatedAt     time.Time `json:"-"`
	ParseTime     time.Time `json:"-"`
}

func (order *Order) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (order *Order) UnmarshalJSON(data []byte) error {
	type Alias Order
	aux := &struct {
		*Alias
		CreatedAt string `json:"createdAt"`
	}{
		Alias: (*Alias)(order),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	} else {
		order.CreatedAt, err = time.Parse(TimeLayout, aux.CreatedAt)
		if err != nil {
			return err
		}
		order.ParseTime = time.Now()
	}
	return nil
}

func (order *Order) GetSymbol() string {
	return order.Market
}

func (order *Order) GetSize() float64 {
	return order.Size
}

func (order *Order) GetPrice() float64 {
	return order.Price
}

func (order *Order) GetFilledSize() float64 {
	return order.FilledSize
}

func (order *Order) GetFilledPrice() float64 {
	return order.AvgFillPrice
}

func (order *Order) GetSide() common.OrderSide {
	if order.Side == OrderSideBuy {
		return common.OrderSideBuy
	} else if order.Side == OrderSideSell {
		return common.OrderSideSell
	}
	return common.OrderSideUnknown
}

func (order *Order) GetClientID() string {
	return order.ClientId
}

func (order *Order) GetID() string {
	return fmt.Sprintf("%d", order.ID)
}

func (order *Order) GetType() common.OrderType {
	switch order.Type {
	case OrderTypeLimit:
		return common.OrderTypeLimit
	case OrderTypeMarket:
		return common.OrderTypeMarket
	default:
		return common.OrderTypeUnknown
	}
}

func (order *Order) GetPostOnly() bool {
	return order.PostOnly
}

func (order *Order) GetReduceOnly() bool {
	return order.ReduceOnly
}

func (order *Order) GetStatus() common.OrderStatus {
	switch order.Status {
	case OrderStatusNew:
		return common.OrderStatusNew
	case OrderStatusOpen:
		return common.OrderStatusOpen
	case OrderStatusClosed:
		if order.FilledSize != 0 {
			return common.OrderStatusFilled
		} else {
			return common.OrderStatusCancelled
		}
	default:
		return common.OrderStatusUnknown
	}
}

type Leverage struct {
	Leverage int `json:"leverage"`
}

//{"type": "error", "code": 400, "msg": "Already logged in"}
type UserDataCap struct {
	Type    string          `json:"type"`
	Code    int             `json:"code"`
	Msg     string          `json:"msg"`
	Channel string          `json:"channel"`
	Data    json.RawMessage `json:"data"`
}

//{
//  "channel": "fills",
//  "type": "update",
//  "data": {
//    "id": 2088010379,
//    "market": "DOGE-PERP",
//    "future": "DOGE-PERP",
//    "baseCurrency": null,
//    "quoteCurrency": null,
//    "type": "order",
//    "side": "sell",
//    "price": 0.527801,
//    "size": 44.0,
//    "orderId": 48026769445,
//    "time": "2021-05-14T09:41:21.708365+00:00",
//    "tradeId": 1035221885,
//    "feeRate": 0.00019,
//    "fee": 0.00441241636,
//    "feeCurrency": "USD",
//    "liquidity": "maker"
//  }
//}

type Fill struct {
	ID     int64  `json:"id"`
	Market string `json:"market"`
	//Future        string    `json:"future"`
	BaseCurrency  string    `json:"baseCurrency"`
	QuoteCurrency string    `json:"quoteCurrency"`
	Type          string    `json:"type"`
	Side          string    `json:"side"`
	FilledPrice   float64   `json:"price"`
	FilledSize    float64   `json:"size"`
	OrderId       int64     `json:"orderId"`
	Time          time.Time `json:"-"`
	TradeId       int64     `json:"tradeId"`
	FeeRate       float64   `json:"feeRate"`
	Fee           float64   `json:"fee"`
	Liquidity     string    `json:"liquidity"`

	OrderType  string  `json:"-"`
	ReduceOnly bool    `json:"-"`
	Ioc        bool    `json:"-"`
	PostOnly   bool    `json:"-"`
	Price      float64 `json:"-"`
	Size       float64 `json:"-"`
	ClientId   string  `json:"-"`
}

func (fill *Fill) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (fill *Fill) UnmarshalJSON(data []byte) error {
	type Alias Fill
	aux := &struct {
		*Alias
		Time string `json:"time"`
	}{
		Alias: (*Alias)(fill),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	} else {
		fill.Time, err = time.Parse(TimeLayout, aux.Time)
		if err != nil {
			return err
		}
	}
	return nil
}

func (fill *Fill) GetSymbol() string {
	return fill.Market
}

func (fill *Fill) GetSize() float64 {
	return fill.Size
}

func (fill *Fill) GetPrice() float64 {
	return fill.Price
}

func (fill *Fill) GetFilledSize() float64 {
	return fill.FilledSize
}

func (fill *Fill) GetFilledPrice() float64 {
	return fill.FilledPrice
}

func (fill *Fill) GetSide() common.OrderSide {
	switch fill.Side {
	case OrderSideBuy:
		return common.OrderSideBuy
	case OrderSideSell:
		return common.OrderSideSell
	default:
		return common.OrderSideUnknown
	}
}

func (fill *Fill) GetClientID() string {
	return fill.ClientId
}

func (fill *Fill) GetID() string {
	return fmt.Sprintf("%d", fill.OrderId)
}

func (fill *Fill) GetType() common.OrderType {
	switch fill.Type {
	case OrderTypeLimit:
		return common.OrderTypeLimit
	case OrderTypeMarket:
		return common.OrderTypeMarket
	default:
		return common.OrderTypeUnknown
	}
}

func (fill *Fill) GetPostOnly() bool {
	return fill.PostOnly
}

func (fill *Fill) GetReduceOnly() bool {
	return fill.PostOnly
}

func (fill *Fill) GetStatus() common.OrderStatus {
	return common.OrderStatusFilled
}

type FutureStats struct {
	Future                   string    `json:"-"`
	Volume                   float64   `json:"volume"`
	NextFundingRate          float64   `json:"nextFundingRate"`
	NextFundingTime          time.Time `json:"-"`
	ExpirationPrice          float64   `json:"expirationPrice"`
	PredictedExpirationPrice float64   `json:"predictedExpirationPrice"`
	StrikePrice              float64   `json:"strikePrice"`
	OpenInterest             float64   `json:"openInterest"`
}

func (fs *FutureStats) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (fs *FutureStats) GetSymbol() string {
	return fs.Future
}

func (fs *FutureStats) GetFundingRate() float64 {
	return fs.NextFundingRate * 8.0
}

func (fs *FutureStats) GetNextFundingTime() time.Time {
	return fs.NextFundingTime
}

func (fs *FutureStats) UnmarshalJSON(data []byte) error {
	type Alias FutureStats
	aux := &struct {
		NextFundingTime string `json:"nextFundingTime"`
		*Alias
	}{
		Alias: (*Alias)(fs),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	} else {
		fs.NextFundingTime, err = time.Parse(TimeLayout, aux.NextFundingTime)
		if err != nil {
			return err
		}
	}
	return nil
}

//{"channel": "ticker", "market": "DOGE-PERP", "type": "update", "data": {"bid": 0.278362, "ask": 0.2784135, "bidSize": 107.0, "askSize": 5600.0, "last": 0.2783695, "time": 1624183024.08771}} 189

type TickerData struct {
	Channel string `json:"channel"`
	Market  string `json:"market"`
	Type    string `json:"type"`
	Data    Ticker `json:"data"`
}

type Ticker struct {
	Bid     float64   `json:"bid"`
	Ask     float64   `json:"ask"`
	BidSize float64   `json:"bidSize"`
	AskSize float64   `json:"askSize"`
	Symbol  string    `json:"-"`
	Time    time.Time `json:"-"`
}

func (t *Ticker) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (t *Ticker) GetSymbol() string {
	panic("implement me")
}

func (t *Ticker) GetTime() time.Time {
	return t.Time
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

func (t *Ticker) UnmarshalJSON(data []byte) error {
	type Alias Ticker
	aux := &struct {
		Time float64 `json:"time"`
		*Alias
	}{
		Alias: (*Alias)(t),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	} else {
		t.Time = time.Unix(0, int64(aux.Time*1000000000))
	}
	return nil
}
