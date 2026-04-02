package dydx_v4_usdfuture

import (
	"encoding/json"
	"github.com/geometrybase/hft-micro/common"
	"strconv"
	"time"
)

// --- WebSocket subscription messages ---

type WSSubscribe struct {
	Type    string `json:"type"`
	Channel string `json:"channel"`
	ID      string `json:"id,omitempty"`
	Batched bool   `json:"batched,omitempty"`
}

// --- REST: Perpetual Markets ---

type PerpetualMarket struct {
	ClobPairID                string `json:"clobPairId"`
	Ticker                    string `json:"ticker"`
	Status                    string `json:"status"`
	OraclePrice               string `json:"oraclePrice"`
	PriceChange24H            string `json:"priceChange24H"`
	Volume24H                 string `json:"volume24H"`
	Trades24H                 int    `json:"trades24h"`
	NextFundingRate           string `json:"nextFundingRate"`
	InitialMarginFraction     string `json:"initialMarginFraction"`
	MaintenanceMarginFraction string `json:"maintenanceMarginFraction"`
	OpenInterest              string `json:"openInterest"`
	AtomicResolution          int    `json:"atomicResolution"`
	QuantumConversionExponent int    `json:"quantumConversionExponent"`
	TickSize                  string `json:"tickSize"`
	StepSize                  string `json:"stepSize"`
	StepBaseQuantums          int    `json:"stepBaseQuantums"`
	SubticksPerTick           int    `json:"subticksPerTick"`
}

func (m *PerpetualMarket) TickSizeFloat() float64 {
	v, _ := strconv.ParseFloat(m.TickSize, 64)
	return v
}

func (m *PerpetualMarket) StepSizeFloat() float64 {
	v, _ := strconv.ParseFloat(m.StepSize, 64)
	return v
}

func (m *PerpetualMarket) OraclePriceFloat() float64 {
	v, _ := strconv.ParseFloat(m.OraclePrice, 64)
	return v
}

func (m *PerpetualMarket) NextFundingRateFloat() float64 {
	v, _ := strconv.ParseFloat(m.NextFundingRate, 64)
	return v
}

type PerpetualMarketsResp struct {
	Markets map[string]PerpetualMarket `json:"markets"`
}

// --- FundingRate ---

type V4FundingRate struct {
	Symbol          string
	FundingRate     float64
	NextFundingTime time.Time
}

func (f *V4FundingRate) GetSymbol() string              { return f.Symbol }
func (f *V4FundingRate) GetFundingRate() float64         { return f.FundingRate }
func (f *V4FundingRate) GetNextFundingTime() time.Time   { return f.NextFundingTime }
func (f *V4FundingRate) GetExchange() common.ExchangeID  { return DydxV4UsdFutureExchangeID }

// --- REST: Subaccount (balance/equity) ---

type SubaccountResp struct {
	Subaccount Subaccount `json:"subaccount"`
}

type Subaccount struct {
	Address                string                  `json:"address"`
	SubaccountNumber       int                     `json:"subaccountNumber"`
	Equity                 string                  `json:"equity"`
	FreeCollateral         string                  `json:"freeCollateral"`
	OpenPerpetualPositions map[string]V4Position    `json:"openPerpetualPositions"`
	AssetPositions         map[string]AssetPosition `json:"assetPositions"`
	MarginEnabled          bool                    `json:"marginEnabled"`
	ParseTime              time.Time               `json:"-"`
}

func (a *Subaccount) UnmarshalJSON(data []byte) error {
	type Alias Subaccount
	aux := &struct{ *Alias }{Alias: (*Alias)(a)}
	err := json.Unmarshal(data, aux)
	if err != nil {
		return err
	}
	a.ParseTime = time.Now()
	return nil
}

func (a *Subaccount) EquityFloat() float64 {
	v, _ := strconv.ParseFloat(a.Equity, 64)
	return v
}

func (a *Subaccount) FreeCollateralFloat() float64 {
	v, _ := strconv.ParseFloat(a.FreeCollateral, 64)
	return v
}

func (a *Subaccount) GetCurrency() string              { return "USDC" }
func (a *Subaccount) GetBalance() float64               { return a.EquityFloat() }
func (a *Subaccount) GetFree() float64                  { return a.FreeCollateralFloat() }
func (a *Subaccount) GetUsed() float64                  { return a.EquityFloat() - a.FreeCollateralFloat() }
func (a *Subaccount) GetTime() time.Time                { return a.ParseTime }
func (a *Subaccount) GetExchange() common.ExchangeID    { return DydxV4UsdFutureExchangeID }

