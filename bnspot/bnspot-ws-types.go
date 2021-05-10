package bnspot

import "fmt"

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
	EventType               string      `json:"e"`
	EventTime               int64       `json:"E"`
	TimeOfLastAccountUpdate int64       `json:"u"`
	Balances                []WSBalance `json:"B"`
}

type WSBalance struct {
	Asset        string  `json:"a"`
	FreeAmount   float64 `json:"f,string"`
	LockedAmount float64 `json:"l,string"`
}

func (wsb *WSBalance) ToString() string {
	return fmt.Sprintf("Asset=%s,Free=%f,Locked=%f", wsb.Asset, wsb.FreeAmount, wsb.LockedAmount)
}

func (wsb *WSBalance) ToBalance() Balance {
	return Balance{
		Asset:  wsb.Asset,
		Free:   wsb.FreeAmount,
		Locked: wsb.LockedAmount,
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
	EventType                              string  `json:"e"`
	EventTime                              int64   `json:"E"`
	Symbol                                 string  `json:"s"`
	ClientOrderID                          string  `json:"c"`
	Side                                   string  `json:"S"`
	OrderType                              string  `json:"o"`
	TimeInForce                            string  `json:"f"`
	Quantity                               float64 `json:"q,string"`
	OrderPrice                             float64 `json:"p,string"`
	StopPrice                              float64 `json:"P,string"`
	IcebergQuantity                        float64 `json:"F,string"`
	OrderListId                            int     `json:"g"`
	OriginalClientOrderID                  string  `json:"C"`
	CurrentExecutionType                   string  `json:"x"`
	CurrentOrderStatus                     string  `json:"X"`
	OrderRejectReason                      string  `json:"r"`
	OrderID                                int64   `json:"i"`
	LastExecutedQuantity                   float64 `json:"l,string"`
	CumulativeFilledQuantity               float64 `json:"z,string"`
	LastExecutedPrice                      float64 `json:"L,string"`
	CommissionAmount                       float64 `json:"n,string"`
	CommissionAsset                        string  `json:"N"`
	TransactionTime                        int64   `json:"T"`
	TradeID                                int64   `json:"t"`
	IsTheOrderOnTheBook                    bool    `json:"w"`
	IsThisTradeTheMakerSide                bool    `json:"m"`
	OrderCreationTime                      int64   `json:"O"`
	CumulativeQuoteAssetTransactedQuantity float64 `json:"Z,string"`
	LastQuoteAssetTransactedQuantity       float64 `json:"Y,string"`
	QuoteOrderQty                          float64 `json:"Q,string"`
}
