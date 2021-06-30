package okex_usdtspot

import (
	"context"
	"fmt"
	"strconv"
	"testing"
)

func TestAPI_GetInstruments(t *testing.T) {
	api, err := NewAPI(&Credentials{}, "socks5://127.0.0.1:1083")
	if err != nil {
		t.Fatal(err)
	}
	var instruments []Instrument
	instruments, err = api.GetInstruments(context.Background())
	if err != nil {
		return
	}
	tickSizes := make(map[string]float64)
	stepSizes := make(map[string]float64)
	minSizes := make(map[string]float64)
	for _, instrument := range instruments {
		if len(instrument.InstrumentId) < 5 {
			continue
		}
		if instrument.InstrumentId[len(instrument.InstrumentId)-5:] != "-USDT" {
			continue
		}
		tickSizes[instrument.InstrumentId] = instrument.TickSize
		stepSizes[instrument.InstrumentId] = instrument.SizeIncrement
		minSizes[instrument.InstrumentId] = instrument.MinSize
	}
	str := "var TickSizes = map[string]float64{\n"
	for symbol, value := range tickSizes {
		str += fmt.Sprintf("  \"%s\": %s,\n", symbol, strconv.FormatFloat(value, 'f', -1, 64))
	}
	str += "}\n\n"
	str += "var StepSizes = map[string]float64{\n"
	for symbol, value := range stepSizes {
		str += fmt.Sprintf("  \"%s\": %s,\n", symbol, strconv.FormatFloat(value, 'f', -1, 64))
	}
	str += "}\n\n"
	str += "var MinSizes = map[string]float64{\n"
	for symbol, value := range minSizes {
		str += fmt.Sprintf("  \"%s\": %s,\n", symbol, strconv.FormatFloat(value, 'f', -1, 64))
	}
	str += "}\n\n"
	fmt.Printf("%s", str)
	return
}
