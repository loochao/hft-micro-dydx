package gtspot

import (
	"encoding/json"
	"github.com/geometrybase/hft-micro/common"
	"time"
)

var (
	TradeSideBuy  = "buy"
	TradeSideSell = "sell"
)

type WSRequest struct {
	Time    int64    `json:"time"`
	ID      int64    `json:"id,omitempty"`
	Channel string   `json:"channel"`
	Event   string   `json:"event"`
	Payload []string `json:"payload,omitempty"`
}

//{
//  "time": 1606292218,
//  "channel": "spot.trades",
//  "event": "update",
//  "result": {
//    "id": 309143071,
//    "create_time": 1606292218,
//    "create_time_ms": "1606292218213.4578",
//    "side": "sell",
//    "currency_pair": "GT_USDT",
//    "amount": "16.4700000000",
//    "price": "0.4705000000"
//  }
//}

type WSTrade struct {
	Channel string `json:"channel"`
	Event   string `json:"event"`
	Result  Trade  `json:"result"`
}

type Trade struct {
	ID           int64     `json:"-"`
	CreateTime   time.Time `json:"-"`
	CreateTimeMs string    `json:"-"`
	Side         string    `json:"side"`
	CurrencyPair string    `json:"currency_pair"`
	Amount       float64   `json:"-"`
	Price        float64   `json:"-"`
}

func (trade *Trade) GetSymbol() string  { return trade.CurrencyPair }
func (trade *Trade) GetSize() float64   { return trade.Amount }
func (trade *Trade) GetPrice() float64  { return trade.Price }
func (trade *Trade) GetTime() time.Time { return trade.CreateTime }
func (trade *Trade) IsUpTick() bool { return trade.Side == TradeSideBuy }

func (trade *Trade) UnmarshalJSON(data []byte) error {
	type Alias Trade
	aux := struct {
		ID         json.RawMessage `json:"id"`
		CreateTime json.RawMessage `json:"create_time_ms"`
		Amount     json.RawMessage `json:"amount"`
		Price      json.RawMessage `json:"price"`
		*Alias
	}{Alias: (*Alias)(trade)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	} else {
		createTime, err := common.ParseFloat(aux.CreateTime[1 : len(aux.CreateTime)-1])
		if err != nil {
			return err
		}
		trade.CreateTime = time.Unix(0, int64(createTime*1000000))
		trade.ID, err = common.ParseInt(aux.ID)
		if err != nil {
			return err
		}
		trade.Price, err = common.ParseFloat(aux.Price[1 : len(aux.Price)-1])
		if err != nil {
			return err
		}
		trade.Amount, err = common.ParseFloat(aux.Amount[1 : len(aux.Amount)-1])
		if err != nil {
			return err
		}
		return nil
	}
}