type AssetPosition struct {
	Symbol  string `json:"symbol"`
	Side    string `json:"side"`
	Size    string `json:"size"`
	AssetID string `json:"assetId"`
}

// --- REST: Perpetual Position ---

type V4Position struct {
	Market        string `json:"market"`
	Status        string `json:"status"`
	Side          string `json:"side"`
	Size          string `json:"size"`
	MaxSize       string `json:"maxSize"`
	EntryPrice    string `json:"entryPrice"`
	ExitPrice     string `json:"exitPrice"`
	RealizedPnl   string `json:"realizedPnl"`
	UnrealizedPnl string `json:"unrealizedPnl"`
	CreatedAt     string `json:"createdAt"`
	ClosedAt      string `json:"closedAt"`
	SumOpen       string `json:"sumOpen"`
	SumClose      string `json:"sumClose"`
	NetFunding    string `json:"netFunding"`
	ParseTime     time.Time `json:"-"`
}

func (p *V4Position) UnmarshalJSON(data []byte) error {
	type Alias V4Position
	aux := &struct{ *Alias }{Alias: (*Alias)(p)}
	err := json.Unmarshal(data, aux)
	if err != nil {
		return err
	}
	p.ParseTime = time.Now()
	return nil
}

func (p *V4Position) SizeFloat() float64 {
	v, _ := strconv.ParseFloat(p.Size, 64)
	if p.Side == "SHORT" {
		return -v
	}
	return v
}

func (p *V4Position) EntryPriceFloat() float64 {
	v, _ := strconv.ParseFloat(p.EntryPrice, 64)
	return v
}

func (p *V4Position) GetSymbol() string              { return p.Market }
func (p *V4Position) GetSize() float64               { return p.SizeFloat() }
func (p *V4Position) GetPrice() float64              { return p.EntryPriceFloat() }
func (p *V4Position) GetEventTime() time.Time        { return p.ParseTime }
func (p *V4Position) GetParseTime() time.Time        { return p.ParseTime }
func (p *V4Position) GetExchange() common.ExchangeID { return DydxV4UsdFutureExchangeID }

// --- REST: Order ---

type V4Order struct {
	ID                string `json:"id"`
	SubaccountID      string `json:"subaccountId"`
	ClientID          string `json:"clientId"`
	ClobPairID        string `json:"clobPairId"`
	Side              string `json:"side"`
	Size              string `json:"size"`
	TotalFilled       string `json:"totalFilled"`
	Price             string `json:"price"`
	Type              string `json:"type"`
	ReduceOnly        bool   `json:"reduceOnly"`
	OrderFlags        string `json:"orderFlags"`
	GoodTilBlock      string `json:"goodTilBlock"`
	GoodTilBlockTime  string `json:"goodTilBlockTime"`
	CreatedAtHeight   string `json:"createdAtHeight"`
	ClientMetadata    string `json:"clientMetadata"`
	TriggerPrice      string `json:"triggerPrice"`
	TimeInForce       string `json:"timeInForce"`
	Status            string `json:"status"`
	PostOnly          bool   `json:"postOnly"`
	Ticker            string `json:"ticker"`
	RemainderQuantums string `json:"remainderQuantums"`
	UpdatedAt         string `json:"updatedAt"`
	UpdatedAtHeight   string `json:"updatedAtHeight"`
}

func (o *V4Order) SizeFloat() float64 {
	v, _ := strconv.ParseFloat(o.Size, 64)
	return v
}

func (o *V4Order) PriceFloat() float64 {
	v, _ := strconv.ParseFloat(o.Price, 64)
	return v
}

func (o *V4Order) TotalFilledFloat() float64 {
	v, _ := strconv.ParseFloat(o.TotalFilled, 64)
	return v
}

func (o *V4Order) GetSymbol() string      { return o.Ticker }
func (o *V4Order) GetSize() float64       { return o.SizeFloat() }
func (o *V4Order) GetPrice() float64      { return o.PriceFloat() }
func (o *V4Order) GetFilledSize() float64  { return o.TotalFilledFloat() }
func (o *V4Order) GetFilledPrice() float64 { return o.PriceFloat() }

func (o *V4Order) GetSide() common.OrderSide {
	switch o.Side {
	case OrderSideBuy:
		return common.OrderSideBuy
	case OrderSideSell:
		return common.OrderSideSell
	default:
		return common.OrderSideUnknown
	}
}

