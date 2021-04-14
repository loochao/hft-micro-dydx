package hbswap

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
	Symbol            string  `json:"symbol"`
	ContractCode      string  `json:"contract_code"`
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
	ContractCode    string    `json:"contract_code"`
	Symbol          string    `json:"symbol"`
	FeeAsset        string   `json:"fee_asset"`
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
