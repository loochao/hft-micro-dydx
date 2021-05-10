package common

import (
	"context"
	"sort"
	"time"
)

type Account interface {
	GetCurrency() string
	GetBalance() float64
	GetAvailable() float64
	GetLocked() float64
	GetTime() time.Time
}

type Balance interface {
	GetCurrency() string
	GetBalance() float64
	GetFree() float64
	GetUsed() float64
	GetTime() time.Time
}

type Position interface {
	GetSymbol() string
	GetSize() float64
	GetPrice() float64
	GetTime() time.Time
}

type Exchange interface {
	Initial(ctx context.Context, settings ExchangeSettings) error
	Start(ctx context.Context)
	SubmitOrder(param NewOrderParam) error
	CancelOrder(param CancelOrderParam) error
	OrderCh() chan Order
	AccountCh() chan Account
	RestartCh() chan interface{}
	StatusCh() chan bool
	Done() chan interface{}
}

type ExchangeSettings struct {
	Proxy         *string  `yaml:"proxy" json:"proxy"`
	ApiKey        *string  `yaml:"apiKey" json:"apiKey"`
	ApiSecret     *string  `yaml:"apiSecret" json:"apiSecret"`
	ApiPassphrase *string  `yaml:"apiPassphrase" json:"apiPassphrase"`
	Symbols       []string `yaml:"symbols" json:"symbols"`
}

type SpotExchange interface {
	Exchange
	BalanceCh() chan Balance
}

type SwapExchange interface {
	Exchange
	PositionCh() chan Position
}

type OrderSide string
type OrderTimeInForce string
type OrderType string
type OrderStatus string

type NewOrderParam struct {
	Symbol      string
	Side        string
	Type        string
	Size        float64
	Price       float64
	PostOnly    bool
	ReduceOnly  bool
	TimeInForce string
	ClientID    string
}

type CancelOrderParam struct {
	Symbol   string
	ClientID string
}

type Order interface {
	GetSymbol() float64
	GetSize() float64
	GetPrice() float64
	GetFilledSize() float64
	GetFilledPrice() float64
	GetSide() float64
	GetClientID() float64
	GetStatus() string
	GetType() string
	GetPostOnly() bool
	GetReduceOnly() bool
}

var OrderSideBuy = OrderSide("BUY")
var OrderSideSell = OrderSide("SELL")
var OrderSideUnknown = OrderSide("ORDER_SIDE_UNKNOWN")

var OrderTypeMarket = OrderType("MARKET")
var OrderTypeLimit = OrderType("LIMIT")
var OrderTypeUnknown = OrderType("ORDER_TYPE_UNKNOWN")

var OrderTimeInForceGTC = OrderTimeInForce("GTC")
var OrderTimeInForceIOC = OrderTimeInForce("IOC")
var OrderTimeInForceFOK = OrderTimeInForce("FOK")

var OrderStatusNew = OrderStatus("NEW")
var OrderStatusOpen = OrderStatus("OPEN")
var OrderStatusPartiallyFilled = OrderStatus("PARTIALLY_FILLED")
var OrderStatusCancelled = OrderStatus("CANCELED")
var OrderStatusPendingCancel = OrderStatus("PENDING_CANCEL")
var OrderStatusReject = OrderStatus("REJECTED")
var OrderStatusExpired = OrderStatus("EXPIRED")
var OrderStatusFilled = OrderStatus("FILLED")
var OrderStatusClosed = OrderStatus("CLOSED") // filled or cancelled or expired or rejected
var OrderStatusUnknown = OrderStatus("ORDER_STATUS_UNKNOWN")

type Bids [][2]float64

func (bids Bids) Len() int {
	return len(bids)
}
func (bids Bids) Swap(i, j int) {
	bids[i], bids[j] = bids[j], bids[i]
}
func (bids Bids) Less(i, j int) bool {
	return bids[i][0] > bids[j][0]
}
func (bids Bids) Search(price float64) int {
	return sort.Search(len(bids), func(i int) bool {
		return bids[i][0] <= price
	})
}
func (bids Bids) SearchAfter(otherBids Bids, n int, price float64) int {
	return sort.Search(len(otherBids)-n, func(i int) bool {
		return otherBids[i+n][0] <= price
	}) + n
}

