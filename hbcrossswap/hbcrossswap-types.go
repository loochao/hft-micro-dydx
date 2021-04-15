package hbcrossswap

import (
	"encoding/json"
	"github.com/geometrybase/hft-micro/logger"
	"strings"
	"time"
)

type DataCap struct {
	Status    string          `json:"status"`
	Data      json.RawMessage `json:"data,omitempty"`
	ErrCode   int             `json:"err_code,omitempty"`
	ErrMsg    string          `json:"err_msg,omitempty"`
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

type Contract struct {
	BaseSymbol        string  `json:"symbol"`
	Symbol            string  `json:"contract_code"`
	ContractSize      float64 `json:"contract_size"`
	PriceTick         float64 `json:"price_tick"`
	CreateDate        string  `json:"create_date"`
	DeliveryTime      string  `json:"delivery_time"`
	ContractStatus    int     `json:"contract_status"`
	SettlementDate    string  `json:"settlement_date"`
	SupportMarginMode string  `json:"support_margin_mode"`
}

type Kline struct {
	ID            int64   `json:"id"`
	Vol           float64 `json:"vol"`
	Count         float64 `json:"count"`
	Open          float64 `json:"open"`
	Close         float64 `json:"close"`
	Low           float64 `json:"low"`
	High          float64 `json:"high"`
	Amount        float64 `json:"amount"`
	TradeTurnover float64 `json:"trade_turnover"`
}

type Depth20 struct {
	Symbol    string         `json:"-"`
	Ch        string         `json:"ch"`
	Bids      [20][2]float64 `json:"bids,omitempty"`
	Asks      [20][2]float64 `json:"asks,omitempty"`
	Version   int64          `json:"version"`
	ID        int64          `json:"id"`
	MRID      int64          `json:"mrid"`
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
	depth.Symbol = strings.Split(depth.Ch, ".")[1]
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

type FundingRate struct {
	EstimatedRate   float64   `json:"estimated_rate,string"`
	FundingRate     float64   `json:"funding_rate,string"`
	Symbol          string    `json:"contract_code"`
	BaseSymbol      string    `json:"symbol"`
	FeeAsset        string    `json:"fee_asset"`
	FundingTime     time.Time `json:"-"`
	NextFundingTime time.Time `json:"-"`
}

func (fr *FundingRate) UnmarshalJSON(data []byte) error {
	type Alias FundingRate
	aux := struct {
		FundingTime     int64 `json:"funding_time,string,omitempty"`
		NextFundingTime int64 `json:"next_funding_time,string,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(fr),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	fr.FundingTime = time.Unix(0, aux.FundingTime*1000000)
	fr.NextFundingTime = time.Unix(0, aux.NextFundingTime*1000000)
	return nil
}

type Position struct {
	BaseSymbol     string  `json:"symbol"`
	Symbol         string  `json:"contract_code"`
	Volume         float64 `json:"volume"`
	Available      float64 `json:"available"`
	Frozen         float64 `json:"frozen"`
	CostOpen       float64 `json:"cost_open"`
	CostHold       float64 `json:"cost_hold"`
	ProfitUnreal   float64 `json:"profit_unreal"`
	ProfitRate     float64 `json:"profit_rate"`
	LeverRate      float64 `json:"lever_rate"`
	PositionMargin float64 `json:"position_margin"`
	Direction      string  `json:"direction"`
	Profit         float64 `json:"profit"`
	LastPrice      float64 `json:"last_price"`
	MarginAsset    string  `json:"margin_asset"`
	MarginMode     string  `json:"margin_mode"`
	MarginAccount  string  `json:"margin_account"`
}

//{"status":"ok","data":{"order_id":832225072378114048,"client_order_id":16184595178081,"order_id_str":"832225072378114048"},"ts":1618459519269}

type NewOrderResponse struct {
	OrderID       int64  `json:"order_id"`
	ClientOrderID int64  `json:"client_order_id"`
	OrderIDStr    string `json:"order_id_str"`
}

type CancelAllResponse struct {
	Errors    []json.RawMessage `json:"errors"`
	Successes []string          `json:"-"`
}

func (fr *CancelAllResponse) UnmarshalJSON(data []byte) error {
	type Alias CancelAllResponse
	aux := struct {
		Successes string `json:"successes"`
		*Alias
	}{
		Alias: (*Alias)(fr),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	fr.Successes = strings.Split(aux.Successes, ",")
	return nil
}

type Account struct {
	MarginMode        string           `json:"margin_mode"`
	MarginAccount     string           `json:"margin_account"`
	MarginAsset       string           `json:"margin_asset"`
	MarginBalance     float64          `json:"margin_balance,omitempty"`
	MarginPosition    float64          `json:"margin_position"`
	MarginStatic      float64          `json:"margin_static,omitempty"`
	MarginFrozen      float64          `json:"margin_frozen,omitempty"`
	ProfitReal        float64          `json:"profit_real,omitempty"`
	ProfitUnreal      float64          `json:"profit_unreal,omitempty"`
	WithdrawAvailable float64          `json:"withdraw_available,omitempty"`
	RiskRate          float64          `json:"risk_rate,omitempty"`
	ContractDetail    []ContractDetail `json:"contract_detail"`
}

type ContractDetail struct {
	BaseSymbol       string  `json:"symbol"`
	Symbol           string  `json:"contract_code"`
	MarginPosition   float64 `json:"margin_position,omitempty"`
	MarginFrozen     float64 `json:"margin_frozen,omitempty"`
	MarginAvailable  float64 `json:"margin_available,omitempty"`
	ProfitUnreal     float64 `json:"profit_unreal,omitempty"`
	LiquidationPrice float64 `json:"liquidation_price,omitempty"`
	LeverRate        float64 `json:"lever_rate,omitempty"`
	AdjustFactor     float64 `json:"adjust_factor,omitempty"`
}

//{
//  "op": "notify",
//  "topic": "orders_cross.fil-usdt",
//  "ts": 1618465548105,
//  "symbol": "FIL",
//  "contract_code": "FIL-USDT",
//  "volume": 10,
//  "price": 175,
//  "order_price_type": "limit",
//  "direction": "sell",
//  "offset": "open",
//  "status": 3,
//  "lever_rate": 3,
//  "order_id": 832250359040368640,
//  "order_id_str": "832250359040368640",
//  "client_order_id": null,
//  "order_source": "web",
//  "order_type": 1,
//  "created_at": 1618465548054,
//  "trade_volume": 0,
//  "trade_turnover": 0,
//  "fee": 0,
//  "trade_avg_price": 0.000,
//  "margin_frozen": 58.333333333333333333,
//  "profit": 0,
//  "trade": [],
//  "canceled_at": 0,
//  "fee_asset": "USDT",
//  "margin_asset": "USDT",
//  "uid": "211055476",
//  "liquidation_type": "0",
//  "margin_mode": "cross",
//  "margin_account": "USDT",
//  "is_tpsl": 0,
//  "real_profit": 0
//}

type WSOrder struct {
	Op             string    `json:"op"`
	Topic          string    `json:"topic"`
	EventTime      time.Time `json:"-"`
	BaseSymbol     string    `json:"symbol"`
	Symbol         string    `json:"contract_code"`
	Volume         int64     `json:"volume"`
	Price          float64   `json:"price"`
	OrderPriceType string    `json:"order_price_type"`
	Direction      string    `json:"direction"`
	Offset         string    `json:"offset"`
	Status         int       `json:"status"`
	LeverRate      int       `json:"lever_rate"`
	OrderID        int64     `json:"order_id"`
	OrderIDStr     string    `json:"order_id_str"`
	ClientOrderID  int64     `json:"client_order_id"`
	OrderSource    string    `json:"order_source"`
	OrderType      int       `json:"order_type"`
	CreatedAt      time.Time `json:"created_at"`
	TradeVolume    int64     `json:"trade_volume"`
	TradeTurnover  int64     `json:"trade_turnover"`
	Fee            float64   `json:"fee"`
	TradeAvgPrice  float64   `json:"trade_avg_price"`
	MarginFrozen   float64   `json:"margin_frozen"`
	Profit         float64   `json:"profit"`
	Trade          []Trade   `json:"trade"`
}

func (wsOrder *WSOrder) UnmarshalJSON(data []byte) error {
	type Alias WSOrder
	aux := struct {
		CreatedAt int64 `json:"created_at"`
		EventTime int64 `json:"ts"`
		*Alias
	}{
		Alias: (*Alias)(wsOrder),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	wsOrder.CreatedAt = time.Unix(0, aux.CreatedAt*1000000)
	wsOrder.EventTime = time.Unix(0, aux.EventTime*1000000)
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
	Op        string          `json:"op"`
	Type      string          `json:"type"`
	ErrCode   int             `json:"err_code,omitempty"`
	ErrMsg    string          `json:"err_msg,omitempty"`
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

type User struct {
	UserID string `json:"user-id"`
}

type WsUser struct {
	Op        string    `json:"op"`
	Type      string    `json:"type"`
	ErrCode   int       `json:"err_code,omitempty"`
	ErrMsg    string    `json:"err_msg,omitempty"`
	Timestamp time.Time `json:"-"`
	User      User      `json:"data,omitempty"`
}

func (wsCap *WsUser) UnmarshalJSON(data []byte) error {
	type Alias WsUser
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

type SubResp struct {
	Op        string    `json:"op"`
	Topic     string    `json:"topic"`
	ErrCode   int       `json:"err_code,omitempty"`
	ErrMsg    string    `json:"err_msg,omitempty"`
	Timestamp time.Time `json:"-"`
}

func (wsCap *SubResp) UnmarshalJSON(data []byte) error {
	type Alias SubResp
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

type WSPositions struct {
	Op        string     `json:"op"`
	Topic     string     `json:"topic"`
	Event     string     `json:"event"`
	Timestamp time.Time  `json:"-"`
	Positions []Position `json:"data"`
}

func (wsPosition *WSPositions) UnmarshalJSON(data []byte) error {
	type Alias WSPositions
	aux := struct {
		Timestamp int64 `json:"ts,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(wsPosition),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	wsPosition.Timestamp = time.Unix(0, aux.Timestamp*1000000)
	return nil
}

type WSAccounts struct {
	Op        string    `json:"op"`
	Topic     string    `json:"topic"`
	Event     string    `json:"event"`
	Timestamp time.Time `json:"-"`
	Accounts  []Account `json:"data"`
}

func (wsAccounts *WSAccounts) UnmarshalJSON(data []byte) error {
	type Alias WSAccounts
	aux := struct {
		Timestamp int64 `json:"ts,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(wsAccounts),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	wsAccounts.Timestamp = time.Unix(0, aux.Timestamp*1000000)
	return nil
}
