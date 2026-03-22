package bybit_usdtfuture

import (
	"encoding/json"
	"github.com/geometrybase/hft-micro/common"
	"time"
)

// {
//      "data": {
//        "user_id": 6265365,
//        "symbol": "AAVEUSDT",
//        "side": "Buy",
//        "size": 0,
//        "position_value": 0,
//        "entry_price": 0,
//        "liq_price": 0,
//        "bust_price": 0,
//        "leverage": 25,
//        "auto_add_margin": 0,
//        "is_isolated": false,
//        "position_margin": 0,
//        "occ_closing_fee": 0,
//        "realised_pnl": 0,
//        "cum_realised_pnl": 0,
//        "free_qty": 0,
//        "tp_sl_mode": "Full",
//        "unrealised_pnl": 0,
//        "deleverage_indicator": 0,
//        "risk_id": 191,
//        "stop_loss": 0,
//        "take_profit": 0,
//        "trailing_stop": 0
//      },
//      "is_valid": true
// }

type WSPosition struct {
	UserID              string    `json:"user_id"`
	Symbol              string    `json:"symbol"`
	Side                string    `json:"side"`
	Size                float64   `json:"size"`
	PositionValue       float64   `json:"position_value"`
	EntryPrice          float64   `json:"entry_price"`
	LiqPrice            float64   `json:"liq_price"`
	BustPrice           float64   `json:"bust_price"`
	Leverage            float64   `json:"leverage"`
	AutoAddMargin       float64   `json:"auto_add_margin,string"`
	IsIsolated          bool      `json:"is_isolated"`
	PositionMargin      float64   `json:"position_margin"`
	OccClosingFee       float64   `json:"occ_closing_fee"`
	RealisedPnl         float64   `json:"realised_pnl"`
	CumRealisedPnl      float64   `json:"cum_realised_pnl"`
	FreeQty             float64   `json:"free_qty"`
	TpSlMode            string    `json:"tp_sl_mode"`
	UnrealisedPnl       float64   `json:"unrealised_pnl"`
	DeleverageIndicator float64   `json:"deleverage_indicator"`
	RiskID              float64   `json:"risk_id,string"`
	StopLoss            float64   `json:"stop_loss"`
	TakeProfit          float64   `json:"take_profit"`
	TrailingStop        float64   `json:"trailing_stop"`
	ParseTime           time.Time `json:"-"`
}

func (pos *WSPosition) UnmarshalJSON(data []byte) error {
	type Alias WSPosition
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(pos),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	} else {
		pos.ParseTime = time.Now()
		return nil
	}
}

type Position struct {
	UserID              int64     `json:"user_id"`
	Symbol              string    `json:"symbol"`
	Side                string    `json:"side"`
	Size                float64   `json:"size"`
	PositionValue       float64   `json:"position_value"`
	EntryPrice          float64   `json:"entry_price"`
	LiqPrice            float64   `json:"liq_price"`
	BustPrice           float64   `json:"bust_price"`
	Leverage            float64   `json:"leverage"`
	AutoAddMargin       float64   `json:"auto_add_margin"`
	IsIsolated          bool      `json:"is_isolated"`
	PositionMargin      float64   `json:"position_margin"`
	OccClosingFee       float64   `json:"occ_closing_fee"`
	RealisedPnl         float64   `json:"realised_pnl"`
	CumRealisedPnl      float64   `json:"cum_realised_pnl"`
	FreeQty             float64   `json:"free_qty"`
	TpSlMode            string    `json:"tp_sl_mode"`
	UnrealisedPnl       float64   `json:"unrealised_pnl"`
	DeleverageIndicator float64   `json:"deleverage_indicator"`
	RiskID              float64   `json:"risk_id"`
	StopLoss            float64   `json:"stop_loss"`
	TakeProfit          float64   `json:"take_profit"`
	TrailingStop        float64   `json:"trailing_stop"`
	ParseTime           time.Time `json:"-"`
}

