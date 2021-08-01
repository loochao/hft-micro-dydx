package binance_busdfuture

var TickSizes = map[string]float64{
	"BTCBUSD": 0.1,
	"ETHBUSD": 0.01,
	"BNBBUSD": 0.01,
}

var StepSizes = map[string]float64{
	"BNBBUSD": 0.01,
	"BTCBUSD": 0.001,
	"ETHBUSD": 0.001,
}

var MinSizes = map[string]float64{
	"BTCBUSD": 0.001,
	"ETHBUSD": 0.001,
	"BNBBUSD": 0.01,
}

var MinNotional = map[string]float64{
	"BTCBUSD": 5,
	"ETHBUSD": 5,
	"BNBBUSD": 5,
}

var MultiplierUps = map[string]float64{
	"BTCBUSD": 1.1,
	"ETHBUSD": 1.1,
	"BNBBUSD": 1.05,
}

var MultiplierDowns = map[string]float64{
	"BNBBUSD": 0.95,
	"BTCBUSD": 0.9,
	"ETHBUSD": 0.9,
}
