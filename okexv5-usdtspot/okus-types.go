package okexv5_usdtspot

import (
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"strconv"
	"time"
)

//    {
//      "alias": "",
//      "baseCcy": "BTC",
//      "category": "1",
//      "ctMult": "",
//      "ctType": "",
//      "ctVal": "",
//      "ctValCcy": "",
//      "expTime": "",
//      "instId": "BTC-USDT",
//      "instType": "SPOT",
//      "lever": "10",
//      "listTime": "1548133413000",
//      "lotSz": "0.00000001",
//      "minSz": "0.00001",
//      "optType": "",
//      "quoteCcy": "USDT",
//      "settleCcy": "",
//      "state": "live",
//      "stk": "",
//      "tickSz": "0.1",
//      "uly": ""
//    }

type Instrument struct {
	Alias    string  `json:"alias"`
	BaseCcy  string  `json:"baseCcy"`
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

//     {
//        "availBal": "",
//        "availEq": "227.32204128222",
//        "cashBal": "227.32204128222",
//        "ccy": "USDT",
//        "crossLiab": "",
//        "disEq": "227.3743253517149",
//        "eq": "227.32204128222",
//        "eqUsd": "227.3743253517149",
//        "frozenBal": "0",
//        "interest": "",
//        "isoEq": "0",
//        "isoLiab": "",
//        "isoUpl": "0",
//        "liab": "",
//        "maxLoan": "",
//        "mgnRatio": "",
//        "notionalLever": "0",
//        "ordFrozen": "0",
//        "stgyEq": "0",
//        "twap": "0",
//        "uTime": "1636700783723",
//        "upl": "0",
//        "uplLiab": ""
//      }

type Balance struct {
	Ccy       string    `json:"ccy"`
	Eq        float64   `json:"eq,string"`        //币种总权益
	CashBal   float64   `json:"cashBal,string"`   //币种余额
	FrozenBal float64   `json:"frozenBal,string"` //币种占用金额
	OrdFrozen float64   `json:"ordFrozen,string"` //挂单冻结数量
	UTime     time.Time `json:"-"`
	ParseTime time.Time `json:"-"`
}

func (b *Balance) UnmarshalJSON(data []byte) error {
	type Alias Balance
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

func (b *Balance) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (b *Balance) GetSymbol() string {
	return b.Ccy + "-USDT"
}

func (b *Balance) GetSize() float64 {
	return b.Eq
}

func (b *Balance) GetPrice() float64 {
	return 0.0
}

func (b *Balance) GetEventTime() time.Time {
	return b.UTime
}

func (b *Balance) GetParseTime() time.Time {
	return b.ParseTime
}

func (b *Balance) GetCurrency() string {
	return b.Ccy
}

func (b *Balance) GetBalance() float64 {
	return b.Eq
}

func (b *Balance) GetFree() float64 {
	return b.CashBal
}

func (b *Balance) GetUsed() float64 {
	return b.Eq - b.CashBal
}

func (b *Balance) GetTime() time.Time {
	return b.UTime
}

// "imr": "3372.2942371050594217",
//            "isoEq": "0",
//            "mgnRatio": "70375.35408747017",
//            "mmr": "134.8917694842024",
//            "notionalUsd": "33722.9423710505978888",
//            "ordFroz": "0",
//            "totalEq": "11172992.1657531589092577",
//            "uTime": "1623392334718"

type BalancesData struct {
	AdjEq     string    `json:"adjEq"`
	Details   []Balance `json:"details"`
	TotalEq   float64   `json:"totalEq,string"`
	UTime     time.Time `json:"-"`
	ParseTime time.Time `json:"-"`
}

func (b *BalancesData) UnmarshalJSON(data []byte) error {
	type Alias BalancesData
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
	UTime     time.Time `json:"-"`
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
	b.UTime = time.Unix(0, aux.UTime*1000000)
	b.ParseTime = time.Now()
	return nil
}

type BalanceAndPosition struct {
	BalData   []CashBalance `json:"balData"`
	EventType string        `json:"eventType"`
	PTime     time.Time     `json:"-"`
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

func (depth *Depth5) GetBidPrice() float64 {
	return depth.Bids[0][0]
}

func (depth *Depth5) GetAskPrice() float64 {
	return depth.Bids[1][0]
}

func (depth *Depth5) GetBidSize() float64 {
	return depth.Bids[0][1]
}

func (depth *Depth5) GetAskSize() float64 {
	return depth.Bids[1][1]
}

func (depth *Depth5) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (depth *Depth5) GetBids() common.Bids { return depth.Bids[:] }
func (depth *Depth5) GetAsks() common.Asks { return depth.Asks[:] }
func (depth *Depth5) GetSymbol() string    { return depth.InstId }
func (depth *Depth5) GetTime() time.Time   { return depth.EventTime }
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
