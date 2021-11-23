package okexv5_usdtswap

import (
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"strconv"
	"time"
)

// {
//      "alias": "",
//      "baseCcy": "",
//      "category": "1",
//      "ctMult": "1",
//      "ctType": "linear",
//      "ctVal": "0.01",
//      "ctValCcy": "BTC",
//      "expTime": "",
//      "instId": "BTC-USDT-SWAP",
//      "instType": "SWAP",
//      "lever": "125",
//      "listTime": "1636620075000",
//      "lotSz": "1",
//      "minSz": "1",
//      "optType": "",
//      "quoteCcy": "",
//      "settleCcy": "USDT",
//      "state": "live",
//      "stk": "",
//      "tickSz": "0.1",
//      "uly": "BTC-USDT"
//    }

type Instrument struct {
	CtVal    float64 `json:"ctVal,string"`
	CtValCcy string  `json:"ctValCcy"`
	CtMult   float64 `json:"ctMult,string"`
	Lever    int64   `json:"lever,string"`
	InstId   string  `json:"instId"`
	InstType string  `json:"instType"`
	ListTime int64   `json:"listTime,string"`
	LotSz    float64 `json:"lotSz,string"`
	MinSz    float64 `json:"MinSz,string"`
	QuoteCcy string  `json:"quoteCcy"`
	State    string  `json:"state"`
	TickSz   float64 `json:"tickSz,string"`
}

//{
//      "adl": "2",
//      "availPos": "1",
//      "avgPx": "32.65",
//      "cTime": "1636814100137",
//      "ccy": "USDT",
//      "deltaBS": "",
//      "deltaPA": "",
//      "gammaBS": "",
//      "gammaPA": "",
//      "imr": "32.645704200632",
//      "instId": "ATOM-USDT-SWAP",
//      "instType": "SWAP",
//      "interest": "0",
//      "last": "32.653",
//      "lever": "1",
//      "liab": "",
//      "liabCcy": "",
//      "liqPx": "241.42399513235674",
//      "margin": "",
//      "markPx": "32.645704200632",
//      "mgnMode": "cross",
//      "mgnRatio": "628.2958655980798",
//      "mmr": "0.32645704200632",
//      "notionalUsd": "32.65288625555614",
//      "optVal": "",
//      "pos": "1",
//      "posCcy": "",
//      "posId": "379761092775538690",
//      "posSide": "short",
//      "thetaBS": "",
//      "thetaPA": "",
//      "tradeId": "20034596",
//      "uTime": "1636814108239",
//      "upl": "0.004295799368002",
//      "uplRatio": "0.0001315711904442",
//      "usdPx": "",
//      "vegaBS": "",
//      "vegaPA": ""
//    }

type Position struct {
	//Adl         int       `json:"adl,string"`
	AvgPx float64 `json:"avgPx,string"`
	//CTime       time.Time `json:"-"`
	Ccy string `json:"ccy"`
	//Imr         float64   `json:"imr,string"` //初始保证金，仅适用于全仓
	InstId   string `json:"instId"`
	InstType string `json:"instType"`
	//Last        float64   `json:"last,string"`
	//Lever       float64   `json:"lever,string"`
	//LiqPx       float64   `json:"liqPx,string"`
	//MarkPx      float64   `json:"markPx,string"`
	MgnMode  string  `json:"mgnMode"`
	MgnRatio float64 `json:"mgnRatio,string"`
	Mmr      float64 `json:"mmr,string"` //维持保证金
	//NotionalUsd float64   `json:"notionalUsd,string"` //以美金价值为单位的持仓数量
	Pos     float64   `json:"pos,string"`
	PosId   string    `json:"posId"`
	PosSide string    `json:"posSide"`
	TradeId string    `json:"tradeId"`
	UTime   time.Time `json:"-"`
	//Upl         float64   `json:"upl,string"`
	//UplRatio    float64   `json:"uplRatio,string"`
	ParseTime time.Time `json:"-"`
}

