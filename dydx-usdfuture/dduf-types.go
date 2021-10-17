package dydx_usdfuture

import (
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"net/url"
	"os"
	"strconv"
	"time"
)

type WSOrderBookSubscribe struct {
	Type           string `json:"type"`
	Channel        string `json:"channel"`
	Id             string `json:"id"`
	IncludeOffsets bool   `json:"includeOffsets,omitempty"`
}

type WSAccountSubscribe struct {
	Type          string `json:"type"`
	Channel       string `json:"channel"`
	AccountNumber string `json:"accountNumber"`
	ApiKey        string `json:"apiKey"`
	Signature     string `json:"signature"`
	Timestamp     string `json:"timestamp"`
	Passphrase    string `json:"passphrase"`
}

type Credentials struct {
	ApiKey          string
	ApiSecret       string
	ApiPassphrase   string
	StarkPrivateKey string
	StarkPublicKey  string
	AccountID       string
	AccountNumber   string
}

type Market struct {
	Market                           string    `json:"market"`
	Status                           string    `json:"status"`
	BaseAsset                        string    `json:"baseAsset"`
	QuoteAsset                       string    `json:"quoteAsset"`
	StepSize                         float64   `json:"stepSize,string"`
	TickSize                         float64   `json:"tickSize,string"`
	IndexPrice                       float64   `json:"indexPrice,string"`
	OraclePrice                      float64   `json:"oraclePrice,string"`
	PriceChange24H                   float64   `json:"priceChange24H,string"`
	NextFundingRate                  float64   `json:"nextFundingRate,string"`
	NextFundingAt                    time.Time `json:"nextFundingAt,string"`
	MinOrderSize                     float64   `json:"minOrderSize,string"`
	Type                             string    `json:"type"`
	InitialMarginFraction            float64   `json:"initialMarginFraction,string"`
	MaintenanceMarginFraction        float64   `json:"maintenanceMarginFraction,string"`
	BaselinePositionSize             float64   `json:"baselinePositionSize,string"`
	IncrementalPositionSize          float64   `json:"incrementalPositionSize,string"`
	IncrementalInitialMarginFraction float64   `json:"incrementalInitialMarginFraction,string"`
	Volume24H                        float64   `json:"volume24H,string"`
	Trades24H                        float64   `json:"trades24H,string"`
	OpenInterest                     float64   `json:"openInterest,string"`
	MaxPositionSize                  float64   `json:"maxPositionSize,string"`
	AssetResolution                  float64   `json:"assetResolution,string"`
}

func (m *Market) GetSymbol() string {
	return m.Market
}

func (m *Market) GetFundingRate() float64 {
	if os.Getenv("DYDX_FUNDING_RATE_1H") != "" {
		return m.NextFundingRate
	} else {
		return m.NextFundingRate * 8.0
	}
}

func (m *Market) GetNextFundingTime() time.Time {
	return m.NextFundingAt
}

func (m Market) GetExchange() common.ExchangeID {
	return ExchangeID
}

//{
//        "market": "BTC-USD",
//        "status": "OPEN",
//        "side": "LONG",
//        "size": "1000",
//        "maxSize": "1050",
//        "entryPrice": "100",
//        "exitPrice": null,
//        "unrealizedPnl": "50",
//        "realizedPnl": "100",
//        "createdAt": "2021-01-04T23:44:59.690Z",
//        "closedAt": null,
//        "netFunding": "500",
//        "sumOpen": "1050",
//        "sumClose": "50"
//      }

type Position struct {
	Market        string     `json:"market"`
	Status        string     `json:"status"`
	Side          string     `json:"side"`
	Size          float64    `json:"size,string"`
	MaxSize       float64    `json:"maxSize,string"`
	EntryPrice    float64    `json:"entryPrice,string"`
	ExitPrice     *float64   `json:"exitPrice,string,omitempty"`
	UnrealizedPnl float64    `json:"unrealizedPnl,string"`
	CreatedAt     time.Time  `json:"createdAt,string"`
	ClosedAt      *time.Time `json:"closedAt,string,omitempty"`
	NetFunding    float64    `json:"netFunding,string"`
	SumOpen       float64    `json:"sumOpen,string"`
	SumClose      float64    `json:"sumClose,string"`
	ParseTime     time.Time  `json:"-"`
}