func (pos *Position) UnmarshalJSON(data []byte) error {
	type Alias Position
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(pos),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	} else {
		pos.ParseTime = time.Now()
		return nil
	}
}

type PositionData struct {
	Data    Position `json:"data"`
	IsValid bool     `json:"is_valid"`
}

type PriceFilter struct {
	MinPrice float64 `json:"min_price,string"`
	MaxPrice float64 `json:"max_price,string"`
	TickSize float64 `json:"tick_size,string"`
}

type LotSizeFilter struct {
	MaxTradingQty float64 `json:"max_trading_qty"`
	MinTradingQty float64 `json:"min_trading_qty"`
	QtyStep       float64 `json:"qty_step"`
}

type LeverageFilter struct {
	MinLeverage  float64 `json:"min_leverage"`
	MaxLeverage  float64 `json:"max_leverage"`
	LeverageStep float64 `json:"leverage_step,string"`
}

type Symbol struct {
	Name           string         `json:"name"`
	Alias          string         `json:"alias"`
	Status         string         `json:"status"`
	BaseCurrency   string         `json:"base_currency"`
	QuoteCurrency  string         `json:"quote_currency"`
	PriceScale     float64        `json:"price_scale"`
	TakerFee       float64        `json:"taker_fee,string"`
	MakerFee       float64        `json:"maker_fee,string"`
	LeverageFilter LeverageFilter `json:"leverage_filter"`
	PriceFilter    PriceFilter    `json:"price_filter"`
	LotSizeFilter  LotSizeFilter  `json:"lot_size_filter"`
}

type ResponseCap struct {
	RetCode int64           `json:"ret_code"`
	RetMsg  string          `json:"ret_msg"`
	ExtCode string          `json:"ext_code"`
	ExtInfo string          `json:"ext_info"`
	TimeNow float64         `json:"time_now,string"`
	Result  json.RawMessage `json:"result"`
}

type OrderBookLevel struct {
	Price  float64 `json:"price,string"`
	Symbol string  `json:"symbol"`
	ID     int64   `json:"id,string"`
	Side   string  `json:"side"`
	Size   float64 `json:"size"`
}

type OrderBookData struct {
	OrderBook      []OrderBookLevel `json:"order_book"`
	Update         []OrderBookLevel `json:"update"`
	Insert         []OrderBookLevel `json:"insert"`
	Delete         []OrderBookLevel `json:"delete"`
	TransactTimeE6 int64            `json:"transactTimeE6"`
}

type OrderBookMsg struct {
	Topic       string        `json:"topic"`
	Type        string        `json:"type"`
	Data        OrderBookData `json:"data"`
	CrossSeq    int64         `json:"cross_seq,string"`
	TimestampE6 int64         `json:"timestamp_e6,string"`
}

type OrderBook struct {
	Bids      common.Bids
	Asks      common.Asks
	EventTime time.Time
	ParseTime time.Time
	bid       [2]float64
	ask       [2]float64
	Symbol    string
}

func (o OrderBook) GetBidOffset() float64 {
	if tickSize, ok := TickSizes[o.Symbol]; ok && len(o.Bids) > 0 && o.Bids[0][0] != 0{
		return tickSize*0.5/o.Bids[0][0]
	}else{
		return 1.0
	}
}

func (o OrderBook) GetAskOffset() float64 {
	if tickSize, ok := TickSizes[o.Symbol]; ok && len(o.Asks) > 0 && o.Asks[0][0] != 0{
		return tickSize*0.5/o.Asks[0][0]
	}else{
		return 1.0
	}
}

func (o OrderBook) GetBidPrice() float64 {
	return o.Bids[0][0]
}

func (o OrderBook) GetAskPrice() float64 {
	return o.Asks[0][0]
}

func (o OrderBook) GetBidSize() float64 {
	return o.Bids[0][1]
}

func (o OrderBook) GetAskSize() float64 {
	return o.Asks[0][1]
}