func (p *Position) GetSymbol() string {
	return p.InstId
}

func (p *Position) GetSize() float64 {
	return p.Pos
}

func (p *Position) GetPrice() float64 {
	return p.AvgPx
}

func (p *Position) GetEventTime() time.Time {
	return p.UTime
}

func (p *Position) GetParseTime() time.Time {
	return p.ParseTime
}

func (p *Position) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (p *Position) UnmarshalJSON(data []byte) error {
	type Alias Position
	aux := &struct {
		UTime int64 `json:"uTime,string"`
		*Alias
	}{
		Alias: (*Alias)(p),
	}
	var err error
	if err = json.Unmarshal(data, &aux); err != nil {
		return err
	}
	p.UTime = time.Unix(0, aux.UTime*1000000)
	p.ParseTime = time.Now()
	return nil
}

//{
//        "title": "Spot System Upgrade",
//        "state": "scheduled",
//        "begin": "1620723600000",
//        "end": "1620724200000",
//        "href": "",
//        "serviceType": "1",
//        "system": "classic",
//        "scheDesc": ""
//    }

type Status struct {
	Title       string `json:"title"`
	State       string `json:"state"`
	Begin       int64  `json:"begin,string"`
	End         int64  `json:"end,string"`
	Href        string `json:"href"`
	ServiceType int    `json:"serviceType,string"`
	System      string `json:"system"`
	ScheDesc    string `json:"scheDesc"`
}

//        {
//          "availBal": "",
//          "availEq": "200.196998228728",
//          "cashBal": "211.14357218222",
//          "ccy": "USDT",
//          "crossLiab": "",
//          "disEq": "211.1789526319414",
//          "eq": "211.149391717101",
//          "eqUsd": "211.1789526319414",
//          "frozenBal": "10.952393488373",
//          "interest": "",
//          "isoEq": "0",
//          "isoLiab": "",
//          "isoUpl": "0",
//          "liab": "",
//          "maxLoan": "",
//          "mgnRatio": "623.910531981601",
//          "notionalLever": "0.155611059060692",
//          "ordFrozen": "0",
//          "stgyEq": "0",
//          "twap": "0",
//          "uTime": "1636815658786",
//          "upl": "0.0058195348809988",
//          "uplLiab": ""
//        }

type Account struct {
	Ccy           string    `json:"ccy"`
	AvailEq       float64   `json:"availEq,string"`
	CashBal       float64   `json:"cashBal,string"`
	DisEq         float64   `json:"disEq,string"`
	Eq            float64   `json:"eq,string"`    //币种总权益
	EqUsd         float64   `json:"eqUsd,string"` //币种总权益
	FrozenBal     float64   `json:"frozenBal,string"`
	MgnRatio      float64   `json:"-"`
	NotionalLever float64   `json:"-"`
	Upl           float64   `json:"upl,string"`
	UTime         time.Time `json:"-"`
	ParseTime     time.Time `json:"-"`
}

func (b *Account) UnmarshalJSON(data []byte) error {
	type Alias Account
	aux := &struct {
		UTime         int64  `json:"uTime,string"`
		MgnRatio      string `json:"mgnRatio"`
		NotionalLever string `json:"notionalLever"`
		*Alias
	}{
		Alias: (*Alias)(b),
	}
	var err error
	if err = json.Unmarshal(data, &aux); err != nil {
		return err
	}
	b.UTime = time.Unix(0, aux.UTime*1000000)
	b.ParseTime = time.Now()
	if aux.MgnRatio != "" {
		b.MgnRatio, err = strconv.ParseFloat(aux.MgnRatio, 64)
	}
	if aux.NotionalLever != "" {
		b.NotionalLever, err = strconv.ParseFloat(aux.NotionalLever, 64)
	}
	return nil
}

func (b *Account) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (b *Account) GetEventTime() time.Time {
	return b.UTime
}

