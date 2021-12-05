package main

import (
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"math"
	"time"
)

type Config struct {
	Name *string `yaml:"name"`

	CpuProfile  string `yaml:"cpuProfile"`
	HeapProfile string `yaml:"heapProfile"`
	DryRun      bool   `yaml:"dryRun"`
	ReduceOnly  bool   `yaml:"reduceOnly"`

	InternalInflux common.InfluxSettings `yaml:"internalInflux"`
	ExternalInflux common.InfluxSettings `yaml:"externalInflux"`

	XExchange common.ExchangeSettings `yaml:"xExchange"`
	YExchange common.ExchangeSettings `yaml:"yExchange"`

	StreamBatchSize int `yaml:"streamBatchSize"`

	RestartSilent    time.Duration `yaml:"restartSilent"`
	RestartInterval  time.Duration `yaml:"restartInterval"`
	LogInterval      time.Duration `yaml:"logInterval"`
	TurnoverLookback time.Duration `yaml:"turnoverLookback"`
	AccountMaxAge    time.Duration `yaml:"accountMaxAge"`

	SpreadEnterOffset   float64 `yaml:"spreadEnterOffset"`
	SpreadLeaveOffset   float64 `yaml:"spreadLeaveOffset"`
	LongEnterThreshold  float64 `yaml:"longEnterThreshold"`
	LongLeaveThreshold  float64 `yaml:"longLeaveThreshold"`
	ShortEnterThreshold float64 `yaml:"shortEnterThreshold"`
	ShortLeaveThreshold float64 `yaml:"shortLeaveThreshold"`

	StatsRootPath       string        `yaml:"statsRootPath"`
	StatsSampleInterval time.Duration `yaml:"statsSampleInterval"`
	StatsSaveInterval   time.Duration `yaml:"statsSaveInterval"`

	TimeDeltaLookback time.Duration `yaml:"timeDeltaLookback"`

	SpreadTDLookback    time.Duration `yaml:"spreadTDLookback"`
	SpreadTDSubInterval time.Duration `yaml:"spreadTDSubInterval"`
	SpreadTDCompression uint32        `yaml:"spreadTDCompression"`

	SpreadShortEnterQuantileTop float64 `yaml:"spreadShortEnterQuantileTop"`
	SpreadShortLeaveQuantileBot float64 `yaml:"spreadShortLeaveQuantileBot"`
	SpreadLongEnterQuantileBot  float64 `yaml:"spreadLongEnterQuantileBot"`
	SpreadLongLeaveQuantileTop  float64 `yaml:"spreadLongLeaveQuantileTop"`
	SpreadMiddleMin             float64 `yaml:"spreadMiddleMin"`
	SpreadMiddleMax             float64 `yaml:"spreadMiddleMax"`

	XTimeDeltaOffsetTop  time.Duration `yaml:"xTimeDeltaOffsetTop"`
	XTimeDeltaOffsetBot  time.Duration `yaml:"xTimeDeltaOffsetBot"`
	YTimeDeltaOffsetTop  time.Duration `yaml:"yTimeDeltaOffsetTop"`
	YTimeDeltaOffsetBot  time.Duration `yaml:"yTimeDeltaOffsetBot"`
	XYTimeDeltaOffsetTop time.Duration `yaml:"xyTimeDeltaOffsetTop"`
	XYTimeDeltaOffsetBot time.Duration `yaml:"xyTimeDeltaOffsetBot"`

	FundingRateOpenShortMin float64             `yaml:"fundingRateOpenShortMin"`
	FundingRateOpenLongMax  float64             `yaml:"fundingRateOpenLongMax"`
	FundingRateOffsetMin    float64             `yaml:"fundingRateOffsetMin"`
	FundingRateOffsetMax    float64             `yaml:"fundingRateOffsetMax"`
	XFundingRateEaseFnName  string              `yaml:"xFundingRateEaseFnName"`
	XFundingRateEaseFn      common.EaseFunction `yaml:"-"`
	YFundingRateEaseFnName  string              `yaml:"yFundingRateEaseFnName"`
	YFundingRateEaseFn      common.EaseFunction `yaml:"-"`
	FundingRateSilentTime   time.Duration       `yaml:"fundingRateSilentTime"`
	XFundingRateWeight      float64             `yaml:"xFundingRateWeight"`
	YFundingRateWeight      float64             `yaml:"yFundingRateWeight"`

	XFundingRateInterval   time.Duration `yaml:"xFundingRateInterval"`
	YFundingRateInterval   time.Duration `yaml:"yFundingRateInterval"`
	XFundingRateTimeOffset time.Duration `yaml:"xFundingRateTimeOffset"`
	YFundingRateTimeOffset time.Duration `yaml:"yFundingRateTimeOffset"`

	TickerMaxRemoteLocalTimeDiff time.Duration `yaml:"tickerMaxRemoteLocalTimeDiff"` //控制时间上限
	TickerMaxXYTimeDiff          time.Duration `yaml:"tickerMaxXYTimeDiff"`          //控制时差上限

	SpreadMaxAge    time.Duration `yaml:"spreadMaxAge"`
	SpreadLookback  time.Duration `yaml:"spreadLookback"`
	SpreadWalkDelay time.Duration `yaml:"spreadWalkDelay"`

	XOrderSilent           time.Duration           `yaml:"xOrderSilent"`
	XOrderCancelAfter      time.Duration           `yaml:"xOrderCancelAfter"`
	XOrderTimeInForce      common.OrderTimeInForce `yaml:"xOrderTimeInForce"`
	YOrderTimeInForce      common.OrderTimeInForce `yaml:"yOrderTimeInForce"`
	YOrderSilent           time.Duration           `yaml:"yOrderSilent"`
	XEnterTimeout          time.Duration           `yaml:"xEnterTimeout"`
	XEnterSilent           time.Duration           `yaml:"xEnterSilent"`
	HedgeRatio             float64                 `yaml:"hedgeRatio"`
	HedgeDelay             time.Duration           `yaml:"hedgeDelay"`
	HedgeCheckDuration     time.Duration           `yaml:"hedgeCheckDuration"`
	HedgeCheckInterval     time.Duration           `yaml:"hedgeCheckInterval"`
	HedgeByLimit           bool                    `yaml:"hedgeByLimit"`
	RealisedSpreadLogDelay time.Duration           `yaml:"realisedSpreadLogDelay"`

	StartValue  float64 `yaml:"startValue"`
	MinXFree    float64 `yaml:"minXFree"`
	MinYFree    float64 `yaml:"minYFree"`
	MaxPosValue float64 `yaml:"maxPosValue"`

	BestSizeFactor        float64       `yaml:"bestSizeFactor"`
	EnterFreePct          float64       `yaml:"enterFreePct"`
	EnterSlippage         float64       `yaml:"enterSlippage"`
	EnterSlippageLookback time.Duration `yaml:"enterSlippageLookback"`
	EnterMinimalStep      float64       `yaml:"enterMinimalStep"`
	EnterTargetFactor     float64       `yaml:"enterTargetFactor"`
	//EnterWithProfitConfirms    bool               `yaml:"enterWithProfitConfirms"`
	StartValues map[string]float64 `yaml:"startValues"`
	//TargetWeightUpdateInterval time.Duration      `yaml:"targetWeightUpdateInterval"`

	XYPairs            map[string]string  `yaml:"xyPairs"`
	MaxPosSizes        map[string]float64 `yaml:"maxPosSizes,omitempty"`
	MaxPosValues       map[string]float64 `yaml:"maxPosValues,omitempty"`
	PosWeights         map[string]float64 `yaml:"posWeights,omitempty"`
	ReduceOnlyBySymbol map[string]bool    `yaml:"reduceOnlyBySymbol,omitempty"`
}

