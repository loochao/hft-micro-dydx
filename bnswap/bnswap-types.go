package bnswap

import (
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"net/url"
	"strconv"
	"time"
)

type Depth20 struct {
	EventTime    time.Time      `json:"-"`
	Symbol       string         `json:"s,omitempty"`
	LastUpdateId int64          `json:"u,omitempty"`
	Bids         [20][2]float64 `json:"b,omitempty"`
	Asks         [20][2]float64 `json:"a,omitempty"`
	ParseTime    time.Time      `json:"-"`
}

func (depth *Depth20) UnmarshalJSON(data []byte) error {
	type Alias Depth20
	aux := &struct {
		Bids      [20][2]string `json:"b,omitempty"`
		Asks      [20][2]string `json:"a,omitempty"`
		EventName string      `json:"e,omitempty"`
		EventTime int64       `json:"E,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(depth),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		logger.Debugf("ERR %v", err)
		return err
	}
	depth.Bids = [20][2]float64{}
	depth.Asks = [20][2]float64{}
	for i, d := range aux.Bids {
		price, _ := strconv.ParseFloat(d[0], 64)
		size, _ := strconv.ParseFloat(d[1], 64)
		depth.Bids[i][0] = price
		depth.Bids[i][1] = size
	}
	for i, d := range aux.Asks {
		price, _ := strconv.ParseFloat(d[0], 64)
		size, _ := strconv.ParseFloat(d[1], 64)
		depth.Asks[i][0] = price
		depth.Asks[i][1] = size
	}
	depth.EventTime = time.Unix(0, aux.EventTime*1000000)
	return nil
}

type DepthFast struct {
	EventTime    time.Time    `json:"-"`
	Symbol       string       `json:"s,omitempty"`
	LastUpdateId int64        `json:"-"`
	Bids         [][2]float64 `json:"-"`
	Asks         [][2]float64 `json:"-"`
	ArrivalTime  time.Time    `json:"-"`
}

func (depth *DepthFast) UnmarshalJSON(data []byte) error {
	type Alias DepthFast
	aux := &struct {
		Bids         [][2]json.RawMessage `json:"b,omitempty"`
		Asks         [][2]json.RawMessage `json:"a,omitempty"`
		EventName    string               `json:"e,omitempty"`
		EventTime    json.RawMessage      `json:"E,omitempty"`
		LastUpdateId json.RawMessage      `json:"u,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(depth),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	depth.Bids = make([][2]float64, 0)
	depth.Asks = make([][2]float64, 0)
	for _, d := range aux.Bids {
		price, _ := common.ParseFloat(d[0])
		size, _ := common.ParseFloat(d[1])
		depth.Bids = append(depth.Bids, [2]float64{price, size})
	}
	for _, d := range aux.Asks {
		price, _ := common.ParseFloat(d[0])
		size, _ := common.ParseFloat(d[1])
		depth.Asks = append(depth.Asks, [2]float64{price, size})
	}
	eventTime, _ := common.ParseInt(aux.EventTime)
	depth.EventTime = time.Unix(0, eventTime*1000000)
	depth.LastUpdateId, _ = common.ParseInt(aux.LastUpdateId)
	return nil
}

type DepthStream struct {
	Stream string  `json:"stream"`
	Data   Depth20 `json:"data"`
}

type DepthFastStream struct {
	Stream string    `json:"stream"`
	Data   DepthFast `json:"data"`
}

type MarkPrice struct {
	EventTime            time.Time `json:"-"`
	Symbol               string    `json:"s,omitempty"`
	MarkPrice            float64   `json:"p,string,omitempty"`
	IndexPrice           float64   `json:"i,string,omitempty"`
	EstimatedSettlePrice float64   `json:"P,string,omitempty"`
	FundingRate          float64   `json:"r,string,omitempty"`
	NextFundingTime      time.Time `json:"-"`
	ArrivalTime          time.Time `json:"-"`
}

func (mpu *MarkPrice) ToString() string {
	return fmt.Sprintf(
		"S=%s, FR=%v, MP=%f, IP=%f, ESP=%f, AT=%v, ET=%v, NFT=%v",
		mpu.Symbol, mpu.FundingRate,
		mpu.MarkPrice,
		mpu.IndexPrice,
		mpu.EstimatedSettlePrice,
		mpu.ArrivalTime,
		mpu.EventTime,
		mpu.NextFundingTime,
	)
}

func (mpu *MarkPrice) UnmarshalJSON(data []byte) error {
	type Alias MarkPrice
	aux := &struct {
		EventName       string `json:"e,omitempty"`
		NextFundingTime int64  `json:"T,omitempty"`
		EventTime       int64  `json:"E,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(mpu),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	mpu.NextFundingTime = time.Unix(aux.NextFundingTime/1000, 0)
	mpu.EventTime = time.Unix(aux.EventTime/1000, 0)
	return nil
}

type MarkPriceStream struct {
	Stream string    `json:"stream"`
	Data   MarkPrice `json:"data"`
}

type KlineParams struct {
	Symbol    string `json:"symbol,omitempty"`
	Interval  string `json:"interval,omitempty"`
	Limit     int64  `json:"limit,omitempty"`
	StartTime int64  `json:"startTime,omitempty"`
	EndTime   int64  `json:"endTime,omitempty"`
}

func (kp *KlineParams) ToUrlValues() url.Values {
	values := url.Values{}
	values.Set("symbol", kp.Symbol)
	values.Set("interval", kp.Interval)
	if kp.Limit > 0 {
		values.Set("limit", fmt.Sprintf("%d", kp.Limit))
	}
	if kp.StartTime > 0 {
		values.Set("startTime", fmt.Sprintf("%d", kp.StartTime))
	}
	if kp.EndTime > 0 {
		values.Set("endTime", fmt.Sprintf("%d", kp.EndTime))
	}
	return values
}

type ServerTime struct {
	ServerTime int64 `json:"serverTime"`
}

//{
//"entryPrice": "0.00000",
//"marginType": "isolated",
//"isAutoAddMargin": "false",
//"isolatedMargin": "0.00000000",
//"leverage": "10",
//"liquidationPrice": "0",
//"markPrice": "6679.50671178",
//"maxNotionalValue": "20000000",
//"positionAmt": "0.000",
//"symbol": "BTCUSDT",
//"unRealizedProfit": "0.00000000",
//"positionSide": "BOTH",
//},

type Position struct {
	EntryPrice       float64 `json:"entryPrice,string"`
	MarginType       string  `json:"marginType"`
	IsAutoAddMargin  bool    `json:"isAutoAddMargin,string"`
	IsolatedMargin   float64 `json:"isolatedMargin,string"`
	Leverage         int64   `json:"leverage,string"`
	LiquidationPrice float64 `json:"liquidationPrice,string"`
	MarkPrice        float64 `json:"markPrice,string"`
	MaxNotionalValue int64   `json:"maxNotionalValue,string"`
	PositionAmt      float64 `json:"positionAmt,string"`
	Symbol           string
	UnRealizedProfit float64   `json:"unRealizedProfit,string"`
	PositionSide     string    `json:"positionSide"`
	UpdateTime       time.Time `json:"-"`
}

func (position *Position) ToString() string {
	return fmt.Sprintf("Symbol=%s,EntryPrice=%f,PositionAmt=%f", position.Symbol, position.EntryPrice, position.PositionAmt)
}

type Asset struct {
	Asset                  string
	InitialMargin          *float64 `json:"initialMargin,string,omitempty"`
	MaintMargin            *float64 `json:"maintMargin,string,omitempty"`
	MarginBalance          *float64 `json:"marginBalance,string,omitempty"`
	MaxWithdrawAmount      *float64 `json:"maxWithdrawAmount,string,omitempty"`
	OpenOrderInitialMargin *float64 `json:"openOrderInitialMargin,string,omitempty"`
	PositionInitialMargin  *float64 `json:"positionInitialMargin,string,omitempty"`
	UnrealizedProfit       *float64 `json:"unrealizedProfit,string,omitempty"`
	WalletBalance          *float64 `json:"walletBalance,string"`
	CrossWalletBalance     *float64 `json:"crossWalletBalance,string"`
	CrossUnPnl             *float64 `json:"crossUnPnl,string,omitempty"`
	AvailableBalance       *float64 `json:"availableBalance,string,omitempty"`
}

//{
//    "assets": [
//        {
//            "asset": "USDT",
//            "initialMargin": "0.33683000",
//            "maintMargin": "0.02695000",
//            "marginBalance": "8.74947592",
//            "maxWithdrawAmount": "8.41264592",
//            "openOrderInitialMargin": "0.00000000",
//            "positionInitialMargin": "0.33683000",
//            "unrealizedProfit": "-0.44537584",
//            "walletBalance": "9.19485176"
//        }
//     ],
//     "canDeposit": True,
//     "canTrade": True,
//     "canWithdraw": True,
//     "feeTier": 2,
//     "maxWithdrawAmount": "8.41264592",
//     "positions": [
//         {
//            "leverage": "20",
//            "initialMargin": "0.33683",
//            "maintMargin": "0.02695",
//            "openOrderInitialMargin": "0.00000",
//            "positionInitialMargin": "0.33683",
//            "symbol": "BTCUSDT",
//            "unrealizedProfit": "-0.44537584"
//         }
//     ],
//     "totalInitialMargin": "0.33683000",
//     "totalMaintMargin": "0.02695000",
//     "totalMarginBalance": "8.74947592",
//     "totalOpenOrderInitialMargin": "0.00000000",
//     "totalPositionInitialMargin": "0.33683000",
//     "totalUnrealizedProfit": "-0.44537584",
//     "totalWalletBalance": "9.19485176",
//     "updateTime": 0
// }

type Account struct {
	Assets                      []Asset    `json:"assets"`
	CanDeposit                  bool       `json:"canDeposit"`
	CanTrade                    bool       `json:"canTrade"`
	CanWithdraw                 bool       `json:"canWithdraw"`
	FeeTier                     int        `json:"feeTier"`
	MaxWithdrawAmount           float64    `json:"maxWithdrawAmount,string"`
	Positions                   []Position `json:"positions"`
	TotalInitialMargin          float64    `json:"totalInitialMargin,string"`
	TotalMaintMargin            float64    `json:"totalMaintMargin,string"`
	TotalMarginBalance          float64    `json:"totalMarginBalance,string"`
	TotalOpenOrderInitialMargin float64    `json:"totalOpenOrderInitialMargin,string"`
	TotalPositionInitialMargin  float64    `json:"totalPositionInitialMargin,string"`
	TotalUnrealizedProfit       float64    `json:"totalUnrealizedProfit,string"`
	TotalWalletBalance          float64    `json:"totalWalletBalance,string"`
	UpdateTime                  int        `json:"updateTime"`
}

type ListenKey struct {
	ListenKey string `json:"listenKey"`
}

func (lk *ListenKey) ToUrlValues() url.Values {
	values := url.Values{}
	values.Set("listenKey", lk.ListenKey)
	return values
}

type UpdateLeverageParams struct {
	Symbol   string `json:"symbol,omitempty"`
	Leverage int64  `json:"leverage,omitempty"`
}

func (ulp *UpdateLeverageParams) ToUrlValues() url.Values {
	values := url.Values{}
	values.Set("symbol", ulp.Symbol)
	values.Set("leverage", fmt.Sprintf("%d", ulp.Leverage))
	return values
}

// Response holds basic response data
type Response struct {
	Code int64
	Msg  string `json:"msg"`
}

type UpdateMarginTypeParams struct {
	Symbol     string `json:"symbol,omitempty"`
	MarginType string `json:"marginType,omitempty"`
}

func (ulp *UpdateMarginTypeParams) ToUrlValues() url.Values {
	values := url.Values{}
	values.Set("symbol", ulp.Symbol)
	values.Set("marginType", ulp.MarginType)
	return values
}

type ExchangeInfo struct {
	Code       int    `json:"code"`
	Msg        string `json:"msg"`
	Timezone   string `json:"timezone"`
	ServerTime int64  `json:"serverTime"`
	RateLimits []struct {
		RateLimitType string `json:"rateLimitType"`
		Interval      string `json:"interval"`
		Limit         int    `json:"limit"`
	} `json:"rateLimits"`
	ExchangeFilters interface{} `json:"exchangeFilters"`
	Symbols         []struct {
		Symbol             string   `json:"symbol"`
		ContractType       string   `json:"contractType"`
		Status             string   `json:"status"`
		BaseAsset          string   `json:"baseAsset"`
		BaseAssetPrecision int      `json:"baseAssetPrecision"`
		QuoteAsset         string   `json:"quoteAsset"`
		QuotePrecision     int      `json:"quotePrecision"`
		OrderTypes         []string `json:"orderTypes"`
		IcebergAllowed     bool     `json:"icebergAllowed"`
		Filters            []struct {
			FilterType          string  `json:"filterType"`
			MinPrice            float64 `json:"minPrice,string"`
			MaxPrice            float64 `json:"maxPrice,string"`
			TickSize            float64 `json:"tickSize,string"`
			MultiplierUp        float64 `json:"multiplierUp,string"`
			MultiplierDown      float64 `json:"multiplierDown,string"`
			AvgPriceMins        int64   `json:"avgPriceMins"`
			MinQty              float64 `json:"minQty,string"`
			MaxQty              float64 `json:"maxQty,string"`
			StepSize            float64 `json:"stepSize,string"`
			Notional            float64 `json:"notional,string"`
			ApplyToMarket       bool    `json:"applyToMarket"`
			Limit               int64   `json:"limit"`
			MaxNumAlgoOrders    int64   `json:"maxNumAlgoOrders"`
			MaxNumIcebergOrders int64   `json:"maxNumIcebergOrders"`
		} `json:"filters"`
	} `json:"symbols"`
}

type Order struct {
	Symbol        string  `json:"symbol"`
	OrderId       int64   `json:"orderId"`
	ClientOrderId string  `json:"clientOrderId"`
	Price         float64 `json:"price,string"`
	ReduceOnly    bool    `json:"reduceOnly"`
	OrigQty       float64 `json:"origQty,string"`
	CumQty        float64 `json:"cumQty,string"`
	CumQuote      float64 `json:"cumQuote,string"`
	Status        string  `json:"status"`
	TimeInForce   string  `json:"timeInForce"`
	Type          string  `json:"type"`
	Side          string  `json:"side"`
	StopPrice     float64 `json:"stopPrice,string"`
	Time          int64   `json:"time"`
	UpdateTime    int64   `json:"updateTime"`
	WorkingType   string  `json:"workingType"`
	Code          int64   `json:"code"`
	Msg           string  `json:"msg"`
}

func (order *Order) ToString() string {
	str := ""
	str += fmt.Sprintf("Symbl=%s, ", order.Symbol)
	str += fmt.Sprintf("OrderId=%d, ", order.OrderId)
	str += fmt.Sprintf("ClientOrderId=%s, ", order.ClientOrderId)
	str += fmt.Sprintf("Price=%f, ", order.Price)
	str += fmt.Sprintf("ReduceOnly=%v, ", order.ReduceOnly)
	str += fmt.Sprintf("OrigQty=%v, ", order.OrigQty)
	str += fmt.Sprintf("CumQuote=%v, ", order.CumQuote)
	str += fmt.Sprintf("Status=%v, ", order.Status)
	str += fmt.Sprintf("TimeInForce=%v, ", order.TimeInForce)
	str += fmt.Sprintf("Type=%v, ", order.Type)
	str += fmt.Sprintf("Side=%v, ", order.Side)
	str += fmt.Sprintf("StopPrice=%v, ", order.StopPrice)
	str += fmt.Sprintf("WorkingType=%v, ", order.WorkingType)
	str += fmt.Sprintf("Time=%v ", order.Time)
	str += fmt.Sprintf("Code=%v ", order.Code)
	str += fmt.Sprintf("Msg=%v ", order.Msg)
	return str
}

type NewOrderParams struct {
	Symbol           string  `json:"symbol,omitempty"`
	Side             string  `json:"side,omitempty"`
	Type             string  `json:"type,omitempty"`
	ReduceOnly       bool    `json:"reduceOnly,omitempty"`
	Quantity         float64 `json:"quantity,omitempty"`
	Price            float64 `json:"price,omitempty"`
	NewClientOrderId string  `json:"newClientOrderId,omitempty"`
	TimeInForce      string  `json:"timeInForce,omitempty"`
	NewOrderRespType string  `json:"newOrderRespType,omitempty"`
}

func (no *NewOrderParams) ToUrlValues() url.Values {
	values := url.Values{}
	values.Set("symbol", no.Symbol)
	values.Set("side", no.Side)
	values.Set("type", no.Type)
	values.Set("reduceOnly", strconv.FormatBool(no.ReduceOnly))
	values.Set("quantity", strconv.FormatFloat(no.Quantity, 'f', 8, 64))
	values.Set("price", strconv.FormatFloat(no.Price, 'f', 8, 64))
	values.Set("newClientOrderId", no.NewClientOrderId)
	values.Set("timeInForce", no.TimeInForce)
	values.Set("newOrderRespType", no.NewOrderRespType)
	return values
}

func (no NewOrderParams) ToString() string {
	return fmt.Sprintf(
		"Symbol=%s, Side=%s, Type=%s, ReduceOnly=%v, "+
			"Quantity=%f, Price=%f, NewClientOrderId=%s, "+
			"TimeInForce=%s",
		no.Symbol, no.Side, no.Type, no.ReduceOnly,
		no.Quantity, no.Price, no.NewClientOrderId,
		no.TimeInForce,
	)
}

//{
//  "e": "aggTrade",  // Event type
//  "E": 123456789,   // Event time
//  "s": "BTCUSDT",    // Symbol
//  "a": 5933014,     // Aggregate trade ID
//  "p": "0.001",     // Price
//  "q": "100",       // Quantity
//  "f": 100,         // First trade ID
//  "l": 105,         // Last trade ID
//  "T": 123456785,   // Trade time
//  "m": true,        // Is the buyer the market maker?
//}

type Trade struct {
	//EventType                string  `json:"e,omitempty"`
	EventTime time.Time  `json:"-"`
	Symbol    string `json:"s,omitempty"`
	//AggregateTradeID         int64   `json:"a,omitempty"`
	Price    float64 `json:"p,string,omitempty"`
	Quantity float64 `json:"q,string,omitempty"`
	//FirstTradeId             int64   `json:"f,omitempty"`
	//LastTradeId              int64   `json:"l,omitempty"`
	//TradeTime                int64   `json:"T,omitempty"`
	IsTheBuyerTheMarketMaker bool `json:"m,omitempty"`
}

//{
//"code": "200",
//"msg": "The operation of cancel all open order is done."
//}

type CancelAllOrderResponse struct {
	Code   int    `json:"code"`
	Msg    string `json:"msg"`
	Symbol string `json:"-"`
}

type CancelAllOrderParams struct {
	Symbol string `json:"symbol"`
}

func (c *CancelAllOrderParams) ToUrlValues() url.Values {
	values := url.Values{}
	values.Set("symbol", c.Symbol)
	return values
}

type PremiumIndex struct {
	EventTime            time.Time `json:"-"`
	Symbol               string    `json:"symbol,omitempty"`
	MarkPrice            float64   `json:"markPrice,string,omitempty"`
	IndexPrice           float64   `json:"indexPrice,string,omitempty"`
	EstimatedSettlePrice float64   `json:"estimatedSettlePrice,string,omitempty"`
	FundingRate          float64   `json:"lastFundingRate,string,omitempty"`
	NextFundingTime      time.Time `json:"-"`
	ParseTime            time.Time `json:"-"`
}

func (mpu *PremiumIndex) ToString() string {
	return fmt.Sprintf(
		"S=%s, FR=%v, MP=%f, IP=%f, ESP=%f, AT=%v, ET=%v, NFT=%v",
		mpu.Symbol, mpu.FundingRate,
		mpu.MarkPrice,
		mpu.IndexPrice,
		mpu.EstimatedSettlePrice,
		mpu.ParseTime,
		mpu.EventTime,
		mpu.NextFundingTime,
	)
}

func (mpu *PremiumIndex) UnmarshalJSON(data []byte) error {
	type Alias PremiumIndex
	aux := &struct {
		NextFundingTime int64  `json:"nextFundingTime,omitempty"`
		EventTime       int64  `json:"time,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(mpu),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	mpu.NextFundingTime = time.Unix(0, aux.NextFundingTime*1000000)
	mpu.EventTime = time.Unix(0,aux.EventTime*1000000)
	mpu.ParseTime = time.Now()
	return nil
}
