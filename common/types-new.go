package common

import (
	"context"
	"sort"
	"time"
)

type Balance interface {
	GetCurrency() string
	GetBalance() float64
	GetFree() float64
	GetUsed() float64
	GetTime() time.Time
	GetExchange() ExchangeID
}

type Position interface {
	GetSymbol() string
	GetSize() float64
	GetPrice() float64
	GetEventTime() time.Time
	GetParseTime() time.Time
	GetExchange() ExchangeID
}

type CoinExchange interface {
	Done() chan interface{}
	Stop()

	Setup(ctx context.Context, settings ExchangeSettings) error

	GetMinNotional(symbol string) (float64, error)
	GetMinSize(symbol string) (float64, error)
	GetStepSize(symbol string) (float64, error)
	GetTickSize(symbol string) (float64, error)
	GetMultiplier(symbol string) (float64, error)

	StreamBasic(ctx context.Context, statusCh chan SystemStatus, balanceChMap map[string]chan Balance, positionChMap map[string]chan Position, orderCh map[string]chan Order, )
	StreamSymbolStatus(ctx context.Context, channels map[string]chan SymbolStatusMsg, batchSize int)
	StreamDepth(ctx context.Context, channels map[string]chan Depth, batchSize int)
	StreamTrade(ctx context.Context, channels map[string]chan Trade, batchSize int)
	StreamTicker(ctx context.Context, channels map[string]chan Ticker, batchSize int)
	StreamKLine(ctx context.Context, channels map[string]chan []KLine, batchSize int, interval, lookback time.Duration)
	StreamFundingRate(ctx context.Context, channels map[string]chan FundingRate, batchSize int)

	GetExchange() ExchangeID

	WatchOrders(ctx context.Context, requestChannels map[string]chan OrderRequest, responseChannels map[string]chan Order, errorChannels map[string]chan OrderError, )
	//WatchBatchOrders(ctx context.Context, requestChannels map[string]chan BatchOrderRequest, responseChannels map[string]chan Order, errorChannels map[string]chan OrderError, )
	GenerateClientID() string
	IsSpot() bool
}

type UsdExchange interface {
	Done() chan interface{}
	Stop()

	GetExchange() ExchangeID
	Setup(ctx context.Context, settings ExchangeSettings) error

	GetMinNotional(symbol string) (float64, error)
	GetMinSize(symbol string) (float64, error)
	GetStepSize(symbol string) (float64, error)
	GetTickSize(symbol string) (float64, error)
	GetMultiplier(symbol string) (float64, error)

	StreamBasic(ctx context.Context, statusCh chan SystemStatus, accountCh chan Balance, commissionAssetValueCh chan float64, positionChMap map[string]chan Position, orderCh map[string]chan Order, )
	StreamSystemStatus(ctx context.Context, statusCh chan SystemStatus)
	StreamSymbolStatus(ctx context.Context, channels map[string]chan SymbolStatusMsg, batchSize int)
	StreamDepth(ctx context.Context, channels map[string]chan Depth, batchSize int)
	StreamTrade(ctx context.Context, channels map[string]chan Trade, batchSize int)
	StreamTicker(ctx context.Context, channels map[string]chan Ticker, batchSize int)
	StreamKLine(ctx context.Context, channels map[string]chan []KLine, batchSize int, interval, lookback time.Duration)
	StreamFundingRate(ctx context.Context, channels map[string]chan FundingRate, batchSize int)

	WatchOrders(ctx context.Context, requestChannels map[string]chan OrderRequest, responseChannels map[string]chan Order, errorChannels map[string]chan OrderError, )
	WatchBatchOrders(ctx context.Context, requestChannels map[string]chan BatchOrderRequest, responseChannels map[string]chan Order, errorChannels map[string]chan OrderError, )
	GenerateClientID() string
	IsSpot() bool
	StartSideLoop()
}

type OrderError struct {
	New    *NewOrderParam
	Cancel *CancelOrderParam
	Error  error
}

type FundingRate interface {
	GetSymbol() string
	GetFundingRate() float64
	GetNextFundingTime() time.Time
	GetExchange() ExchangeID
}

type Ticker interface {
	GetSymbol() string
	GetTime() time.Time
	GetBidPrice() float64
	GetAskPrice() float64
	GetBidSize() float64
	GetAskSize() float64
	GetExchange() ExchangeID
}

