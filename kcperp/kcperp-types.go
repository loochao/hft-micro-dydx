package kcperp

import "encoding/json"

//      {
//         "symbol":"XTZUSDTM",
//         "rootSymbol":"USDT",
//         "type":"FFWCSX",
//         "firstOpenDate":1617955200000,
//         "expireDate":null,
//         "settleDate":null,
//         "baseCurrency":"XTZ",
//         "quoteCurrency":"USDT",
//         "settleCurrency":"USDT",
//         "maxOrderQty":1000000,
//         "maxPrice":1000000.0000000000,
//         "lotSize":1,
//         "tickSize":0.001,
//         "indexPriceTickSize":0.001,
//         "multiplier":1.0,
//         "initialMargin":0.05,
//         "maintainMargin":0.025,
//         "maxRiskLimit":200000,
//         "minRiskLimit":200000,
//         "riskStep":100000,
//         "makerFeeRate":0.00020,
//         "takerFeeRate":0.00060,
//         "takerFixFee":0.0000000000,
//         "makerFixFee":0.0000000000,
//         "settlementFee":null,
//         "isDeleverage":true,
//         "isQuanto":false,
//         "isInverse":false,
//         "markMethod":"FairPrice",
//         "fairMethod":"FundingRate",
//         "fundingBaseSymbol":".XTZINT8H",
//         "fundingQuoteSymbol":".USDTINT8H",
//         "fundingRateSymbol":".XTZUSDTMFPI8H",
//         "indexSymbol":".KXTZUSDT",
//         "settlementSymbol":"",
//         "status":"Open",
//         "fundingFeeRate":0.002212,
//         "predictedFundingFeeRate":0.001441,
//         "openInterest":"392327",
//         "turnoverOf24h":1717520.38173818,
//         "volumeOf24h":263020.00000000,
//         "markPrice":6.543,
//         "indexPrice":6.538,
//         "lastTradePrice":6.5530000000,
//         "nextFundingRateTime":9103241,
//         "maxLeverage":20,
//         "sourceExchanges":[
//            "huobi",
//            "Okex",
//            "Binance",
//            "Kucoin",
//            "Poloniex",
//            "Bittrex"
//         ],
//         "premiumsSymbol1M":".XTZUSDTMPI",
//         "premiumsSymbol8H":".XTZUSDTMPI8H",
//         "fundingBaseSymbol1M":".XTZINT",
//         "fundingQuoteSymbol1M":".USDTINT",
//         "lowPrice":6.263,
//         "highPrice":6.828,
//         "priceChgPct":-0.0149,
//         "priceChg":-0.099
//      },

type Contract struct {
	Symbol        string `json:"symbol,omitempty"`
	RootSymbol    string `json:"rootSymbol,omitempty"`
	Type          string `json:"type,omitempty"`
	FirstOpenDate int64  `json:"firstOpenDate,omitempty"`

	BaseCurrency            string   `json:"baseCurrency,omitempty"`
	QuoteCurrency           string   `json:"quoteCurrency,omitempty"`
	MaxOrderQty             float64  `json:"maxOrderQty,omitempty"`
	MaxPrice                float64  `json:"maxPrice,omitempty"`
	LotSize                 float64  `json:"lotSize,omitempty"`
	TickSize                float64  `json:"tickSize,omitempty"`
	IndexPriceTickSize      float64  `json:"indexPriceTickSize,omitempty"`
	Multiplier              float64  `json:"multiplier,omitempty"`
	InitialMargin           float64  `json:"initialMargin,omitempty"`
	MaintainMargin          float64  `json:"maintainMargin,omitempty"`
	MaxRiskLimit            float64  `json:"maxRiskLimit,omitempty"`
	MinRiskLimit            float64  `json:"minRiskLimit,omitempty"`
	RiskStep                float64  `json:"riskStep,omitempty"`
	MakerFeeRate            float64  `json:"MakerFeeRate,omitempty"`
	TakerFeeRate            float64  `json:"takerFeeRate,omitempty"`
	TakerFixFee             float64  `json:"takerFixFee,omitempty"`
	MakerFixFee             float64  `json:"makerFixFee,omitempty"`
	IsDeleverage            bool     `json:"isDeleverage,omitempty"`
	IsQuanto                bool     `json:"isQuanto,omitempty"`
	IsInverse               bool     `json:"isInverse,omitempty"`
	MarkMethod              string   `json:"markMethod,omitempty"`
	FairMethod              string   `json:"fairMethod,omitempty"`
	FundingBaseSymbol       string   `json:"fundingBaseSymbol,omitempty"`
	FundingQuoteSymbol      string   `json:"fundingQuoteSymbol,omitempty"`
	FundingRateSymbol       string   `json:"fundingRateSymbol,omitempty"`
	IndexSymbol             string   `json:"indexSymbol,omitempty"`
	Status                  string   `json:"status,omitempty"`
	FundingFeeRate          float64  `json:"fundingFeeRate,omitempty"`
	PredictedFundingFeeRate float64  `json:"predictedFundingFeeRate,omitempty"`
	OpenInterest            float64  `json:"openInterest,string,omitempty"`
	TurnoverOf24h           float64  `json:"turnoverOf24h,omitempty"`
	VolumeOf24h             float64  `json:"volumeOf24h,omitempty"`
	MarkPrice               float64  `json:"markPrice,omitempty"`
	IndexPrice              float64  `json:"indexPrice,omitempty"`
	LastTradePrice          float64  `json:"lastTradePrice,omitempty"`
	NextFundingRateTime     int64    `json:"nextFundingRateTime,omitempty"`
	MaxLeverage             float64  `json:"maxLeverage,omitempty"`
	SourceExchanges         []string `json:"sourceExchanges,omitempty"`
	PremiumsSymbol1M        string   `json:"premiumsSymbol1M,omitempty"`
	PremiumsSymbol8H        string   `json:"premiumsSymbol8H,omitempty"`
	FundingBaseSymbol1M     string   `json:"fundingBaseSymbol1M,omitempty"`
	FundingQuoteSymbol1M    string   `json:"fundingQuoteSymbol1M,omitempty"`
	LowPrice                float64  `json:"lowPrice,omitempty"`
	HighPrice               float64  `json:"highPrice,omitempty"`
	PriceChgPct             float64  `json:"priceChgPct,omitempty"`
	PriceChg                float64  `json:"priceChg,omitempty"`
}

type DataCap struct {
	Code int             `json:"code,string,omitempty"`
	Msg  string          `json:"msg,omitempty"`
	Data json.RawMessage `json:"data,omitempty"`
}