func (b *Account) GetParseTime() time.Time {
	return b.ParseTime
}

func (b *Account) GetCurrency() string {
	return b.Ccy
}

func (b *Account) GetBalance() float64 {
	return b.Eq
}

func (b *Account) GetFree() float64 {
	return b.AvailEq
}

func (b *Account) GetUsed() float64 {
	return b.Eq - b.AvailEq
}

func (b *Account) GetTime() time.Time {
	return b.UTime
}

//{
//      "adjEq": "",
//      "details": [
//        {
//          "availBal": "",
//          "availEq": "200.196998228728",
//          "cashBal": "211.14357218222",
//          "ccy": "USDT",
//          "crossLiab": "",
//          "disEq": "211.1789526319414",
//          "eq": "211.149391717101",
//          "eqUsd": "211.1789526319414",
//          "frozenBal": "10.952393488373",
//          "interest": "",
//          "isoEq": "0",
//          "isoLiab": "",
//          "isoUpl": "0",
//          "liab": "",
//          "maxLoan": "",
//          "mgnRatio": "623.910531981601",
//          "notionalLever": "0.155611059060692",
//          "ordFrozen": "0",
//          "stgyEq": "0",
//          "twap": "0",
//          "uTime": "1636815658786",
//          "upl": "0.0058195348809988",
//          "uplLiab": ""
//        }
//      ],
//      "imr": "",
//      "isoEq": "0",
//      "mgnRatio": "",
//      "mmr": "",
//      "notionalUsd": "",
//      "ordFroz": "",
//      "totalEq": "227.60212008766186",
//      "uTime": "1636817309566"
//    }

type AccountData struct {
	AdjEq     string    `json:"adjEq"`
	Details   []Account `json:"details"`
	TotalEq   float64   `json:"totalEq,string"`
	UTime     time.Time `json:"-"`
	ParseTime time.Time `json:"-"`
}

func (b *AccountData) UnmarshalJSON(data []byte) error {
	type Alias AccountData
	aux := &struct {
		UTime int64 `json:"uTime,string"`
		*Alias
	}{
		Alias: (*Alias)(b),
	}
	var err error
	if err = json.Unmarshal(data, &aux); err != nil {
		return err
	}
	b.UTime = time.Unix(0, aux.UTime*1000000)
	b.ParseTime = time.Now()
	return nil
}

// {
//          "cashBal": "0.00000051",
//          "ccy": "LTC",
//          "uTime": "1625739325118"
//        }
type CashBalance struct {
	CashBal   float64   `json:"cashBal,string"`
	Ccy       string    `json:"ccy"`
	EventTime time.Time `json:"-"`
	ParseTime time.Time `json:"-"`
}

func (b *CashBalance) UnmarshalJSON(data []byte) error {
	type Alias CashBalance
	aux := &struct {
		UTime int64 `json:"uTime,string"`
		*Alias
	}{
		Alias: (*Alias)(b),
	}
	var err error
	if err = json.Unmarshal(data, &aux); err != nil {
		return err
	}
	b.EventTime = time.Unix(0, aux.UTime*1000000)
	b.ParseTime = time.Now()
	return nil
}

type BalanceAndPosition struct {
	PosData   []Position `json:"posData"`
	EventType string     `json:"eventType"`
	PTime     time.Time  `json:"-"`
}

func (b *BalanceAndPosition) UnmarshalJSON(data []byte) error {
	type Alias BalanceAndPosition
	aux := &struct {
		PTime int64 `json:"pTime,string"`
		*Alias
	}{
		Alias: (*Alias)(b),
	}
	var err error
	if err = json.Unmarshal(data, &aux); err != nil {
		return err
	}
	b.PTime = time.Unix(0, aux.PTime*1000000)
	return nil
}

