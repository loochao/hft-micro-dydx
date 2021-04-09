package bnswap

import "fmt"

//   {
//          "a":"BNB",
//          "wb":"1.00000000",
//          "cw":"0.00000000"
//   }

type WSBalance struct {
	Asset              string  `json:"a"`
	WalletBalance      float64 `json:"wb,string"`
	CrossWalletBalance float64 `json:"cwb,string"`
}

//        {
//          "s":"BTCUSDT",            // Symbol
//          "pa":"0",                 // Position Amount
//          "ep":"0.00000",            // Entry Price
//          "cr":"200",               // (Pre-fee) Accumulated Realized
//          "up":"0",                     // Unrealized PnL
//          "mt":"isolated",              // Margin Type
//          "iw":"0.00000000",            // Isolated Wallet (if isolated position)
//          "ps":"BOTH"                   // Position Side
//        }

type WSPosition struct {
	Symbol              string  `json:"s"`
	PositionAmt         float64 `json:"pa,string"`
	EntryPrice          float64 `json:"ep,string"`
	AccumulatedRealized float64 `json:"cr,string"`
	UnRealizedProfit    float64 `json:"up,string"`
	MarginType          string  `json:"mt"`
	IsolatedWallet      float64 `json:"iw,string"`
	PositionSide        string  `json:"ps"`
}

func (wsp *WSPosition) ToString() string {
	return fmt.Sprintf("Symbol=%s,EntryPrice=%f,PositionAmt=%f", wsp.Symbol, wsp.EntryPrice, wsp.PositionAmt)
}

type WSAccount struct {
	EventReasonType string       `json:"m"`
	Balances        []WSBalance  `json:"B"`
	Positions       []WSPosition `json:"P"`
}

const (
	EventReasonTypeDeposit             = "DEPOSIT"
	EventReasonTypeWithdraw            = "WITHDRAW"
	EventReasonTypeOrder               = "ORDER"
	EventReasonTypeFundingFee          = "FUNDING_FEE"
	EventReasonTypeWithdrawReject      = "WITHDRAW_REJECT"
	EventReasonTypeAdjustment          = "ADJUSTMENT"
	EventReasonTypeInsuranceClear      = "INSURANCE_CLEAR"
	EventReasonTypeAdminDeposit        = "ADMIN_DEPOSIT"
	EventReasonTypeAdminWithdraw       = "ADMIN_WITHDRAW"
	EventReasonTypeMarginTransfer      = "MARGIN_TRANSFER"
	EventReasonTypeMarginTypeChange    = "MARGIN_TYPE_CHANGE"
	EventReasonTypeAssetTransfer       = "ASSET_TRANSFER"
	EventReasonTypeOptionsPremiumFee   = "OPTIONS_PREMIUM_FEE"
	EventReasonTypeOptionsSettleProfit = "OPTIONS_SETTLE_PROFIT"
)

type BalanceAndPositionUpdateEvent struct {
	Event           string    `json:"e"`
	Time            int64     `json:"E"`
	TransactionTime int64     `json:"T"`
	Account         WSAccount `json:"a"`
}

