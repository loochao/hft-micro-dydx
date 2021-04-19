package main

import (
	"fmt"
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
	CpuProfile   *string `yaml:"cpuProfile,omitempty"`

	InternalInflux *InfluxConfig `yaml:"internalInflux"`
	ExternalInflux *InfluxConfig `yaml:"externalInflux"`

	HbApiKey    *string `yaml:"hbApiKey,omitempty"`
	HbApiSecret *string `yaml:"hbApiSecret,omitempty"`
	BnApiKey    *string `yaml:"bnApiKey,omitempty"`
	BnApiSecret *string `yaml:"bnApiSecret,omitempty"`

	Leverage       *int    `yaml:"leverage,omitempty"`
	ChangeLeverage *bool   `yaml:"changeLeverage,omitempty"`
	MarginType     *string `yaml:"marginType,omitempty"`

	LoopInterval          *time.Duration `yaml:"loopInterval,omitempty"`
	LogInterval           *time.Duration `yaml:"logInterval,omitempty"`
	PullInterval          *time.Duration `yaml:"pullInterval,omitempty"`
	RequestInterval       *time.Duration `yaml:"requestInterval,omitempty"`
	BalancePositionMaxAge *time.Duration `yaml:"balancePositionMaxAge,omitempty"`

	MakerTakerSymbolsMap map[string]string `yaml:"makerTakerSymbolsMap,omitempty"`

	PullBarsInterval      *time.Duration `yaml:"pullBarsInterval,omitempty"`
	PullBarsRetryInterval *time.Duration `yaml:"pullBarsRetryInterval,omitempty"`
	BarsLookback          *int           `yaml:"barsLookback,omitempty"`

	TopQuantile       *float64 `yaml:"topQuantile,omitempty"`
	BotQuantile       *float64 `yaml:"botQuantile,omitempty"`
	MinimalBandOffset *float64 `yaml:"minimalBandOffset,omitempty"`
	MinimalEnterDelta *float64 `yaml:"minimalEnterDelta,omitempty"`
	MaximalExitDelta  *float64 `yaml:"maximalExitDelta,omitempty"`
	TopBandScale      *float64 `yaml:"topBandScale,omitempty"`
	BotBandScale      *float64 `yaml:"botBandScale,omitempty"`

	MinimalEnterFundingRate *float64 `yaml:"minimalEnterFundingRate,omitempty"`
	MinimalKeepFundingRate  *float64 `yaml:"minimalKeepFundingRate,omitempty"`

	OrderBookTakerDecay         *float64       `yaml:"orderBookTakerDecay,omitempty"`
	OrderBookMakerDecay         *float64       `yaml:"orderBookMakerDecay,omitempty"`
	OrderBookTakerBias          *float64       `yaml:"orderBookTakerBias,omitempty"`
	OrderBookMakerBias          *float64       `yaml:"orderBookMakerBias,omitempty"`
	OrderBookImpact             *float64       `yaml:"orderBookImpact,omitempty"`
	OrderBookMaxAgeDiff         *time.Duration `yaml:"orderBookMaxAgeDiff,omitempty"`
	OrderBookMaxAge             *time.Duration `yaml:"orderBookMaxAge,omitempty"`
	SpreadTimeToLive            *time.Duration `yaml:"spreadTimeToLive,omitempty"`
	SpreadLookbackDuration      *time.Duration `yaml:"spreadLookbackDuration,omitempty"`
	SpreadLookbackMinimalWindow *int           `yaml:"spreadLookbackMinimalWindow,omitempty"`

	StartValue        *float64           `yaml:"startValue,omitempty"`
	EnterFreePct      *float64           `yaml:"enterFreePct,omitempty"`
	EnterMinimalStep  *float64           `yaml:"enterMinimalStep,omitempty"`
	EnterTargetFactor *float64           `yaml:"enterTargetFactor,omitempty"`
	MaxUnHedgeValue   *float64           `yaml:"maxUnHedgeValue,omitempty"`
	StartValues       map[string]float64 `yaml:"startValues,omitempty"`

	OrderTimeout        *time.Duration `yaml:"orderTimeout,omitempty"`
	OrderSilent         *time.Duration `yaml:"orderSilent,omitempty"`
	OrderCancelSilent   *time.Duration `yaml:"orderCancelSilent,omitempty"`
	OrderMaxCancelCount *int           `yaml:"orderMaxCancelCount,omitempty"`
	EnterSlippage       *float64       `yaml:"enterSlippage,omitempty"`
	EnterSilent         *time.Duration `yaml:"enterSilent,omitempty"`
	RestartSilent       *time.Duration `yaml:"restartSilent,omitempty"`
	HttpSilent          *time.Duration `yaml:"httpSilent,omitempty"`
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
	if config.CpuProfile == nil {
		var cpuProfile = ""
		config.CpuProfile = &cpuProfile
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
				split := strings.Split(jsonTag, ",")
				outgoingTag = split[0]
			}
		}
		if outgoingTag == "" || outgoingTag == "-" {
			outgoingTag = structType.Field(i).Name
		}
		outgoingTag = strings.ToLower(outgoingTag)
		if structField.IsNil() {
			if strings.ToLower(outgoingTag) != "proxyaddress" &&
				strings.ToLower(outgoingTag) != "cpuprofile" {
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
				if *v == "" &&
					strings.ToLower(outgoingTag) != "proxyaddress" &&
					strings.ToLower(outgoingTag) != "cpuprofile" {
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
				split := strings.Split(jsonTag, ",")
				outgoingTag = split[0]
			}
		}
		if outgoingTag == "" || outgoingTag == "-" {
			outgoingTag = structType.Field(i).Name
		}
		outgoingTag = strings.ToLower(outgoingTag)
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