func (o *V4Order) GetClientID() string { return o.ClientID }
func (o *V4Order) GetID() string       { return o.ID }

func (o *V4Order) GetStatus() common.OrderStatus {
	switch o.Status {
	case OrderStatusCanceled, OrderStatusBestEffortCanceled:
		return common.OrderStatusCancelled
	case OrderStatusFilled:
		return common.OrderStatusFilled
	case OrderStatusOpen, OrderStatusBestEffortOpened:
		if o.TotalFilledFloat() > 0 {
			return common.OrderStatusPartiallyFilled
		}
		return common.OrderStatusOpen
	case OrderStatusPending:
		return common.OrderStatusNew
	default:
		return common.OrderStatusUnknown
	}
}

func (o *V4Order) GetType() common.OrderType {
	switch o.Type {
	case OrderTypeMarket:
		return common.OrderTypeMarket
	default:
		return common.OrderTypeLimit
	}
}

func (o *V4Order) GetPostOnly() bool               { return o.PostOnly }
func (o *V4Order) GetReduceOnly() bool              { return o.ReduceOnly }
func (o *V4Order) GetExchange() common.ExchangeID   { return DydxV4UsdFutureExchangeID }

// --- REST: Orderbook ---

type OrderbookResp struct {
	Asks []OrderbookLevel `json:"asks"`
	Bids []OrderbookLevel `json:"bids"`
}

type OrderbookLevel struct {
	Price string `json:"price"`
	Size  string `json:"size"`
}

// --- Depth type for internal use ---

type V4Depth struct {
	Bids             common.Bids
	Asks             common.Asks
	Market           string
	ParseTime        time.Time
	WithSnapshotData bool
}

func (d *V4Depth) GetParseTime() time.Time          { return d.ParseTime }
func (d *V4Depth) GetEventTime() time.Time           { return d.ParseTime }
func (d *V4Depth) GetAsks() common.Asks              { return d.Asks[:] }
func (d *V4Depth) GetBids() common.Bids              { return d.Bids[:] }
func (d *V4Depth) GetSymbol() string                 { return d.Market }
func (d *V4Depth) GetExchange() common.ExchangeID    { return DydxV4UsdFutureExchangeID }

func (d *V4Depth) GetBidPrice() float64 {
	if len(d.Bids) > 0 {
		return d.Bids[0][0]
	}
	return 0.0
}
func (d *V4Depth) GetAskPrice() float64 {
	if len(d.Asks) > 0 {
		return d.Asks[0][0]
	}
	return 0.0
}
func (d *V4Depth) GetBidSize() float64 {
	if len(d.Bids) > 0 {
		return d.Bids[0][1]
	}
	return 0.0
}
func (d *V4Depth) GetAskSize() float64 {
	if len(d.Asks) > 0 {
		return d.Asks[0][1]
	}
	return 0.0
}
func (d *V4Depth) GetBidOffset() float64 {
	if len(d.Bids) > 0 && len(d.Asks) > 0 && d.Bids[0][0] != 0 {
		return (d.Asks[0][0] - d.Bids[0][0]) * 0.5 / d.Bids[0][0]
	}
	return common.DefaultBidAskOffset
}
func (d *V4Depth) GetAskOffset() float64 {
	if len(d.Bids) > 0 && len(d.Asks) > 0 && d.Asks[0][0] != 0 {
		return (d.Asks[0][0] - d.Bids[0][0]) * 0.5 / d.Asks[0][0]
	}
	return common.DefaultBidAskOffset
}
func (d *V4Depth) IsValid() bool {
	if !d.WithSnapshotData || len(d.Asks) == 0 || len(d.Bids) == 0 || d.Asks[0][0] < d.Bids[0][0] {
		return false
	}
	return true
}

// --- V4 Ticker (derived from depth BBO) ---

type V4Ticker struct {
	Symbol     string
	BidPrice   float64
	AskPrice   float64
	BidSize    float64
	AskSize    float64
	EventTime  time.Time
	ParseTime_ time.Time
}