func (p *Position) UnmarshalJSON(data []byte) error {
	type Alias Position
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(p),
	}
	err := json.Unmarshal(data, aux)
	if err != nil {
		return err
	}
	p.ParseTime = time.Now()
	return nil
}

func (p Position) GetSymbol() string {
	return p.Market
}

func (p Position) GetSize() float64 {
	return p.Size
}

func (p Position) GetPrice() float64 {
	return p.EntryPrice
}

func (p Position) GetEventTime() time.Time {
	return p.ParseTime
}

func (p Position) GetParseTime() time.Time {
	return p.ParseTime
}

func (p Position) GetExchange() common.ExchangeID {
	return ExchangeID
}

type Account struct {
	StarkKey           string              `json:"starkKey"`
	PositionId         string              `json:"positionId"`
	Equity             float64             `json:"equity,string"`
	FreeCollateral     float64             `json:"freeCollateral,string"`
	QuoteBalance       float64             `json:"quoteBalance,string"`
	PendingDeposits    float64             `json:"pendingDeposits,string"`
	PendingWithdrawals float64             `json:"pendingWithdrawals,string"`
	OpenPositions      map[string]Position `json:"openPositions"`
	AccountNumber      json.RawMessage     `json:"accountNumber"`
	ID                 string              `json:"id"`
	ParseTime          time.Time           `json:"-"`
}

func (a *Account) GetCurrency() string {
	return "USDC"
}

func (a *Account) GetBalance() float64 {
	return a.Equity
}

func (a *Account) GetFree() float64 {
	return a.FreeCollateral
}

func (a *Account) GetUsed() float64 {
	return a.Equity - a.FreeCollateral
}

func (a *Account) GetTime() time.Time {
	return a.ParseTime
}

func (a *Account) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (a *Account) UnmarshalJSON(data []byte) error {
	type Alias Account
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	err := json.Unmarshal(data, aux)
	if err != nil {
		return err
	}
	a.ParseTime = time.Now()
	return nil
}

type AccountResp struct {
	Account Account `json:"account"`
}

type AccountsResp struct {
	Accounts []Account `json:"accounts"`
}

// {
//      "value": "1c69867ef434431103da0a6cc9432fe34c09d3706662ffa3729a95ac27d1ae6c",
//      "msg": "signature must be a valid string of length 44 in headers",
//      "param": "dydx-signature",
//      "location": "headers"
//    }

type Error struct {
	Value    string `json:"value"`
	Msg      string `json:"msg"`
	Param    string `json:"param"`
	Location string `json:"headers"`
}

type ErrorsCap struct {
	Errors []Error `json:"errors"`
}

//  {
//      "id": "id",
//      "clientId": "foo",
//      "accountId": "afoo",
//      "market": "BTC-USD",
//      "side": "SELL",
//      "price": "29000",
//      "triggerPrice": null,
//      "trailingPercent": null,
//      "size": "0.500",
//      "remainingSize": "0.500",
//      "type": "LIMIT",
//      "createdAt": "2021-01-04T23:44:59.690Z",
//      "unfillableAt": null,
//      "expiresAt": "2021-02-04T23:44:59.690Z",
//      "status": "OPEN",
//      "timeInForce": "GTT",
//      "postOnly": false,
//      "cancelReason": null
//    }

type Order struct {
	ID              string     `json:"id"`
	ClientID        string     `json:"clientId"`
	Market          string     `json:"market"`
	Side            string     `json:"side"`
	Price           float64    `json:"price,string,omitempty"`
	TriggerPrice    *float64   `json:"triggerPrice,string,omitempty"`
	TrailingPercent *float64   `json:"trailingPercent,string,omitempty"`
	Size            float64    `json:"size,string,omitempty"`
	RemainingSize   float64    `json:"remainingSize,string,omitempty"`
	CreatedAt       time.Time  `json:"createdAt,string"`
	ExpiresAt       time.Time  `json:"expiresAt,string"`
	UnfillableAt    *time.Time `json:"unfillableAt,string"`
	Status          string     `json:"status"`
	TimeInForce     string     `json:"timeInForce"`
	PostOnly        bool       `json:"postOnly"`
	CancelReason    *string    `json:"cancelReason"`
}

