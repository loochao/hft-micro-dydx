package main

import (
	"fmt"
	"github.com/geometrybase/hft/common"
	"reflect"
	"strings"
	"time"
)

type InfluxConfig struct {
	Address      *string        `yaml:"address,omitempty"`
	Username     *string        `yaml:"username,omitempty"`
	Password     *string        `yaml:"password,omitempty"`
	Database     *string        `yaml:"database,omitempty"`
	Measurement  *string        `yaml:"measurement,omitempty"`
	BatchSize    *int           `yaml:"batchSize,omitempty"`
	SaveInterval *time.Duration `yaml:"saveInterval,omitempty"`
}

type Config struct {
	Name *string `yaml:"name"`

	DryRun       *bool   `yaml:"dryRun"`
	ProxyAddress *string `yaml:"proxyAddress,omitempty"`

	InternalInflux *InfluxConfig `yaml:"internalInflux"`
	ExternalInflux *InfluxConfig `yaml:"externalInflux"`
	MarginType     *string       `yaml:"marginType,omitempty"`

	BnApiKey    *string `yaml:"bnApiKey,omitempty"`
	BnApiSecret *string `yaml:"bnApiSecret,omitempty"`

	OkApiKey     *string `yaml:"okApiKey,omitempty"`
	OkApiSecret  *string `yaml:"okApiSecret,omitempty"`
	OkPassphrase *string `yaml:"okPassphrase,omitempty"`

	OkApiUrl *string `yaml:"okApiUrl,omitempty"`
	OkWsUrl  *string `yaml:"okWsUrl,omitempty"`

	Leverage       *float64 `yaml:"leverage,omitempty"`
	ChangeLeverage *bool    `yaml:"changeLeverage,omitempty"`

	LoopInterval *time.Duration `yaml:"loopInterval,omitempty"`
	PullInterval *time.Duration `yaml:"pullInterval,omitempty"`

	ResetUnrealisedPnlInterval *time.Duration `yaml:"resetUnrealisedPnlInterval,omitempty"`
	ResetUnrealisedTriggerPct  *float64       `yaml:"resetUnrealisedTriggerPct,omitempty"`
	ResetCount                 *int           `yaml:"resetCount,omitempty"`

	Symbols []string `yaml:"symbols,omitempty"`

	PullBarsInterval      *time.Duration `yaml:"pullBarsInterval,omitempty"`
	PullBarsRetryInterval *time.Duration `yaml:"pullBarsRetryInterval,omitempty"`
	BarsLookback          *int           `yaml:"barsLookback,omitempty"`

	TopQuantile       *float64 `yaml:"topQuantile,omitempty"`
	BotQuantile       *float64 `yaml:"botQuantile,omitempty"`
	MinimalBandOffset *float64 `yaml:"minimalBandOffset,omitempty"`
	MinimalEnterDelta *float64 `yaml:"minimalEnterDelta,omitempty"`
	MaximalExitDelta  *float64 `yaml:"maximalExitDelta,omitempty"`
	TradeCount        *int     `yaml:"tradeCount,omitempty"`
	TopBandScale      *float64 `yaml:"topBandScale,omitempty"`
	BotBandScale      *float64 `yaml:"botBandScale,omitempty"`

	MinimalEnterFundingRate    *float64       `yaml:"minimalEnterFundingRate,omitempty"`
	MinimalKeepFundingRate     *float64       `yaml:"minimalKeepFundingRate,omitempty"`
	MinimalDeltaWindow         *int           `yaml:"minimalDeltaWindow,omitempty"`
	MinimalOrderBookValue      *float64       `yaml:"minimalOrderBookValue,omitempty"`
	OrderBookType              *string        `yaml:"orderBookType,omitempty"`
	SwapSpotTimeDeltaTolerance *time.Duration `yaml:"swapSpotTimeDeltaTolerance,omitempty"`
	SystemTimeDeltaTolerance   *time.Duration `yaml:"systemTimeDeltaTolerance,omitempty"`
	DeltaLookback              *time.Duration `yaml:"deltaLookback,omitempty"`

	StartValue  *float64 `yaml:"startValue,omitempty"`
	EnterStep   *float64 `yaml:"enterStep,omitempty"`
	EnterTarget *float64 `yaml:"enterTarget,omitempty"`

	InsuranceFundingRatio    *float64       `yaml:"insuranceFundingRatio,omitempty"`
	ReBalanceInterval        *time.Duration `yaml:"reBalanceInterval,omitempty"`
	ReBalanceMinimalNotional *float64       `yaml:"reBalanceMinimalNotional,omitempty"`

	EnterSlippage *float64       `yaml:"enterSlippage,omitempty"`
	OrderTimeout  *time.Duration `yaml:"orderTimeout,omitempty"`
	OrderSilent   *time.Duration `yaml:"orderSilent,omitempty"`
	EnterSilent   *time.Duration `yaml:"enterSilent,omitempty"`
	ExitSilent    *time.Duration `yaml:"exitSilent,omitempty"`
}

func (config *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type Alias Config
	aux := (*Alias)(config)
	var err error
	if err = unmarshal(aux); err != nil {
		return err
	}
	if config.ProxyAddress == nil {
		var proxyAddress = ""
		config.ProxyAddress = &proxyAddress
	}
	return nil
}

func (config *Config) IsValid() (bool, string) {
	return config.isValid(config, "")
}