//otherBids need to be ordered by price descending
func (bids Bids) UpdateBatch(otherBids Bids) Bids {
	n := 0
	for _, bid := range otherBids {
		n = bids.SearchAfter(bids, n, bid[0])
		if bid[1] == 0 {
			if n < len(bids) && bid[0] == bids[n][0] {
				copy(bids[n:], bids[n+1:])
				bids = bids[:len(bids)-1]
			}
		} else {
			if n < len(bids) {
				if bid[0] == bids[n][0] {
					bids[n][1] = bid[1]
				} else {
					bids = append(bids, [2]float64{})
					copy(bids[n+1:], bids[n:])
					bids[n] = bid
				}
			} else {
				bids = append(bids, bid)
			}
		}
	}
	return bids
}

func (bids Bids) Update(bid [2]float64) Bids {
	n := bids.Search(bid[0])
	if bid[1] == 0 {
		if n < len(bids) && bid[0] == bids[n][0] {
			copy(bids[n:], bids[n+1:])
			bids[len(bids)-1] = [2]float64{}
			bids = bids[:len(bids)-1]
		}
	} else {
		if n < len(bids) {
			if bid[0] == bids[n][0] {
				bids[n][1] = bid[1]
			} else {
				bids = append(bids, [2]float64{})
				copy(bids[n+1:], bids[n:])
				bids[n] = bid
			}
		} else {
			bids = append(bids, bid)
		}
	}
	return bids
}

type Asks [][2]float64

func (asks Asks) Len() int {
	return len(asks)
}
func (asks Asks) Swap(i, j int) {
	asks[i], asks[j] = asks[j], asks[i]
}
func (asks Asks) Less(i, j int) bool {
	return asks[i][0] < asks[j][0]
}
func (asks Asks) Search(price float64) int {
	return sort.Search(len(asks), func(i int) bool {
		return asks[i][0] >= price
	})
}

func (asks Asks) SearchAfter(otherAsks Asks, n int, price float64) int {
	return sort.Search(len(otherAsks)-n, func(i int) bool {
		return otherAsks[i+n][0] >= price
	}) + n
}

//otherAsks need to be ordered by price ascending
func (asks Asks) UpdateBatch(otherAsks Asks) Asks {
	n := 0
	for _, ask := range otherAsks {
		n = asks.SearchAfter(asks, n, ask[0])
		if ask[1] == 0 {
			if n < len(asks) && asks[n][0] == ask[0] {
				copy(asks[n:], asks[n+1:])
				asks = asks[:len(asks)-1]
			}
		} else {
			if n < len(asks) {
				if ask[0] == asks[n][0] {
					asks[n][1] = ask[1]
				} else {
					asks = append(asks, [2]float64{})
					copy(asks[n+1:], asks[n:])
					asks[n] = ask
				}
			} else {
				asks = append(asks, ask)
			}
		}
	}
	return asks
}

func (asks Asks) Update(ask [2]float64) Asks {
	n := asks.Search(ask[0])
	if ask[1] == 0 {
		if n < len(asks) && asks[n][0] == ask[0] {
			copy(asks[n:], asks[n+1:])
			asks[len(asks)-1] = [2]float64{}
			asks = asks[:len(asks)-1]
		}
	} else {
		if n < len(asks) {
			if ask[0] == asks[n][0] {
				asks[n][1] = ask[1]
			} else {
				asks = append(asks, [2]float64{})
				copy(asks[n+1:], asks[n:])
				asks[n] = ask
			}
		} else {
			asks = append(asks, ask)
		}
	}
	return asks
}

type Depth interface {
	GetTime() time.Time
	GetAsks() Asks
	GetBids() Bids
	GetSymbol() string
}