func (o *Order) GetSymbol() string {
	return o.Market
}

func (o *Order) GetSize() float64 {
	return o.Size
}

func (o *Order) GetPrice() float64 {
	return o.Price
}

func (o *Order) GetFilledSize() float64 {
	return o.Size - o.RemainingSize
}

func (o *Order) GetFilledPrice() float64 {
	return o.Price
}

func (o *Order) GetSide() common.OrderSide {
	switch o.Side {
	case OrderSideBuy:
		return common.OrderSideBuy
	case OrderSideSell:
		return common.OrderSideSell
	default:
		return common.OrderSideUnknown
	}
}

func (o *Order) GetClientID() string {
	return o.ClientID
}

func (o *Order) GetID() string {
	return o.ID
}

func (o *Order) GetStatus() common.OrderStatus {
	switch o.Status {
	case OrderStatusCanceled:
		return common.OrderStatusCancelled
	case OrderStatusFilled:
		return common.OrderStatusFilled
	case OrderStatusOpen:
		return common.OrderStatusOpen
	case OrderStatusPending:
		return common.OrderStatusNew
	default:
		return common.OrderStatusUnknown
	}
}

func (o *Order) GetType() common.OrderType {
	return common.OrderTypeLimit
}

func (o *Order) GetPostOnly() bool {
	return o.PostOnly
}

func (o *Order) GetReduceOnly() bool {
	return false
}

func (o *Order) GetExchange() common.ExchangeID {
	return ExchangeID
}

type OrdersResp struct {
	Orders []Order `json:"orders"`
}

type CancelOrdersParam struct {
	Market string `json:"market,omitempty"`
}

func (c CancelOrdersParam) ToUrlValues() url.Values {
	v := url.Values{}
	if c.Market != "" {
		v.Set("market", c.Market)
	}
	return v
}

type CancelOrdersResp struct {
	CancelOrders []Order `json:"cancelOrders"`
}

type NewOrderParams struct {
	PositionID  string  `json:"position_id,omitempty"`
	Market      string  `json:"market,omitempty"`
	Side        string  `json:"side,omitempty"`
	Type        string  `json:"order_type,omitempty"`
	PostOnly    bool    `json:"post_only,omitempty"`
	Size        float64 `json:"size,string,omitempty"`
	Price       float64 `json:"price,string,omitempty"`
	LimitFee    float64 `json:"limit_fee,string,omitempty"`
	Expiration  string  `json:"expiration,omitempty"`
	ClientId    string  `json:"client_id,omitempty"`
	TimeInForce string  `json:"time_in_force"`
}

func (nop *NewOrderParams) MarshalJSON() ([]byte, error) {
	jsonStr := fmt.Sprintf(
		`{"position_id": "%s", "market": "%s", "side": "%s", "order_type": "%s", "post_only": %v, "size": "%s", "price": "%s", "limit_fee": "%.4f", "expiration": "%s", "client_id": "%s", "time_in_force": "%s"}`,
		nop.PositionID,
		nop.Market,
		nop.Side,
		nop.Type,
		nop.PostOnly,
		common.FormatByPrecision(nop.Size, StepPrecisions[nop.Market]),
		common.FormatByPrecision(nop.Price, TickPrecisions[nop.Market]),
		nop.LimitFee,
		nop.Expiration,
		nop.ClientId,
		nop.TimeInForce,
	)
	logger.Debugf("%s", jsonStr)
	return []byte(jsonStr), nil
}

type CreateOrderResp struct {
	Order Order `json:"order"`
}

type Depth struct {
	Bids             common.Bids
	Asks             common.Asks
	Market           string
	ParseTime        time.Time
	WithSnapshotData bool
	Offset           int64
}

