package okexv5_usdtswap

import (
	"encoding/json"
	"github.com/geometrybase/hft-micro/common"
	"time"
)

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


//{"arg":{"channel":"tickers","instId":"DOGE-USDT"},"data":[{"instType":"SPOT","instId":"DOGE-USDT","last":"0.254381","lastSz":"600","askPx":"0.254381","askSz":"1400","bidPx":"0.25438","bidSz":"400","open24h":"0.263668","high24h":"0.268614","low24h":"0.248601","sodUtc0":"0.260658","sodUtc8":"0.253989","volCcy24h":"125310776.54685","vol24h":"486148293.462458","ts":"1636737706397"}]}
type Ticker struct {
	InstId    string    `json:"instId"`
	AskPx     float64   `json:"askPx,string"`
	AskSz     float64   `json:"askSz,string"`
	BidPx     float64   `json:"bidPx,string"`
	BidSz     float64   `json:"bidSz,string"`
	TS time.Time `json:"-"`
}

func (ticker *Ticker) GetBidOffset() float64 {
	panic("implement me")
}

func (ticker *Ticker) GetAskOffset() float64 {
	panic("implement me")
}

func (ticker *Ticker) GetSymbol() string {
	return ticker.InstId
}

func (ticker *Ticker) GetTime() time.Time {
	return ticker.TS
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
	}
	return
}

