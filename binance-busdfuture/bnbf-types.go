package binance_busdfuture

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

func (depth *Depth20) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (depth *Depth20) GetBids() common.Bids {
	return depth.Bids[:]
}
func (depth *Depth20) GetAsks() common.Asks {
	return depth.Asks[:]
}
func (depth *Depth20) GetSymbol() string {
	return depth.Symbol
}
func (depth *Depth20) GetTime() time.Time {
	return depth.EventTime
}

func (depth *Depth20) UnmarshalJSON(data []byte) error {
	type Alias Depth20
	aux := &struct {
		Bids      [20][2]string `json:"b,omitempty"`
		Asks      [20][2]string `json:"a,omitempty"`
		EventName string        `json:"e,omitempty"`
		EventTime int64         `json:"E,omitempty"`
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
		price, _ := common.ParseDecimal(d[0])
		size, _ := common.ParseDecimal(d[1])
		depth.Bids = append(depth.Bids, [2]float64{price, size})
	}
	for _, d := range aux.Asks {
		price, _ := common.ParseDecimal(d[0])
		size, _ := common.ParseDecimal(d[1])
		depth.Asks = append(depth.Asks, [2]float64{price, size})
	}
	eventTime, _ := common.ParseInt(aux.EventTime)
	depth.EventTime = time.Unix(0, eventTime*1000000)
	depth.LastUpdateId, _ = common.ParseInt(aux.LastUpdateId)
	return nil
}

type Depth20Stream struct {
	Stream string  `json:"stream"`
	Data   Depth20 `json:"data"`
}