func (config *Config) SetDefaultIfNotSet() error {
	if config.XExchange.Leverage == 0 {
		config.XExchange.Leverage = 1.0
	}
	if config.YExchange.Leverage == 0 {
		config.YExchange.Leverage = 1.0
	}
	if config.InternalInflux.SaveInterval == 0 {
		config.InternalInflux.SaveInterval = time.Minute
	}
	if config.ExternalInflux.SaveInterval == 0 {
		config.ExternalInflux.SaveInterval = time.Minute
	}
	if config.LogInterval == 0 {
		config.LogInterval = time.Minute
	}
	if config.AccountMaxAge == 0 {
		config.AccountMaxAge = time.Minute * 3
	}
	if config.RealisedSpreadLogDelay == 0 {
		config.RealisedSpreadLogDelay = time.Second
	}
	if config.RestartSilent == 0 {
		config.RestartSilent = time.Minute * 3
	}
	if config.StreamBatchSize <= 0 {
		config.StreamBatchSize = 20
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
	if config.SpreadLookback == 0 {
		config.SpreadLookback = time.Second
	}
	//if config.TargetWeightUpdateInterval == 0 {
	//	config.TargetWeightUpdateInterval = time.Hour
	//}
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
	if config.FundingRateSilentTime == 0 {
		config.FundingRateSilentTime = time.Minute
	}
	if config.XFundingRateInterval == 0 {
		config.XFundingRateInterval = time.Hour * 4
	}
	if config.XFundingRateWeight == 0 {
		config.XFundingRateWeight = 1.0
	}
	if config.YFundingRateWeight == 0 {
		config.YFundingRateWeight = 1.0
	}
	config.XFundingRateEaseFn = common.GetEaseFnByName(config.XFundingRateEaseFnName)
	config.YFundingRateEaseFn = common.GetEaseFnByName(config.YFundingRateEaseFnName)

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
	if config.TimeDeltaLookback == 0 {
		config.TimeDeltaLookback = time.Hour * 4
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
	if config.XTimeDeltaOffsetTop == 0 {
		config.XTimeDeltaOffsetTop = time.Nanosecond
	}
	if config.XTimeDeltaOffsetBot == 0 {
		config.XTimeDeltaOffsetBot = -time.Nanosecond
	}
	if config.YTimeDeltaOffsetTop == 0 {
		config.YTimeDeltaOffsetTop = time.Nanosecond
	}
	if config.YTimeDeltaOffsetBot == 0 {
		config.YTimeDeltaOffsetBot = -time.Nanosecond
	}
	if config.XYTimeDeltaOffsetTop == 0 {
		config.XYTimeDeltaOffsetTop = time.Nanosecond
	}
	if config.XYTimeDeltaOffsetBot == 0 {
		config.XYTimeDeltaOffsetBot = -time.Nanosecond
	}
	if config.StatsSampleInterval == 0 {
		config.StatsSampleInterval = time.Second
	}
	if config.MaxPosValues == nil {
		config.MaxPosValues = make(map[string]float64)
	}
	for xSymbol := range config.XYPairs {
		if posValue, ok := config.MaxPosValues[xSymbol]; !ok {
			config.MaxPosValues[xSymbol] = config.MaxPosValue
		} else {
			config.MaxPosValues[xSymbol] = math.Min(config.MaxPosValue, posValue)
		}
		if _, ok := config.MaxPosSizes[xSymbol]; !ok {
			return fmt.Errorf("miss max pos size for %s", xSymbol)
		}
		if _, ok := config.PosWeights[xSymbol]; !ok {
			return fmt.Errorf("miss pos weight for %s", xSymbol)
		}
	}
	if config.ReduceOnlyBySymbol == nil {
		config.ReduceOnlyBySymbol = make(map[string]bool)
	}
	for xSymbol := range config.XYPairs {
		//如果是全局减仓，那所有币的减仓，如果全局不减，可以有的币自定规则
		if reduce, ok := config.ReduceOnlyBySymbol[xSymbol]; !ok || config.ReduceOnly {
			config.ReduceOnlyBySymbol[xSymbol] = config.ReduceOnly
		} else {
			config.ReduceOnlyBySymbol[xSymbol] = reduce
		}
	}
	if config.SpreadMiddleMax == 0 &&
		config.SpreadMiddleMin == 0 {
		config.SpreadMiddleMax = 0.01
		config.SpreadMiddleMin = -0.01
	}
	if config.BestSizeFactor == 0 {
		config.BestSizeFactor = 1.0
	}
	if config.XOrderTimeInForce == "" {
		config.XOrderTimeInForce = common.OrderTimeInForceIOC
	}
	if config.YOrderTimeInForce == "" {
		config.YOrderTimeInForce = common.OrderTimeInForceIOC
	}
	if config.HedgeRatio == 0 {
		config.HedgeRatio = 1.0
	}
	if config.EnterSlippageLookback == 0 {
		config.EnterSlippageLookback = time.Hour * 24
	}
	return nil
}