func (d *Depth) GetBidPrice() float64 {
	//if len(d.Bids) > 1 {
	//	return d.Bids[1][0]
	//} else
	if len(d.Bids) > 0 {
		return d.Bids[0][0]
	} else {
		return 0.0
	}
}

func (d *Depth) GetAskPrice() float64 {
	//if len(d.Asks) > 1 {
	//	return d.Asks[1][0]
	//} else
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

func (d *Depth) GetTime() time.Time {
	return d.ParseTime
}

func (d *Depth) GetAsks() common.Asks {
	return d.Asks[:]
}

func (d *Depth) GetBids() common.Bids {
	return d.Bids[:]
}

func (d *Depth) GetSymbol() string {
	return d.Market
}

func (d *Depth) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (d *Depth) IsValid() bool {
	if !d.WithSnapshotData || (len(d.Asks) > 0 && len(d.Bids) > 0 && d.Asks[0][0] < d.Bids[0][0]) {
		return false
	}
	return true
}

type OrderBookSnapshot struct {
	Type         string `json:"type"`
	ConnectionID string `json:"connection_id"`
	MessageID    int    `json:"message_id"`
	Channel      string `json:"channel"`
	ID           string `json:"id"`
	Contents     struct {
		Asks []struct {
			Size  float64 `json:"size,string"`
			Price float64 `json:"price,string"`
		} `json:"asks"`
		Bids []struct {
			Size  float64 `json:"size,string"`
			Price float64 `json:"price,string"`
		} `json:"bids"`
	} `json:"contents"`
}

type OrderBookUpdate struct {
	Type         string `json:"type"`
	ConnectionID string `json:"connection_id"`
	MessageID    int    `json:"message_id"`
	Channel      string `json:"channel"`
	ID           string `json:"id"`
	Contents     struct {
		Asks   [][2]float64 `json:"-"`
		Bids   [][2]float64 `json:"-"`
		Offset int64        `json:"offset"`
	} `json:"-"`
}

func (obd *OrderBookUpdate) UnmarshalJSON(data []byte) error {
	type Alias OrderBookUpdate
	aux := &struct {
		Contents struct {
			Asks [][2]string `json:"asks"`
			Bids [][2]string `json:"bids"`
		} `json:"contents"`
		*Alias
	}{
		Alias: (*Alias)(obd),
	}
	err := json.Unmarshal(data, aux)
	if err != nil {
		return err
	}
	obd.Contents.Asks = make([][2]float64, len(aux.Contents.Asks))
	obd.Contents.Bids = make([][2]float64, len(aux.Contents.Bids))
	for i, ask := range aux.Contents.Asks {
		obd.Contents.Asks[i][0], err = strconv.ParseFloat(ask[0], 64)
		if err != nil {
			return err
		}
		obd.Contents.Asks[i][1], err = strconv.ParseFloat(ask[1], 64)
		if err != nil {
			return err
		}
	}
	for i, bid := range aux.Contents.Bids {
		obd.Contents.Bids[i][0], err = strconv.ParseFloat(bid[0], 64)
		if err != nil {
			return err
		}
		obd.Contents.Bids[i][1], err = strconv.ParseFloat(bid[1], 64)
		if err != nil {
			return err
		}
	}
	return nil
}

type WSUserSubscribed struct {
	Orders  []Order `json:"orders"`
	Account Account `json:"account"`
}
type WSUserChannelData struct {
	Orders    []Order    `json:"orders"`
	Accounts  []Account  `json:"accounts"`
	Positions []Position `json:"positions"`
}

type WSUserCap struct {
	Type         string          `json:"type"`
	ConnectionID string          `json:"connection_id"`
	MessageID    int             `json:"message_id"`
	Channel      string          `json:"channel"`
	Message      string          `json:"message"`
	Contents     json.RawMessage `json:"contents"`
}

//{
//  "iso": "2021-02-02T18:35:45Z",
//  "epoch": "1611965998.515",
//}
type ServerTime struct {
	ISO   time.Time `json:"iso,string"`
	Epoch float64   `json:"epoch,string"`
}
