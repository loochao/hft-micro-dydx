package common

import (
	"context"
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
	GetAvailable() float64
	GetLocked() float64
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
	GetSide() float64
	GetClientID() float64
	GetType() string
	GetPostOnly() bool
	GetReduceOnly() bool
}

var OrderSideBuy = OrderSide("BUY")
var OrderSideSell = OrderSide("SELL")
var OrderTypeMarket = OrderType("MARKET")
var OrderTypeLimit = OrderType("LIMIT")
var OrderTimeInForceGTC = OrderTimeInForce("GTC")
var OrderTimeInForceIOC = OrderTimeInForce("IOC")
var OrderTimeInForceFOK = OrderTimeInForce("FOK")