func (o OrderBook) IsValidate() bool {
	if len(o.Asks) > 25 || len(o.Bids) > 25 {
		return false
	}
	if len(o.Asks) > 0 && len(o.Bids) > 0 && o.Asks[0][0] <= o.Bids[0][0] {
		return false
	}
	//for i := 0; i < len(o.Asks)-1; i++ {
	//	if o.Asks[i][0] >= o.Asks[i+1][0] {
	//		return false
	//	}
	//}
	//for i := 0; i < len(o.Bids)-1; i++ {
	//	if o.Bids[i][0] <= o.Bids[i+1][0] {
	//		return false
	//	}
	//}
	return true
}

func (o OrderBook) GetEventTime() time.Time {
	return o.EventTime
}

func (o OrderBook) GetAsks() common.Asks {
	return o.Asks
}

func (o OrderBook) GetBids() common.Bids {
	return o.Bids
}

func (o OrderBook) GetSymbol() string {
	return o.Symbol
}

func (o OrderBook) GetExchange() common.ExchangeID {
	return ExchangeID
}

type SubscribeParam struct {
	Op   string   `json:"op"`
	Args []string `json:"args"`
}

//{
//"symbol":"BTCUSDT",
//"funding_rate":-0.00005965,
//"funding_rate_timestamp":"2020-04-07T08:00:00.000Z"
//}

type FundingRate struct {
	Symbol      string    `json:"symbol"`
	FundingRate float64   `json:"funding_rate"`
	Timestamp   time.Time `json:"timestamp"`
}

func (fr *FundingRate) GetSymbol() string {
	return fr.Symbol
}

func (fr *FundingRate) GetFundingRate() float64 {
	return fr.FundingRate
}

func (fr *FundingRate) GetNextFundingTime() time.Time {
	return fr.Timestamp
}

func (fr *FundingRate) GetExchange() common.ExchangeID {
	panic("implement me")
}

func (fr *FundingRate) UnmarshalJSON(data []byte) error {
	type Alias FundingRate
	aux := &struct {
		Timestamp string `json:"funding_rate_timestamp"`
		*Alias
	}{
		Alias: (*Alias)(fr),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	} else {
		//"2020-04-07T08:00:00.000Z"
		fr.Timestamp, err = time.Parse("2006-01-02T15:04:05.999999999Z", aux.Timestamp)
		if err != nil {
			return err
		} else {
			return nil
		}
	}
}

//{
//      "equity": 0,
//      "available_balance": 0,
//      "used_margin": 0,
//      "order_margin": 0,
//      "position_margin": 0,
//      "occ_closing_fee": 0,
//      "occ_funding_fee": 0,
//      "wallet_balance": 0,
//      "realised_pnl": 0,
//      "unrealised_pnl": 0,
//      "cum_realised_pnl": 0,
//      "given_cash": 0,
//      "service_cash": 0
//    }
type Balance struct {
	Equity           float64   `json:"equity"`
	AvailableBalance float64   `json:"available_balance"`
	UsedMargin       float64   `json:"used_margin"`
	OrderMargin      float64   `json:"order_margin"`
	PositionMargin   float64   `json:"position_margin"`
	OccClosingFee    float64   `json:"occ_closing_fee"`
	OccFundingFee    float64   `json:"occ_funding_fee"`
	WalletBalance    float64   `json:"wallet_balance"`
	RealisedPnl      float64   `json:"realised_pnl"`
	UnrealisedPnl    float64   `json:"unrealised_pnl"`
	CumRealisedPnl   float64   `json:"cum_realised_pnl"`
	GivenCash        float64   `json:"given_cash"`
	ServiceCash      float64   `json:"service_cash"`
	Timestamp        time.Time `json:"-"`
}

func (b *Balance) GetCurrency() string {
	return "USDT"
}

func (b *Balance) GetBalance() float64 {
	return b.Equity
}

func (b *Balance) GetFree() float64 {
	return b.AvailableBalance
}

func (b *Balance) GetUsed() float64 {
	return b.UsedMargin
}

func (b *Balance) GetTime() time.Time {
	return b.Timestamp
}

