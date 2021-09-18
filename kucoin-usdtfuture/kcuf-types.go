package kucoin_usdtfuture

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"os"
	"strconv"
	"time"
)

const (
	Granularity1Min   = 1
	Granularity5Min   = 5
	Granularity15Min  = 15
	Granularity30Min  = 30
	Granularity1Hour  = 60
	Granularity2Hour  = 120
	Granularity4Hour  = 480
	Granularity12Hour = 720
	Granularity1Day   = 1440
	Granularity1Week  = 10080

	//       "type": "match",  //消息类型，取值列表: "open", "match", "filled", "canceled", "update"
	OrderStatusOpen     = "open"
	OrderStatusMatch    = "match"
	OrderStatusDone     = "done"
	OrderStatusFilled   = "filled"
	OrderStatusCanceled = "canceled"
	OrderStatusUpdate   = "update"

	SystemStatusOpen       = "open"
	SystemStatusCancelOnly = "cancelonly"
	SystemStatusClose      = "close"
)

var GranularityDurations = map[int]time.Duration{
	Granularity1Min:   time.Minute,
	Granularity5Min:   time.Minute * 5,
	Granularity15Min:  time.Minute * 15,
	Granularity30Min:  time.Minute * 30,
	Granularity1Hour:  time.Hour,
	Granularity2Hour:  time.Hour * 2,
	Granularity4Hour:  time.Hour * 4,
	Granularity12Hour: time.Hour * 12,
	Granularity1Day:   time.Hour * 24,
	Granularity1Week:  time.Hour * 168,
}

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

type Depth50 struct {
	Symbol    string         `json:"-"`
	Bids      [50][2]float64 `json:"bids,omitempty"`
	Asks      [50][2]float64 `json:"asks,omitempty"`
	Sequence  int64          `json:"sequence"`
	ParseTime time.Time      `json:"-"`
	EventTime time.Time      `json:"-"`
}

