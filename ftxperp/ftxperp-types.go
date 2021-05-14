package ftxperp

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

type Future struct {
	Ask                 float64 `json:"ask"`
	Bid                 float64 `json:"bid"`
	Change1h            float64 `json:"change1h"`
	Change24h           float64 `json:"change24h"`
	ChangeBod           float64 `json:"changeBod"`
	VolumeUSD24h        float64 `json:"volumeUsd24h"`
	Volume              float64 `json:"volume"`
	Description         string  `json:"description"`
	Enabled             bool    `json:"enabled"`
	Expired             bool    `json:"expired"`
	Expiry              string  `json:"expiry"`
	Index               float64 `json:"index"`
	ImfFactor           float64 `json:"imfFactor"`
	Last                float64 `json:"last"`
	LowerBound          float64 `json:"lowerBound"`
	Mark                float64 `json:"mark"`
	Name                string  `json:"name"`
	Perpetual           bool    `json:"perpetual"`
	PositionLimitWeight float64 `json:"positionLimitWeight"`
	PostOnly            bool    `json:"postOnly"`
	PriceIncrement      float64 `json:"priceIncrement"`
	SizeIncrement       float64 `json:"sizeIncrement"`
	Underlying          string  `json:"underlying"`
	UpperBound          float64 `json:"upperBound"`
	Type                string  `json:"type"`
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
	//if bidLen > 100 {
	//	bidLen = 100
	//}
	//if askLen > 100 {
	//	askLen = 100
	//}
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

type Position struct {
	Cost                         float64 `json:"cost"`
	EntryPrice                   float64 `json:"entryPrice"`
	EstimatedLiquidationPrice    float64 `json:"estimatedLiquidationPrice"`
	Market                       string  `json:"future"`
	InitialMarginRequirement     float64 `json:"initialMarginRequirement"`
	LongOrderSize                float64 `json:"longOrderSize"`
	MaintenanceMarginRequirement float64 `json:"maintenanceMarginRequirement"`
	NetSize                      float64 `json:"netSize"`
	OpenSize                     float64 `json:"openSize"`
	RealizedPnl                  float64 `json:"realizedPnl"`
	ShortOrderSize               float64 `json:"shortOrderSize"`
	Side                         string  `json:"side"`
	Size                         float64 `json:"size"`
	UnrealizedPnl                float64 `json:"unrealizedPnl"`
	CollateralUsed               float64   `json:"collateralUsed"`
	ParseTime                    time.Time `json:"-"`
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
		return position.Cost / position.NetSize
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
	return account.Collateral
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
	Liquidation   string    `json:"liquidation"`
	AvgFillPrice  float64   `json:"avgFillPrice"`
	PostOnly      bool      `json:"postOnly"`
	Ioc           bool      `json:"ioc"`
	CreatedAt     time.Time `json:"-"`
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
		}else{
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
	ID            int64     `json:"id"`
	Market        string    `json:"market"`
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

func (fs *FutureStats) GetSymbol() string {
	return fs.Future
}

func (fs *FutureStats) GetFundingRate() float64 {
	return fs.NextFundingRate
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