type ExchangeSettings struct {
	Name                                string        `yaml:"name" json:"name"`
	DryRun                              bool          `yaml:"-" json:"-"`
	Proxy                               string        `yaml:"proxy" json:"proxy"`
	ApiKey                              string        `yaml:"apiKey" json:"apiKey"`
	ApiSecret                           string        `yaml:"apiSecret" json:"apiSecret"`
	ApiPassphrase                       string        `yaml:"apiPassphrase" json:"apiPassphrase"`
	ApiSubAccount                       string        `yaml:"apiSubAccount" json:"apiSubAccount"`
	ApiUrl                              string        `yaml:"apiUrl" json:"apiUrl"`
	Symbols                             []string      `yaml:"symbols" json:"symbols"`
	PullInterval                        time.Duration `yaml:"pullInterval" json:"httpPullInterval"`
	HttpRequestInterval                 time.Duration `yaml:"httpRequestInterval" json:"httpRequestInterval"`
	MarginType                          string        `yaml:"marginType" json:"marginType"`
	ChangeMarginType                    bool          `yaml:"changeMarginType" json:"changeMarginType"`
	Leverage                            float64       `yaml:"leverage" json:"leverage"`
	ChangeLeverage                      bool          `yaml:"changeLeverage" json:"changeLeverage"`
	AutoAddCommissionDiscountAsset      bool          `yaml:"autoAddCommissionDiscountAsset" json:"autoAddCommissionDiscountAsset"`
	MinimalCommissionDiscountAssetValue float64       `yaml:"minimalCommissionDiscountAssetValue" json:"minimalCommissionDiscountAsset"`
}

type SpotExchange interface {
	UsdExchange
	BalanceCh() chan Balance
}

type SwapExchange interface {
	UsdExchange
	PositionCh() chan Position
}

type OrderSide string
type OrderTimeInForce string
type OrderType string
type OrderStatus string

type NewOrderParam struct {
	Symbol      string
	Side        OrderSide
	Type        OrderType
	Size        float64
	Price       float64
	PostOnly    bool
	ReduceOnly  bool
	TimeInForce OrderTimeInForce
	ClientID    string
}

type CancelOrderParam struct {
	Symbol   string
	ClientID string
}

type OrderRequest struct {
	New    *NewOrderParam
	Cancel *CancelOrderParam
}

type BatchOrderRequest struct {
	New    []NewOrderParam
	Cancel *CancelOrderParam
}

type Order interface {
	GetSymbol() string
	GetSize() float64
	GetPrice() float64
	GetFilledSize() float64
	GetFilledPrice() float64
	GetSide() OrderSide
	GetClientID() string
	GetID() string
	GetStatus() OrderStatus
	GetType() OrderType
	GetPostOnly() bool
	GetReduceOnly() bool
	GetExchange() ExchangeID
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

//var OrderStatusClosed = OrderStatus("CLOSED") // filled or cancelled or expired or rejected
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
	GetExchange() ExchangeID
}

type SystemStatus string

var (
	SystemStatusNotReady = SystemStatus("NOTREADY")
	SystemStatusReady    = SystemStatus("READY")
	SystemStatusRestart  = SystemStatus("RESTART")
	SystemStatusClosed   = SystemStatus("CLOSED")
	SystemStatusError    = SystemStatus("ERROR")
)

type SymbolStatusMsg string
type SymbolStatus struct {
	Symbol string
	Status SymbolStatusMsg
}

var (
	SymbolStatusReady    = SymbolStatusMsg("READY")
	SymbolStatusNotReady = SymbolStatusMsg("NOT_READY")
)

type InfluxSettings struct {
	Address      string        `yaml:"address"`
	Username     string        `yaml:"username"`
	Password     string        `yaml:"password"`
	Database     string        `yaml:"database"`
	Measurement  string        `yaml:"measurement"`
	BatchSize    int           `yaml:"batchSize"`
	SaveInterval time.Duration `yaml:"saveInterval"`
}

var (
	TickSizeNotFoundError     = "tick size for %s not found"
	StepSizeNotFoundError     = "step size for %s not found"
	MinSizeNotFoundError      = "min size for %s not found"
	MinNotionalNotFoundError  = "min notional for %s not found"
	ContractSizeNotFoundError = "contract size for %s not found"
	MultiplierNotFoundError   = "multipliers for %s not found"
)
