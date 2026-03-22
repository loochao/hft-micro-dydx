package main

import (
	"context"
	"encoding/json"
	"flag"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/influx/client"
	"github.com/geometrybase/hft-micro/logger"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"strings"
	"time"
)

func main() {
	var xyGlobalCtx context.Context
	var xyGlobalCancel context.CancelFunc
	var xyInfluxWriter *common.InfluxWriter
	var xyConfig *Config

	configPath := flag.String("config", "", "config path")
	flag.Parse()

	if *configPath == "" {
		logger.Fatal("config is empty")
	}

	configFile, err := ioutil.ReadFile(*configPath)
	if err != nil {
		logger.Fatal(err)
	}
	var config Config
	err = yaml.Unmarshal(configFile, &config)
	if err != nil {
		logger.Fatal(err)
	}
	config.SetDefaultIfNotSet()
	xyConfig = &config

	configStr, err := yaml.Marshal(xyConfig)
	if err != nil {
		logger.Fatal(err)
	}
	logger.Debug("\n\nCONFIG:")
	for _, l := range strings.Split(string(configStr), "\n") {
		logger.Debugf("%s", l)
	}
	logger.Debug("\n\n")

	xyGlobalCtx, xyGlobalCancel = context.WithCancel(context.Background())
	defer xyGlobalCancel()

	if xyConfig.Influx.Address == "" {
		logger.Fatal("miss influx address")
	}

	xyInfluxWriter, err = common.NewInfluxWriter(
		xyGlobalCtx,
		xyConfig.Influx.Address,
		xyConfig.Influx.Username,
		xyConfig.Influx.Password,
		xyConfig.Influx.Database,
		xyConfig.Influx.BatchSize,
	)
	if err != nil {
		logger.Debugf("common.NewInfluxWriter error %v", err)
		return
	}
	defer xyInfluxWriter.Stop()

	updateTimer := time.NewTimer(
		time.Now().Truncate(
			xyConfig.UpdateInterval,
		).Add(
			xyConfig.UpdateInterval,
		).Sub(time.Now()),
	)
	defer updateTimer.Stop()

	xyInfluxClient, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     xyConfig.Influx.Address,
		Username: xyConfig.Influx.Username,
		Password: xyConfig.Influx.Password,
		Timeout:  time.Minute * 5,
	})
	if err != nil {
		logger.Fatal(err)
	}

mainLoop:
	for {
		select {
		case <-xyInfluxWriter.Done():
			xyGlobalCancel()
			break mainLoop
		case <-updateTimer.C:
			updateValues(
				xyInfluxClient,
				xyConfig.Influx,
				xyConfig.ValueField,
				xyConfig.StartValues,
				xyInfluxWriter,
			)
			updateTimer.Reset(xyConfig.UpdateInterval)
			break
		}
	}

	logger.Debugf("stop waiting 5s")
	<-time.After(time.Second * 5)
	logger.Debugf("exit 0")
}

func updateValues(xyInfluxClient client.Client, influxConfig common.InfluxSettings, valueField string, startValues map[string]float64, writer *common.InfluxWriter) {
	queryStr := `select last("` + valueField + `") from "accounts" group by "name";`
	query := client.NewQuery(queryStr, influxConfig.Database, "ns")
	resp, err := xyInfluxClient.Query(query)
	if err != nil {
		logger.Debugf("xyInfluxClient.Query error %v", err)
		return
	}

	values := make(map[string]float64)
	for _, r := range resp.Results {
		for _, s := range r.Series {
			jn := s.Values[0][1].(json.Number)
			jf, err := jn.Float64()
			if err != nil {
				logger.Debugf("jn.Float64() error %v", err)
				continue
			}
			logger.Debugf("%s %f", s.Tags["name"], jf)
			values[s.Tags["name"]] = jf
		}
	}

	startSum := 0.0
	currentSum := 0.0
	fields := make(map[string]interface{})
	hasAll := true
	for name, sv := range startValues {
		if ev, ok := values[name]; ok {
			startSum += sv
			currentSum += ev
			fields["account_value_"+name] = ev
			if sv != 0 {
				fields["account_networth_"+name] = ev / sv
			}
		} else {
			hasAll = false
			logger.Debugf("miss value for %s", name)
		}
	}
	if startSum != 0 && hasAll {
		fields["total_start_value"] = startSum
		fields["total_current_value"] = currentSum
		fields["total_networth"] = currentSum / startSum
		logger.Debugf("START %f END %f NETWORTH %f", startSum, currentSum, currentSum / startSum)
	}

	if len(fields) > 0 {
		pt, err := client.NewPoint(
			influxConfig.Measurement,
			map[string]string{
				"period": valueField,
			},
			fields,
			time.Now().UTC(),
		)
		if err != nil {
			logger.Debugf("client.NewPoint error %v", err)
		} else {
			err = writer.PushPoint(pt)
			if err != nil {
				logger.Debugf("xyInfluxWriter.PushPoint error %v", err)
			}
		}
	}

}
