package ftx_usdspot

import (
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
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

//  {
//      "name": "AAVE/USD",
//      "enabled": true,
//      "postOnly": false,
//      "priceIncrement": 0.01,
//      "sizeIncrement": 0.01,
//      "minProvideSize": 0.01,
//      "last": 329.0,
//      "bid": 329.34,
//      "ask": 329.44,
//      "price": 329.34,
//      "type": "spot",
//      "baseCurrency": "AAVE",
//      "quoteCurrency": "USD",
//      "underlying": null,
//      "restricted": false,
//      "highLeverageFeeExempt": true,
//      "change1h": 0.0029845291752954076,
//      "change24h": 0.026748971193415638,
//      "changeBod": -0.0017882581153578032,
//      "quoteVolume24h": 7081249.2787,
//      "volumeUsd24h": 7081249.2787
//    }

type Market struct {
	Name                  string  `json:"name"`
	Enabled               bool    `json:"enabled"`
	PostOnly              bool    `json:"postOnly"`
	PriceIncrement        float64 `json:"priceIncrement"`
	SizeIncrement         float64 `json:"sizeIncrement"`
	MinProvideSize        float64 `json:"minProvideSize"`
	Last                  float64 `json:"last"`
	Ask                   float64 `json:"ask"`
	Bid                   float64 `json:"bid"`
	Price                 float64 `json:"price"`
	Type                  string  `json:"type"`
	BaseCurrency          string  `json:"baseCurrency"`
	QuoteCurrency         string  `json:"quoteCurrency"`
	Underlying            string  `json:"underlying"`
	Restricted            bool    `json:"restricted"`
	HighLeverageFeeExempt bool    `json:"highLeverageFeeExempt"`

	Change1h  float64 `json:"change1h"`
	Change24h float64 `json:"change24h"`
	ChangeBod float64 `json:"changeBod"`

	VolumeUSD24h float64 `json:"volumeUsd24h"`
	Volume       float64 `json:"volume"`
}

type FundingRate struct {
	Market string    `json:"market"`
	Time   time.Time `json:"-"`
}

func (fr *FundingRate) GetSymbol() string {
	return fr.Market
}

func (fr *FundingRate) GetFundingRate() float64 {
	return 0.0
}

func (fr *FundingRate) GetNextFundingTime() time.Time {
	return fr.Time
}

func (fr *FundingRate) GetExchange() common.ExchangeID {
	return ExchangeID
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
	Action    string      `json:"action"`
	Time      time.Time   `json:"-"`
	Bids      common.Bids `json:"bids"`
	Asks      common.Asks `json:"asks"`
	Checksum  uint32      `json:"checksum"`
	Market    string      `json:"-"`
	ParseTime time.Time   `json:"-"`
}

func (orderBook *OrderBook) GetParseTime() time.Time {
	return orderBook.ParseTime
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
func (orderBook *OrderBook) GetEventTime() time.Time {
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

//{
//"coin": "USDTBEAR",
//"free": 2320.2,
//"spotBorrow": 0.0,
//"total": 2340.2,
//"usdValue": 2340.2,
//"availableWithoutBorrow": 2320.2
//}

type Balance struct {
	Coin                   string    `json:"coin"`
	Free                   float64   `json:"free"`
	SpotBorrow             float64   `json:"spotBorrow"`
	Total                  float64   `json:"total"`
	UsdValue               float64   `json:"usdValue"`
	AvailableWithoutBorrow float64   `json:"availableWithoutBorrow"`
	ParseTime              time.Time `json:"-"`
	EventTime              time.Time `json:"-"`
}

func (position *Balance) GetCurrency() string {
	return position.Coin
}

func (position *Balance) GetBalance() float64 {
	return position.Total
}

func (position *Balance) GetFree() float64 {
	return position.Free
}

func (position *Balance) GetUsed() float64 {
	return position.Total - position.Free
}

func (position *Balance) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (position *Balance) GetEventTime() time.Time {
	return position.EventTime
}

func (position *Balance) GetParseTime() time.Time {
	return position.ParseTime
}

func (position *Balance) UnmarshalJSON(data []byte) error {
	type Alias Balance
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

func (position *Balance) GetSymbol() string {
	return position.Coin + "/USD"
}
func (position *Balance) GetSize() float64 {
	return position.Total
}
func (position *Balance) GetPrice() float64 {
	return 0.0
}
func (position *Balance) GetTime() time.Time {
	return position.ParseTime
}

type BalancesHttpResponse struct {
	Success bool      `json:"success"`
	Result  []Balance `json:"result"`
}

type Account struct {
	BackstopProvider             bool      `json:"backstopProvider"`
	Collateral                   float64   `json:"collateral"`
	FreeCollateral               float64   `json:"freeCollateral"`
	CollateralUsed               float64   `json:"collateralUsed"`
	Leverage                     float64   `json:"leverage"`
	Liquidating                  bool      `json:"liquidating"`
	MaintenanceMarginRequirement float64   `json:"maintenanceMarginRequirement"`
	MakerFee                     float64   `json:"makerFee"`
	MarginFraction               float64   `json:"marginFraction"`
	OpenMarginFraction           float64   `json:"openMarginFraction"`
	TakerFee                     float64   `json:"takerFee"`
	TotalAccountValue            float64   `json:"totalAccountValue"`
	TotalPositionSize            float64   `json:"totalPositionSize"`
	Username                     string    `json:"username"`
	Positions                    []Balance `json:"positions"`
	ParseTime                    time.Time `json:"-"`
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
	logger.Debugf("USD ACCOUNT %f %f %f", account.TotalAccountValue, account.FreeCollateral, account.CollateralUsed)
	return nil
}
func (account *Account) GetTime() time.Time {
	return account.ParseTime
}
func (account *Account) GetCurrency() string {
	return "USD"
}
func (account *Account) GetBalance() float64 {
	//return account.TotalAccountValue
	return account.FreeCollateral
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
	//Market        string    `json:"future"`
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

//{"channel": "ticker", "market": "DOGE-PERP", "type": "update", "data": {"bid": 0.278362, "ask": 0.2784135, "bidSize": 107.0, "askSize": 5600.0, "last": 0.2783695, "time": 1624183024.08771}} 189

type TickerData struct {
	Channel string `json:"channel"`
	Market  string `json:"market"`
	Type    string `json:"type"`
	Data    Ticker `json:"data"`
}

type Ticker struct {
	Bid       float64   `json:"bid"`
	Ask       float64   `json:"ask"`
	BidSize   float64   `json:"bidSize"`
	AskSize   float64   `json:"askSize"`
	Symbol    string    `json:"-"`
	Time      time.Time `json:"-"`
	ParseTime time.Time `json:"-"`
}

func (t *Ticker) GetEventTime() time.Time {
	return t.Time
}

func (t *Ticker) GetParseTime() time.Time {
	return t.ParseTime
}

func (t *Ticker) GetBidOffset() float64 {
	return (t.Ask - t.Bid) / (t.Ask + t.Bid)
}

func (t *Ticker) GetAskOffset() float64 {
	return (t.Ask - t.Bid) / (t.Ask + t.Bid)
}

func (t *Ticker) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (t *Ticker) GetSymbol() string {
	return t.Symbol
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

type Depth struct {
	Bids      common.Bids
	Asks      common.Asks
	Symbol    string
	EventTime time.Time
	ParseTime time.Time
}

func (d *Depth) GetParseTime() time.Time {
	return d.ParseTime
}

func (d *Depth) GetBidOffset() float64 {
	panic("implement me")
}

func (d *Depth) GetAskOffset() float64 {
	panic("implement me")
}

func (d *Depth) GetBidPrice() float64 {
	if len(d.Bids) > 0 {
		return d.Bids[0][0]
	} else {
		return 0.0
	}
}

func (d *Depth) GetAskPrice() float64 {
	if len(d.Asks) > 0 {
		return d.Asks[0][0]
	} else {
		return 0.0
	}
}

func (d *Depth) GetBidSize() float64 {
	if len(d.Bids) > 0 {
		return d.Bids[0][1]
	} else {
		return 0.0
	}
}

func (d *Depth) GetAskSize() float64 {
	if len(d.Asks) > 0 {
		return d.Asks[0][1]
	} else {
		return 0.0
	}
}

func (d *Depth) GetEventTime() time.Time {
	return d.EventTime
}

func (d *Depth) GetAsks() common.Asks {
	return d.Asks[:]
}

func (d *Depth) GetBids() common.Bids {
	return d.Bids[:]
}

func (d *Depth) GetSymbol() string {
	return d.Symbol
}

func (d *Depth) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (d *Depth) IsValid() bool {
	if len(d.Asks) > 0 && len(d.Bids) > 0 && d.Asks[0][0] <= d.Bids[0][0] {
		return false
	}
	return true
}
