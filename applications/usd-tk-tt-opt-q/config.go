package main

import (
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"strconv"
	"strings"
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

	QuantileLookback       time.Duration `yaml:"quantileLookback"`
	QuantileSubInterval    time.Duration `yaml:"quantileSubInterval"`
	QuantilePath           string        `yaml:"quantilePath"`
	QuantileSaveInterval   time.Duration `yaml:"quantileSaveInterval"`
	QuantileSampleInterval time.Duration `yaml:"quantileSampleInterval"`

	MinimalEnterFundingRate float64       `yaml:"minimalEnterFundingRate"`
	FundingRateOffsetFactor float64       `yaml:"fundingRateOffsetFactor"`
	FundingRateSilentTime   time.Duration `yaml:"fundingRateSilentTime"`
	FundingRateInterval     time.Duration `yaml:"fundingRateInterval"`
	FundingRateTimeOffset   time.Duration `yaml:"fundingRateTimeOffset"`

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

	EnterFreePct     float64            `yaml:"enterFreePct"`
	BestSizeFactor   float64            `yaml:"bestSizeFactor"`
	EnterSlippage    float64            `yaml:"enterSlippage"`
	EnterMinimalStep float64            `yaml:"enterMinimalStep"`
	StartValues      map[string]float64 `yaml:"startValues"`

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

	XYPairs map[string]string `yaml:"xyPairs"`

	Offsets        map[string]string  `yaml:"offsets,omitempty"`
	TargetWeights  map[string]float64 `yaml:"targetWeights,omitempty"`
	MaxOrderValues map[string]float64 `yaml:"maxOrderValues,omitempty"`

	DefaultEnterOffset float64            `yaml:"defaultEnterOffset,omitempty"`
	DefaultLeaveOffset float64            `yaml:"defaultLeaveOffset,omitempty"`
	EnterOffsets       map[string]float64 `yaml:"enterOffsets,omitempty"`
	LeaveOffsets       map[string]float64 `yaml:"leaveOffsets,omitempty"`
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
	if config.XOrderTimeInForce == "" {
		config.XOrderTimeInForce = common.OrderTimeInForceFOK
	}
	if config.XEnterTimeout == 0 {
		config.XEnterTimeout = time.Minute
	}
	if config.DefaultEnterOffset == 0 {
		config.DefaultEnterOffset = 0.004
	}
	if config.DefaultLeaveOffset == 0 {
		config.DefaultLeaveOffset = 0.001
	}
	config.EnterOffsets = make(map[string]float64)
	config.LeaveOffsets = make(map[string]float64)
	var err error
	for xSymbol := range config.XYPairs {
		if offset, ok := config.Offsets[xSymbol]; !ok {
			config.EnterOffsets[xSymbol] = config.DefaultEnterOffset
			config.LeaveOffsets[xSymbol] = config.DefaultLeaveOffset
			logger.Debugf("miss offset for %s", xSymbol)
		} else {
			segs := strings.Split(offset, ",")
			if len(segs) != 2 {
				logger.Fatalf("bad offset %s", offset)
			}
			config.EnterOffsets[xSymbol], err = strconv.ParseFloat(segs[0], 64)
			if err != nil {
				logger.Fatalf("bad offset %s", offset)
			}
			config.LeaveOffsets[xSymbol], err = strconv.ParseFloat(segs[1], 64)
			if err != nil {
				logger.Fatalf("bad offset %s", offset)
			}
		}
	}
}