func (depth *Depth50) UnmarshalJSON(data []byte) error {
	type Alias Depth50
	aux := struct {
		EventTime int64 `json:"timestamp,omitempty"`
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

type Depth5 struct {
	Symbol string        `json:"-"`
	Bids   [5][2]float64 `json:"bids,omitempty"`
	Asks   [5][2]float64 `json:"asks,omitempty"`
	//Sequence  int64         `json:"sequence"`
	EventTime time.Time `json:"-"`
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
	aux := struct {
		EventTime int64 `json:"timestamp,omitempty"`
		*Alias
	}{Alias: (*Alias)(depth)}
	if err := json.Unmarshal(data, &aux); err != nil {
		logger.Debugf("ERR %v", err)
		return err
	}
	depth.EventTime = time.Unix(0, aux.EventTime*1000000)
	return nil
}

type Position struct {
	ID                string    `json:"id"`
	Symbol            string    `json:"symbol"`
	AutoDeposit       bool      `json:"autoDeposit"`
	MaintMarginReq    float64   `json:"maintMarginReq"`
	RiskLimit         float64   `json:"riskLimit"`
	RealLeverage      float64   `json:"realLeverage"`
	CrossMode         bool      `json:"crossMode"`
	DelevPercentage   float64   `json:"delevPercentage"`
	CurrentQty        float64   `json:"currentQty"`
	CurrentCost       float64   `json:"currentCost"`
	CurrentComm       float64   `json:"currentComm"`
	UnrealisedCost    float64   `json:"unrealisedCost"`
	RealisedGrossCost float64   `json:"realisedGrossCost"`
	RealisedCost      float64   `json:"realisedCost"`
	IsOpen            bool      `json:"isOpen"`
	MarkPrice         float64   `json:"markPrice"`
	MarkValue         float64   `json:"markValue"`
	PosCost           float64   `json:"posCost"`
	PosCross          float64   `json:"posCross"`
	PosInit           float64   `json:"posInit"`
	PosComm           float64   `json:"posComm"`
	PosLoss           float64   `json:"posLoss"`
	PosMargin         float64   `json:"posMargin"`
	PosMaint          float64   `json:"posMaint"`
	MaintMargin       float64   `json:"maintMargin"`
	RealisedGrossPnl  float64   `json:"realisedGrossPnl"`
	RealisedPnl       float64   `json:"realisedPnl"`
	UnrealisedPnl     float64   `json:"unrealisedPnl"`
	UnrealisedPnlPcnt float64   `json:"unrealisedPnlPcnt"`
	UnrealisedRoePcnt float64   `json:"unrealisedRoePcnt"`
	AvgEntryPrice     float64   `json:"avgEntryPrice"`
	LiquidationPrice  float64   `json:"liquidationPrice"`
	BankruptPrice     float64   `json:"bankruptPrice"`
	SettleCurrency    string    `json:"settleCurrency"`
	OpeningTimestamp  time.Time `json:"-"`
	EventTime         time.Time `json:"-"`
	ParseTime         time.Time `json:"-"`
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
	return position.CurrentQty
}

func (position *Position) GetPrice() float64 {
	return position.AvgEntryPrice
}

func (position *Position) UnmarshalJSON(data []byte) error {
	type Alias Position
	aux := struct {
		OpeningTimestamp int64 `json:"openingTimestamp"`
		CurrentTimestamp int64 `json:"currentTimestamp"`
		*Alias
	}{
		Alias: (*Alias)(position),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		logger.Debugf("UnmarshalJSON Position error %v", err)
		return err
	}
	position.EventTime = time.Unix(0, aux.CurrentTimestamp*1000000)
	position.ParseTime = time.Now()
	position.OpeningTimestamp = time.Unix(0, aux.OpeningTimestamp*1000000)
	return nil
}

//{"data":{"realisedGrossPnl":0E-8,"symbol":"MANAUSDTM","crossMode":false,"liquidationPrice":1.077,"posLoss":0E-8,"avgEntryPrice":0.824,"unrealisedPnl":-0.05000000,"markPrice":0.826,"posMargin":6.88314667,"autoDeposit":false,"riskLimit":200000,"unrealisedCost":-20.60000000,"posComm":0.01648000,"posMaint":0.53953460,"posCost":-20.60000000,"maintMarginReq":0.025,"bankruptPrice":1.099,"realisedCost":0.01236000,"markValue":-20.65000000,"posInit":6.86666667,"realisedPnl":-0.01236000,"maintMargin":6.83314667,"realLeverage":3.0220048903,"changeReason":"positionChange","currentCost":-20.60000000,"settleCurrency":"USDT","currentQty":-25,"currentComm":0.01236000,"realisedGrossCost":0E-8,"isOpen":true,"posCross":0E-8,"currentTimestamp":1621615230736,"unrealisedRoePcnt":-0.0073,"unrealisedPnlPcnt":-0.0024},"subject":"position.change","topic":"/contract/position:MANAUSDTM","channelType":"private","type":"message","userId":"60a3c060c0b3d10006aa9428"}
//      "realisedGrossPnl": 0E-8,                //累加已实现毛利
//      "crossMode": false,                      //是否全仓
//      "liquidationPrice": 1000000.0,           //强平价格
//      "posLoss": 0E-8,                         //手动追加的保证金
//      "avgEntryPrice": 7508.22,                //平均开仓价格
//      "unrealisedPnl": -0.00014735,            //未实现盈亏
//      "markPrice": 7947.83,                    //标记价格
//      "posMargin": 0.00266779,                 //仓位保证金
//      "riskLimit": 200,                        //风险限额
//      "unrealisedCost": 0.00266375,            //未实现价值
//      "posComm": 0.00000392,                   //破产费用
//      "posMaint": 0.00001724,                  //维持保证金
//      "posCost": 0.00266375,                   //仓位价值
//      "maintMarginReq": 0.005,                 //维持保证金比例
//      "bankruptPrice": 1000000.0,              //破产价格
//      "realisedCost": 0.00000271,              //当前累计已实现仓位价值
//      "markValue": 0.00251640,                 //标记价值
//      "posInit": 0.00266375,                   //杠杆保证金
//      "realisedPnl": -0.00000253,              //已实现盈亏
//      "maintMargin": 0.00252044,               //仓位保证金
//      "realLeverage": 1.06,                    //杠杆倍数
//      "currentCost": 0.00266375,               //当前总仓位价值
//      "openingTimestamp": 1558433191000,       //开仓时间
//      "currentQty": -20,                       //当前仓位
//      "delevPercentage": 0.52,                 //ADL分位数
//      "currentComm": 0.00000271,               //当前总费用
//      "realisedGrossCost": 0E-8,               //累计已实现毛利价值
//      "isOpen": true,                          //是否开仓
//      "posCross": 1.2E-7,                      //手动追加的保证金
//      "currentTimestamp": 1558506060394,       //当前时间戳
//      "unrealisedRoePcnt": -0.0553,            //投资回报率
//      "unrealisedPnlPcnt": -0.0553,            //仓位盈亏率
//      "settleCurrency": "XBT"                  //结算币种
type WSPosition struct {
	Symbol            string    `json:"-"`
	RealisedGrossPnl  *float64  `json:"realisedGrossPnl,omitempty"`
	CrossMode         *bool     `json:"crossMode,omitempty"`
	LiquidationPrice  *float64  `json:"liquidationPrice,omitempty"`
	PosLoss           *float64  `json:"posLoss,omitempty"`
	AvgEntryPrice     *float64  `json:"avgEntryPrice,omitempty"`
	UnrealisedPnl     *float64  `json:"unrealisedPnl,omitempty"`
	MarkPrice         *float64  `json:"markPrice,omitempty"`
	PosMargin         *float64  `json:"posMargin,omitempty"`
	RiskLimit         *float64  `json:"riskLimit,omitempty"`
	UnrealisedCost    *float64  `json:"unrealisedCost,omitempty"`
	PosComm           *float64  `json:"posComm,omitempty"`
	PosMaint          *float64  `json:"posMaint,omitempty"`
	PosCost           *float64  `json:"posCost,omitempty"`
	MaintMarginReq    *float64  `json:"maintMarginReq,omitempty"`
	BankruptPrice     *float64  `json:"bankruptPrice,omitempty"`
	RealisedCost      *float64  `json:"realisedCost,omitempty"`
	MarkValue         *float64  `json:"markValue,omitempty"`
	PosInit           *float64  `json:"posInit,omitempty"`
	RealisedPnl       *float64  `json:"realisedPnl,omitempty"`
	MaintMargin       *float64  `json:"maintMargin,omitempty"`
	RealLeverage      *float64  `json:"realLeverage,omitempty"`
	CurrentCost       *float64  `json:"currentCost,omitempty"`
	CurrentQty        *float64  `json:"currentQty,omitempty"`
	DelevPercentage   *float64  `json:"delevPercentage,omitempty"`
	CurrentComm       *float64  `json:"currentComm,omitempty"`
	RealisedGrossCost *float64  `json:"realisedGrossCost,omitempty"`
	IsOpen            *bool     `json:"isOpen,omitempty"`
	PosCross          *float64  `json:"posCross,omitempty"`
	SettleCurrency    *string   `json:"settleCurrency,omitempty"`
	UnrealisedPnlPcnt *float64  `json:"unrealisedPnlPcnt,omitempty"`
	UnrealisedRoePcnt *float64  `json:"unrealisedRoePcnt,omitempty"`
	EventTime         time.Time `json:"-"`
	ParseTime         time.Time `json:"-"`
}

func (wsPosition *WSPosition) UnmarshalJSON(data []byte) error {
	type Alias WSPosition
	aux := struct {
		CurrentTimestamp *int64 `json:"currentTimestamp"`
		*Alias
	}{
		Alias: (*Alias)(wsPosition),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		logger.Debugf("UnmarshalJSON Position error %v", err)
		return err
	}
	if aux.CurrentTimestamp != nil {
		wsPosition.EventTime = time.Unix(0, *aux.CurrentTimestamp*1000000)
	}
	wsPosition.ParseTime = time.Now()
	return nil
}

// Signer interface contains Sign() method.
type Signer interface {
	Sign(plain []byte) []byte
}

// Sha256Signer is the sha256 Signer.
type Sha256Signer struct {
	key []byte
}

// Sign makes a signature by sha256.
func (ss *Sha256Signer) Sign(plain []byte) []byte {
	hm := hmac.New(sha256.New, ss.key)
	hm.Write(plain)
	return hm.Sum(nil)
}

// KcSigner is the implement of Signer for KuCoin.
type KcSigner struct {
	Sha256Signer
	apiKey        string
	apiSecret     string
	apiPassPhrase string
	apiKeyVersion string
}

// Sign makes a signature by sha256 with `apiKey` `apiSecret` `apiPassPhrase`.
func (ks *KcSigner) Sign(plain []byte) []byte {
	s := ks.Sha256Signer.Sign(plain)
	return []byte(base64.StdEncoding.EncodeToString(s))
}

// Headers returns a map of signature header.
func (ks *KcSigner) Headers(plain string) map[string]string {
	t := fmt.Sprintf("%d", time.Now().UnixNano()/1000000)
	p := []byte(t + plain)
	s := string(ks.Sign(p))
	ksHeaders := map[string]string{
		"KC-API-KEY":        ks.apiKey,
		"KC-API-PASSPHRASE": ks.apiPassPhrase,
		"KC-API-TIMESTAMP":  t,
		"KC-API-SIGN":       s,
	}

	if ks.apiKeyVersion != "" && ks.apiKeyVersion != "1" {
		ksHeaders["KC-API-KEY-VERSION"] = ks.apiKeyVersion
	}

	return ksHeaders
}

// NewKcSigner creates a instance of KcSigner.
func NewKcSigner(key, secret, passPhrase string) *KcSigner {
	ks := &KcSigner{
		apiKey:        key,
		apiSecret:     secret,
		apiPassPhrase: passPhraseEncrypt([]byte(secret), []byte(passPhrase)),
		apiKeyVersion: "2",
	}
	ks.key = []byte(secret)
	return ks
}

type OrderResponse struct {
	OrderId string `json:"orderId"`
}

type CancelAllOrdersResponse struct {
	CancelledOrderIds []string `json:"cancelledOrderIds"`
}

type ConnectToken struct {
	InstanceServers []InstanceServer `json:"instanceServers"`
	Token           string           `json:"token"`
}

type InstanceServer struct {
	Endpoint     string `json:"endpoint"`
	Protocol     string `json:"protocol"`
	Encrypt      bool   `json:"encrypt"`
	PingInterval int    `json:"pingInterval"`
	PingTimeout  int    `json:"pingTimeout"`
}

type Ping struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type WSOrder struct {
	OrderId      string    `json:"orderId"`
	Symbol       string    `json:"symbol"`
	EventType    string    `json:"type"`
	Status       string    `json:"status"`
	MatchSize    float64   `json:"matchSize,string,omitempty"`
	MatchPrice   float64   `json:"matchPrice,string,omitempty"`
	OrderType    string    `json:"orderType"`
	Side         string    `json:"side"`
	Price        float64   `json:"-"`
	Size         float64   `json:"-"`
	RemainSize   float64   `json:"remainSize,string,omitempty"`
	FilledSize   float64   `json:"filledSize,string,omitempty"`
	FilledPrice  float64   `json:"-"`
	CanceledSize float64   `json:"canceledSize,string"`
	TradeId      string    `json:"tradeId"`
	ClientOid    string    `json:"clientOid"`
	OldSize      float64   `json:"oldSize,string"`
	OrderTime    time.Time `json:"-`
	Liquidity    string    `json:"liquidity"`
	EventTime    time.Time `json:"-"`
	ParseTime    time.Time `json:"-"`
}

func (wsOrder *WSOrder) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (wsOrder *WSOrder) GetSymbol() string {
	return wsOrder.Symbol
}

func (wsOrder *WSOrder) GetSize() float64 {
	return wsOrder.Size
}

func (wsOrder *WSOrder) GetPrice() float64 {
	return wsOrder.Price
}

func (wsOrder *WSOrder) GetFilledSize() float64 {
	return wsOrder.FilledSize
}

func (wsOrder *WSOrder) GetFilledPrice() float64 {
	if wsOrder.FilledPrice != 0 {
		return wsOrder.FilledPrice
	} else {
		return wsOrder.MatchPrice
	}
}

func (wsOrder *WSOrder) GetSide() common.OrderSide {
	switch wsOrder.Side {
	case OrderSideSell:
		return common.OrderSideSell
	case OrderSideBuy:
		return common.OrderSideBuy
	default:
		return common.OrderSideUnknown
	}
}

func (wsOrder *WSOrder) GetClientID() string {
	return wsOrder.ClientOid
}

func (wsOrder *WSOrder) GetID() string {
	return wsOrder.OrderId
}

func (wsOrder *WSOrder) GetStatus() common.OrderStatus {
	switch wsOrder.EventType {
	case OrderStatusOpen:
		return common.OrderStatusOpen
	case OrderStatusDone:
		if wsOrder.FilledSize != 0 {
			return common.OrderStatusFilled
		} else {
			return common.OrderStatusCancelled
		}
	case OrderStatusFilled:
		return common.OrderStatusFilled
	case OrderStatusMatch:
		return common.OrderStatusFilled
	case OrderStatusUpdate:
		return common.OrderStatusOpen
	default:
		return common.OrderStatusUnknown
	}
}

func (wsOrder *WSOrder) GetType() common.OrderType {
	switch wsOrder.OrderType {
	case OrderTypeMarket:
		return common.OrderTypeMarket
	case OrderTypeLimit:
		return common.OrderTypeLimit
	default:
		return common.OrderTypeUnknown
	}
}

func (wsOrder *WSOrder) GetPostOnly() bool {
	return false
}

func (wsOrder *WSOrder) GetReduceOnly() bool {
	return false
}

func (wsOrder *WSOrder) UnmarshalJSON(data []byte) error {
	type Alias WSOrder
	aux := struct {
		OrderTime int64  `json:"orderTime,omitempty"`
		EventTime int64  `json:"ts,omitempty"`
		Price     string `json:"price"`
		Size      string `json:"size"`
		*Alias
	}{
		Alias: (*Alias)(wsOrder),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		logger.Debugf("UnmarshalJSON WsOrder error %v", err)
		return err
	} else {
		if aux.Price != "" {
			wsOrder.Price, err = strconv.ParseFloat(aux.Price, 64)
			if err != nil {
				return err
			}
		}
		if aux.Size != "" {
			wsOrder.Size, err = strconv.ParseFloat(aux.Size, 64)
			if err != nil {
				return err
			}
		}
	}
	wsOrder.EventTime = time.Unix(0, aux.EventTime)
	wsOrder.ParseTime = time.Now()
	wsOrder.OrderTime = time.Unix(0, aux.OrderTime)
	return nil
}

type WsCap struct {
	ID          string          `json:"id"`
	Type        string          `json:"type"`
	Topic       string          `json:"topic"`
	Subject     string          `json:"subject"`
	ChannelType string          `json:"channelType"`
	Data        json.RawMessage `json:"data"`
}

//{"data":{"currency":"USDT","orderMargin":"0.5907469921","timestamp":"1618323715531"},"subject":"orderMargin.change","topic":"/contractAccount/wallet","channelType":"private","id":"6075a90346259d00062dde01","type":"message","userId":"6072bd6950d6480006756fa7"}

type WsBalanceEvent struct {
	Currency         *string   `json:"currency,omitempty"`
	OrderMargin      *float64  `json:"orderMargin,string,omitempty"`
	AvailableBalance *float64  `json:"availableBalance,string,omitempty"`
	HoldBalance      *float64  `json:"holdBalance,string,omitempty"`
	WithdrawHold     *float64  `json:"withdrawHold,string,omitempty"`
	EventTime        time.Time `json:"-"`
	ParseTime        time.Time `json:"-"`
	Subject          string    `json:"-"`
}

func (wsBalanceEvent *WsBalanceEvent) UnmarshalJSON(data []byte) error {
	type Alias WsBalanceEvent
	aux := struct {
		Timestamp int64 `json:"timestamp,string,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(wsBalanceEvent),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		logger.Debugf("UnmarshalJSON MarkPrice error %v", err)
		return err
	}
	wsBalanceEvent.EventTime = time.Unix(0, aux.Timestamp*1000000)
	wsBalanceEvent.ParseTime = time.Now()
	return nil
}

type Account struct {
	AccountEquity    float64   `json:"accountEquity"`
	UnrealisedPNL    float64   `json:"unrealisedPNL"`
	MarginBalance    float64   `json:"marginBalance"`
	PositionMargin   float64   `json:"positionMargin"`
	OrderMargin      float64   `json:"orderMargin"`
	FrozenFunds      float64   `json:"frozenFunds"`
	AvailableBalance float64   `json:"availableBalance"`
	Currency         string    `json:"currency"`
	EventTime        time.Time `json:"-"`
	ParseTime        time.Time `json:"-"`
}

func (a *Account) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (a *Account) UnmarshalJSON(data []byte) error {
	type Alias Account
	aux := struct {
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	a.EventTime = time.Now()
	a.ParseTime = time.Now()
	return nil
}

func (a Account) GetCurrency() string {
	return a.Currency
}

func (a Account) GetBalance() float64 {
	return a.MarginBalance + a.UnrealisedPNL
}

func (a Account) GetFree() float64 {
	return a.AvailableBalance
}

func (a Account) GetUsed() float64 {
	return a.FrozenFunds
}

func (a Account) GetTime() time.Time {
	return a.EventTime
}

type MarkPrice struct {
	Symbol      string    `json:"-"`
	Granularity int       `json:"granularity"`
	IndexPrice  float64   `json:"indexPrice"`
	MarkPrice   float64   `json:"markPrice"`
	EventTime   time.Time `json:"-"`
	ParseTime   time.Time `json:"-"`
}

func (wsCap *MarkPrice) UnmarshalJSON(data []byte) error {
	type Alias MarkPrice
	aux := struct {
		Timestamp int64 `json:"timestamp,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(wsCap),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		logger.Debugf("UnmarshalJSON MarkPrice error %v", err)
		return err
	}
	wsCap.EventTime = time.Unix(0, aux.Timestamp*1000000)
	wsCap.ParseTime = time.Now()
	return nil
}

type FundingRate struct {
	Symbol      string    `json:"-"`
	Granularity int       `json:"granularity"`
	FundingRate float64   `json:"fundingRate"`
	EventTime   time.Time `json:"-"`
	ParseTime   time.Time `json:"-"`
}

func (wsCap *FundingRate) UnmarshalJSON(data []byte) error {
	type Alias FundingRate
	aux := struct {
		Timestamp int64 `json:"timestamp,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(wsCap),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		logger.Debugf("UnmarshalJSON FundingRate error %v", err)
		return err
	}
	wsCap.EventTime = time.Unix(0, aux.Timestamp*1000000)
	wsCap.ParseTime = time.Now()
	return nil
}

//{
//"symbol": ".XBTUSDMFPI8H",              //资金费率symbol
//"granularity": 28800000,               //粒度(毫秒)
//"timePoint": 1558000800000,            //时间点(毫秒)
//"value": 0.00375,                      //资金费率
//"predictedValue": 0.00375              //预测资金费率
//}

type CurrentFundingRate struct {
	Symbol         string    `json:"symbol"`
	Granularity    int       `json:"granularity"`
	Value          float64   `json:"value"`
	PredictedValue float64   `json:"predictedValue"`
	TimePoint      time.Time `json:"-"`
	ParseTime      time.Time `json:"-"`
}

func (fr *CurrentFundingRate) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (fr *CurrentFundingRate) GetSymbol() string {
	return fr.Symbol
}

func (fr *CurrentFundingRate) GetFundingRate() float64 {
	if os.Getenv("KC_FR_WITH_PREDICTED") != "" {
		if math.Abs(fr.Value) > math.Abs(fr.PredictedValue) {
			//logger.Debugf("%s use predicted", fr.Symbol)
			return fr.Value
		} else {
			return fr.PredictedValue
		}
	} else {
		return fr.Value
	}
}

func (fr *CurrentFundingRate) GetNextFundingTime() time.Time {
	return fr.TimePoint
}

func (fr *CurrentFundingRate) UnmarshalJSON(data []byte) error {
	type Alias CurrentFundingRate
	aux := struct {
		TimePoint int64 `json:"timePoint,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(fr),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		logger.Debugf("UnmarshalJSON CurrentFundingRate error %v", err)
		return err
	}
	fr.TimePoint = time.Unix(0, aux.TimePoint*1000000)
	fr.ParseTime = time.Now()
	return nil
}

type SystemStatus struct {
	Msg    string `json:"msg"`
	Status string `json:"status"`
}

type MatchWS struct {
	Data    Match  `json:"data"`
	Topic   string `json:"topic"`
	Type    string `json:"type"`
	Subject string `json:"subject"`
}

//{"data":{"makerUserId":"6087a3ac5ef8260006c4d8b9","symbol":"LINKUSDTM","sequence":1656767,"side":"buy","size":20,"price":49.511,"takerOrderId":"60931ad274332e00062a1846","makerOrderId":"60931ad235ff0c00063e0046","takerUserId":"60634b1c27acdc000609d7b8","tradeId":"60931ad23c7feb74160ff882","ts":1620253394937842861},"subject":"match","topic":"/contractMarket/execution:LINKUSDTM","type":"message"}
type Match struct {
	MakerUserID  string    `json:"makerUserId"`
	Symbol       string    `json:"symbol"`
	Sequence     int64     `json:"sequence"`
	Side         string    `json:"side"`
	Size         float64   `json:"size"`
	Price        float64   `json:"price"`
	TakerOrderId string    `json:"takerOrderId"`
	MakerOrderId string    `json:"makerOrderId"`
	TakerUserID  string    `json:"takerUserId"`
	Timestamp    time.Time `json:"-"`
}

func (match *Match) UnmarshalJSON(data []byte) error {
	type Alias Match
	aux := struct {
		Timestamp int64 `json:"ts"`
		*Alias
	}{
		Alias: (*Alias)(match),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		logger.Debugf("json.Unmarshal error %v", err)
		return err
	}
	match.Timestamp = time.Unix(0, aux.Timestamp)
	return nil
}

// {
//    "code": "200000",
//    "data": {
//      "sequence": 1001,             // 顺序号
//      "symbol": "XBTUSDM",              // 合约
//      "side": "buy",                    // 成交方向 - taker
//      "size": 10,                       // 成交数量
//      "price": "7000.0",                // 成交价格
//      "bestBidSize": 20,                // 最佳买一价总量
//      "bestBidPrice": "7000.0",     // 最佳买一价
//      "bestAskSize": 30,                // 最佳卖一价总量
//      "bestAskPrice": "7001.0",     // 最佳卖一价
//      "tradeId": "5cbd7377a6ffab0c7ba98b26",  // 交易号
//      "ts": 1550653727731              // 成交时间 - 纳秒
//    }
//  }
type TickerData struct {
	Data Ticker `json:"data"`
}

type Ticker struct {
	Symbol       string    `json:"symbol"`
	BestBidSize  float64   `json:"bestBidSize"`
	BestBidPrice float64   `json:"bestBidPrice,string"`
	BestAskSize  float64   `json:"bestAskSize"`
	BestAskPrice float64   `json:"bestAskPrice,string"`
	Timestamp    time.Time `json:"-"`
}

func (ticker *Ticker) GetSymbol() string {
	return ticker.Symbol
}

func (ticker *Ticker) GetTime() time.Time {
	return ticker.Timestamp
}

func (ticker *Ticker) GetBidPrice() float64 {
	return ticker.BestBidPrice
}

func (ticker *Ticker) GetAskPrice() float64 {
	return ticker.BestAskPrice
}

func (ticker *Ticker) GetBidSize() float64 {
	return ticker.BestBidSize
}

func (ticker *Ticker) GetAskSize() float64 {
	return ticker.BestAskSize
}

func (ticker *Ticker) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (ticker *Ticker) UnmarshalJSON(data []byte) error {
	type Alias Ticker
	aux := struct {
		Timestamp int64 `json:"ts"`
		*Alias
	}{
		Alias: (*Alias)(ticker),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		logger.Debugf("json.Unmarshal error %v", err)
		return err
	}
	ticker.Timestamp = time.Unix(0, aux.Timestamp)
	return nil
}
