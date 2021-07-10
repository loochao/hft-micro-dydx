package binance_usdtspot

import (
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"net/url"
	"strconv"
	"time"
)

const (
	OrderTimeInForceGTC  = "GTC"
	OrderTimeInForceIOC  = "IOC"
	OrderTimeInForceFOK  = "FOK"
	OrderRespTypeAck     = "ACK"
	OrderRespTypeResult  = "RESULT"
	OrderRespTypeFull    = "FULL"
	OrderIsIsolatedTrue  = "TRUE"
	OrderIsIsolatedFalse = "FALSE"

	OrderSideBuy  = "BUY"
	OrderSideSell = "SELL"

	OrderTypeLimit           = "LIMIT"
	OrderTypeMarket          = "MARKET"
	OrderTypeStopLoss        = "STOP_LOSS"
	OrderTypeStopLossLimit   = "STOP_LOSS_LIMIT"
	OrderTypeTakeProfit      = "TAKE_PROFIT"
	OrderTypeTakeProfitLimit = "TAKE_PROFIT_LIMIT"
	OrderTypeLimitMarker     = "LIMIT_MAKER"

	TimeIntervalMinute         = "1m"
	TimeIntervalThreeMinutes   = "3m"
	TimeIntervalFiveMinutes    = "5m"
	TimeIntervalFifteenMinutes = "15m"
	TimeIntervalThirtyMinutes  = "30m"
	TimeIntervalHour           = "1h"
	TimeIntervalTwoHours       = "2h"
	TimeIntervalFourHours      = "4h"
	TimeIntervalSixHours       = "6h"
	TimeIntervalEightHours     = "8h"
	TimeIntervalTwelveHours    = "12h"
	TimeIntervalDay            = "1d"
	TimeIntervalThreeDays      = "3d"
	TimeIntervalWeek           = "1w"
	TimeIntervalMonth          = "1M"

	OrderStatusNew             = "NEW"
	OrderStatusPartiallyFilled = "PARTIALLY_FILLED"
	OrderStatusFilled          = "FILLED"
	OrderStatusCancelled       = "CANCELED"
	OrderStatusPendingCancel   = "PENDING_CANCEL"
	OrderStatusReject          = "REJECTED"
	OrderStatusExpired         = "EXPIRED"
)

type Depth20 struct {
	Symbol       string         `json:"s,omitempty"`
	LastUpdateId int64          `json:"lastUpdateId,omitempty"`
	Bids         [20][2]float64 `json:"-"`
	Asks         [20][2]float64 `json:"_"`
	ParseTime    time.Time      `json:"-"`
}

