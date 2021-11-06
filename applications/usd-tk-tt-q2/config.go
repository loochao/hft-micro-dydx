package main

import (
	"github.com/geometrybase/hft-micro/common"
	"time"
)

type Config struct {
	Name *string `yaml:"name"`

	CpuProfile string `yaml:"cpuProfile"`
	DryRun     bool   `yaml:"dryRun"`
	ReduceOnly bool   `yaml:"reduceOnly"`

	InternalInflux common.InfluxSettings `yaml:"internalInflux"`
	ExternalInflux common.InfluxSettings `yaml:"externalInflux"`

	XExchange common.ExchangeSettings `yaml:"xExchange"`
	YExchange common.ExchangeSettings `yaml:"yExchange"`

	SpreadWalkDelay       time.Duration `yaml:"spreadWalkDelay"`
	LogInterval           time.Duration `yaml:"logInterval"`
	TurnoverLookback      time.Duration `yaml:"turnoverLookback"`
	BalancePositionMaxAge time.Duration `yaml:"balancePositionMaxAge"`

	EnterOffset     float64 `yaml:"enterOffset"`
	ExitOffset      float64 `yaml:"exitOffset"`
	LongEnterDelta  float64 `yaml:"longEnterDelta"`
	ShortEnterDelta float64 `yaml:"shortEnterDelta"`
	LongExitDelta   float64 `yaml:"longExitDelta"`
	ShortExitDelta  float64 `yaml:"shortExitDelta"`

	TDRootPath     string        `yaml:"tdRootPath"`
	TDSaveInterval time.Duration `yaml:"tdSaveInterval"`

	SpreadTDLookback        time.Duration `yaml:"spreadTDLookback"`
	SpreadTDSubInterval     time.Duration `yaml:"spreadTDSubInterval"`
	SpreadTDCompression     uint32        `yaml:"spreadTDCompression"`

	XLiquidityTDLookback    time.Duration `yaml:"xLiquidityTDLookback"`
	XLiquidityTDSubInterval time.Duration `yaml:"xLiquidityTDSubInterval"`
	XLiquidityTDCompression uint32        `yaml:"xLiquidityTDCompression"`

	YLiquidityTDLookback    time.Duration `yaml:"yLiquidityTDLookback"`
	YLiquidityTDSubInterval time.Duration `yaml:"yLiquidityTDSubInterval"`
	YLiquidityTDCompression uint32        `yaml:"yLiquidityTDCompression"`

	TimeDeltaTDLookback     time.Duration `yaml:"timeDeltaTDLookback"`
	TimeDeltaTDSubInterval  time.Duration `yaml:"timeDeltaTDSubInterval"`
	TimeDeltaTDCompression  uint32        `yaml:"timeDeltaTDCompression"`

	SpreadShortEnterQuantileTop float64 `yaml:"spreadShortEnterQuantileTop"`
	SpreadShortExitQuantileBot  float64 `yaml:"spreadShortExitQuantileBot"`
	SpreadLongEnterQuantileBot  float64 `yaml:"spreadLongEnterQuantileBot"`
	SpreadLongExitQuantileTop   float64 `yaml:"spreadLongExitQuantileTop"`

	XLiquidityQuantile   float64 `yaml:"xLiquidityQuantile"`
	YLiquidityQuantile   float64 `yaml:"yLiquidityQuantile"`
	TimeDeltaQuantileTop float64 `yaml:"spreadShortEnterQuantileTop"`
	TimeDeltaQuantileBot float64 `yaml:"spreadShortExitQuantileBot"`

	StatsOutputInterval time.Duration `yaml:"statsOutputInterval"`
	StatsSampleInterval time.Duration `yaml:"statsSampleInterval"`

	MinimalEnterFundingRate float64       `yaml:"minimalEnterFundingRate"`
	FundingRateOffsetMin    float64       `yaml:"fundingRateOffsetMin"`
	FundingRateOffsetMax    float64       `yaml:"fundingRateOffsetMax"`
	FundingRateSilentTime   time.Duration `yaml:"fundingRateSilentTime"`
	FundingRateInterval     time.Duration `yaml:"fundingRateInterval"`
	FundingRateTimeOffset   time.Duration `yaml:"fundingRateTimeOffset"`
	XFundingRateFactor      float64       `yaml:"xFundingRateFactor"`
	YFundingRateFactor      float64       `yaml:"yFundingRateFactor"`

	TickerMaxTimeDelta   time.Duration `yaml:"tickerTimeDeltaMax"`
	TickerMinTimeDelta   time.Duration `yaml:"tickerTimeDeltaMin"`
	TickerYDecay         float64       `yaml:"tickerYDecay"`
	TickerXDecay         float64       `yaml:"tickerXDecay"`
	TickerYBias          time.Duration `yaml:"tickerYBias"`
	TickerXBias          time.Duration `yaml:"tickerXBias"`
	TickerMaxAgeDiffBias time.Duration `yaml:"tickerMaxAgeDiffBias"`
	TickerReportCount    int           `yaml:"tickerReportCount"`

	SpreadTimeToEnter time.Duration `yaml:"spreadTimeToEnter"`
	SpreadLookback    time.Duration `yaml:"spreadLookback"`
	BatchSize         int           `yaml:"batchSize"`

	StartValue float64 `yaml:"startValue"`

	MinimalXFree     float64 `yaml:"minimalXFree"`
	MinimalYFree     float64 `yaml:"minimalYFree"`
	MaximalXPosValue float64 `yaml:"maximalXPosValue"`
	MaximalYPosValue float64 `yaml:"maximalYPosValue"`

	EnterFreePct      float64            `yaml:"enterFreePct"`
	BestSizeFactor    float64            `yaml:"bestSizeFactor"`
	EnterSlippage     float64            `yaml:"enterSlippage"`
	EnterMinimalStep  float64            `yaml:"enterMinimalStep"`
	EnterTargetFactor float64            `yaml:"enterTargetFactor"`
	StartValues       map[string]float64 `yaml:"startValues"`

	XOrderSilent           time.Duration           `yaml:"xOrderSilent"`
	XOrderTimeInForce      common.OrderTimeInForce `yaml:"xOrderTimeInForce"`
	YOrderSilent           time.Duration           `yaml:"yOrderSilent"`
	XEnterTimeout          time.Duration           `yaml:"xEnterTimeout"`
	XEnterSilent           time.Duration           `yaml:"xEnterSilent"`
	HedgeDelay             time.Duration           `yaml:"hedgeDelay"`
	HedgeCheckDuration     time.Duration           `yaml:"hedgeCheckDuration"`
	HedgeCheckInterval     time.Duration           `yaml:"hedgeCheckInterval"`
	RealisedSpreadLogDelay time.Duration           `yaml:"realisedSpreadLogDelay"`
	RestartSilent          time.Duration           `yaml:"restartSilent"`
	RestartInterval        time.Duration           `yaml:"restartInterval"`

	XYPairs        map[string]string  `yaml:"xyPairs"`
	TargetWeights  map[string]float64 `yaml:"targetWeights,omitempty"`
	MaxOrderValues map[string]float64 `yaml:"maxOrderValues,omitempty"`
}

