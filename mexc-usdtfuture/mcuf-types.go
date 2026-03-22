package mexc_usdtfuture

import (
	"encoding/json"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

type DataCap struct {
	Code    int             `json:"code"`
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
}

//响应示例
//
//{
//    "success":true,
//    "code":0,
//    "data":[
//        {
//            "symbol":"BTC_USDT",
//            "displayName":"BTC_USDT永续",
//            "displayNameEn":"BTC_USDT SWAP",
//            "positionOpenType":3,
//            "baseCoin":"BTC",
//            "quoteCoin":"USDT",
//            "settleCoin":"USDT",
//            "contractSize":0.0001,
//            "minLeverage":1,
//            "maxLeverage":125,
//            "priceScale":2,
//            "volScale":0,
//            "amountScale":4,
//            "priceUnit":0.5,
//            "volUnit":1,
//            "minVol":1,
//            "maxVol":5000000,
//            "bidLimitPriceRate":0.03,
//            "askLimitPriceRate":0.03,
//            "takerFeeRate":0.0006,
//            "makerFeeRate":0.0002,
//            "maintenanceMarginRate":0.004,
//            "initialMarginRate":0.008,
//            "riskBaseVol":150000,
//            "riskIncrVol":150000,
//            "riskIncrMmr":0.004,
//            "riskIncrImr":0.004,
//            "riskLevelLimit":5,
//            "priceCoefficientVariation":0.05,
//            "indexOrigin":[
//                "Binance",
//                "GATEIO",
//                "HUOBI",
//                "MXC"
//            ],
//            "state":0,
//            "isNew":false,
//            "isHot":true,
//            "isHidden":false
//        },
//    ]
//}

type Contract struct {
	Symbol                    string   `json:"symbol"`
	DisplayName               string   `json:"displayName"`
	DisplayNameEn             string   `json:"displayNameEn"`
	PositionOpenType          float64   `json:"positionOpenType"`
	BaseCoin                  string   `json:"baseCoin"`
	QuoteCoin                 string   `json:"quoteCoin"`
	SettleCoin                string   `json:"settleCoin"`
	ContractSize              float64  `json:"contractSize"`
	MinLeverage               float64  `json:"minLeverage"`
	MaxLeverage               float64  `json:"maxLeverage"`
	PriceScale                float64  `json:"priceScale"`
	VolScale                  float64  `json:"volScale"`
	AmountScale               float64  `json:"amountScale"`
	PriceUnit                 float64  `json:"priceUnit"`
	VolUnit                   float64  `json:"volUnit"`
	MinVol                    float64  `json:"minVol"`
	MaxVol                    float64  `json:"maxVol"`
	BidLimitPriceRate         float64  `json:"bidLimitPriceRate"`
	AskLimitPriceRate         float64  `json:"askLimitPriceRate"`
	TakerFeeRate              float64  `json:"takerFeeRate"`
	MakerFeeRate              float64  `json:"makerFeeRate"`
	MaintenanceMarginRate     float64  `json:"maintenanceMarginRate"`
	InitialMarginRate         float64  `json:"initialMarginRate"`
	RiskBaseVol               float64  `json:"riskBaseVol"`
	RiskIncrVol               float64  `json:"riskIncrVol"`
	RiskIncrMmr               float64  `json:"riskIncrMmr"`
	RiskIncrImr               float64  `json:"riskIncrImr"`
	RiskLevelLimit            float64  `json:"riskLevelLimit"`
	PriceCoefficientVariation float64  `json:"priceCoefficientVariation"`
	IndexOrigin               []string `json:"indexOrigin"`
	State                     int64    `json:"state"`
	IsNew                     bool     `json:"isNew"`
	IsHot                     bool     `json:"isHot"`
	IsHidden                  bool     `json:"isHidden"`
}

type WSParam struct {
	Symbol string `json:"symbol,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

type WSReq struct {
	Method string  `json:"method"`
	Param  WSParam `json:"param,omitempty"`
}

type Ticker struct {
	Symbol string  `json:"symbol"`
	Ask1   float64 `json:"ask1"`
	Bid1   float64 `json:"bid1"`
}

type Depth5 struct {
	Symbol    string        `json:"-"`
	Bids      [5][2]float64 `json:"-"`
	Asks      [5][2]float64 `json:"_"`
	ParseTime time.Time     `json:"-"`
	EventTime time.Time     `json:"-"`
}

func (depth *Depth5) GetBidPrice() float64 {
	return depth.Bids[0][0]
}

func (depth *Depth5) GetAskPrice() float64 {
	return depth.Asks[0][0]
}

func (depth *Depth5) GetBidSize() float64 {
	return depth.Asks[0][1]
}

func (depth *Depth5) GetAskSize() float64 {
	return depth.Asks[0][1]
}

func (depth *Depth5) GetAsks() common.Asks {
	return depth.Asks[:]
}

func (depth *Depth5) GetBids() common.Bids {
	return depth.Bids[:]
}

func (depth *Depth5) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (depth *Depth5) GetSymbol() string       { return depth.Symbol }
func (depth *Depth5) GetEventTime() time.Time { return depth.EventTime }
func (depth *Depth5) UnmarshalJSON(data []byte) error {
	type Alias Depth5
	aux := struct {
		Data struct {
			Bids [5][3]float64 `json:"bids,omitempty"`
			Asks [5][3]float64 `json:"asks,omitempty"`
		} `json:"data,omitempty"`
		Channel   string `json:"channel,omitempty"`
		EventTime int64  `json:"ts,omitempty"`
		Symbol    string `json:"symbol"`
	}{}
	if err := json.Unmarshal(data, &aux); err != nil {
		logger.Debugf("ERR %v", err)
		return err
	}
	depth.Bids = [5][2]float64{}
	depth.Asks = [5][2]float64{}
	for i, d := range aux.Data.Bids {
		depth.Bids[i][0] = d[0]
		depth.Bids[i][1] = d[1]
	}
	for i, d := range aux.Data.Asks {
		depth.Asks[i][0] = d[0]
		depth.Asks[i][1] = d[1]
	}
	depth.EventTime = time.Unix(0, aux.EventTime*1000000)
	depth.ParseTime = time.Now()
	depth.Symbol = aux.Symbol
	return nil
}
