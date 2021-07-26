package gate_usdtfuture

type Contract struct {
	FundingRateIndicative float64 `json:"funding_rate_indicative,string"`
	MarkPriceRound        float64 `json:"mark_price_round,string"`
	FundingOffset         float64 `json:"funding_offset"`
	InDelisting           bool    `json:"in_delisting"`
	RiskLimitBase         float64 `json:"risk_limit_base,string"`
	InterestRate          float64 `json:"interest_rate,string"`
	IndexPrice            float64 `json:"index_price,string"`
	OrderPriceRound       float64 `json:"order_price_round,string"`
	OrderSizeMin          float64 `json:"order_size_min"`
	RefRebateRate         float64 `json:"ref_rebate_rate,string"`
	Name                  string  `json:"name"`
	RefDiscountRate       float64 `json:"ref_discount_rate,string"`
	OrderPriceDeviate     float64 `json:"order_price_deviate,string"`
	MaintenanceRate       float64 `json:"maintenance_rate,string"`
	MarkType              string  `json:"mark_type"`
	FundingInterval       int64   `json:"funding_interval"`
	Type                  string  `json:"type"`
	RiskLimitStep         float64 `json:"risk_limit_step,string"`
	LeverageMin           float64 `json:"leverage_min,string"`
	FundingRate           float64 `json:"funding_rate,string"`
	LastPrice             float64 `json:"last_price,string"`
	MarkPrice             float64 `json:"mark_price,string"`
	OrderSizeMax          float64 `json:"order_size_max"`
	FundingNextApply      float64 `json:"funding_next_apply"`
	ShortUsers            int64   `json:"short_users"`
	ConfigChangeTime      int64   `json:"config_change_time"`
	TradeSize             float64 `json:"trade_size"`
	PositionSize          float64 `json:"position_size"`
	LongUsers             int64   `json:"long_users"`
	QuantoMultiplier      float64 `json:"quanto_multiplier,string"`
	FundingImpactValue    float64 `json:"funding_impact_value,string"`
	LeverageMax           float64 `json:"leverage_max,string"`
	RiskLimitMax          float64 `json:"risk_limit_max,string"`
	MakerFeeRate          float64 `json:"maker_fee_rate,string"`
	TakerFeeRate          float64 `json:"taker_fee_rate,string"`
	OrdersLimit           float64 `json:"orders_limit"`
	TradeID               float64 `json:"trade_id"`
	OrderbookID           float64 `json:"orderbook_id"`
}
