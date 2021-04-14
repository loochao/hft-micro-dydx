package kcperp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/logger"
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

	OrderStatusOpen  = "open"
	OrderStatusMatch = "match"
	OrderStatusDone  = "done"

	OrderTypeOpen     = "open"
	OrderTypeMatch    = "match"
	OrderTypeFilled   = "filled"
	OrderTypeCanceled = "canceled"
	OrderTypeUpdate   = "update"
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
	Symbol    string         `json:"-"`
	Bids      [5][2]float64 `json:"bids,omitempty"`
	Asks      [5][2]float64 `json:"asks,omitempty"`
	Sequence  int64          `json:"sequence"`
	ParseTime time.Time      `json:"-"`
	EventTime time.Time      `json:"-"`
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
	depth.ParseTime = time.Now()
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

func (wsCap *Position) UnmarshalJSON(data []byte) error {
	type Alias Position
	aux := struct {
		OpeningTimestamp int64 `json:"openingTimestamp"`
		CurrentTimestamp int64 `json:"currentTimestamp"`
		*Alias
	}{
		Alias: (*Alias)(wsCap),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		logger.Debugf("UnmarshalJSON Position error %v", err)
		return err
	}
	wsCap.EventTime = time.Unix(0, aux.CurrentTimestamp*1000000)
	wsCap.ParseTime = time.Now()
	wsCap.OpeningTimestamp = time.Unix(0, aux.OpeningTimestamp*1000000)
	return nil
}

type WSPosition struct {
	ID                string    `json:"id"`
	Symbol            string    `json:"-"`
	CurrentQty        *float64   `json:"currentQty"`
	AvgEntryPrice     *float64   `json:"avgEntryPrice"`
	UnrealisedPnl     *float64   `json:"unrealisedPnl"`
	UnrealisedPnlPcnt *float64   `json:"unrealisedPnlPcnt"`
	UnrealisedRoePcnt *float64   `json:"unrealisedRoePcnt"`
	EventTime         time.Time `json:"-"`
	ParseTime         time.Time `json:"-"`
}

func (wsCap *WSPosition) UnmarshalJSON(data []byte) error {
	type Alias WSPosition
	aux := struct {
		CurrentTimestamp *int64 `json:"currentTimestamp"`
		*Alias
	}{
		Alias: (*Alias)(wsCap),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		logger.Debugf("UnmarshalJSON Position error %v", err)
		return err
	}
	if aux.CurrentTimestamp != nil {
		wsCap.EventTime = time.Unix(0, *aux.CurrentTimestamp*1000000)
	}
	wsCap.ParseTime = time.Now()
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
	Type         string    `json:"type"`
	Status       string    `json:"status"`
	MatchSize    float64   `json:"matchSize,string"`
	MatchPrice   float64   `json:"matchPrice,string"`
	OrderType    string    `json:"orderType"`
	Side         string    `json:"side"`
	Price        float64   `json:"price,string"`
	Size         float64   `json:"size,string"`
	RemainSize   float64   `json:"remainSize,string"`
	FilledSize   float64   `json:"filledSize,string"`
	CanceledSize float64   `json:"canceledSize,string"`
	TradeId      string    `json:"tradeId"`
	ClientOid    string    `json:"clientOid"`
	OldSize      float64   `json:"oldSize,string"`
	OrderTime    time.Time `json:"-`
	Liquidity    string    `json:"liquidity"`
	EventTime    time.Time `json:"-"`
	ParseTime    time.Time `json:"-"`
}

func (wsCap *WSOrder) UnmarshalJSON(data []byte) error {
	type Alias WSOrder
	aux := struct {
		OrderTime int64 `json:"orderTime,omitempty"`
		EventTime int64 `json:"ts,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(wsCap),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		logger.Debugf("UnmarshalJSON WsOrder error %v", err)
		return err
	}
	wsCap.EventTime = time.Unix(0, aux.EventTime)
	wsCap.ParseTime = time.Now()
	wsCap.OrderTime = time.Unix(0, aux.OrderTime)
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
	AvailableBalance *float64   `json:"availableBalance,string,omitempty"`
	HoldBalance      *float64   `json:"holdBalance,string,omitempty"`
	WithdrawHold     *float64   `json:"withdrawHold,string,omitempty"`
	EventTime        time.Time `json:"-"`
	Subject          string    `json:"-"`
}

type Account struct {
	AccountEquity    float64 `json:"accountEquity"`
	UnrealisedPNL    float64 `json:"unrealisedPNL"`
	MarginBalance    float64 `json:"marginBalance"`
	PositionMargin   float64 `json:"positionMargin"`
	OrderMargin      float64 `json:"orderMargin"`
	FrozenFunds      float64 `json:"frozenFunds"`
	AvailableBalance float64 `json:"availableBalance"`
	Currency         string  `json:"currency"`
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
