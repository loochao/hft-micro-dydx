package bnspot

import (
	"encoding/json"
	"fmt"
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

func (depth Depth20) GetBids() [20][2]float64 { return depth.Bids }
func (depth Depth20) GetAsks() [20][2]float64 { return depth.Asks }
func (depth Depth20) GetSymbol() string       { return depth.Symbol }
func (depth Depth20) GetTime() time.Time      { return depth.ParseTime }
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
	UpdateTime       time.Time `json:"-"`
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
	at.UpdateTime = time.Unix(0, aux.UpdateTime*1e6)
	return nil
}

// Balance holds query order data
type Balance struct {
	Asset  string  `json:"asset"`
	Free   float64 `json:"free,string"`
	Locked float64 `json:"locked,string"`
}

func (b *Balance) ToString() string {
	return fmt.Sprintf("Asset=%s,Free=%f,Locked=%f", b.Asset, b.Free, b.Locked)
}

type TransferResponse struct {
	TranId int64 `json:"tranId"`
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
	values.Set("quantity", strconv.FormatFloat(o.Quantity, 'f', 8, 64))
	values.Set("price", strconv.FormatFloat(o.Price, 'f', 8, 64))
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

func (depth Depth5) GetBids() [5][2]float64 { return depth.Bids }
func (depth Depth5) GetAsks() [5][2]float64 { return depth.Asks }
func (depth Depth5) GetSymbol() string      { return depth.Symbol }
func (depth Depth5) GetTime() time.Time     { return depth.ParseTime }
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