func (config *Config) isValid(v interface{}, prefix string) (bool, string) {
	structVal := reflect.ValueOf(v).Elem()
	structType := structVal.Type()
	errors := ""
	for i := 0; i < structVal.NumField(); i++ {
		structField := structVal.Field(i)
		var outgoingTag string
		if structType.Field(i).Tag != "" {
			jsonTag := structType.Field(i).Tag.Get("yaml")
			if jsonTag != "" {
				split := common.SplitStrings(jsonTag, ",")
				outgoingTag = split[0]
			}
		}
		if outgoingTag == "" || outgoingTag == "-" {
			outgoingTag = structType.Field(i).Name
		}
		outgoingTag = common.LcFirst(outgoingTag)
		if structField.IsNil() {
			if strings.ToLower(outgoingTag) != "proxyaddress" {
				errors += fmt.Sprintf("%s%s is empty;\n", prefix, outgoingTag)
			}
		} else {
			switch v := structField.Interface().(type) {
			case *InfluxConfig:
				isValid, reason := config.isValid(v, "influx.")
				if !isValid {
					errors += fmt.Sprintf("\n%s", reason)
				}
			case *string:
				if *v == "" && strings.ToLower(outgoingTag) != "proxyaddress" {
					errors += fmt.Sprintf("%s%s is empty;\n", prefix, outgoingTag)
				}
			default:
			}
		}
	}
	if errors != "" {
		return false, errors
	}
	return true, ""
}
func (config *Config) toString(v interface{}, prefix string) string {
	structVal := reflect.ValueOf(v).Elem()
	structType := structVal.Type()
	output := ""
	for i := 0; i < structVal.NumField(); i++ {
		structField := structVal.Field(i)
		var outgoingTag string
		if structType.Field(i).Tag != "" {
			jsonTag := structType.Field(i).Tag.Get("yaml")
			if jsonTag != "" {
				split := common.SplitStrings(jsonTag, ",")
				outgoingTag = split[0]
			}
		}
		if outgoingTag == "" || outgoingTag == "-" {
			outgoingTag = structType.Field(i).Name
		}
		outgoingTag = common.LcFirst(outgoingTag)
		if structField.IsNil() {
			output += fmt.Sprintf("%s%s=\n", prefix, outgoingTag)
		} else {
			switch d := structField.Interface().(type) {
			case *InfluxConfig:
				output += fmt.Sprintf("%s", config.toString(d, "influx."))
			case time.Time:
				output += fmt.Sprintf("%s%s=%v\n", prefix, outgoingTag, d)
			case time.Duration:
				output += fmt.Sprintf("%s%s=%v\n", prefix, outgoingTag, d)
			case int:
				output += fmt.Sprintf("%s%s=%d\n", prefix, outgoingTag, d)
			case int64:
				output += fmt.Sprintf("%s%s=%d\n", prefix, outgoingTag, d)
			case float64:
				output += fmt.Sprintf("%s%s=%f\n", prefix, outgoingTag, d)
			case bool:
				output += fmt.Sprintf("%s%s=%v\n", prefix, outgoingTag, d)
			case string:
				output += fmt.Sprintf("%s%s=%v\n", prefix, outgoingTag, d)
			case *time.Time:
				output += fmt.Sprintf("%s%s=%v\n", prefix, outgoingTag, *d)
			case *time.Duration:
				output += fmt.Sprintf("%s%s=%v\n", prefix, outgoingTag, *d)
			case *int:
				output += fmt.Sprintf("%s%s=%d\n", prefix, outgoingTag, *d)
			case *int64:
				output += fmt.Sprintf("%s%s=%d\n", prefix, outgoingTag, *d)
			case *float64:
				output += fmt.Sprintf("%s%s=%f\n", prefix, outgoingTag, *d)
			case *bool:
				output += fmt.Sprintf("%s%s=%v\n", prefix, outgoingTag, *d)
			case *string:
				output += fmt.Sprintf("%s%s=%v\n", prefix, outgoingTag, *d)
			case []int:
				output += fmt.Sprintf("%s%s=%d\n", prefix, outgoingTag, d)
			case []int64:
				output += fmt.Sprintf("%s%s=%d\n", prefix, outgoingTag, d)
			case []float64:
				output += fmt.Sprintf("%s%s=%f\n", prefix, outgoingTag, d)
			case []bool:
				output += fmt.Sprintf("%s%s=%v\n", prefix, outgoingTag, d)
			case []string:
				output += fmt.Sprintf("%s%s=%s\n", prefix, outgoingTag, d)
			case map[string]int:
				output += fmt.Sprintf("%s%s=%v\n", prefix, outgoingTag, d)
			case map[string]int64:
				output += fmt.Sprintf("%s%s=%v\n", prefix, outgoingTag, d)
			case map[string]float64:
				output += fmt.Sprintf("%s%s=%v\n", prefix, outgoingTag, d)
			case map[string]bool:
				output += fmt.Sprintf("%s%s=%v\n", prefix, outgoingTag, d)
			case map[string]string:
				output += fmt.Sprintf("%s%s=%v\n", prefix, outgoingTag, d)
			case map[string][]int:
				output += fmt.Sprintf("%s%s=%v\n", prefix, outgoingTag, d)
			}
		}
	}
	return output + "\n"
}

func (config *Config) ToString() string {
	return config.toString(config, "")
}