func (b *Balance) GetExchange() common.ExchangeID {
	return ExchangeID
}

func (b *Balance) UnmarshalJSON(data []byte) error {
	type Alias Balance
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(b),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	} else {
		b.Timestamp = time.Now()
		return nil
	}
}

//{
//        "order_id":"bd1844f-f3c0-4e10-8c25-10fea03763f6",
//        "user_id": 1,
//        "symbol": "BTCUSDT",
//        "side": "Sell",
//        "order_type": "Limit",
//        "price": 8083,
//        "qty": 10,
//        "time_in_force": "GoodTillCancel",
//        "order_status": "New",
//        "last_exec_price": 8083,    //Last execution price
//        "cum_exec_qty": 0,          //Cumulative qty of trading
//        "cum_exec_value": 0,        //Cumulative value of trading
//        "cum_exec_fee": 0,          //Cumulative trading fees
//        "reduce_only": false,       //true means close order, false means open position
//        "close_on_trigger": false
//        "order_link_id": "",
//        "created_time": "2019-10-21T07:28:19.396246Z",
//        "updated_time": "2019-10-21T07:28:19.396246Z",
//    }
type Order struct {
	OrderID        string    `json:"order_id"`
	UserID         int64     `json:"user_id"`
	Symbol         string    `json:"symbol"`
	Side           string    `json:"side"`
	OrderType      string    `json:"order_type"`
	Price          float64   `json:"price"`
	Qty            float64   `json:"qty"`
	TimeInForce    string    `json:"time_in_force"`
	OrderStatus    string    `json:"order_status"`
	LastExecPrice  float64   `json:"last_exec_price"`
	CumExecQty     float64   `json:"cum_exec_qty"`
	CumExecValue   float64   `json:"cum_exec_value"`
	CumExecFee     float64   `json:"cum_exec_fee"`
	ReduceOnly     bool      `json:"reduce_only"`
	CloseOnTrigger bool      `json:"close_on_trigger"`
	OrderLinkID    string    `json:"order_link_id"`
	CreatedTime    time.Time `json:"-"`
	UpdatedTime    time.Time `json:"-"`
}

func (o Order) GetSymbol() string {
	return o.Symbol
}

func (o Order) GetSize() float64 {
	return o.Qty
}

func (o Order) GetPrice() float64 {
	return o.Price
}

func (o Order) GetFilledSize() float64 {
	return o.CumExecQty
}

func (o Order) GetFilledPrice() float64 {
	if o.CumExecQty != 0 {
		return o.CumExecValue / o.CumExecQty
	} else {
		return o.Price
	}
}

func (o Order) GetSide() common.OrderSide {
	if o.Side == OrderSideSell {
		return common.OrderSideSell
	} else if o.Side == OrderSideBuy {
		return common.OrderSideBuy
	} else {
		return common.OrderSideUnknown
	}
}

func (o Order) GetClientID() string {
	return o.OrderLinkID
}

func (o Order) GetID() string {
	return o.OrderID
}

func (o Order) GetStatus() common.OrderStatus {
	switch o.OrderStatus {
	case OrderStatusCreated:
		return common.OrderStatusOpen
	case OrderStatusRejected:
		return common.OrderStatusReject
	case OrderStatusNew:
		return common.OrderStatusNew
	case OrderStatusPartiallyFilled:
		return common.OrderStatusPartiallyFilled
	case OrderStatusFilled:
		return common.OrderStatusFilled
	case OrderStatusCancelled:
		return common.OrderStatusCancelled
	case OrderStatusPendingCancel:
		return common.OrderStatusPendingCancel
	default:
		return common.OrderStatusUnknown
	}
}

func (o Order) GetType() common.OrderType {
	if o.OrderType == OrderTypeLimit {
		return common.OrderTypeLimit
	} else if o.OrderType == OrderTypeMarket {
		return common.OrderTypeMarket
	} else {
		return common.OrderTypeUnknown
	}
}

func (o Order) GetPostOnly() bool {
	if o.TimeInForce == TimeInForcePostOnly {
		return true
	} else {
		return false
	}
}