func (config *Config) SetDefaultIfNotSet() {
	if config.LogInterval == 0 {
		config.LogInterval = time.Minute
	}
	if config.BalancePositionMaxAge == 0 {
		config.BalancePositionMaxAge = time.Minute * 3
	}
	if config.RealisedSpreadLogDelay == 0 {
		config.RealisedSpreadLogDelay = time.Second
	}
	if config.RestartSilent == 0 {
		config.RestartSilent = time.Minute * 3
	}
	if config.BatchSize <= 0 {
		config.BatchSize = 20
	}
	if config.XEnterSilent == 0 {
		config.XEnterSilent = time.Second
	}
	if config.TickerMaxAgeDiffBias == 0 {
		config.TickerMaxAgeDiffBias = time.Millisecond * 100
	}
	if config.TickerReportCount == 0 {
		config.RestartSilent = 1000
	}
	if config.SpreadLookback == 0 {
		config.SpreadLookback = time.Second
	}
	if config.RestartInterval == 0 {
		config.RestartInterval = time.Hour * 9999
	}
	if config.TurnoverLookback == 0 {
		config.TurnoverLookback = time.Hour * 24
	}
	if config.XOrderSilent == 0 {
		config.XOrderSilent = time.Second
	}
	if config.YOrderSilent == 0 {
		config.YOrderSilent = time.Second * 5
	}
	if config.FundingRateSilentTime == 0 {
		config.FundingRateSilentTime = time.Minute
	}
	if config.FundingRateInterval == 0 {
		config.FundingRateInterval = time.Hour * 4
	}
	config.XExchange.DryRun = config.DryRun
	config.YExchange.DryRun = config.DryRun
	if config.BestSizeFactor == 0 {
		config.BestSizeFactor = 1.0
	}
	if config.XFundingRateFactor == 0 {
		config.XFundingRateFactor = 1.0
	}
	if config.YFundingRateFactor == 0 {
		config.YFundingRateFactor = 1.0
	}
	if config.XOrderTimeInForce == "" {
		config.XOrderTimeInForce = common.OrderTimeInForceFOK
	}
	if config.XEnterTimeout == 0 {
		config.XEnterTimeout = time.Minute
	}

	if config.SpreadLongEnterQuantileBot == 0 {
		config.SpreadLongEnterQuantileBot = 0.005
	}
	if config.SpreadShortEnterQuantileTop == 0 {
		config.SpreadShortEnterQuantileTop = 0.995
	}
	if config.SpreadShortExitQuantileBot == 0 {
		config.SpreadShortExitQuantileBot = 0.2
	}
	if config.SpreadLongExitQuantileTop == 0 {
		config.SpreadLongExitQuantileTop = 0.8
	}

	if config.SpreadTDLookback == 0 {
		config.SpreadTDLookback = time.Hour * 72
	}
	if config.SpreadTDSubInterval == 0 {
		config.SpreadTDSubInterval = time.Hour
	}
	if config.SpreadTDCompression == 0 {
		config.SpreadTDCompression = 10
	}
	if config.XLiquidityTDLookback == 0 {
		config.XLiquidityTDLookback = time.Hour * 4
	}
	if config.XLiquidityTDSubInterval == 0 {
		config.XLiquidityTDSubInterval = time.Minute * 5
	}
	if config.XLiquidityTDCompression == 0 {
		config.XLiquidityTDCompression = 10
	}
	if config.YLiquidityTDLookback == 0 {
		config.YLiquidityTDLookback = time.Hour * 4
	}
	if config.YLiquidityTDSubInterval == 0 {
		config.YLiquidityTDSubInterval = time.Minute * 5
	}
	if config.YLiquidityTDCompression == 0 {
		config.YLiquidityTDCompression = 10
	}
	if config.TimeDeltaTDLookback == 0 {
		config.TimeDeltaTDLookback = time.Hour * 4
	}
	if config.TimeDeltaTDSubInterval == 0 {
		config.TimeDeltaTDSubInterval = time.Minute * 5
	}
	if config.TimeDeltaTDCompression == 0 {
		config.TimeDeltaTDCompression = 10
	}
	if config.SpreadShortEnterQuantileTop == 0 {
		config.SpreadShortEnterQuantileTop = 0.995
	}
	if config.SpreadShortExitQuantileBot == 0 {
		config.SpreadShortExitQuantileBot = 0.2
	}
	if config.SpreadLongEnterQuantileBot == 0 {
		config.SpreadLongEnterQuantileBot = 0.005
	}
	if config.SpreadLongExitQuantileTop == 0 {
		config.SpreadLongExitQuantileTop = 0.8
	}
	if config.XLiquidityQuantile == 0 {
		config.XLiquidityQuantile = 0.8
	}
	if config.YLiquidityQuantile == 0 {
		config.YLiquidityQuantile = 0.8
	}
	if config.TimeDeltaQuantileTop == 0 {
		config.TimeDeltaQuantileTop = 0.95
	}
	if config.TimeDeltaQuantileBot == 0 {
		config.TimeDeltaQuantileBot = 0.2
	}
	if config.StatsOutputInterval == 0 {
		config.StatsOutputInterval = time.Second * 15
	}
	if config.StatsSampleInterval == 0 {
		config.StatsSampleInterval = time.Second
	}
}
