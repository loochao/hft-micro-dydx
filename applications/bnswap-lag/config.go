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

	ApiKey    *string `yaml:"apiKey,omitempty"`
	ApiSecret *string `yaml:"apiSecret,omitempty"`

	Leverage       *int    `yaml:"leverage,omitempty"`
	ChangeLeverage *bool   `yaml:"changeLeverage,omitempty"`
	MarginType     *string `yaml:"marginType,omitempty"`

	LoopInterval    *time.Duration `yaml:"loopInterval,omitempty"`
	LogInterval     *time.Duration `yaml:"logInterval,omitempty"`
	PullInterval    *time.Duration `yaml:"pullInterval,omitempty"`
	RequestInterval *time.Duration `yaml:"requestInterval,omitempty"`
	PositionMaxAge  *time.Duration `yaml:"positionMaxAge,omitempty"`

	Symbols          []string `yaml:"symbols,omitempty"`
	TradeSymbolIndex *int     `yaml:"tradeSymbolIndex,omitempty"`

	PullBarsInterval      *time.Duration `yaml:"pullBarsInterval,omitempty"`
	PullBarsRetryInterval *time.Duration `yaml:"pullBarsRetryInterval,omitempty"`
	BarsLookback          *int           `yaml:"barsLookback,omitempty"`

	QuantileOffset *float64 `yaml:"quantileOffset,omitempty"`
	TimeDecay      *float64 `yaml:"timeDecay,omitempty"`
	TimeBias       *float64 `yaml:"timeBias,omitempty"`
	DepthBatchSize *int     `yaml:"depthBatchSize,omitempty"`
	DirWindow      *int     `yaml:"dirWindow,omitempty"`

	StartValue        *float64           `yaml:"startValue,omitempty"`
	EnterThreshold    *float64           `yaml:"enterThreshold,omitempty"`
	LeaveThreshold    *float64           `yaml:"leaveThreshold,omitempty"`
	EnterFreePct      *float64           `yaml:"enterFreePct,omitempty"`
	EnterMinimalStep  *float64           `yaml:"enterMinimalStep,omitempty"`
	EnterTargetFactor *float64           `yaml:"enterTargetFactor,omitempty"`
	StartValues       map[string]float64 `yaml:"startValues,omitempty"`

	OrderTimeout       *time.Duration `yaml:"orderTimeout,omitempty"`
	OrderSilent        *time.Duration `yaml:"orderSilent,omitempty"`
	SymbolLoopInterval *time.Duration `yaml:"symbolLoopInterval,omitempty"`
	EnterSilent        *time.Duration `yaml:"enterSilent,omitempty"`
	RestartSilent      *time.Duration `yaml:"restartSilent,omitempty"`
	HttpSilent         *time.Duration `yaml:"httpSilent,omitempty"`
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