func (depth Depth20) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (depth Depth20) GetBids() common.Bids { return depth.Bids[:] }
func (depth Depth20) GetAsks() common.Asks { return depth.Asks[:] }
func (depth Depth20) GetSymbol() string    { return depth.Symbol }
func (depth Depth20) GetTime() time.Time   { return depth.ParseTime }
func (depth *Depth20) UnmarshalJSON(data []byte) error {
	type Alias Depth20
	aux := &struct {
		Bids [20][2]string `json:"bids"`
		Asks [20][2]string `json:"asks"`
		*Alias
	}{
		Alias: (*Alias)(depth),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	aux.Alias.Bids = [20][2]float64{}
	aux.Alias.Asks = [20][2]float64{}
	for i, d := range aux.Bids {
		aux.Alias.Bids[i][0], _ = strconv.ParseFloat(d[0], 64)
		aux.Alias.Bids[i][1], _ = strconv.ParseFloat(d[1], 64)
	}
	for i, d := range aux.Asks {
		aux.Alias.Asks[i][0], _ = strconv.ParseFloat(d[0], 64)
		aux.Alias.Asks[i][1], _ = strconv.ParseFloat(d[1], 64)
	}
	return nil
}

type Depth20Stream struct {
	Stream string  `json:"stream,omitempty"`
	Data   Depth20 `json:"data,omitempty"`
}

type ExchangeInfo struct {
	Code       int    `json:"code"`
	Msg        string `json:"msg"`
	Timezone   string `json:"timezone"`
	ServerTime int64  `json:"serverTime"`
	RateLimits []struct {
		RateLimitType string `json:"rateLimitType"`
		Interval      string `json:"interval"`
		Limit         int    `json:"limit"`
	} `json:"rateLimits"`
	ExchangeFilters interface{} `json:"exchangeFilters"`
	Symbols         []Symbol    `json:"symbols"`
}

type Symbol struct {
	Symbol             string   `json:"symbol"`
	Status             string   `json:"status"`
	BaseAsset          string   `json:"baseAsset"`
	BaseAssetPrecision int      `json:"baseAssetPrecision"`
	QuoteAsset         string   `json:"quoteAsset"`
	QuotePrecision     int      `json:"quotePrecision"`
	OrderTypes         []string `json:"orderTypes"`
	IcebergAllowed     bool     `json:"icebergAllowed"`
	Filters            []struct {
		FilterType          string  `json:"filterType"`
		MinPrice            float64 `json:"minPrice,string"`
		MaxPrice            float64 `json:"maxPrice,string"`
		TickSize            float64 `json:"tickSize,string"`
		MultiplierUp        float64 `json:"multiplierUp,string"`
		MultiplierDown      float64 `json:"multiplierDown,string"`
		AvgPriceMins        int64   `json:"avgPriceMins"`
		MinQty              float64 `json:"minQty,string"`
		MaxQty              float64 `json:"maxQty,string"`
		StepSize            float64 `json:"stepSize,string"`
		MinNotional         float64 `json:"minNotional,string"`
		ApplyToMarket       bool    `json:"applyToMarket"`
		Limit               int64   `json:"limit"`
		MaxNumAlgoOrders    int64   `json:"maxNumAlgoOrders"`
		MaxNumIcebergOrders int64   `json:"maxNumIcebergOrders"`
	} `json:"filters"`
}

type KlineParams struct {
	Symbol    string `json:"symbol,omitempty"`
	Interval  string `json:"interval,omitempty"`
	Limit     int64  `json:"limit,omitempty"`
	StartTime int64  `json:"startTime,omitempty"`
	EndTime   int64  `json:"endTime,omitempty"`
}

func (bkp *KlineParams) ToUrlValues() url.Values {
	values := url.Values{}
	values.Set("symbol", bkp.Symbol)
	values.Set("interval", bkp.Interval)
	if bkp.Limit > 0 {
		values.Set("limit", fmt.Sprintf("%d", bkp.Limit))
	}
	if bkp.StartTime > 0 {
		values.Set("startTime", fmt.Sprintf("%d", bkp.StartTime))
	}
	if bkp.EndTime > 0 {
		values.Set("endTime", fmt.Sprintf("%d", bkp.EndTime))
	}
	return values
}

type ListenKey struct {
	ListenKey string `json:"listenKey"`
}

func (lk *ListenKey) ToUrlValues() url.Values {
	values := url.Values{}
	values.Set("listenKey", lk.ListenKey)
	return values
}

// Account holds the account data
type Account struct {
	MakerCommission  int       `json:"makerCommission"`
	TakerCommission  int       `json:"takerCommission"`
	BuyerCommission  int       `json:"buyerCommission"`
	SellerCommission int       `json:"sellerCommission"`
	CanTrade         bool      `json:"canTrade"`
	CanWithdraw      bool      `json:"canWithdraw"`
	CanDeposit       bool      `json:"canDeposit"`
	EventTime        time.Time `json:"-"`
	ParseTime        time.Time `json:"-"`
	Balances         []Balance `json:"balances"`
}

func (at *Account) UnmarshalJSON(data []byte) error {
	type Alias Account
	aux := &struct {
		UpdateTime int64 `json:"updateTime"`
		*Alias
	}{
		Alias: (*Alias)(at),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	at.EventTime = time.Unix(0, aux.UpdateTime*1000000)
	at.ParseTime = time.Now()
	for i := 0; i < len(at.Balances); i++ {
		at.Balances[i].EventTime = at.EventTime
		at.Balances[i].ParseTime = at.ParseTime
	}
	return nil
}

// Balance holds query order data
type Balance struct {
	Asset     string    `json:"asset"`
	Free      float64   `json:"free,string"`
	Locked    float64   `json:"locked,string"`
	EventTime time.Time `json:"-"`
	ParseTime time.Time `json:"-"`
}

func (b *Balance) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (b *Balance) GetTime() time.Time {
	return b.EventTime
}

func (b *Balance) GetEventTime() time.Time {
	return b.EventTime
}

func (b *Balance) GetParseTime() time.Time {
	return b.ParseTime
}

func (b *Balance) GetSymbol() string {
	return b.Asset + "USDT"
}

func (b *Balance) GetSize() float64 {
	return b.Free + b.Locked
}

func (b *Balance) GetPrice() float64 {
	return 0.0
}

func (b *Balance) GetCurrency() string {
	return b.Asset
}

func (b *Balance) GetBalance() float64 {
	return b.Free + b.Locked
}

func (b *Balance) GetFree() float64 {
	return b.Free
}

func (b *Balance) GetUsed() float64 {
	return b.Locked
}

func (b *Balance) ToString() string {
	return fmt.Sprintf("Asset=%s,Free=%f,Locked=%f,Time=%v", b.Asset, b.Free, b.Locked, b.EventTime)
}

type TransferResponse struct {
	TranId int64 `json:"tranId"`
}

type FundingRate struct {
	Symbol string
}

func (f FundingRate) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (f FundingRate) GetSymbol() string {
	return f.Symbol
}

func (f FundingRate) GetFundingRate() float64 {
	return 0
}

func (f FundingRate) GetNextFundingTime() time.Time {
	return time.Time{}
}

type NewOrderParams struct {
	Symbol           string  `json:"symbol,omitempty"`
	Side             string  `json:"side,omitempty"`
	Type             string  `json:"type,omitempty"`
	TimeInForce      string  `json:"timeInForce,omitempty"`
	Quantity         float64 `json:"quantity,omitempty"`
	Price            float64 `json:"price,omitempty"`
	IcebergQty       float64 `json:"icebergQty,omitempty"`
	NewClientOrderID string  `json:"newClientOrderId,omitempty"`
	NewOrderRespType string  `json:"newOrderRespType,omitempty"`
}

func (o *NewOrderParams) ToUrlValues() url.Values {
	values := url.Values{}
	values.Set("symbol", o.Symbol)
	values.Set("side", o.Side)
	values.Set("type", o.Type)
	if o.TimeInForce != "" {
		values.Set("timeInForce", o.TimeInForce)
	}
	if o.Quantity != 0.0 {
		values.Set("quantity", strconv.FormatFloat(o.Quantity, 'f', 8, 64))
	}
	if o.Price != 0.0 && o.Type != OrderTypeMarket {
		values.Set("price", strconv.FormatFloat(o.Price, 'f', 8, 64))
	}
	values.Set("newClientOrderId", o.NewClientOrderID)
	values.Set("newOrderRespType", o.NewOrderRespType)
	if o.IcebergQty != 0 {
		values.Set("icebergQty", strconv.FormatFloat(o.IcebergQty, 'f', 8, 64))
	}
	return values
}

//{
//  "symbol": "BTCUSDT",
//  "orderId": 28,
//  "clientOrderId": "6gCrw2kRUAF9CvJDGP16IP",
//  "transactTime": 1507725176595,
//  "price": "1.00000000",
//  "origQty": "10.00000000",
//  "executedQty": "10.00000000",
//  "cummulativeQuoteQty": "10.00000000",
//  "status": "FILLED",
//  "timeInForce": "GTC",
//  "type": "MARKET",
//  "side": "SELL",
//  "marginBuyBorrowAmount": 5,       // will not return if no margin trade happens
//  "marginBuyBorrowAsset": "BTC",    // will not return if no margin trade happens
//  "isIsolated": true,       // if isolated margin
//  "fills": [
//    {
//      "price": "4000.00000000",
//      "qty": "1.00000000",
//      "commission": "4.00000000",
//      "commissionAsset": "USDT"
//    },
//  ]
//}
type NewOrderResponse struct {
	Symbol                string  `json:"symbol"`
	OrderID               int64   `json:"orderId"`
	ClientOrderID         string  `json:"clientOrderId"`
	TransactionTime       int64   `json:"transactTime"`
	Price                 float64 `json:"price,string"`
	OrigQty               float64 `json:"origQty,string"`
	ExecutedQty           float64 `json:"executedQty,string"`
	CummulativeQuoteQty   float64 `json:"cummulativeQuoteQty,string"`
	Status                string  `json:"status"`
	TimeInForce           string  `json:"timeInForce"`
	Type                  string  `json:"type"`
	Side                  string  `json:"side"`
	MarginBuyBorrowAmount float64 `json:"marginBuyBorrowAmount,string"`
	MarginBuyBorrowAsset  string  `json:"marginBuyBorrowAsset"`
	Fills                 []struct {
		Price           float64 `json:"price,string"`
		Qty             float64 `json:"qty,string"`
		Commission      float64 `json:"commission,string"`
		CommissionAsset string  `json:"commissionAsset"`
	} `json:"fills"`
	EventTime time.Time `json:"-"`
	ParseTime time.Time `json:"-"`
}

func (order *NewOrderResponse) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (order *NewOrderResponse) UnmarshalJSON(data []byte) error {
	type Alias NewOrderResponse
	aux := &struct {
		TransactionTime int64 `json:"transactTime"`
		*Alias
	}{
		Alias: (*Alias)(order),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	order.EventTime = time.Unix(0, aux.TransactionTime*1000000)
	order.ParseTime = time.Now()
	return nil
}

func (order NewOrderResponse) GetSymbol() string {
	return order.Symbol
}

func (order NewOrderResponse) GetSize() float64 {
	return order.OrigQty
}

func (order NewOrderResponse) GetPrice() float64 {
	return order.Price
}

func (order NewOrderResponse) GetFilledSize() float64 {
	size := 0.0
	for _, f := range order.Fills {
		size += f.Qty
	}
	return size
}

func (order NewOrderResponse) GetFilledPrice() float64 {
	size := 0.0
	value := 0.0
	for _, f := range order.Fills {
		size += f.Qty
		value += f.Qty * f.Price
	}
	if size != 0.0 {
		return value / size
	} else {
		return 0.0
	}
}

func (order NewOrderResponse) GetSide() common.OrderSide {
	switch order.Side {
	case OrderSideBuy:
		return common.OrderSideBuy
	case OrderSideSell:
		return common.OrderSideSell
	default:
		return common.OrderSideUnknown
	}
}

func (order NewOrderResponse) GetClientID() string {
	return order.ClientOrderID
}

func (order NewOrderResponse) GetID() string {
	return fmt.Sprintf("%d", order.OrderID)
}

func (order NewOrderResponse) GetStatus() common.OrderStatus {
	switch order.Status {
	case OrderStatusNew:
		return common.OrderStatusNew
	case OrderStatusFilled:
		return common.OrderStatusFilled
	case OrderStatusCancelled:
		return common.OrderStatusCancelled
	case OrderStatusReject:
		return common.OrderStatusReject
	case OrderStatusExpired:
		return common.OrderStatusExpired
	case OrderStatusPartiallyFilled:
		return common.OrderStatusFilled
	default:
		return common.OrderStatusUnknown
	}
}

func (order NewOrderResponse) GetType() common.OrderType {
	switch order.Type {
	case OrderTypeMarket:
		return common.OrderTypeMarket
	case OrderTypeLimit:
		return common.OrderTypeLimit
	case OrderTypeLimitMarker:
		return common.OrderTypeLimit
	default:
		return common.OrderTypeUnknown
	}
}

func (order NewOrderResponse) GetPostOnly() bool {
	if order.Type == OrderTypeLimitMarker {
		return false
	} else {
		return true
	}
}

func (order NewOrderResponse) GetReduceOnly() bool {
	return false
}

type CancelAllOrderParams struct {
	Symbol string `json:"symbol"`
}

func (c *CancelAllOrderParams) ToUrlValues() url.Values {
	values := url.Values{}
	values.Set("symbol", c.Symbol)
	return values
}

// CancelOrderResponse is the return structured response from the exchange
type CancelOrderResponse struct {
	Symbol            string `json:"symbol"`
	OrigClientOrderID string `json:"origClientOrderId"`
	OrderID           int64  `json:"orderId"`
	ClientOrderID     string `json:"clientOrderId"`
}

type FutureAccountTransferParams struct {
	Asset  string  `json:"asset"`
	Amount float64 `json:"amount"`
	Type   int     `json:"type"`
}

func (fat *FutureAccountTransferParams) ToUrlValues() url.Values {
	values := url.Values{}
	values.Set("asset", fat.Asset)
	values.Set("amount", strconv.FormatFloat(fat.Amount, 'f', 8, 64))
	values.Set("type", fmt.Sprintf("%d", fat.Type))
	return values
}

type Depth5 struct {
	Symbol       string        `json:"s,omitempty"`
	LastUpdateId int64         `json:"lastUpdateId,omitempty"`
	Bids         [5][2]float64 `json:"-"`
	Asks         [5][2]float64 `json:"_"`
	ParseTime    time.Time     `json:"-"`
}

func (depth *Depth5) GetBidPrice() float64 {
	return depth.Bids[0][0]
}

func (depth *Depth5) GetAskPrice() float64 {
	return depth.Asks[0][0]
}

func (depth *Depth5) GetBidSize() float64 {
	return depth.Bids[0][1]
}

func (depth *Depth5) GetAskSize() float64 {
	return depth.Asks[0][1]
}

func (depth *Depth5) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (depth *Depth5) GetBids() common.Bids { return depth.Bids[:] }
func (depth *Depth5) GetAsks() common.Asks { return depth.Asks[:] }
func (depth *Depth5) GetSymbol() string    { return depth.Symbol }
func (depth *Depth5) GetTime() time.Time   { return depth.ParseTime }
func (depth *Depth5) UnmarshalJSON(data []byte) error {
	type Alias Depth5
	aux := &struct {
		Bids [5][2]string `json:"bids"`
		Asks [5][2]string `json:"asks"`
		*Alias
	}{
		Alias: (*Alias)(depth),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	aux.Alias.Bids = [5][2]float64{}
	aux.Alias.Asks = [5][2]float64{}
	for i, d := range aux.Bids {
		aux.Alias.Bids[i][0], _ = strconv.ParseFloat(d[0], 64)
		aux.Alias.Bids[i][1], _ = strconv.ParseFloat(d[1], 64)
	}
	for i, d := range aux.Asks {
		aux.Alias.Asks[i][0], _ = strconv.ParseFloat(d[0], 64)
		aux.Alias.Asks[i][1], _ = strconv.ParseFloat(d[1], 64)
	}
	return nil
}

type Depth5Stream struct {
	Stream string `json:"stream,omitempty"`
	Data   Depth5 `json:"data,omitempty"`
}

type Ping struct {
}

//{
//  "e": "trade",     // Event type
//  "E": 123456789,   // Event time
//  "s": "BNBBTC",    // Market
//  "t": 12345,       // Trade ID
//  "p": "0.001",     // Price
//  "q": "100",       // Quantity
//  "b": 88,          // Buyer order ID
//  "a": 50,          // Seller order ID
//  "T": 123456785,   // Trade time
//  "m": true,        // Is the buyer the market maker?
//  "M": true         // Ignore
//}

type Trade struct {
	EventType                string    `json:"e"`
	EventTime                time.Time `json:"-"`
	Symbol                   string    `json:"s"`
	Price                    float64   `json:"-"`
	Quantity                 float64   `json:"-"`
	TradeTime                time.Time `json:"-"`
	IsTheBuyerTheMarketMaker bool      `json:"m"`
	Ignore                   bool      `json:"M"`
}

func (trade *Trade) GetSymbol() string  { return trade.Symbol }
func (trade *Trade) GetSize() float64   { return trade.Quantity }
func (trade *Trade) GetPrice() float64  { return trade.Price }
func (trade *Trade) GetTime() time.Time { return trade.EventTime }
func (trade *Trade) IsUpTick() bool     { return !trade.IsTheBuyerTheMarketMaker }

func (trade *Trade) UnmarshalJSON(data []byte) error {
	type Alias Trade
	aux := struct {
		EventTime json.RawMessage `json:"E"`
		TradeTime json.RawMessage `json:"T"`
		Price     json.RawMessage `json:"p"`
		Quantity  json.RawMessage `json:"q"`
		*Alias
	}{Alias: (*Alias)(trade)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	} else {
		eventTime, err := common.ParseInt(aux.EventTime)
		if err != nil {
			return err
		}
		tradeTime, err := common.ParseInt(aux.TradeTime)
		if err != nil {
			return err
		}
		trade.Price, err = common.ParseFloat(aux.Price[1 : len(aux.Price)-1])
		if err != nil {
			return err
		}
		trade.Quantity, err = common.ParseFloat(aux.Quantity[1 : len(aux.Quantity)-1])
		if err != nil {
			return err
		}
		trade.EventTime = time.Unix(0, eventTime*1000000)
		trade.TradeTime = time.Unix(0, tradeTime*1000000)
		return nil
	}
}

type WSTrade struct {
	Stream string `json:"stream"`
	Data   Trade  `json:"data"`
}

type TickerParam struct {
	Symbol string `json:"symbol"`
}

func (lk *TickerParam) ToUrlValues() url.Values {
	values := url.Values{}
	values.Set("symbol", lk.Symbol)
	return values
}

type Ticker struct {
	Symbol string  `json:"symbol"`
	Price  float64 `json:"price,string"`
}

//{
//  "u":400900217,     // order book updateId
//  "s":"BNBUSDT",     // symbol
//  "b":"25.35190000", // best bid price
//  "B":"31.21000000", // best bid qty
//  "a":"25.36520000", // best ask price
//  "A":"40.66000000"  // best ask qty
//}

type BookTicker struct {
	Symbol       string    `json:"s"`
	BestBidPrice float64   `json:"b,string"`
	BestBidQty   float64   `json:"B,string"`
	BestAskPrice float64   `json:"a,string"`
	BestAskQty   float64   `json:"A,string"`
	ParseTime    time.Time `json:"-"`
}

func (bt *BookTicker) GetTime() time.Time {
	return bt.ParseTime
}

func (bt *BookTicker) GetBidPrice() float64 {
	return bt.BestBidPrice
}

func (bt *BookTicker) GetAskPrice() float64 {
	return bt.BestAskPrice
}

func (bt *BookTicker) GetBidSize() float64 {
	return bt.BestBidQty
}

func (bt *BookTicker) GetAskSize() float64 {
	return bt.BestAskQty
}

func (bt *BookTicker) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (bt *BookTicker) GetSymbol() string {
	return bt.Symbol
}

func (bt *BookTicker) UnmarshalJSON(data []byte) error {
	type Alias BookTicker
	aux := struct{ *Alias }{Alias: (*Alias)(bt)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	} else {
		bt.ParseTime = time.Now()
		return nil
	}
}

type BookTickerStream struct {
	Stream string     `json:"stream"`
	Data   BookTicker `json:"data"`
}