type OrderResponse struct {
	ClOrdId string `json:"clOrdId,omitempty"`
	OrdId   string `json:"ordId,omitempty"`
	Tag     string `json:"tag,omitempty"`
	SCode   string `json:"sCode,omitempty"`
	SMsg    string `json:"sMsg,omitempty"`
}

//{
//      "accFillSz": "0",
//      "amendResult": "",
//      "avgPx": "0",
//      "cTime": "1636718644502",
//      "category": "normal",
//      "ccy": "",
//      "clOrdId": "",
//      "code": "0",
//      "execType": "",
//      "fee": "0",
//      "feeCcy": "USDT",
//      "fillFee": "0",
//      "fillFeeCcy": "",
//      "fillNotionalUsd": "",
//      "fillPx": "",
//      "fillSz": "0",
//      "fillTime": "",
//      "instId": "ATOM-USDT",
//      "instType": "SPOT",
//      "lever": "0",
//      "msg": "",
//      "notionalUsd": "17.493419453650002",
//      "ordId": "379360722823835652",
//      "ordType": "limit",
//      "pnl": "0",
//      "posSide": "",
//      "px": "35",
//      "rebate": "0",
//      "rebateCcy": "ATOM",
//      "reqId": "",
//      "side": "sell",
//      "slOrdPx": "",
//      "slTriggerPx": "",
//      "slTriggerPxType": "last",
//      "state": "live",
//      "sz": "0.499777",
//      "tag": "",
//      "tdMode": "cash",
//      "tgtCcy": "",
//      "tpOrdPx": "",
//      "tpTriggerPx": "",
//      "tpTriggerPxType": "last",
//      "tradeId": "",
//      "uTime": "1636718644502"
//    }

type Order struct {
	InstId   string    `json:"instId"`
	UTime    time.Time `json:"uTime"`
	OrdId    string    `json:"ordId"`
	InstType string    `json:"instType"`
	TdMode   string    `json:"tdMode"`
	Price    *float64  `json:"-"`
	Size     float64   `json:"sz,string"`
	Side     string    `json:"side"`
	OrdType  string    `json:"ordType"`
	ClOrdId  string    `json:"clOrdId"`
	FillSz   *float64  `json:"-"`
	FillPx   *float64  `json:"-"`
	AvgPx    *float64  `json:"-"`
	State    string    `json:"state"`
}

func (o *Order) GetSymbol() string {
	return o.InstId
}

func (o *Order) GetSize() float64 {
	return o.Size
}

func (o *Order) GetPrice() float64 {
	if o.Price != nil {
		return *o.Price
	} else {
		return 0.0
	}
}

func (o *Order) GetFilledSize() float64 {
	if o.FillSz != nil {
		return *o.FillSz
	} else {
		return 0.0
	}
}

func (o *Order) GetFilledPrice() float64 {
	if o.AvgPx != nil {
		return *o.AvgPx
	} else if o.FillPx != nil {
		return *o.FillPx
	} else {
		return 0.0
	}
}

func (o *Order) GetSide() common.OrderSide {
	switch o.Side {
	case OrderSideBuy:
		return common.OrderSideBuy
	case OrderSideSell:
		return common.OrderSideSell
	default:
		return common.OrderSideUnknown
	}
}

func (o *Order) GetClientID() string {
	return o.ClOrdId
}

func (o *Order) GetID() string {
	return o.OrdId
}

func (o *Order) GetStatus() common.OrderStatus {
	switch o.State {
	case OrderStateCanceled:
		return common.OrderStatusCancelled
	case OrderStateLive:
		return common.OrderStatusOpen
	case OrderStatePartiallyFilled:
		return common.OrderStatusPartiallyFilled
	case OrderStateFilled:
		return common.OrderStatusFilled
	default:
		return common.OrderStatusUnknown
	}
}

