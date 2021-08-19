package binance_coinfuture

import (
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"strconv"
	"time"
)

type PositionMode struct {
	DualSidePosition bool `json:"dualSidePosition"`
}

type Depth20 struct {
	Symbol       string         `json:"s,omitempty"`
	LastUpdateId int64          `json:"u,omitempty"`
	Bids         [20][2]float64 `json:"-"`
	Asks         [20][2]float64 `json:"-"`
	EventTime    time.Time      `json:"-"`
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

type ServerTime struct {
	ServerTime int64 `json:"serverTime"`
}

//  {
//      "symbol": "BNBUSD_PERP",
//      "initialMargin": "0.00292518",
//      "maintMargin": "0.00073129",
//      "unrealizedProfit": "-0.00153241",
//      "positionInitialMargin": "0.00292518",
//      "openOrderInitialMargin": "0",
//      "leverage": "10",
//      "isolated": true,
//      "positionSide": "BOTH",
//      "entryPrice": "360.75800000",
//      "maxQty": "4000",
//      "notionalValue": "0.02925182",
//      "isolatedWallet": "0.00276464"
//    }

type AccountPosition struct {
	Symbol                 string    `json:"symbol"`
	InitialMargin          float64   `json:"initialMargin,string"`
	MaintMargin            float64   `json:"maintMargin,string"`
	UnrealizedProfit       float64   `json:"unrealizedProfit,string"`
	PositionInitialMargin  float64   `json:"positionInitialMargin,string"`
	OpenOrderInitialMargin float64   `json:"openOrderInitialMargin,string"`
	Leverage               int64     `json:"leverage,string"`
	Isolated               bool      `json:"isolated"`
	PositionSide           string    `json:"positionSide"`
	EntryPrice             float64   `json:"entryPrice,string"`
	MaxQty                 float64   `json:"maxQty,string"`
	NotionalValue          float64   `json:"notionalValue,string"`
	IsolatedWallet         float64   `json:"isolatedWallet,string"`
	ParseTime              time.Time `json:"-"`
	EventTime              time.Time `json:"-"`
}

func (position *AccountPosition) UnmarshalJSON(data []byte) error {
	type Alias AccountPosition
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

func (position *AccountPosition) GetEventTime() time.Time {
	return position.EventTime
}

func (position *AccountPosition) GetParseTime() time.Time {
	return position.ParseTime
}

func (position *AccountPosition) GetSymbol() string {
	return position.Symbol
}

func (position *AccountPosition) GetSize() float64 {
	return math.Round(position.NotionalValue*position.EntryPrice/ContractSizes[position.Symbol])
}

func (position *AccountPosition) GetPrice() float64 {
	return position.EntryPrice
}

type HttpPosition struct {
	Symbol           string    `json:"symbol"`
	PositionAmt      float64   `json:"positionAmt,string"`
	EntryPrice       float64   `json:"entryPrice,string"`
	MarkPrice        float64   `json:"markPrice,string"`
	UnRealizedProfit float64   `json:"unRealizedProfit,string"`
	LiquidationPrice float64   `json:"liquidationPrice,string"`
	Leverage         int64     `json:"leverage,string"`
	MaxQty           int64     `json:"maxQty,string"`
	MarginType       string    `json:"marginType"`
	IsolatedMargin   float64   `json:"isolatedMargin,string"`
	IsAutoAddMargin  bool      `json:"isAutoAddMargin,string"`
	PositionSide     string    `json:"positionSide"`
	NotionalValue    float64   `json:"notionalValue,string"`
	IsolatedWallet   float64   `json:"isolatedWallet,string"`
	ParseTime        time.Time `json:"-"`
	EventTime        time.Time `json:"-"`
}

func (position *HttpPosition) GetEventTime() time.Time {
	return position.EventTime
}

func (position *HttpPosition) GetParseTime() time.Time {
	return position.ParseTime
}

func (position *HttpPosition) GetSymbol() string {
	return position.Symbol
}

func (position *HttpPosition) GetSize() float64 {
	return position.PositionAmt
}

func (position *HttpPosition) GetPrice() float64 {
	return position.EntryPrice
}

func (position *HttpPosition) UnmarshalJSON(data []byte) error {
	type Alias HttpPosition
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

type AccountAsset struct {
	Asset                  string
	WalletBalance          float64   `json:"walletBalance,string"`
	UnrealizedProfit       float64   `json:"unrealizedProfit,string,omitempty"`
	MarginBalance          float64   `json:"marginBalance,string,omitempty"`
	MaintMargin            float64   `json:"maintMargin,string,omitempty"`
	InitialMargin          float64   `json:"initialMargin,string,omitempty"`
	PositionInitialMargin  float64   `json:"positionInitialMargin,string,omitempty"`
	OpenOrderInitialMargin float64   `json:"openOrderInitialMargin,string,omitempty"`
	MaxWithdrawAmount      float64   `json:"maxWithdrawAmount,string,omitempty"`
	CrossWalletBalance     float64   `json:"crossWalletBalance,string"`
	CrossUnPnl             float64   `json:"crossUnPnl,string,omitempty"`
	AvailableBalance       float64   `json:"availableBalance,string,omitempty"`
	EventTime              time.Time `json:"-"`
	ParseTime              time.Time `json:"-"`
}

func (a *AccountAsset) UnmarshalJSON(data []byte) error {
	type Alias AccountAsset
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
	a.EventTime = time.Now() // time.Unix(0, aux.EventTime*1000000)
	return nil
}

func (a AccountAsset) GetCurrency() string {
	return a.Asset
}

func (a AccountAsset) GetBalance() float64 {
	return a.MarginBalance
}

func (a AccountAsset) GetFree() float64 {
	return a.AvailableBalance
}

func (a AccountAsset) GetUsed() float64 {
	return a.WalletBalance - a.AvailableBalance
}

func (a AccountAsset) GetTime() time.Time {
	return a.EventTime
}

type Account struct {
	FeeTier     float64           `json:"feeTier"`
	CanTrade    bool              `json:"canTrade"`
	CanDeposit  bool              `json:"canDeposit"`
	CanWithdraw bool              `json:"canWithdraw"`
	Assets      []AccountAsset    `json:"assets"`
	Positions   []AccountPosition `json:"positions"`
	EventTime   time.Time         `json:"-"`
	ParseTime   time.Time         `json:"-"`
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

type Response struct {
	Code int64
	Msg  string `json:"msg"`
}

type Symbol struct {
	Symbol                string    `json:"symbol"`
	Pair                  string    `json:"pair"`
	ContractType          string    `json:"contractType"`
	DeliveryDate          time.Time `json:"-"`
	OnboardDate           time.Time `json:"-"`
	ContractStatus        string    `json:"contractStatus"`
	ContractSize          int64     `json:"contractSize"`
	MarginAsset           string    `json:"marginAsset"`
	MaintMarginPercent    float64   `json:"maintMarginPercent,string"`
	RequiredMarginPercent float64   `json:"requiredMarginPercent,string"`
	BaseAsset             string    `json:"baseAsset"`
	QuoteAsset            string    `json:"quoteAsset"`
	PricePrecision        int       `json:"pricePrecision"`
	QuantityPrecision     int       `json:"quantityPrecision"`
	BaseAssetPrecision    int       `json:"baseAssetPrecision"`
	QuotePrecision        int       `json:"quotePrecision"`
	EqualQtyPrecision     int       `json:"equalQtyPrecision"`
	TriggerProtect        float64   `json:"triggerProtect,string"`
	UnderlyingType        string    `json:"underlyingType"`
	Filters               []Filter  `json:"filters"`
	OrderTypes            []string  `json:"orderTypes"`
	TimeInForce           []string  `json:"timeInForce"`
}

func (symbol *Symbol) UnmarshalJSON(data []byte) error {
	type Alias Symbol
	aux := &struct {
		DeliveryDate int64 `json:"deliveryDate"`
		OnboardDate  int64 `json:"onboardDate"`
		*Alias
	}{
		Alias: (*Alias)(symbol),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	symbol.DeliveryDate = time.Unix(0, aux.DeliveryDate*1000000)
	symbol.OnboardDate = time.Unix(0, aux.OnboardDate*1000000)
	return nil
}

type Filter struct {
	FilterType        string  `json:"filterType"`
	MinPrice          float64 `json:"minPrice,string"`
	MaxPrice          float64 `json:"maxPrice,string"`
	TickSize          float64 `json:"tickSize,string"`
	StepSize          float64 `json:"stepSize,string"`
	MaxQty            float64 `json:"maxQty,string"`
	MinQty            float64 `json:"minQty,string"`
	Limit             float64 `json:"limit"`
	MultiplierDown    float64 `json:"multiplierDown,string"`
	MultiplierUp      float64 `json:"multiplierUp,string"`
	MultiplierDecimal float64 `json:"multiplierDecimal,string"`
}

type RateLimit struct {
	RateLimitType string `json:"rateLimitType"`
	Interval      string `json:"interval"`
	IntervalNum   int    `json:"intervalNum"`
	Limit         int    `json:"limit"`
}

type ExchangeInfo struct {
	TimeZone        string          `json:"timezone"`
	ServerTime      time.Time       `json:"serverTime"`
	RateLimits      []RateLimit     `json:"rateLimits"`
	ExchangeFilters json.RawMessage `json:"exchangeFilters"`
	Symbols         []Symbol        `json:"symbols"`
}

func (exchangeInfo *ExchangeInfo) UnmarshalJSON(data []byte) error {
	type Alias ExchangeInfo
	aux := &struct {
		ServerTime int64 `json:"serverTime"`
		*Alias
	}{
		Alias: (*Alias)(exchangeInfo),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	exchangeInfo.ServerTime = time.Unix(0, aux.ServerTime*1000000)
	return nil
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

type CancelAllOrderResponse struct {
	Code   int    `json:"code"`
	Msg    string `json:"msg"`
	Symbol string `json:"-"`
}

type PremiumIndex struct {
	Symbol               string    `json:"symbol"`
	Pair                 string    `json:"pair"`
	MarkPrice            float64   `json:"markPrice,string"`
	IndexPrice           float64   `json:"indexPrice,string"`
	EstimatedSettlePrice float64   `json:"estimatedSettlePrice,string"`
	FundingRate          float64   `json:"-"`
	InterestRate         float64   `json:"-"`
	NextFundingTime      time.Time `json:"-"`
	EventTime            time.Time `json:"-"`
	ParseTime            time.Time `json:"-"`
}

func (mpu *PremiumIndex) MarshalJSON() ([]byte, error) {
	type Alias PremiumIndex
	return json.Marshal(&struct {
		NextFundingTime int64 `json:"nextFundingTime,omitempty"`
		EventTime       int64 `json:"time,omitempty"`
		*Alias
	}{
		Alias:           (*Alias)(mpu),
		NextFundingTime: mpu.NextFundingTime.UnixNano() / 1000000,
		EventTime:       mpu.EventTime.UnixNano() / 1000000,
	})
}

func (pi *PremiumIndex) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (pi *PremiumIndex) GetSymbol() string {
	return pi.Symbol
}

func (pi *PremiumIndex) GetFundingRate() float64 {
	return pi.FundingRate
}

func (pi *PremiumIndex) GetNextFundingTime() time.Time {
	return pi.NextFundingTime
}

func (pi *PremiumIndex) UnmarshalJSON(data []byte) error {
	type Alias PremiumIndex
	aux := &struct {
		NextFundingTime int64  `json:"nextFundingTime,omitempty"`
		EventTime       int64  `json:"time,omitempty"`
		FundingRate     string `json:"lastFundingRate"`
		InterestRate    string `json:"interestRate"`
		*Alias
	}{
		Alias: (*Alias)(pi),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	pi.NextFundingTime = time.Unix(0, aux.NextFundingTime*1000000)
	pi.EventTime = time.Unix(0, aux.EventTime*1000000)
	pi.ParseTime = time.Now()
	if aux.FundingRate != "" {
		fr, err := strconv.ParseFloat(aux.FundingRate, 64)
		if err != nil {
			return err
		}
		pi.FundingRate = fr
	}
	if aux.InterestRate != "" {
		ir, err := strconv.ParseFloat(aux.InterestRate, 64)
		if err != nil {
			return err
		}
		pi.InterestRate = ir
	}
	return nil
}

type Depth5 struct {
	EventTime    time.Time     `json:"-"`
	Symbol       string        `json:"s,omitempty"`
	LastUpdateId int64         `json:"u,omitempty"`
	Bids         [5][2]float64 `json:"b,omitempty"`
	Asks         [5][2]float64 `json:"a,omitempty"`
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

type Leverage struct {
	Symbol   string `json:"symbol"`
	Leverage int64  `json:"leverage"`
	MaxQty   int64  `json:"maxQty,string"`
}
