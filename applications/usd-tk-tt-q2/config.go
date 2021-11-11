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

	LogInterval           time.Duration `yaml:"logInterval"`
	TurnoverLookback      time.Duration `yaml:"turnoverLookback"`
	BalancePositionMaxAge time.Duration `yaml:"balancePositionMaxAge"`

	SpreadEnterOffset   float64 `yaml:"spreadEnterOffset"`
	SpreadLeaveOffset   float64 `yaml:"spreadLeaveOffset"`
	LongEnterThreshold  float64 `yaml:"longEnterThreshold"`
	LongLeaveThreshold  float64 `yaml:"longLeaveThreshold"`
	ShortEnterThreshold float64 `yaml:"shortEnterThreshold"`
	ShortLeaveThreshold float64 `yaml:"shortLeaveThreshold"`

	StatsRootPath       string        `yaml:"statsRootPath"`
	StatsSampleInterval time.Duration `yaml:"statsSampleInterval"`
	StatsSaveInterval   time.Duration `yaml:"statsSaveInterval"`

	TimeDeltaTDLookback    time.Duration `yaml:"timeDeltaTDLookback"`
	TimeDeltaTDSubInterval time.Duration `yaml:"timeDeltaTDSubInterval"`
	TimeDeltaTDCompression uint32        `yaml:"timeDeltaTDCompression"`

	XLiquidityTDLookback    time.Duration `yaml:"xLiquidityTDLookback"`
	XLiquidityTDSubInterval time.Duration `yaml:"xLiquidityTDSubInterval"`
	XLiquidityTDCompression uint32        `yaml:"xLiquidityTDCompression"`

	YLiquidityTDLookback    time.Duration `yaml:"yLiquidityTDLookback"`
	YLiquidityTDSubInterval time.Duration `yaml:"yLiquidityTDSubInterval"`
	YLiquidityTDCompression uint32        `yaml:"yLiquidityTDCompression"`

	SpreadTDLookback    time.Duration `yaml:"spreadTDLookback"`
	SpreadTDSubInterval time.Duration `yaml:"spreadTDSubInterval"`
	SpreadTDCompression uint32        `yaml:"spreadTDCompression"`

	SpreadShortEnterQuantileTop float64 `yaml:"spreadShortEnterQuantileTop"`
	SpreadShortLeaveQuantileBot float64 `yaml:"spreadShortLeaveQuantileBot"`
	SpreadLongEnterQuantileBot  float64 `yaml:"spreadLongEnterQuantileBot"`
	SpreadLongLeaveQuantileTop  float64 `yaml:"spreadLongLeaveQuantileTop"`

	XSizeQuantile        float64 `yaml:"xLiquidityQuantile"`
	YSizeQuantile        float64 `yaml:"yLiquidityQuantile"`
	TimeDeltaQuantileTop float64 `yaml:"timeDeltaQuantileTop"`
	TimeDeltaQuantileBot float64 `yaml:"timeDeltaQuantileBot"`

	MinimalEnterFundingRate float64             `yaml:"minimalEnterFundingRate"`
	FundingRateOffsetMin    float64             `yaml:"fundingRateOffsetMin"`
	FundingRateOffsetMax    float64             `yaml:"fundingRateOffsetMax"`
	FundingRateEaseFnName   string              `yaml:"fundingRateEaseFnName"`
	FundingRateEaseFn       common.EaseFunction `yaml:"-"`
	FundingRateSilentTime   time.Duration       `yaml:"fundingRateSilentTime"`

	XFundingRateInterval   time.Duration `yaml:"xFundingRateInterval"`
	YFundingRateInterval   time.Duration `yaml:"yFundingRateInterval"`
	XFundingRateTimeOffset time.Duration `yaml:"xFundingRateTimeOffset"`
	YFundingRateTimeOffset time.Duration `yaml:"yFundingRateTimeOffset"`

	XFundingRateFactor float64 `yaml:"xFundingRateFactor"`
	YFundingRateFactor float64 `yaml:"yFundingRateFactor"`

	TickerMaxRemoteLocalTimeDiff time.Duration `yaml:"tickerMaxRemoteLocalTimeDiff"` //控制时间上限
	TickerMaxXYTimeDiff          time.Duration `yaml:"tickerMaxXYTimeDiff"`          //控制时差上限
	TickerReportCount            int           `yaml:"tickerReportCount"`

	SpreadTimeToEnter time.Duration `yaml:"spreadTimeToEnter"`
	SpreadLookback    time.Duration `yaml:"spreadLookback"`
	SpreadWalkDelay   time.Duration `yaml:"spreadWalkDelay"`

	BatchSize int `yaml:"batchSize"`

	StartValue            float64 `yaml:"startValue"`
	MinimalXFree          float64 `yaml:"minimalXFree"`
	MinimalYFree          float64 `yaml:"minimalYFree"`
	GlobalMaximalPosValue float64 `yaml:"GlobalMaximalPosValue"`

	//BestSizeFactor    float64            `yaml:"bestSizeFactor"`
	EnterFreePct      float64            `yaml:"enterFreePct"`
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

	XYPairs          map[string]string  `yaml:"xyPairs"`
	MaximalPosValues map[string]float64 `yaml:"maximalPosValues,omitempty"`
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
	if config.TickerMaxXYTimeDiff == 0 {
		config.TickerMaxXYTimeDiff = time.Second
	}
	if config.TickerMaxRemoteLocalTimeDiff == 0 {
		config.TickerMaxRemoteLocalTimeDiff = time.Second * 5
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
	config.XExchange.DryRun = config.DryRun
	config.YExchange.DryRun = config.DryRun
	//if config.BestSizeFactor == 0 {
	//	config.BestSizeFactor = 1.0
	//}
	if config.FundingRateSilentTime == 0 {
		config.FundingRateSilentTime = time.Minute
	}
	if config.XFundingRateInterval == 0 {
		config.XFundingRateInterval = time.Hour * 4
	}
	if config.XFundingRateFactor == 0 {
		config.XFundingRateFactor = 1.0
	}
	if config.YFundingRateFactor == 0 {
		config.YFundingRateFactor = 1.0
	}
	if config.FundingRateEaseFnName == "" {
		config.FundingRateEaseFn = common.Linear
	}else{
		config.FundingRateEaseFn = common.GetEaseFnByName(config.FundingRateEaseFnName)
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
	if config.SpreadShortLeaveQuantileBot == 0 {
		config.SpreadShortLeaveQuantileBot = 0.2
	}
	if config.SpreadLongLeaveQuantileTop == 0 {
		config.SpreadLongLeaveQuantileTop = 0.8
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
	if config.SpreadShortLeaveQuantileBot == 0 {
		config.SpreadShortLeaveQuantileBot = 0.2
	}
	if config.SpreadLongEnterQuantileBot == 0 {
		config.SpreadLongEnterQuantileBot = 0.005
	}
	if config.SpreadLongLeaveQuantileTop == 0 {
		config.SpreadLongLeaveQuantileTop = 0.8
	}
	if config.XSizeQuantile == 0 {
		config.XSizeQuantile = 0.8
	}
	if config.YSizeQuantile == 0 {
		config.YSizeQuantile = 0.8
	}
	if config.TimeDeltaQuantileTop == 0 {
		config.TimeDeltaQuantileTop = 0.95
	}
	if config.TimeDeltaQuantileBot == 0 {
		config.TimeDeltaQuantileBot = 0.2
	}
	if config.StatsSampleInterval == 0 {
		config.StatsSampleInterval = time.Second
	}
	if config.MaximalPosValues == nil {
		config.MaximalPosValues = make(map[string]float64)
	}
	for xSymbol := range config.XYPairs {
		if v, ok := config.MaximalPosValues[xSymbol]; !ok {
			config.MaximalPosValues[xSymbol] = config.GlobalMaximalPosValue
		} else if v > config.GlobalMaximalPosValue {
			config.MaximalPosValues[xSymbol] = config.GlobalMaximalPosValue
		}
	}
}
