package hbspot

import (
	"encoding/json"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

type DataCap struct {
	Status    string          `json:"status"`
	Data      json.RawMessage `json:"data,omitempty"`
	ErrCode   string          `json:"err-code,omitempty"`
	ErrMsg    string          `json:"err-msg,omitempty"`
	Timestamp time.Time       `json:"-"`
}

func (wsCap *DataCap) UnmarshalJSON(data []byte) error {
	type Alias DataCap
	aux := struct {
		Timestamp int64 `json:"ts,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(wsCap),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	wsCap.Timestamp = time.Unix(0, aux.Timestamp*1000000)
	return nil
}

type HeartBeat struct {
	Heartbeat                       int    `json:"heartbeat"`
	SwapHeartbeat                   int    `json:"swap_heartbeat"`
	OptionHeartbeat                 int    `json:"option_heartbeat"`
	LinearSwapHeartbeat             int    `json:"linear_swap_heartbeat"`
	EstimatedRecoveryTime           *int64 `json:"estimated_recovery_time,omitempty"`
	SwapEstimatedRecoveryTime       *int64 `json:"swap_estimated_recovery_time,omitempty"`
	OptionEstimatedRecoveryTime     *int64 `json:"option_estimated_recovery_time,omitempty"`
	LinearSwapEstimatedRecoveryTime *int64 `json:"linear_swap_estimated_recovery_time,omitempty"`
}

type Symbol struct {
	Currency                 string  `json:"currency"`
	QuoteCurrency            string  `json:"quote-currency"`
	PricePrecision           int     `json:"price-precision"`
	AmountPrecision          int     `json:"amount-precision"`
	SymbolPartition          string  `json:"symbol-partition"`
	Symbol                   string  `json:"symbol"`
	State                    string  `json:"state"`
	ValuePrecision           float64 `json:"value-precision"`
	MinOrderAmt              float64 `json:"min-order-amt"`
	MaxOrderAmt              float64 `json:"max-order-amt"`
	MinOrderValue            float64 `json:"min-order-value"`
	LimitOrderMinOrderAmt    float64 `json:"limit-order-min-order-amt"`
	LimitOrderMaxOrderAmt    float64 `json:"limit-order-max-order-amt"`
	SellMarketMinOrderAmt    float64 `json:"sell-market-min-order-amt"`
	SellMarketMaxOrderAmt    float64 `json:"sell-market-max-order-amt"`
	BuyMarketMaxOrderValue   float64 `json:"buy-market-max-order-value"`
	LeverageRatio            float64 `json:"leverage-ratio"`
	SuperMarginLeverageRatio float64 `json:"super-margin-leverage-ratio"`
	FundingLeverageRatio     float64 `json:"funding-leverage-ratio"`
	ApiTrading               string  `json:"api-trading"`
}

type Kline struct {
	ID     int64   `json:"id"`
	Amount float64 `json:"amount"`
	Count  float64 `json:"count"`
	Open   float64 `json:"open"`
	Close  float64 `json:"close"`
	Low    float64 `json:"low"`
	High   float64 `json:"high"`
	Volume float64 `json:"vol"`
}

type Depth20 struct {
	Symbol    string         `json:"-"`
	Bids      [20][2]float64 `json:"bids,omitempty"`
	Asks      [20][2]float64 `json:"asks,omitempty"`
	Version   int64          `json:"version"`
	ParseTime time.Time      `json:"-"`
	EventTime time.Time      `json:"-"`
}

func (depth *Depth20) UnmarshalJSON(data []byte) error {
	type Alias Depth20
	aux := struct {
		EventTime int64 `json:"ts,omitempty"`
		*Alias
	}{Alias: (*Alias)(depth)}
	if err := json.Unmarshal(data, &aux); err != nil {
		logger.Debugf("ERR %v", err)
		return err
	}
	depth.EventTime = time.Unix(0, aux.EventTime*1000000)
	depth.ParseTime = time.Now()
	return nil
}

type WsDepthCap struct {
	Ch        string          `json:"ch"`
	Tick      json.RawMessage `json:"tick,omitempty"`
	Timestamp time.Time       `json:"-"`
}

func (wsCap *WsDepthCap) UnmarshalJSON(data []byte) error {
	type Alias WsDepthCap
	aux := struct {
		Timestamp int64 `json:"ts,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(wsCap),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	wsCap.Timestamp = time.Unix(0, aux.Timestamp*1000000)
	return nil
}

type WSBalance struct {
	Currency    string     `json:"currency"`
	AccountId   int64      `json:"accountId"`
	Balance     *float64   `json:"balance,string,omitempty"`
	Available   *float64   `json:"available,string,omitempty"`
	ChangeType  string     `json:"changeType"`
	AccountType string     `json:"accountType"`
	ChangeTime  *time.Time `json:"-"`
}

func (balance *WSBalance) UnmarshalJSON(data []byte) error {
	type Alias WSBalance
	aux := struct {
		ChangeTime *int64 `json:"changeTime,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(balance),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.ChangeTime != nil {
		t := time.Unix(0, *aux.ChangeTime*1000000)
		balance.ChangeTime = &t
	}
	return nil
}

type WSBalanceEvent struct {
	Action    string    `json:"action"`
	Code      int       `json:"code"`
	Ch        string    `json:"ch"`
	Timestamp time.Time `json:"-"`
	Balance   WSBalance `json:"data,omitempty"`
}

func (wsCap *WSBalanceEvent) UnmarshalJSON(data []byte) error {
	type Alias WSBalanceEvent
	aux := struct {
		Timestamp int64 `json:"ts,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(wsCap),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	wsCap.Timestamp = time.Unix(0, aux.Timestamp*1000000)
	return nil
}

type Balance struct {
	Symbol    string  `json:"_"`
	Currency  string  `json:"-"`
	Balance   float64 `json:"-"`
	Available float64 `json:"-"`
	Trade     float64 `json:"-"`
	Frozen    float64 `json:"-"`
	Loan      float64 `json:"-"`
	Interest  float64 `json:"-"`
	Lock      float64 `json:"-"`
	Bank      float64 `json:"-"`
}

type NewOrderResponse struct {
	OrderID       int64  `json:"order_id"`
	ClientOrderID int64  `json:"client_order_id"`
	OrderIDStr    string `json:"order_id_str"`
}

type CancelAllResponse struct {
	SuccessCount int64 `json:"success-count"`
	FailedCount  int64 `json:"failed-count"`
	NextID       int64 `json:"next-id"`
}

type Account struct {
	ID       int64            `json:"id"`
	Type     string           `json:"type"`
	State    string           `json:"state"`
	Balances []AccountBalance `json:"list"`
}

type AccountBalance struct {
	Currency string  `json:"currency"`
	Type     string  `json:"type"`
	Balance  float64 `json:"balance,string"`
}

type WSOrderEvent struct {
	Action    string    `json:"action"`
	Code      int       `json:"code"`
	Ch        string    `json:"ch"`
	Timestamp time.Time `json:"-"`
	Order     WSOrder   `json:"data,omitempty"`
}

func (wsCap *WSOrderEvent) UnmarshalJSON(data []byte) error {
	type Alias WSOrderEvent
	aux := struct {
		Timestamp int64 `json:"ts,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(wsCap),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	wsCap.Timestamp = time.Unix(0, aux.Timestamp*1000000)
	return nil
}

type WSOrder struct {
	EventType     string     `json:"eventType,omitempty"`
	Symbol        string     `json:"symbol,omitempty"`
	ClientOrderID string     `json:"clientOrderId,omitempty"`
	OrderSide     *string    `json:"orderSide,omitempty"`
	OrderStatus   *string    `json:"orderStatus,omitempty"`
	ErrCode       *int       `json:"errCode,omitempty"`
	ErrMessage    *string    `json:"errMessage,omitempty"`
	LastActTime   *time.Time `json:"-"`

	AccountId   *int64   `json:"accountId,omitempty"`
	OrderId     *int64   `json:"orderId,omitempty"`
	OrderSource *string  `json:"orderSource,omitempty"`
	OrderPrice  *float64 `json:"orderPrice,string,omitempty"`
	OrderSize   *float64 `json:"orderSize,string,omitempty"`
	OrderValue  *float64 `json:"orderValue,string,omitempty"`
	Type        *string  `json:"type,omitempty"`

	TradePrice  *float64   `json:"tradePrice,string,omitempty"`
	TradeVolume *float64   `json:"tradeVolume,string,omitempty"`
	TradeId     *int64     `json:"tradeId,omitempty"`
	TradeTime   *time.Time `json:"-"`
	Aggressor   *bool      `json:"aggressor,omitempty"`
	RemainAmt   *float64   `json:"remainAmt,string,omitempty"`
	ExecAmt     *float64   `json:"execAmt,string,omitempty"`
}

//{"action":"push","ch":"orders#filusdt","data":{"orderSize":"0.1","remainAmt":"0.1","execAmt":"0","lastActTime":1618486619890,"orderSource":"spot-web","orderPrice":"165","symbol":"filusdt","type":"buy-limit","clientOrderId":"","orderStatus":"canceled","orderId":255726957080695,"eventType":"cancellation"}}

func (wsOrder *WSOrder) UnmarshalJSON(data []byte) error {
	type Alias WSOrder
	aux := struct {
		LastActTime *int64 `json:"lastActTime,omitempty"`
		TradeTime   *int64 `json:"tradeTime,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(wsOrder),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.LastActTime != nil {
		lat := time.Unix(0, *aux.LastActTime*1000000)
		wsOrder.LastActTime = &lat
	}
	if aux.TradeTime != nil {
		tt := time.Unix(0, *aux.TradeTime*1000000)
		wsOrder.TradeTime = &tt
	}
	return nil
}

type Trade struct {
	TradeFee      float64 `json:"trade_fee"`
	FeeAsset      string  `json:"fee_asset"`
	ID            string  `json:"id"`
	TradeVolume   int64   `json:"trade_volume"`
	TradePrice    float64 `json:"trade_price"`
	TradeTurnover float64 `json:"trade_turnover"`
	CreatedAt     int64   `json:"created_at"`
	Profit        float64 `json:"profit"`
	RealProfit    float64 `json:"real_profit"`
	Role          string  `json:"role"`
}

type WsCap struct {
	Action    string          `json:"action"`
	Code      int             `json:"code"`
	Ch        string          `json:"ch"`
	Message   string          `json:"message"`
	Timestamp time.Time       `json:"-"`
	Data      json.RawMessage `json:"data,omitempty"`
}

func (wsCap *WsCap) UnmarshalJSON(data []byte) error {
	type Alias WsCap
	aux := struct {
		Timestamp int64 `json:"ts,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(wsCap),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	wsCap.Timestamp = time.Unix(0, aux.Timestamp*1000000)
	return nil
}
