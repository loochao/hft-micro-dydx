package binance_coinfuture

import (
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"time"
)

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

type WSCap struct {
	Stream string          `json:"stream,omitempty"`
	Data   json.RawMessage `json:"data,omitempty"`
	ID     int64           `json:"id,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
}

type WSResultCap struct {
	ID     int64      `json:"id,omitempty"`
	Result []WSResCap `json:"result,omitempty"`
}

type WSResCap struct {
	Req string          `json:"req,omitempty"`
	Res json.RawMessage `json:"res,omitempty"`
}

type WSRequest struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
	ID     int64    `json:"id"`
}

//    {
//            "entryPrice": "0.0",
//            "marginType": "CROSSED",
//            "isAutoAddMargin": false,
//            "isolatedMargin": "0",
//            "leverage": 20,
//            "liquidationPrice": "0",
//            "markPrice": "0.00000000",
//            "maxQty": "2500",
//            "positionAmt": "0",
//            "symbol": "LTCUSD_210924",
//            "unRealizedProfit": "0.00000000",
//            "positionSide": "BOTH"
//    }

type WSPosition struct {
	EntryPrice       float64   `json:"EntryPrice,string"`
	MarginType       string    `json:"marginType"`
	IsAutoAddMargin  bool      `json:"isAutoAddMargin"`
	IsolatedMargin   float64   `json:"isolatedMargin,string"`
	Leverage         float64   `json:"leverage"`
	LiquidationPrice float64   `json:"liquidationPrice,string"`
	MaxQty           float64   `json:"maxQty,string"`
	PositionAmt      float64   `json:"positionAmt,string"`
	Symbol           string    `json:"symbol"`
	UnRealizedProfit float64   `json:"unRealizedProfit,string"`
	PositionSide     string    `json:"positionSide"`
	ParseTime        time.Time `json:"-"`
	EventTime        time.Time `json:"-"`
}

func (W WSPosition) GetSymbol() string {
	return W.Symbol
}

func (W WSPosition) GetSize() float64 {
	return W.PositionAmt
}

func (W WSPosition) GetPrice() float64 {
	return W.EntryPrice
}

func (W WSPosition) GetEventTime() time.Time {
	return W.EventTime
}

func (W WSPosition) GetParseTime() time.Time {
	return W.ParseTime
}

//         "asset": "BTC",
//            "balance": "0.00000000",
//            "crossWalletBalance": "0.00000000",
//            "crossUnPnl": "0.00000000",
//            "availableBalance": "0.00000000",
//            "maxWithdrawAmount": "0.00000000"
type WSBalance struct {
	Asset              string    `json:"asset"`
	Balance            float64   `json:"balance,string"`
	CrossWalletBalance float64   `json:"crossWalletBalance,string"`
	CrossUnPnl         float64   `json:"crossUnPnl,string"`
	AvailableBalance   float64   `json:"availableBalance,string"`
	MaxWithdrawAmount  float64   `json:"maxWithdrawAmount,string"`
	ParseTime          time.Time `json:"-"`
	EventTime          time.Time `json:"-"`
}

func (W WSBalance) GetCurrency() string {
	return W.Asset
}

func (W WSBalance) GetBalance() float64 {
	return W.CrossWalletBalance + W.CrossUnPnl
}

func (W WSBalance) GetFree() float64 {
	return W.AvailableBalance
}

func (W WSBalance) GetUsed() float64 {
	return W.CrossWalletBalance + W.CrossUnPnl - W.AvailableBalance
}

func (W WSBalance) GetTime() time.Time {
	return W.EventTime
}

type WSBalanceUpdate struct {
	Asset                               string    `json:"a"`
	WalletBalance                       *float64  `json:"wb,string,omitempty"`
	CrossWalletBalance                  *float64  `json:"cwb,string,omitempty"`
	BalanceChangeExceptPnLAndCommission *float64  `json:"bc,string,omitempty"`
	EventTime                           time.Time `json:"-"`
	ParseTime                           time.Time `json:"-"`
}

type WSPositionUpdate struct {
	Symbol              string    `json:"s"`
	PositionAmt         float64   `json:"pa,string"`
	EntryPrice          float64   `json:"ep,string"`
	AccumulatedRealized float64   `json:"cr,string"`
	UnRealizedProfit    float64   `json:"up,string"`
	IsolatedWallet      float64   `json:"iw,string"`
	PositionSide        string    `json:"ps"`
	ParseTime           time.Time `json:"-"`
	EventTime           time.Time `json:"-"`
}

func (wsp *WSPositionUpdate) GetEventTime() time.Time {
	return wsp.EventTime
}

func (wsp *WSPositionUpdate) GetParseTime() time.Time {
	return wsp.ParseTime
}

func (wsp *WSPositionUpdate) GetSymbol() string {
	return wsp.Symbol
}

func (wsp *WSPositionUpdate) GetSize() float64 {
	return wsp.PositionAmt
}

func (wsp *WSPositionUpdate) GetPrice() float64 {
	return wsp.EntryPrice
}

type WSAccount struct {
	EventReasonType string             `json:"m"`
	Balances        []WSBalanceUpdate  `json:"B"`
	Positions       []WSPositionUpdate `json:"P"`
}

type BalanceAndPositionUpdateEvent struct {
	Event   string    `json:"e"`
	Account WSAccount `json:"a"`

	EventTime       time.Time `json:"-"`
	TransactionTime time.Time `json:"-"`
	ParseTime       time.Time `json:"-"`
}

func (bpu *BalanceAndPositionUpdateEvent) UnmarshalJSON(data []byte) error {
	type Alias BalanceAndPositionUpdateEvent
	aux := &struct {
		EventTime       int64 `json:"E"`
		TransactionTime int64 `json:"T"`
		*Alias
	}{
		Alias: (*Alias)(bpu),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	bpu.ParseTime = time.Now()
	bpu.EventTime = time.Unix(0, aux.EventTime*1000000)
	bpu.TransactionTime = time.Unix(0, aux.TransactionTime*1000000)
	for i := range bpu.Account.Positions {
		bpu.Account.Positions[i].EventTime = bpu.EventTime
		bpu.Account.Positions[i].ParseTime = bpu.ParseTime
	}
	for i := range bpu.Account.Balances {
		bpu.Account.Balances[i].EventTime = bpu.EventTime
		bpu.Account.Balances[i].ParseTime = bpu.ParseTime
	}
	return nil
}

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
	MarginAsset               string  `json:"ma,omitempty"`
	CommissionAsset           string  `json:"N,omitempty"`
	Commission                float64 `json:"n,string,omitempty"`
	OrderTradeTime            int64   `json:"T,omitempty"`
	TradeId                   int64   `json:"t,omitempty"`
	RealizedProfitOfTheTrade  float64 `json:"rp,string,omitempty"`
	BidNotional               float64 `json:"b,string,omitempty"`
	AskNotional               float64 `json:"a,string,omitempty"`
	IsThisTradeTheMakerSide   bool    `json:"m,omitempty"`
	ReduceOnly                bool    `json:"R,omitempty"`
	StopPriceWorkingType      string  `json:"wt,omitempty"`
	OriginalOrderType         string  `json:"ot,omitempty"`
	PositionSide              string  `json:"ps,omitempty"`
	CloseAll                  bool    `json:"cp,omitempty"`
	ActivationPrice           float64 `json:"AP,string,omitempty"`
	CallbackRate              float64 `json:"cr,string,omitempty"`
	IsOrderTriggerProtected   bool    `json:"pP,omitempty"`
}

func (order *WSOrder) GetSymbol() string {
	return order.Symbol
}

func (order *WSOrder) GetSize() float64 {
	return order.OriginalQuantity
}

func (order *WSOrder) GetPrice() float64 {
	return order.OriginalPrice
}

func (order *WSOrder) GetFilledSize() float64 {
	return order.FilledAccumulatedQuantity
}

func (order *WSOrder) GetFilledPrice() float64 {
	return order.AveragePrice
}

func (order *WSOrder) GetSide() common.OrderSide {
	switch order.Side {
	case OrderSideSell:
		return common.OrderSideSell
	case OrderSideBuy:
		return common.OrderSideBuy
	default:
		return common.OrderSideUnknown
	}
}

func (order *WSOrder) GetClientID() string {
	return order.ClientOrderId
}

func (order *WSOrder) GetID() string {
	return fmt.Sprintf("%d", order.OrderId)
}

func (order *WSOrder) GetStatus() common.OrderStatus {
	switch order.Status {
	case OrderStatusFilled:
		return common.OrderStatusFilled
	case OrderStatusCancelled:
		return common.OrderStatusCancelled
	case OrderStatusPartiallyFilled:
		return common.OrderStatusPartiallyFilled
	case OrderStatusExpired:
		return common.OrderStatusExpired
	case OrderStatusNew:
		return common.OrderStatusNew
	default:
		return common.OrderStatusUnknown
	}
}

func (order *WSOrder) GetType() common.OrderType {
	switch order.Type {
	case OrderTypeLimit:
		return common.OrderTypeLimit
	case OrderTypeMarket:
		return common.OrderTypeMarket
	default:
		return common.OrderTypeUnknown
	}
}

func (order *WSOrder) GetPostOnly() bool {
	return order.TimeInForce == OrderTimeInForceGTX
}

func (order *WSOrder) GetReduceOnly() bool {
	return order.ReduceOnly
}

type OrderUpdateEvent struct {
	EventType       string    `json:"e"`
	EventTime       time.Time `json:"-"`
	TransactionTime time.Time `json:"-"`
	AccountAlias    string    `json:"i"`
	Order           WSOrder   `json:"o"`
}

func (oue *OrderUpdateEvent) UnmarshalJSON(data []byte) error {
	type Alias OrderUpdateEvent
	aux := &struct {
		EventTime       int64 `json:"E"`
		TransactionTime int64 `json:"T"`
		*Alias
	}{
		Alias: (*Alias)(oue),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	oue.EventTime = time.Unix(0, aux.EventTime*1000000)
	oue.TransactionTime = time.Unix(0, aux.TransactionTime*1000000)
	return nil
}