func (o *Order) GetType() common.OrderType {
	switch o.OrdType {
	case OrderTypeLimit:
		return common.OrderTypeLimit
	case OrderTypeMarket:
		return common.OrderTypeMarket
	case OrderTypeFOK:
		return common.OrderTypeLimit
	case OrderTypeIOC:
		return common.OrderTypeLimit
	case OrderTypePostOnly:
		return common.OrderTypeLimit
	default:
		return common.OrderTypeUnknown
	}
}

func (o *Order) GetPostOnly() bool {
	return o.OrdType == OrderTypePostOnly
}

func (o *Order) GetReduceOnly() bool {
	return false
}

func (o *Order) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (o *Order) UnmarshalJSON(data []byte) error {
	type Alias Order
	aux := &struct {
		UTime  int64  `json:"uTime,string"`
		FillSz string `json:"fillSz"`
		FillPx string `json:"fillPx"`
		AvgPx  string `json:"avgPx"`
		Price  string `json:"px"`
		*Alias
	}{
		Alias: (*Alias)(o),
	}
	var err error
	if err = json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.FillSz != "" {
		o.FillSz = new(float64)
		*o.FillSz, err = strconv.ParseFloat(aux.FillSz, 64)
	}
	if aux.FillPx != "" {
		o.FillPx = new(float64)
		*o.FillPx, err = strconv.ParseFloat(aux.FillPx, 64)
	}
	if aux.AvgPx != "" {
		o.AvgPx = new(float64)
		*o.AvgPx, err = strconv.ParseFloat(aux.AvgPx, 64)
	}
	if aux.Price != "" {
		o.Price = new(float64)
		*o.Price, err = strconv.ParseFloat(aux.Price, 64)
	}
	o.UTime = time.Unix(0, aux.UTime*1000000)
	return nil
}

type Credentials struct {
	Key        string
	Secret     string
	Passphrase string
}

type Depth5 struct {
	InstId    string        `json:"-"`
	Bids      [5][2]float64 `json:"-"`
	Asks      [5][2]float64 `json:"_"`
	ParseTime time.Time     `json:"-"`
	EventTime time.Time     `json:"-"`
}

func (depth *Depth5) GetParseTime() time.Time {
	return depth.ParseTime
}

func (depth *Depth5) GetBidOffset() float64 {
	if depth.Bids[0][0] != 0 {
		return (depth.Asks[0][0] - depth.Bids[0][0]) * 0.5 / depth.Bids[0][0]
	} else {
		return common.DefaultBidAskOffset
	}
}