func (t *V4Ticker) GetBidPrice() float64            { return t.BidPrice }
func (t *V4Ticker) GetAskPrice() float64            { return t.AskPrice }
func (t *V4Ticker) GetBidSize() float64             { return t.BidSize }
func (t *V4Ticker) GetAskSize() float64             { return t.AskSize }
func (t *V4Ticker) GetBidOffset() float64 {
	if t.BidPrice != 0 {
		return (t.AskPrice - t.BidPrice) * 0.5 / t.BidPrice
	}
	return common.DefaultBidAskOffset
}
func (t *V4Ticker) GetAskOffset() float64 {
	if t.AskPrice != 0 {
		return (t.AskPrice - t.BidPrice) * 0.5 / t.AskPrice
	}
	return common.DefaultBidAskOffset
}
func (t *V4Ticker) GetSymbol() string              { return t.Symbol }
func (t *V4Ticker) GetEventTime() time.Time        { return t.EventTime }
func (t *V4Ticker) GetParseTime() time.Time        { return t.ParseTime_ }
func (t *V4Ticker) GetExchange() common.ExchangeID { return DydxV4UsdFutureExchangeID }

// --- V4 Trade ---

type V4Trade struct {
	Symbol    string
	Price     float64
	Size      float64
	Side      string
	CreatedAt time.Time
}

func (t *V4Trade) GetPrice() float64  { return t.Price }
func (t *V4Trade) GetSize() float64   { return t.Size }
func (t *V4Trade) GetTime() time.Time { return t.CreatedAt }
func (t *V4Trade) IsUpTick() bool     { return t.Side == OrderSideBuy }
func (t *V4Trade) GetSymbol() string  { return t.Symbol }

// --- WebSocket response envelopes ---

type WSMessage struct {
	Type         string          `json:"type"`
	ConnectionID string          `json:"connection_id"`
	MessageID    int             `json:"message_id"`
	Channel      string          `json:"channel"`
	ID           string          `json:"id"`
	Contents     json.RawMessage `json:"contents"`
	Message      string          `json:"message"`
}

// --- WS Orderbook ---

type WSOrderbookSnapshot struct {
	Asks []WSOrderbookLevel `json:"asks"`
	Bids []WSOrderbookLevel `json:"bids"`
}

type WSOrderbookLevel struct {
	Price string `json:"price"`
	Size  string `json:"size"`
}

type WSOrderbookUpdate struct {
	Asks []WSOrderbookLevel `json:"asks"`
	Bids []WSOrderbookLevel `json:"bids"`
}

// --- WS Trade ---

type WSTradeUpdate struct {
	Trades []WSTradeEntry `json:"trades"`
}

type WSTradeEntry struct {
	ID        string `json:"id"`
	Side      string `json:"side"`
	Size      string `json:"size"`
	Price     string `json:"price"`
	Type      string `json:"type"`
	CreatedAt string `json:"createdAt"`
}

// --- WS Subaccount ---

type WSSubaccountUpdate struct {
	Orders    []V4Order    `json:"orders"`
	Positions []V4Position `json:"perpetualPositions"`
	Fills     []V4Fill     `json:"fills"`
	Subaccount *Subaccount `json:"subaccount"`
}

type WSSubaccountSubscribed struct {
	Subaccount Subaccount `json:"subaccount"`
	Orders     []V4Order  `json:"orders"`
}

type V4Fill struct {
	ID        string `json:"id"`
	Side      string `json:"side"`
	Liquidity string `json:"liquidity"`
	Type      string `json:"type"`
	Market    string `json:"market"`
	Price     string `json:"price"`
	Size      string `json:"size"`
	Fee       string `json:"fee"`
	CreatedAt string `json:"createdAt"`
	OrderID   string `json:"orderId"`
}

// --- WS Markets channel ---

type WSMarketsSubscribed struct {
	Markets map[string]WSMarketData `json:"markets"`
}

type WSMarketsUpdate struct {
	Trading map[string]WSMarketUpdate `json:"trading"`
	Oracle  map[string]WSMarketUpdate `json:"oraclePrice"`
}

type WSMarketData struct {
	OraclePrice     string `json:"oraclePrice"`
	NextFundingRate string `json:"nextFundingRate"`
	Ticker          string `json:"ticker"`
}

type WSMarketUpdate struct {
	OraclePrice     string `json:"oraclePrice,omitempty"`
	NextFundingRate string `json:"nextFundingRate,omitempty"`
}

// --- Historical Funding ---

type HistoricalFundingEntry struct {
	Ticker            string `json:"ticker"`
	Rate              string `json:"rate"`
	Price             string `json:"price"`
	EffectiveAt       string `json:"effectiveAt"`
	EffectiveAtHeight string `json:"effectiveAtHeight"`
}

type HistoricalFundingResp struct {
	HistoricalFunding []HistoricalFundingEntry `json:"historicalFunding"`
}