func (o Order) GetReduceOnly() bool {
	return o.ReduceOnly
}

func (o Order) GetExchange() common.ExchangeID {
	return ExchangeID
}

type WSCap struct {
	Success bool            `json:"success"`
	RetMsg  string          `json:"ret_msg"`
	ConnID  string          `json:"conn_id"`
	Request *WSRequest      `json:"request"`
	Topic   string          `json:"topic"`
	Action  string          `json:"action"`
	Data    json.RawMessage `json:"data"`
}

//    {
//            "symbol": "BTCUSDT",
//            "side": "Sell",
//            "order_id": "xxxxxxxx-xxxx-xxxx-9a8f-4a973eb5c418",
//            "exec_id": "xxxxxxxx-xxxx-xxxx-8b66-c3d2fcd352f6",
//            "order_link_id": "",
//            "price": 11527.5,
//            "order_qty": 0.001,
//            "exec_type": "Trade",
//            "exec_qty": 0.001,
//            "exec_fee": 0.00864563,
//            "leaves_qty": 0,
//            "is_maker": false,
//            "trade_time": "2020-08-12T21:16:18.142746Z"
//        }

type Execution struct {
	Symbol      string    `json:"symbol"`
	Side        string    `json:"side"`
	OrderID     string    `json:"order_id"`
	ExecID      string    `json:"exec_id"`
	OrderLinkID string    `json:"order_link_id"`
	Price       float64   `json:"price"`
	OrderQty    float64   `json:"order_qty"`
	ExecType    string    `json:"exec_type"`
	ExecQty     float64   `json:"exec_qty"`
	ExecFee     float64   `json:"exec_fee"`
	LeavesQty   float64   `json:"leaves_qty"`
	IsMaker     bool      `json:"is_maker"`
	ParseTime   time.Time `json:"-"`
	TradeTime   time.Time `json:"-"`
}

func (e *Execution) UnmarshalJSON(data []byte) error {
	type Alias Execution
	aux := &struct {
		*Alias
		TradeTime string `json:"trade_time"`
	}{
		Alias: (*Alias)(e),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	} else {
		e.ParseTime = time.Now()
		e.TradeTime, err = time.Parse("2006-01-02T15:04:05.999999999Z", aux.TradeTime)
		return err
	}
}

//{
//   "wallet_balance":429.80713,
//   "available_balance":429.67322
//}
type Wallet struct {
	WalletBalance    float64   `json:"wallet_balance"`
	AvailableBalance float64   `json:"available_balance"`
	Timestamp        time.Time `json:"-"`
}

func (w *Wallet) UnmarshalJSON(data []byte) error {
	type Alias Wallet
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(w),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	} else {
		w.Timestamp = time.Now()
		return nil
	}
}

func (w Wallet) GetCurrency() string {
	return "USDT"
}

func (w Wallet) GetBalance() float64 {
	return w.WalletBalance
}

func (w Wallet) GetFree() float64 {
	return w.AvailableBalance
}

func (w Wallet) GetUsed() float64 {
	return w.WalletBalance - w.AvailableBalance
}

func (w Wallet) GetTime() time.Time {
	return w.Timestamp
}

func (w Wallet) GetExchange() common.ExchangeID {
	return ExchangeID
}

type MergedPosition struct {
	Price     float64
	Size      float64
	Symbol    string
	EventTime time.Time
	ParseTime time.Time
}

func (m MergedPosition) GetSymbol() string {
	return m.Symbol
}

func (m MergedPosition) GetSize() float64 {
	return m.Size
}

func (m MergedPosition) GetPrice() float64 {
	return m.Price
}

func (m MergedPosition) GetEventTime() time.Time {
	return m.EventTime
}

func (m MergedPosition) GetParseTime() time.Time {
	return m.ParseTime
}

func (m MergedPosition) GetExchange() common.ExchangeID {
	return ExchangeID
}

type CancelOrderResp struct {
	OrderID string `json:"order_id"`
}
