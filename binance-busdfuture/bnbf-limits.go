package binance_busdfuture

var TickSizes = map[string]float64{
	"BTCBUSD": 0.1,
	"ETHBUSD": 0.01,
}

var StepSizes = map[string]float64{
	"ETHBUSD": 0.001,
	"BTCBUSD": 0.001,
}

var MinSizes = map[string]float64{
	"BTCBUSD": 0.001,
	"ETHBUSD": 0.001,
}

var MinNotional = map[string]float64{
	"BTCBUSD": 5,
	"ETHBUSD": 5,
}

var MultiplierUps = map[string]float64{
	"BTCBUSD": 1.05,
	"ETHBUSD": 1.15,
}

var MultiplierDowns = map[string]float64{
	"BTCBUSD": 0.95,
	"ETHBUSD": 0.85,
}