func (depth *Depth5) GetAskOffset() float64 {
	if depth.Asks[0][0] != 0 {
		return (depth.Asks[0][0] - depth.Bids[0][0]) * 0.5 / depth.Asks[0][0]
	} else {
		return common.DefaultBidAskOffset
	}
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

func (depth *Depth5) GetBids() common.Bids    { return depth.Bids[:] }
func (depth *Depth5) GetAsks() common.Asks    { return depth.Asks[:] }
func (depth *Depth5) GetSymbol() string       { return depth.InstId }
func (depth *Depth5) GetEventTime() time.Time { return depth.EventTime }
func (depth *Depth5) UnmarshalJSON(data []byte) error {
	type Alias Depth5
	aux := struct {
		Data []struct {
			Bids      [5][4]string `json:"bids"`
			Asks      [5][4]string `json:"asks"`
			InstId    string       `json:"instId"`
			EventTime int64        `json:"ts,string"`
		} `json:"data"`
	}{}
	var err error
	if err = json.Unmarshal(data, &aux); err != nil {
		logger.Debugf("ERR %v", err)
		return err
	}
	if len(aux.Data) != 1 {
		return fmt.Errorf("bad deth5 format %s", data)
	}
	depth.Bids = [5][2]float64{}
	depth.Asks = [5][2]float64{}
	for i, d := range aux.Data[0].Bids {
		price, _ := strconv.ParseFloat(d[0], 64)
		size, _ := strconv.ParseFloat(d[1], 64)
		depth.Bids[i][0] = price
		depth.Bids[i][1] = size
	}
	for i, d := range aux.Data[0].Asks {
		price, _ := strconv.ParseFloat(d[0], 64)
		size, _ := strconv.ParseFloat(d[1], 64)
		depth.Asks[i][0] = price
		depth.Asks[i][1] = size
	}
	depth.InstId = aux.Data[0].InstId
	depth.EventTime = time.Unix(0, aux.Data[0].EventTime*1000000)
	depth.ParseTime = time.Now()
	return nil
}

//    {
//      "instId": "DOGE-USDT",
//      "tradeId": "106645495",
//      "px": "0.256222",
//      "sz": "14.19554",
//      "side": "sell",
//      "ts": "1636778780284"
//    }

type Trade struct {
	InstId string    `json:"instId"`
	Px     float64   `json:"px,string"`
	Sz     float64   `json:"sz,string"`
	Side   string    `json:"side"`
	TS     time.Time `json:"-"`
}

func (t *Trade) GetPrice() float64 {
	return t.Px
}

func (t *Trade) GetSize() float64 {
	return t.Sz
}

func (t *Trade) GetTime() time.Time {
	return t.TS
}

func (t *Trade) IsUpTick() bool {
	return t.Side == "buy"
}

func (t *Trade) GetSymbol() string {
	return t.InstId
}

func (t *Trade) UnmarshalJSON(data []byte) error {
	type Alias Trade
	aux := &struct {
		TS int64 `json:"ts,string"`
		*Alias
	}{
		Alias: (*Alias)(t),
	}
	var err error
	if err = json.Unmarshal(data, &aux); err != nil {
		return err
	}
	t.TS = time.Unix(0, aux.TS*1000000)
	return nil
}

//{
//        "uid": "43812",
//        "acctLv": "2",
//        "posMode": "long_short_mode",
//        "autoLoan": true,
//        "greeksType": "BS",
//        "level": "lv1",
//        "levelTmp": ""
//    }

type AccountConfig struct {
	Uid        string `json:"uid"`
	AcctLv     string `json:"acctLv"`
	PosMode    string `json:"posMode"`
	AutoLoan   bool   `json:"autoLoan"`
	GreeksType string `json:"greeksType"`
	Level      string `json:"level"`
	LevelTmp   string `json:"levelTmp"`
}

//{
//        "instType": "SWAP",
//        "instId": "BTC-USD-SWAP",
//        "fundingRate": "0.018",
//        "nextFundingRate": "",
//        "fundingTime": "1597026383085"
//    }

type FundingRate struct {
	InstType        string    `json:"instType"`
	InstId          string    `json:"instId"`
	FundingRate     float64   `json:"fundingRate,string"`
	NextFundingRate float64   `json:"nextFundingRate,string"`
	FundingTime     time.Time `json:"-"`
}

func (f *FundingRate) GetSymbol() string {
	return f.InstId
}

func (f *FundingRate) GetFundingRate() float64 {
	return f.FundingRate
}

func (f *FundingRate) GetNextFundingTime() time.Time {
	return f.FundingTime
}

func (f *FundingRate) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (f *FundingRate) UnmarshalJSON(data []byte) error {
	type Alias FundingRate
	aux := &struct {
		FundingTime int64 `json:"fundingTime,string"`
		*Alias
	}{
		Alias: (*Alias)(f),
	}
	var err error
	if err = json.Unmarshal(data, &aux); err != nil {
		return err
	}
	f.FundingTime = time.Unix(0, aux.FundingTime*1000000)
	return nil
}

type PositionTierParam struct {
	InstType string `json:"instType"`
	TdMode   string `json:"tdMode"`
	Uly      string `json:"uly"`
	Tier     string `json:"tier"`
}

type PositionTier struct {
	Uly      string  `json:"uly"`
	InstId   string  `json:"instId"`
	Tier     string  `json:"tier"`
	MinSz    float64 `json:"minSz,string"`
	MaxSz    float64 `json:"maxSz,string"`
	Mmr      float64 `json:"mmr,string"`
	Imr      float64 `json:"imr,string"`
	MaxLever float64 `json:"maxLever,string"`
}

//{"arg":{"channel":"tickers","instId":"DOGE-USDT"},"data":[{"instType":"SPOT","instId":"DOGE-USDT","last":"0.254381","lastSz":"600","askPx":"0.254381","askSz":"1400","bidPx":"0.25438","bidSz":"400","open24h":"0.263668","high24h":"0.268614","low24h":"0.248601","sodUtc0":"0.260658","sodUtc8":"0.253989","volCcy24h":"125310776.54685","vol24h":"486148293.462458","ts":"1636737706397"}]}
type Ticker struct {
	InstId    string    `json:"instId"`
	AskPx     float64   `json:"askPx,string"`
	AskSz     float64   `json:"askSz,string"`
	BidPx     float64   `json:"bidPx,string"`
	BidSz     float64   `json:"bidSz,string"`
	TS        time.Time `json:"-"`
	ParseTime time.Time `json:"-"`
}

func (ticker *Ticker) GetEventTime() time.Time {
	return ticker.TS
}

func (ticker *Ticker) GetParseTime() time.Time {
	return ticker.ParseTime
}

func (ticker *Ticker) GetBidOffset() float64 {
	if ticker.BidPx != 0 {
		return (ticker.AskPx - ticker.BidPx) * 0.5 / ticker.BidPx
	} else {
		return common.DefaultBidAskOffset
	}
}

func (ticker *Ticker) GetAskOffset() float64 {
	if ticker.AskPx != 0 {
		return (ticker.AskPx - ticker.BidPx) * 0.5 / ticker.AskPx
	} else {
		return common.DefaultBidAskOffset
	}
}

func (ticker *Ticker) GetSymbol() string {
	return ticker.InstId
}

func (ticker *Ticker) GetBidPrice() float64 {
	return ticker.BidPx
}

func (ticker *Ticker) GetAskPrice() float64 {
	return ticker.AskPx
}

func (ticker *Ticker) GetBidSize() float64 {
	return ticker.BidSz
}

func (ticker *Ticker) GetAskSize() float64 {
	return ticker.AskSz
}

func (ticker *Ticker) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (ticker *Ticker) UnmarshalJSON(data []byte) (err error) {
	type Alias Ticker
	aux := &struct {
		TS int64 `json:"ts,string"`
		*Alias
	}{
		Alias: (*Alias)(ticker),
	}
	if err = json.Unmarshal(data, &aux); err == nil {
		ticker.TS = time.Unix(0, aux.TS*1000000)
		ticker.ParseTime = time.Now()
	}
	return
}

type WsArgs struct {
	Channel  string `json:"channel"`
	InstType string `json:"instType,omitempty"`
	Uly      string `json:"uly,omitempty"`
	InstId   string `json:"instId,omitempty"`
}

type WsSubUnsub struct {
	Op   string   `json:"op"`
	Args []WsArgs `json:"args"`
}

type WsLoginArgs struct {
	ApiKey     string `json:"apiKey"`
	Passphrase string `json:"passphrase"`
	Timestamp  string `json:"timestamp"`
	Sign       string `json:"sign"`
}

type WsLogin struct {
	Op   string        `json:"op"`
	Args []WsLoginArgs `json:"args"`
}

type CommonCapture struct {
	Table  string `json:"table,omitempty"`
	Action string `json:"action,omitempty"`

	Event string `json:"event,omitempty"`
	Msg   string `json:"msg,omitempty"`
	Code  string `json:"code,omitempty"`
	Arg   struct {
		Channel string `json:"channel,omitempty"`
		//UID int64 `json:"uid"`
	} `json:"arg,omitempty"`
	Data json.RawMessage `json:"data,omitempty"`
}