//{
//
//  "e":"ORDER_TRADE_UPDATE",     // Event Type
//  "E":1568879465651,            // Event Time
//  "T":1568879465650,            // Transaction Time
//  "o":{
//    "s":"BTCUSDT",              // Symbol
//    "c":"TEST",                 // Client Order Id
//      // special client order id:
//      // starts with "autoclose-": liquidation order
//      // "adl_autoclose": ADL auto close order
//    "S":"SELL",                 // Side
//    "o":"TRAILING_STOP_MARKET", // Order Type
//    "f":"GTC",                  // Time in Force
//    "q":"0.001",                // Original Quantity
//    "p":"0",                    // Original Price
//    "ap":"0",                   // Average Price
//    "sp":"7103.04",             // Stop Price. Please ignore with TRAILING_STOP_MARKET order
//    "x":"NEW",                  // Execution Type
//    "X":"NEW",                  // Order Status
//    "i":8886774,                // Order Id
//    "l":"0",                    // Order Last Filled Quantity
//    "z":"0",                    // Order Filled Accumulated Quantity
//    "L":"0",                    // Last Filled Price
//    "N":"USDT",             // Commission Asset, will not push if no commission
//    "n":"0",                // Commission, will not push if no commission
//    "T":1568879465651,          // Order Trade Time
//    "t":0,                      // Trade Id
//    "b":"0",                    // Bids Notional
//    "a":"9.91",                 // Ask Notional
//    "m":false,                  // Is this trade the maker side?
//    "R":false,                  // Is this reduce only
//    "wt":"CONTRACT_PRICE",      // Stop Price Working Type
//    "ot":"TRAILING_STOP_MARKET",    // Original Order Type
//    "ps":"LONG",                        // Position Side
//    "cp":false,                     // If Close-All, pushed with conditional order
//    "AP":"7476.89",             // Activation Price, only puhed with TRAILING_STOP_MARKET order
//    "cr":"5.0",                 // Callback Rate, only puhed with TRAILING_STOP_MARKET order
//    "rp":"0"                            // Realized Profit of the trade
//  }
//
//}

type WSOrder struct {
	Symbol                    string  `json:"s,omitempty"`
	ClientOrderId             string  `json:"c,omitempty"`
	Side                      string  `json:"S,omitempty"`
	Type                      string  `json:"o,omitempty"`
	TimeInForce               string  `json:"f,omitempty"`
	OriginalQuantity          float64 `json:"q,string,omitempty"`
	OriginalPrice             float64 `json:"p,string,omitempty"`
	AveragePrice              float64 `json:"ap,string,omitempty"`
	StopPrice                 float64 `json:"sp,string,,=omitempty"`
	ExecutionType             string  `json:"x,omitempty"`
	Status                    string  `json:"X,omitempty"`
	OrderId                   int64   `json:"i,omitempty"`
	LastFilledQuantity        float64 `json:"l,string,omitempty"`
	FilledAccumulatedQuantity float64 `json:"z,string,omitempty"`
	LastFilledPrice           float64 `json:"L,string,omitempty"`
	CommissionAsset           string  `json:"N,omitempty"`
	Commission                float64 `json:"n,string,omitempty"`
	Time                      int64   `json:"T,omitempty"`
	TradeId                   int64   `json:"t,omitempty"`
	BidNotional               float64 `json:"b,string,omitempty"`
	AskNotional               float64 `json:"a,string,omitempty"`
	MakerSide                 bool    `json:"m,omitempty"`
	ReduceOnly                bool    `json:"R,omitempty"`
	StopPriceWorkingType      string  `json:"wt,omitempty"`
	OriginalOrderType         string  `json:"ot,omitempty"`
	PositionSide              string  `json:"ps,omitempty"`
	CloseAll                  bool    `json:"cp,omitempty"`
	ActivationPrice           float64 `json:"AP,string,omitempty"`
	CallbackRate              float64 `json:"rp,string,omitempty"`
}


func (wso *WSOrder) ToOrder() *Order {
	return &Order{
		Symbol:        wso.Symbol,
		OrderId:       wso.OrderId,
		ClientOrderId: wso.ClientOrderId,
		Price:         wso.OriginalPrice,
		ReduceOnly:    wso.ReduceOnly,
		OrigQty:       wso.OriginalQuantity,
		CumQty:        wso.FilledAccumulatedQuantity,
		CumQuote:      wso.AveragePrice,
		Status:        wso.Status,
		TimeInForce:   wso.TimeInForce,
		Type:          wso.Type,
		Side:          wso.Side,
		StopPrice:     wso.StopPrice,
		Time:          wso.Time,
	}
}


type OrderUpdateEvent struct {
	EventType       string  `json:"e"`
	EventTime       int64   `json:"E"`
	TransactionTime int64   `json:"T"`
	Order           WSOrder `json:"o"`
}
