package kucoin_usdtspot

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"strconv"
	"strings"
	"time"
)

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

type Depth5 struct {
	Symbol    string        `json:"-"`
	Bids      [5][2]float64 `json:"-"`
	Asks      [5][2]float64 `json:"_"`
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

func (depth *Depth5) GetSymbol() string  { return depth.Symbol }
func (depth *Depth5) GetTime() time.Time { return depth.EventTime }
func (depth *Depth5) UnmarshalJSON(data []byte) error {
	type Alias Depth5
	aux := struct {
		Data struct {
			Bids      [5][2]string `json:"bids,omitempty"`
			Asks      [5][2]string `json:"asks,omitempty"`
			EventTime int64        `json:"timestamp,omitempty"`
		} `json:"data,omitempty"`
		Topic string `json:"topic,omitempty"`
	}{}
	if err := json.Unmarshal(data, &aux); err != nil {
		logger.Debugf("ERR %v", err)
		return err
	}
	depth.Bids = [5][2]float64{}
	depth.Asks = [5][2]float64{}
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
	ParseTime time.Time `json:"-"`
}

func (a *Account) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (a *Account) GetSymbol() string {
	return a.Currency + "-USDT"
}

func (a *Account) GetSize() float64 {
	return a.Balance
}

func (a *Account) GetPrice() float64 {
	return 0.0
}

func (a *Account) GetEventTime() time.Time {
	return a.EventTime
}

func (a *Account) GetParseTime() time.Time {
	return a.ParseTime
}

func (a *Account) GetCurrency() string {
	return a.Currency
}

func (a *Account) GetBalance() float64 {
	return a.Balance
}

func (a *Account) GetFree() float64 {
	return a.Available
}

func (a *Account) GetUsed() float64 {
	return a.Holds
}

func (a *Account) GetTime() time.Time {
	return a.EventTime
}

func (a *Account) UnmarshalJSON(data []byte) error {
	type Alias Account
	aux := struct {
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		logger.Debugf("UnmarshalJSON WsOrder error %v", err)
		return err
	}
	a.EventTime = time.Now()
	a.ParseTime = time.Now()
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
	Symbol    string `json:"symbol"`
	OrderType string `json:"orderType"`
	Side      string `json:"side"`
	OrderId   string `json:"orderId"`
	Type      string `json:"type"`
	//OldSize    float64   `json:"oldSize,string"`
	OrderTime   time.Time `json:"-`
	Size        float64   `json:"size,string"`
	FilledSize  float64   `json:"filledSize,string"`
	FilledPrice float64   `json:"-"`
	MatchPrice  float64   `json:"matchPrice,string"`
	MatchSize   float64   `json:"matchSize,string"`
	TradeId     string    `json:"tradeId"`
	Price       float64   `json:"price,string"`
	ClientOid   string    `json:"clientOid"`
	RemainSize  float64   `json:"remainSize,string"`
	Status      string    `json:"status"`
	EventTime   time.Time `json:"-"`
	ParseTime   time.Time `json:"-"`
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
	return wsOrder.FilledPrice
}

func (wsOrder *WSOrder) GetSide() common.OrderSide {
	switch wsOrder.Side {
	case OrderSideBuy:
		return common.OrderSideBuy
	case OrderSideSell:
		return common.OrderSideSell
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
	switch wsOrder.Type {
	case OrderTypeOpen:
		return common.OrderStatusOpen
	case OrderTypeMatch:
		return common.OrderStatusFilled
	case OrderTypeFilled:
		return common.OrderStatusFilled
	default:
		return common.OrderStatusUnknown
	}
}

func (wsOrder *WSOrder) GetType() common.OrderType {
	switch wsOrder.OrderType {
	case OrderTypeLimit:
		return common.OrderTypeLimit
	case OrderTypeMarket:
		return common.OrderTypeMarket
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

func (wsOrder *WSOrder) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (wsOrder *WSOrder) UnmarshalJSON(data []byte) error {
	type Alias WSOrder
	aux := struct {
		OrderTime int64 `json:"orderTime,omitempty"`
		EventTime int64 `json:"ts,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(wsOrder),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		logger.Debugf("UnmarshalJSON WsOrder error %v", err)
		return err
	}
	wsOrder.EventTime = time.Unix(0, aux.EventTime)
	wsOrder.ParseTime = time.Now()
	wsOrder.OrderTime = time.Unix(0, aux.OrderTime)
	return nil
}

type SystemStatus struct {
	Msg    string
	Status string
}

//{
//    "type":"message",
//    "topic":"/market/match:BTC-USDT",
//    "subject":"trade.l3match",
//    "data":{
//
//        "sequence":"1545896669145",
//        "type":"match",
//        "symbol":"BTC-USDT",
//        "side":"buy",
//        "price":"0.08200000000000000000",
//        "size":"0.01022222000000000000",
//        "tradeId":"5c24c5da03aa673885cd67aa",
//        "takerOrderId":"5c24c5d903aa6772d55b371e",
//        "makerOrderId":"5c2187d003aa677bd09d5c93",
//        "time":"1545913818099033203"
//    }
//}

type WSTrade struct {
	Type    string `json:"type"`
	Topic   string `json:"topic"`
	Subject string `json:"subject"`
	Data    Trade  `json:"data"`
}

type Trade struct {
	Sequence     int64     `json:"-"`
	Type         string    `json:"type"`
	Symbol       string    `json:"symbol"`
	Side         string    `json:"side"`
	Price        float64   `json:"-"`
	Size         float64   `json:"-"`
	TradeId      string    `json:"tradeId"`
	TakerOrderId string    `json:"takerOrderId"`
	MakerOrderId string    `json:"makerOrderId"`
	Time         time.Time `json:"-"`
}

func (trade *Trade) UnmarshalJSON(data []byte) error {
	type Alias Trade
	aux := struct {
		Sequence json.RawMessage `json:"Sequence"`
		Price    json.RawMessage `json:"price"`
		Size     json.RawMessage `json:"size"`
		Time     json.RawMessage `json:"time"`
		*Alias
	}{
		Alias: (*Alias)(trade),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	} else {
		timestamp, err := common.ParseInt(aux.Time[1 : len(aux.Time)-1])
		if err != nil {
			return err
		}
		trade.Time = time.Unix(0, timestamp*1000000)
		trade.Price, err = common.ParseFloat(aux.Price[1 : len(aux.Price)-1])
		if err != nil {
			return err
		}
		trade.Size, err = common.ParseFloat(aux.Size[1 : len(aux.Size)-1])
		if err != nil {
			return err
		}
		trade.Sequence, err = common.ParseInt(aux.Sequence[1 : len(aux.Sequence)-1])
		if err != nil {
			return err
		}
		return nil
	}
}

func (trade *Trade) GetSymbol() string  { return trade.Symbol }
func (trade *Trade) GetSize() float64   { return trade.Size }
func (trade *Trade) GetPrice() float64  { return trade.Price }
func (trade *Trade) GetTime() time.Time { return trade.Time }
func (trade *Trade) IsUpTick() bool     { return trade.Side == TradeSideBuy }

type TickerData struct {
	Type    string `json:"type"`
	Topic   string `json:"topic"`
	Subject string `json:"subject"`
	Data    Ticker `json:"data"`
}

func (ticker *TickerData) UnmarshalJSON(data []byte) error {
	type Alias TickerData
	aux := struct {
		*Alias
	}{
		Alias: (*Alias)(ticker),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		logger.Debugf("json.Unmarshal error %v", err)
		return err
	}
	symbols := strings.Split(ticker.Topic, ":")
	if len(symbols) != 2 {
		return fmt.Errorf("bad topic %s", ticker.Topic)
	}
	ticker.Data.Symbol = symbols[1]
	return nil
}

//  "data":{
//
//        "sequence":"1545896668986", //序列号
//        "price":"0.08",             // 最近成交价格
//        "size":"0.011",             // 最近成交数量
//        "bestAsk":"0.08",           //最佳卖一价
//        "bestAskSize":"0.18",       // 最佳卖一数量
//        "bestBid":"0.049",          //最佳买一价
//        "bestBidSize":"0.036",      //最佳买一数量
//    }

type Ticker struct {
	Symbol       string    `json:"-"`
	BestBidSize  float64   `json:"bestBidSize,string"`
	BestBidPrice float64   `json:"bestBid,string"`
	BestAskSize  float64   `json:"bestAskSize,string"`
	BestAskPrice float64   `json:"bestAsk,string"`
	EventTime    time.Time `json:"-"`
}

func (ticker *Ticker) GetSymbol() string {
	return ticker.Symbol
}

func (ticker *Ticker) GetTime() time.Time {
	return ticker.EventTime
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
		*Alias
		Time int64 `json:"time"`
	}{
		Alias: (*Alias)(ticker),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		logger.Debugf("json.Unmarshal error %v", err)
		return err
	}
	ticker.EventTime = time.Unix(0, aux.Time*1000000)
	return nil
}

type FundingRate struct {
	Symbol string
}

func (f FundingRate) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (f FundingRate) GetSymbol() string {
	return f.Symbol
}

func (f FundingRate) GetFundingRate() float64 {
	return 0
}

func (f FundingRate) GetNextFundingTime() time.Time {
	return time.Time{}
}
