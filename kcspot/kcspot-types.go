package kcspot

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/logger"
	"strconv"
	"strings"
	"time"
)

const (
	CandleType1Min   = "1min"
	CandleType3Min   = "3min"
	CandleType5Min   = "5min"
	CandleType15Min  = "15min"
	CandleType30Min  = "30min"
	CandleType1Hour  = "1hour"
	CandleType4Hour  = "4hour"
	CandleType6Hour  = "6hour"
	CandleType8Hour  = "8hour"
	CandleType12Hour = "12hour"
	CandleType1Day   = "1day"
	CandleType1Week  = "1week"

	OrderStatusOpen  = "open"
	OrderStatusMatch = "match"
	OrderStatusDone  = "done"

	OrderTypeOpen     = "open"
	OrderTypeMatch    = "match"
	OrderTypeFilled   = "filled"
	OrderTypeCanceled = "canceled"
	OrderTypeUpdate   = "update"
)

var CandleTypeDurations = map[string]time.Duration{
	CandleType1Min:   time.Minute,
	CandleType3Min:   time.Minute * 3,
	CandleType5Min:   time.Minute * 5,
	CandleType15Min:  time.Minute * 15,
	CandleType30Min:  time.Minute * 30,
	CandleType1Hour:  time.Hour,
	CandleType4Hour:  time.Hour * 4,
	CandleType6Hour:  time.Hour * 6,
	CandleType8Hour:  time.Hour * 8,
	CandleType12Hour: time.Hour * 12,
	CandleType1Day:   time.Hour * 24,
	CandleType1Week:  time.Hour * 168,
}

type Symbol struct {
	Symbol          string  `json:"symbol,omitempty"`
	Name            string  `json:"name,omitempty"`
	BaseCurrency    string  `json:"baseCurrency,omitempty"`
	QuoteCurrency   string  `json:"quoteCurrency,omitempty"`
	BaseMinSize     float64 `json:"baseMinSize,string,omitempty"`
	QuoteMinSize    float64 `json:"quoteMinSize,string,omitempty"`
	BaseMaxSize     float64 `json:"baseMaxSize,string,omitempty"`
	QuoteMaxSize    float64 `json:"quoteMaxSize,string,omitempty"`
	BaseIncrement   float64 `json:"baseIncrement,string,omitempty"`
	QuoteIncrement  float64 `json:"quoteIncrement,string,omitempty"`
	PriceIncrement  float64 `json:"priceIncrement,string,omitempty"`
	FeeCurrency     string  `json:"feeCurrency,omitempty"`
	Market          string  `json:"market,omitempty"`
	EnableTrading   bool    `json:"enableTrading,omitempty"`
	IsMarginEnabled bool    `json:"isMarginEnabled,omitempty"`
	PriceLimitRate  float64 `json:"priceLimitRate,string,omitempty"`
}

type DataCap struct {
	Code int             `json:"code,string,omitempty"`
	Msg  string          `json:"msg,omitempty"`
	Data json.RawMessage `json:"data,omitempty"`
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

type Depth50 struct {
	Symbol    string         `json:"-"`
	Bids      [50][2]float64 `json:"-"`
	Asks      [50][2]float64 `json:"_"`
	ParseTime time.Time      `json:"-"`
	EventTime time.Time      `json:"-"`
}

func (depth *Depth50) UnmarshalJSON(data []byte) error {
	type Alias Depth50
	aux := struct {
		Data struct {
			Bids      [50][2]string `json:"bids,omitempty"`
			Asks      [50][2]string `json:"asks,omitempty"`
			EventTime int64         `json:"timestamp,omitempty"`
		} `json:"data,omitempty"`
		Topic string `json:"topic,omitempty"`
	}{}
	if err := json.Unmarshal(data, &aux); err != nil {
		logger.Debugf("ERR %v", err)
		return err
	}
	depth.Bids = [50][2]float64{}
	depth.Asks = [50][2]float64{}
	for i, d := range aux.Data.Bids {
		price, _ := strconv.ParseFloat(d[0], 64)
		size, _ := strconv.ParseFloat(d[1], 64)
		depth.Bids[i][0] = price
		depth.Bids[i][1] = size
	}
	for i, d := range aux.Data.Asks {
		price, _ := strconv.ParseFloat(d[0], 64)
		size, _ := strconv.ParseFloat(d[1], 64)
		depth.Asks[i][0] = price
		depth.Asks[i][1] = size
	}
	depth.EventTime = time.Unix(0, aux.Data.EventTime*1000000)
	depth.ParseTime = time.Now()
	segs := strings.Split(aux.Topic, ":")
	if len(segs) == 2 {
		depth.Symbol = segs[1]
	}
	return nil
}

type Ping struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type Credentials struct {
	Key        string
	Secret     string
	Passphrase string
}

type Account struct {
	ID        string    `json:"id"`
	Currency  string    `json:"currency"`
	Type      string    `json:"type"`
	Balance   float64   `json:"balance,string"`
	Available float64   `json:"available,string"`
	Holds     float64   `json:"holds,string"`
	EventTime time.Time `json:"-"`
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

func passPhraseEncrypt(key, plain []byte) string {
	hm := hmac.New(sha256.New, key)
	hm.Write(plain)
	return base64.StdEncoding.EncodeToString(hm.Sum(nil))
}

type OrderResponse struct {
	OrderId string `json:"orderId"`
}

type CancelAllOrdersResponse struct {
	CancelledOrderIds []string `json:"cancelledOrderIds"`
}

type WsCap struct {
	ID          string          `json:"id"`
	Type        string          `json:"type"`
	Topic       string          `json:"topic"`
	Subject     string          `json:"subject"`
	ChannelType string          `json:"channelType"`
	Data        json.RawMessage `json:"data"`
}

type WsBalance struct {
	AccountId       string  `json:"accountId"`
	Total           float64 `json:"total,string"`
	Available       float64 `json:"available,string"`
	AvailableChange float64 `json:"availableChange,string"`
	Currency        string  `json:"currency"`
	Hold            float64 `json:"hold,string"`
	HoldChange      float64 `json:"holdChange,string"`
	RelationEvent   string  `json:"relationEvent"`
	RelationEventId string  `json:"relationEventId"`
	RelationContext struct {
		TradeId string `json:"tradeId"`
		OrderId string `json:"orderId"`
		Symbol  string `json:"symbol"`
	} `json:"relationContext"`
	EventTime time.Time `json:"-"`
	ParseTime time.Time `json:"-"`
}

func (wsCap *WsBalance) UnmarshalJSON(data []byte) error {
	type Alias WsBalance
	aux := struct {
		EventTime int64 `json:"time,string"`
		*Alias
	}{
		Alias: (*Alias)(wsCap),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		logger.Debugf("UnmarshalJSON WsOrder error %v", err)
		return err
	}
	wsCap.EventTime = time.Unix(0, aux.EventTime*1000000)
	wsCap.ParseTime = time.Now()
	return nil
}

type WSOrder struct {
	Symbol     string    `json:"symbol"`
	OrderType  string    `json:"orderType"`
	Side       string    `json:"side"`
	OrderId    string    `json:"orderId"`
	Type       string    `json:"type"`
	OldSize    float64   `json:"oldSize,string"`
	OrderTime  time.Time `json:"-`
	Size       float64   `json:"size,string"`
	FilledSize float64   `json:"filledSize,string"`
	Price      float64   `json:"price,string"`
	ClientOid  string    `json:"clientOid"`
	RemainSize float64   `json:"remainSize,string"`
	Status     string    `json:"status"`
	EventTime  time.Time `json:"-"`
	ParseTime  time.Time `json:"-"`
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
