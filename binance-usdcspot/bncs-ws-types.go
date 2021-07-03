package binance_usdcspot

import (
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"time"
)

//{
//  "e": "outboundAccountPosition", //Event type
//  "E": 1564034571105,             //Event Time
//  "u": 1564034571073,             //Time of last account update
//  "B": [                          //Balances Array
//    {
//      "a": "ETH",                 //Asset
//      "f": "10000.000000",        //Free
//      "l": "0.000000"             //Locked
//    }
//  ]
//}

type AccountUpdateEvent struct {
	EventType               string          `json:"e"`
	TimeOfLastAccountUpdate int64           `json:"u"`
	Balances                []BalanceUpdate `json:"B"`
	EventTime               time.Time       `json:"-"`
	ParseTime               time.Time       `json:"-"`
}

func (at *AccountUpdateEvent) UnmarshalJSON(data []byte) error {
	type Alias AccountUpdateEvent
	aux := &struct {
		EventTime int64 `json:"E"`
		*Alias
	}{
		Alias: (*Alias)(at),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	at.EventTime = time.Unix(0, aux.EventTime*1000000)
	at.ParseTime = time.Now()
	for i := 0; i < len(at.Balances); i++ {
		at.Balances[i].EventTime = at.EventTime
		at.Balances[i].ParseTime = at.ParseTime
	}
	return nil
}

type BalanceUpdate struct {
	Asset        string    `json:"a"`
	FreeAmount   float64   `json:"f,string"`
	LockedAmount float64   `json:"l,string"`
	EventTime    time.Time `json:"-"`
	ParseTime    time.Time `json:"-"`
}

func (wsb *BalanceUpdate) ToString() string {
	return fmt.Sprintf("Asset=%s,Free=%f,Locked=%f", wsb.Asset, wsb.FreeAmount, wsb.LockedAmount)
}

func (wsb *BalanceUpdate) ToBalance() Balance {
	return Balance{
		Asset:  wsb.Asset,
		Free:   wsb.FreeAmount,
		Locked: wsb.LockedAmount,
		EventTime: wsb.EventTime,
		ParseTime: wsb.ParseTime,
	}
}

//{
//  "e": "balanceUpdate",         //Event Type
//  "E": 1573200697110,           //Event Time
//  "a": "BTC",                   //Asset
//  "d": "100.00000000",          //Balance Delta
//  "T": 1573200697068            //Clear Time
//}

type BalanceUpdateEvent struct {
	EventType    string  `json:"e"`
	EventTime    int64   `json:"E"`
	Asset        string  `json:"a"`
	BalanceDelta float64 `json:"d,string"`
	ClearTime    int64   `json:"T"`
}

//{
//  "e": "executionReport",        // Event type
//  "E": 1499405658658,            // Event time
//  "s": "ETHBTC",                 // Market
//  "c": "mUvoqJxFIILMdfAW5iGSOW", // Client order ID
//  "S": "BUY",                    // Side
//  "o": "LIMIT",                  // Order type
//  "f": "GTC",                    // Time in force
//  "q": "1.00000000",             // Order quantity
//  "p": "0.10264410",             // Order price
//  "P": "0.00000000",             // Stop price
//  "F": "0.00000000",             // Iceberg quantity
//  "g": -1,                       // OrderListId
//  "C": "",                       // Original client order ID; This is the ID of the order being canceled
//  "x": "NEW",                    // Current execution type
//  "X": "NEW",                    // Current order status
//  "r": "NONE",                   // Order reject reason; will be an error code.
//  "i": 4293153,                  // Order ID
//  "l": "0.00000000",             // Last executed quantity
//  "z": "0.00000000",             // Cumulative filled quantity
//  "L": "0.00000000",             // Last executed price
//  "n": "0",                      // Commission amount
//  "N": null,                     // Commission asset
//  "T": 1499405658657,            // Transaction time
//  "t": -1,                       // Trade ID
//  "I": 8641984,                  // Ignore
//  "w": true,                     // Is the order on the book?
//  "m": false,                    // Is this trade the maker side?
//  "M": false,                    // Ignore
//  "O": 1499405658657,            // Order creation time
//  "Z": "0.00000000",             // Cumulative quote asset transacted quantity
//  "Y": "0.00000000",             // Last quote asset transacted quantity (i.e. lastPrice * lastQty)
//  "Q": "0.00000000"              // Quote Order Qty
//}

type OrderUpdateEvent struct {
	EventType                              string    `json:"e"`
	EventTime                              time.Time `json:"-"`
	ParseTime                              time.Time `json:"-"`
	Symbol                                 string    `json:"s"`
	ClientOrderID                          string    `json:"c"`
	Side                                   string    `json:"S"`
	OrderType                              string    `json:"o"`
	TimeInForce                            string    `json:"f"`
	Quantity                               float64   `json:"q,string"`
	OrderPrice                             float64   `json:"p,string"`
	StopPrice                              float64   `json:"P,string"`
	IcebergQuantity                        float64   `json:"F,string"`
	OrderListId                            int       `json:"g"`
	OriginalClientOrderID                  string    `json:"C"`
	CurrentExecutionType                   string    `json:"x"`
	CurrentOrderStatus                     string    `json:"X"`
	OrderRejectReason                      string    `json:"r"`
	OrderID                                int64     `json:"i"`
	LastExecutedQuantity                   float64   `json:"l,string"`
	CumulativeFilledQuantity               float64   `json:"z,string"`
	LastExecutedPrice                      float64   `json:"L,string"`
	CommissionAmount                       float64   `json:"n,string"`
	CommissionAsset                        string    `json:"N"`
	TransactionTime                        int64     `json:"T"`
	TradeID                                int64     `json:"t"`
	IsTheOrderOnTheBook                    bool      `json:"w"`
	IsThisTradeTheMakerSide                bool      `json:"m"`
	OrderCreationTime                      int64     `json:"O"`
	CumulativeQuoteAssetTransactedQuantity float64   `json:"Z,string"`
	LastQuoteAssetTransactedQuantity       float64   `json:"Y,string"`
	QuoteOrderQty                          float64   `json:"Q,string"`
}

func (o OrderUpdateEvent) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (o OrderUpdateEvent) GetSymbol() string {
	return o.Symbol
}

func (o OrderUpdateEvent) GetSize() float64 {
	return o.Quantity
}

func (o OrderUpdateEvent) GetPrice() float64 {
	return o.OrderPrice
}

func (o OrderUpdateEvent) GetFilledSize() float64 {
	return o.CumulativeFilledQuantity
}

func (o OrderUpdateEvent) GetFilledPrice() float64 {
	if o.CumulativeFilledQuantity != 0 {
		return o.CumulativeQuoteAssetTransactedQuantity / o.CumulativeFilledQuantity
	} else {
		return 0.0
	}
}

func (o OrderUpdateEvent) GetSide() common.OrderSide {
	switch o.Side {
	case OrderSideSell:
		return common.OrderSideSell
	case OrderSideBuy:
		return common.OrderSideBuy
	default:
		return common.OrderSideUnknown
	}
}

func (o OrderUpdateEvent) GetClientID() string {
	if o.CurrentOrderStatus == OrderStatusCancelled {
		return o.OriginalClientOrderID
	} else {
		return o.ClientOrderID
	}
}

func (o OrderUpdateEvent) GetID() string {
	return fmt.Sprintf("%d", o.OrderID)
}

func (o OrderUpdateEvent) GetStatus() common.OrderStatus {
	switch o.CurrentOrderStatus {
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

func (o OrderUpdateEvent) GetType() common.OrderType {
	switch o.OrderType {
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

func (o OrderUpdateEvent) GetPostOnly() bool {
	if o.OrderType == OrderTypeLimitMarker {
		return true
	} else {
		return false
	}
}

func (o OrderUpdateEvent) GetReduceOnly() bool {
	return false
}

func (o *OrderUpdateEvent) UnmarshalJSON(data []byte) error {
	type Alias OrderUpdateEvent
	aux := &struct {
		EventTime int64 `json:"E"`
		*Alias
	}{
		Alias: (*Alias)(o),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	o.EventTime = time.Unix(0, aux.EventTime*1000000)
	o.ParseTime = time.Now()
	return nil
}