type Depth5Stream struct {
	Stream string `json:"stream"`
	Data   Depth5 `json:"data"`
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

//  {
//      "symbol": "BNBUSDT",
//      "initialMargin": "2.00145712",
//      "maintMargin": "0.26018942",
//      "unrealizedProfit": "-2.08062798",
//      "positionInitialMargin": "2.00145712",
//      "openOrderInitialMargin": "0",
//      "leverage": "20",
//      "isolated": false,
//      "entryPrice": "291.91165",
//      "maxNotional": "250000",
//      "positionSide": "BOTH",
//      "positionAmt": "-0.13",
//      "notional": "-40.02914248",
//      "isolatedWallet": "0",
//      "updateTime": 1621859418346
//    }

type Position struct {
	Symbol                 string    `json:"symbol"`
	InitialMargin          float64   `json:"initialMargin,string"`
	MaintMargin            float64   `json:"maintMargin,string"`
	UnrealizedProfit       float64   `json:"unrealizedProfit,string"`
	PositionInitialMargin  float64   `json:"positionInitialMargin,string"`
	OpenOrderInitialMargin float64   `json:"openOrderInitialMargin,string"`
	Leverage               int64     `json:"leverage,string"`
	Isolated               bool      `json:"isolated"`
	EntryPrice             float64   `json:"entryPrice,string"`
	MaxNotional            float64   `json:"maxNotional,string"`
	PositionSide           string    `json:"positionSide"`
	PositionAmt            float64   `json:"positionAmt,string"`
	Notional               float64   `json:"notional,string"`
	IsolatedWallet         float64   `json:"isolatedWallet,string"`
	ParseTime              time.Time `json:"-"`
	EventTime              time.Time `json:"-"`
}

func (position *Position) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (position *Position) GetEventTime() time.Time {
	return position.EventTime
}

func (position *Position) GetParseTime() time.Time {
	return position.ParseTime
}

func (position *Position) GetSymbol() string {
	return position.Symbol
}

func (position *Position) GetSize() float64 {
	return position.PositionAmt
}

func (position *Position) GetPrice() float64 {
	return position.EntryPrice
}

func (position *Position) ToString() string {
	return fmt.Sprintf("Market=%s,EntryPrice=%f,PositionAmt=%f", position.Symbol, position.EntryPrice, position.PositionAmt)
}

func (position *Position) UnmarshalJSON(data []byte) error {
	type Alias Position
	aux := &struct {
		*Alias
		EventTime int64 `json:"updateTime"`
	}{
		Alias: (*Alias)(position),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	position.ParseTime = time.Now()
	position.EventTime = time.Unix(0, aux.EventTime*1000000)
	return nil
}

// {
//      "asset": "BNB",
//      "walletBalance": "0.05630802",
//      "unrealizedProfit": "0.00000000",
//      "marginBalance": "0.05630802",
//      "maintMargin": "0.00000000",
//      "initialMargin": "0.00000000",
//      "positionInitialMargin": "0.00000000",
//      "openOrderInitialMargin": "0.00000000",
//      "maxWithdrawAmount": "0.05630802",
//      "crossWalletBalance": "0.00000000",
//      "crossUnPnl": "0.00000000",
//      "availableBalance": "0.00000000",
//      "marginAvailable": false,
//      "updateTime": 1621863019452
//    }
type Asset struct {
	Asset                  string
	InitialMargin          *float64  `json:"initialMargin,string,omitempty"`
	MaintMargin            *float64  `json:"maintMargin,string,omitempty"`
	MarginBalance          *float64  `json:"marginBalance,string,omitempty"`
	MaxWithdrawAmount      *float64  `json:"maxWithdrawAmount,string,omitempty"`
	OpenOrderInitialMargin *float64  `json:"openOrderInitialMargin,string,omitempty"`
	PositionInitialMargin  *float64  `json:"positionInitialMargin,string,omitempty"`
	UnrealizedProfit       *float64  `json:"unrealizedProfit,string,omitempty"`
	WalletBalance          *float64  `json:"walletBalance,string"`
	CrossWalletBalance     *float64  `json:"crossWalletBalance,string"`
	CrossUnPnl             *float64  `json:"crossUnPnl,string,omitempty"`
	AvailableBalance       *float64  `json:"availableBalance,string,omitempty"`
	EventTime              time.Time `json:"-"`
	ParseTime              time.Time `json:"-"`
}

func (a *Asset) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (a *Asset) UnmarshalJSON(data []byte) error {
	type Alias Asset
	aux := &struct {
		*Alias
		EventTime int64 `json:"updateTime"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	a.ParseTime = time.Now()
	a.EventTime = time.Unix(0, aux.EventTime*1000000)
	return nil
}

func (a Asset) GetCurrency() string {
	return a.Asset
}

func (a Asset) GetBalance() float64 {
	if a.MarginBalance != nil {
		return *a.MarginBalance
	} else {
		return 0.0
	}
}

func (a Asset) GetFree() float64 {
	if a.AvailableBalance != nil {
		return *a.AvailableBalance
	} else {
		return 0.0
	}
}

func (a Asset) GetUsed() float64 {
	if a.WalletBalance != nil && a.AvailableBalance != nil {
		return *a.WalletBalance - *a.AvailableBalance
	} else {
		return 0.0
	}
}

func (a Asset) GetTime() time.Time {
	return a.ParseTime
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

//{
//    "feeTier": 0,       // account commisssion tier
//    "canTrade": true,   // if can trade
//    "canDeposit": true,     // if can transfer in asset
//    "canWithdraw": true,    // if can transfer out asset
//    "updateTime": 0,
//    "totalInitialMargin": "0.00000000",    // total initial margin required with current mark price (useless with isolated positions), only for USDT asset
//    "totalMaintMargin": "0.00000000",     // total maintenance margin required, only for USDT asset
//    "totalWalletBalance": "23.72469206",     // total wallet balance, only for USDT asset
//    "totalUnrealizedProfit": "0.00000000",   // total unrealized profit, only for USDT asset
//    "totalMarginBalance": "23.72469206",     // total margin balance, only for USDT asset
//    "totalPositionInitialMargin": "0.00000000",    // initial margin required for positions with current mark price, only for USDT asset
//    "totalOpenOrderInitialMargin": "0.00000000",   // initial margin required for open orders with current mark price, only for USDT asset
//    "totalCrossWalletBalance": "23.72469206",      // crossed wallet balance, only for USDT asset
//    "totalCrossUnPnl": "0.00000000",      // unrealized profit of crossed positions, only for USDT asset
//    "availableBalance": "23.72469206",       // available balance, only for USDT asset
//    "maxWithdrawAmount": "23.72469206"     // maximum amount for transfer out, only for USDT asset
//    "assets": [
//        {
//            "asset": "USDT",            // asset name
//            "walletBalance": "23.72469206",      // wallet balance
//            "unrealizedProfit": "0.00000000",    // unrealized profit
//            "marginBalance": "23.72469206",      // margin balance
//            "maintMargin": "0.00000000",        // maintenance margin required
//            "initialMargin": "0.00000000",    // total initial margin required with current mark price
//            "positionInitialMargin": "0.00000000",    //initial margin required for positions with current mark price
//            "openOrderInitialMargin": "0.00000000",   // initial margin required for open orders with current mark price
//            "crossWalletBalance": "23.72469206",      // crossed wallet balance
//            "crossUnPnl": "0.00000000"       // unrealized profit of crossed positions
//            "availableBalance": "23.72469206",       // available balance
//            "maxWithdrawAmount": "23.72469206",     // maximum amount for transfer out
//            "marginAvailable": true    // whether the asset can be used as margin in Multi-Assets mode
//        },
//        {
//            "asset": "BUSD",            // asset name
//            "walletBalance": "103.12345678",      // wallet balance
//            "unrealizedProfit": "0.00000000",    // unrealized profit
//            "marginBalance": "103.12345678",      // margin balance
//            "maintMargin": "0.00000000",        // maintenance margin required
//            "initialMargin": "0.00000000",    // total initial margin required with current mark price
//            "positionInitialMargin": "0.00000000",    //initial margin required for positions with current mark price
//            "openOrderInitialMargin": "0.00000000",   // initial margin required for open orders with current mark price
//            "crossWalletBalance": "103.12345678",      // crossed wallet balance
//            "crossUnPnl": "0.00000000"       // unrealized profit of crossed positions
//            "availableBalance": "103.12345678",       // available balance
//            "maxWithdrawAmount": "103.12345678",     // maximum amount for transfer out
//            "marginAvailable": true    // whether the asset can be used as margin in Multi-Assets mode
//        }
//    ],
//    "positions": [  // positions of all sumbols in the market are returned
//        // only "BOTH" positions will be returned with One-way mode
//        // only "LONG" and "SHORT" positions will be returned with Hedge mode
//        {
//            "symbol": "BTCUSDT",    // symbol name
//            "initialMargin": "0",   // initial margin required with current mark price
//            "maintMargin": "0",     // maintenance margin required
//            "unrealizedProfit": "0.00000000",  // unrealized profit
//            "positionInitialMargin": "0",      // initial margin required for positions with current mark price
//            "openOrderInitialMargin": "0",     // initial margin required for open orders with current mark price
//            "leverage": "100",      // current initial leverage
//            "isolated": true,       // if the position is isolated
//            "entryPrice": "0.00000",    // average entry price
//            "maxNotional": "250000",    // maximum available notional with current leverage
//            "positionSide": "BOTH",     // position side
//            "positionAmt": "0"          // position amount
//        }
//    ]
//}

type Account struct {
	FeeTier     float64 `json:"feeTier"`
	Assets      []Asset `json:"assets"`
	CanTrade    bool    `json:"canTrade"`
	CanDeposit  bool    `json:"canDeposit"`
	CanWithdraw bool    `json:"canWithdraw"`

	TotalInitialMargin          float64 `json:"totalInitialMargin,string"`
	TotalMaintMargin            float64 `json:"totalMaintMargin,string"`
	TotalWalletBalance          float64 `json:"totalWalletBalance,string"`
	TotalUnrealizedProfit       float64 `json:"totalUnrealizedProfit,string"`
	TotalMarginBalance          float64 `json:"totalMarginBalance,string"`
	TotalPositionInitialMargin  float64 `json:"totalPositionInitialMargin,string"`
	TotalOpenOrderInitialMargin float64 `json:"totalOpenOrderInitialMargin,string"`
	TotalCrossWalletBalance     float64 `json:"totalCrossWalletBalance,string"`
	TotalCrossUnPnl             float64 `json:"totalCrossUnPnl,string"`
	AvailableBalance            float64 `json:"availableBalance,string"`
	MaxWithdrawAmount           float64 `json:"maxWithdrawAmount,string"`

	Positions []Position `json:"positions"`
	EventTime time.Time  `json:"-"`
	ParseTime time.Time  `json:"-"`
}

func (account *Account) UnmarshalJSON(data []byte) error {
	type Alias Account
	aux := &struct {
		*Alias
		EventTime int64 `json:"updateTime"`
	}{
		Alias: (*Alias)(account),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	account.ParseTime = time.Now()
	account.EventTime = time.Unix(0, aux.EventTime*1000000)
	return nil
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

func (order *Order) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (order *Order) GetSymbol() string {
	return order.Symbol
}

func (order *Order) GetSize() float64 {
	return order.OrigQty
}

func (order *Order) GetPrice() float64 {
	return order.Price
}

func (order *Order) GetFilledSize() float64 {
	return order.CumQty
}

func (order *Order) GetFilledPrice() float64 {
	if order.CumQty != 0 {
		return order.CumQuote / order.CumQty
	} else {
		return 0.0
	}
}

func (order *Order) GetSide() common.OrderSide {
	switch order.Side {
	case OrderSideSell:
		return common.OrderSideSell
	case OrderSideBuy:
		return common.OrderSideBuy
	default:
		return common.OrderSideUnknown
	}
}

func (order *Order) GetClientID() string {
	return order.ClientOrderId
}

func (order *Order) GetID() string {
	return fmt.Sprintf("%d", order.OrderId)
}

func (order *Order) GetStatus() common.OrderStatus {
	switch order.Status {
	case OrderStatusFilled:
		return common.OrderStatusFilled
	case OrderStatusCancelled:
		return common.OrderStatusCancelled
	case OrderStatusReject:
		return common.OrderStatusReject
	case OrderStatusPartiallyFilled:
		return common.OrderStatusPartiallyFilled
	case OrderStatusExpired:
		return common.OrderStatusExpired
	case OrderStatusNew:
		return common.OrderStatusNew
	case OrderStatusPendingCancel:
		return common.OrderStatusPendingCancel
	default:
		return common.OrderStatusUnknown
	}
}

func (order *Order) GetType() common.OrderType {
	switch order.Type {
	case OrderTypeLimit:
		return common.OrderTypeLimit
	case OrderTypeMarket:
		return common.OrderTypeMarket
	default:
		return common.OrderTypeUnknown
	}
}

func (order *Order) GetPostOnly() bool {
	return order.TimeInForce == OrderTimeInForceGTX
}

func (order *Order) GetReduceOnly() bool {
	return order.ReduceOnly
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
	if no.Quantity != 0.0 {
		values.Set("quantity", strconv.FormatFloat(no.Quantity, 'f', 8, 64))
	}
	if no.Price != 0.0 && no.Type != OrderTypeMarket {
		values.Set("price", strconv.FormatFloat(no.Price, 'f', 8, 64))
	}
	values.Set("newClientOrderId", no.NewClientOrderId)
	if no.TimeInForce != "" {
		values.Set("timeInForce", no.TimeInForce)
	}
	values.Set("newOrderRespType", no.NewOrderRespType)
	return values
}

func (no NewOrderParams) ToString() string {
	return fmt.Sprintf(
		"Market=%s, Side=%s, Type=%s, ReduceOnly=%v, "+
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
//  "s": "BTCUSDT",    // Market
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
	EventTime time.Time `json:"-"`
	Symbol    string    `json:"s,omitempty"`
	//AggregateTradeID         int64   `json:"a,omitempty"`
	Price    float64 `json:"p,string,omitempty"`
	Quantity float64 `json:"q,string,omitempty"`
	//FirstTradeId             int64   `json:"f,omitempty"`
	//LastTradeId              int64   `json:"l,omitempty"`
	//TradeTime                int64   `json:"T,omitempty"`
	IsTheBuyerTheMarketMaker bool `json:"m,omitempty"`
}

func (trade *Trade) GetSymbol() string  { return trade.Symbol }
func (trade *Trade) GetSize() float64   { return trade.Quantity }
func (trade *Trade) GetPrice() float64  { return trade.Price }
func (trade *Trade) GetTime() time.Time { return trade.EventTime }
func (trade *Trade) IsUpTick() bool     { return !trade.IsTheBuyerTheMarketMaker }

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

type CancelOrderParam struct {
	Symbol            string `json:"symbol"`
	OrderId           int64  `json:"orderId"`
	OrigClientOrderId string `json:"origClientOrderId"`
}

func (c *CancelOrderParam) ToUrlValues() url.Values {
	values := url.Values{}
	values.Set("symbol", c.Symbol)
	if c.OrderId != 0 {
		values.Set("orderId", fmt.Sprintf("%d", c.OrderId))
	}
	if c.OrigClientOrderId != "" {
		values.Set("origClientOrderId", c.OrigClientOrderId)
	}
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

func (mpu *PremiumIndex) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (mpu *PremiumIndex) GetSymbol() string {
	return mpu.Symbol
}

func (mpu *PremiumIndex) GetFundingRate() float64 {
	return mpu.FundingRate
}

func (mpu *PremiumIndex) GetNextFundingTime() time.Time {
	return mpu.NextFundingTime
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
		NextFundingTime int64 `json:"nextFundingTime,omitempty"`
		EventTime       int64 `json:"time,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(mpu),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	mpu.NextFundingTime = time.Unix(0, aux.NextFundingTime*1000000)
	mpu.EventTime = time.Unix(0, aux.EventTime*1000000)
	mpu.ParseTime = time.Now()
	return nil
}

type Depth5 struct {
	EventTime    time.Time     `json:"-"`
	Symbol       string        `json:"s,omitempty"`
	LastUpdateId int64         `json:"u,omitempty"`
	Bids         [5][2]float64 `json:"b,omitempty"`
	Asks         [5][2]float64 `json:"a,omitempty"`
}

func (depth *Depth5) GetBidPrice() float64 {
	return depth.Bids[0][0]
}

func (depth *Depth5) GetAskPrice() float64 {
	return depth.Asks[0][0]
}

func (depth *Depth5) GetBidSize() float64 {
	return depth.Bids[0][1]
}

func (depth *Depth5) GetAskSize() float64 {
	return depth.Asks[0][1]
}

func (depth *Depth5) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (depth *Depth5) GetBids() common.Bids {
	return depth.Bids[:]
}
func (depth *Depth5) GetAsks() common.Asks {
	return depth.Asks[:]
}
func (depth *Depth5) GetSymbol() string {
	return depth.Symbol
}
func (depth *Depth5) GetTime() time.Time {
	return depth.EventTime
}

func (depth *Depth5) UnmarshalJSON(data []byte) error {
	type Alias Depth5
	aux := &struct {
		Bids      [5][2]string `json:"b,omitempty"`
		Asks      [5][2]string `json:"a,omitempty"`
		EventName string       `json:"e,omitempty"`
		EventTime int64        `json:"E,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(depth),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		logger.Debugf("ERR %v", err)
		return err
	}
	depth.Bids = [5][2]float64{}
	depth.Asks = [5][2]float64{}
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

type Ping struct {
}

type Depth Depth20

func (depth *Depth) GetSymbol() string {
	return depth.Symbol
}
func (depth *Depth) GetTime() time.Time {
	return depth.EventTime
}
func (depth *Depth) GetAsks() common.Asks {
	return depth.Asks[:]
}
func (depth *Depth) GetBids() common.Bids {
	return depth.Bids[:]
}

type MultiAssetsMarginParam struct {
	MultiAssetsMargin bool
}

func (cpmp *MultiAssetsMarginParam) ToUrlValues() url.Values {
	values := url.Values{}
	if cpmp.MultiAssetsMargin {
		values.Set("multiAssetsMargin", "true")
	} else {
		values.Set("multiAssetsMargin", "false")
	}
	return values
}

type MultiAssetsMargin struct {
	MultiAssetsMargin bool `json:"multiAssetsMargin"`
}

//{
//  "e":"bookTicker",         // event type
//  "u":400900217,            // order book updateId
//  "E": 1568014460893,       // event time
//  "T": 1568014460891,       // transaction time
//  "s":"BNBUSDT",            // symbol
//  "b":"25.35190000",        // best bid price
//  "B":"31.21000000",        // best bid qty
//  "a":"25.36520000",        // best ask price
//  "A":"40.66000000"         // best ask qty
//}

type BookTicker struct {
	EventType string `json:"e"`
	//OrderBookUpdateId int64     `json:"u"`
	EventTime time.Time `json:"-"`
	//TransactionTime   time.Time `json:"-"`
	Symbol       string  `json:"s"`
	BestBidPrice float64 `json:"b,string"`
	BestBidQty   float64 `json:"B,string"`
	BestAskPrice float64 `json:"a,string"`
	BestAskQty   float64 `json:"A,string"`
}

func (bt *BookTicker) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (bt *BookTicker) GetSymbol() string {
	return bt.Symbol
}

func (bt *BookTicker) GetTime() time.Time {
	return bt.EventTime
}

func (bt *BookTicker) GetBidPrice() float64 {
	return bt.BestBidPrice
}

func (bt *BookTicker) GetAskPrice() float64 {
	return bt.BestAskPrice
}

func (bt *BookTicker) GetBidSize() float64 {
	return bt.BestBidQty
}

func (bt *BookTicker) GetAskSize() float64 {
	return bt.BestAskQty
}

func (bt *BookTicker) UnmarshalJSON(data []byte) error {
	type Alias BookTicker
	aux := &struct {
		EventTime int64 `json:"E"`
		//TransactionTime int64 `json:"T"`
		*Alias
	}{
		Alias: (*Alias)(bt),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	bt.EventTime = time.Unix(0, aux.EventTime*1000000)
	//bt.TransactionTime = time.Unix(0, aux.TransactionTime*1000000)
	return nil
}

