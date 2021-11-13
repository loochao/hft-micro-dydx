package okexv5_usdtspot

import (
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"sort"
	"strconv"
	"testing"
)

func TestAPI_GetInstruments(t *testing.T) {
	api, err := NewAPI(&Credentials{}, os.Getenv("OK_PROXY"))
	if err != nil {
		t.Fatal(err)
	}
	var instruments []Instrument
	instruments, err = api.GetInstruments(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	tickSizes := make(map[string]float64)
	stepSizes := make(map[string]float64)
	minSizes := make(map[string]float64)
	tickPrecisions := make(map[string]int)
	stepPrecisions := make(map[string]int)
	ids := make([]string, 0)
	for _, instrument := range instruments {
		if instrument.State != "live" {
			continue
		}
		if len(instrument.InstId) < 5 {
			continue
		}
		if instrument.InstId[len(instrument.InstId)-5:] != "-USDT" {
			continue
		}
		tickSizes[instrument.InstId] = instrument.TickSz
		stepSizes[instrument.InstId] = instrument.LotSz
		minSizes[instrument.InstId] = instrument.MinSz
		tickPrecisions[instrument.InstId] = common.GetFloatPrecision(instrument.TickSz)
		stepPrecisions[instrument.InstId] = common.GetFloatPrecision(instrument.LotSz)
		ids = append(ids, instrument.InstId)
	}
	sort.Strings(ids)
	str := "var TickSizes = map[string]float64{\n"
	for _, symbol := range ids {
		value := tickSizes[symbol]
		str += fmt.Sprintf("  \"%s\": %s,\n", symbol, strconv.FormatFloat(value, 'f', -1, 64))
	}
	str += "}\n\n"
	str += "var StepSizes = map[string]float64{\n"
	for _, symbol := range ids {
		value := stepSizes[symbol]
		str += fmt.Sprintf("  \"%s\": %s,\n", symbol, strconv.FormatFloat(value, 'f', -1, 64))
	}
	str += "}\n\n"
	str += "var MinSizes = map[string]float64{\n"
	for _, symbol := range ids {
		value := minSizes[symbol]
		str += fmt.Sprintf("  \"%s\": %s,\n", symbol, strconv.FormatFloat(value, 'f', -1, 64))
	}
	str += "}\n\n"
	str += "var StepPrecisions = map[string]float64{\n"
	for _, symbol := range ids {
		value := stepPrecisions[symbol]
		str += fmt.Sprintf("  \"%s\": %d,\n", symbol, value)
	}
	str += "}\n\n"
	str += "var TickPrecisions = map[string]float64{\n"
	for _, symbol := range ids {
		value := tickPrecisions[symbol]
		str += fmt.Sprintf("  \"%s\": %d,\n", symbol, value)
	}
	str += "}\n\n"
	fmt.Printf("%s", str)
	return
}

func TestAPI_GetStatus(t *testing.T) {
	logger.Debugf("%s",  os.Getenv("OK_PROXY"))
	api, err := NewAPI(&Credentials{}, os.Getenv("OK_PROXY"))
	if err != nil {
		t.Fatal(err)
	}
	status, err := api.GetStatus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range status {
		logger.Debugf("%v", s)
	}
}
