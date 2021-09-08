package binance_busdfuture

var TickSizes = map[string]float64{
	"ETHBUSD":  0.01,
	"BNBBUSD":  0.01,
	"ADABUSD":  0.0001,
	"XRPBUSD":  0.0001,
	"DOGEBUSD": 0.00001,
	"SOLBUSD":  0.001,
	"FTTBUSD":  0.001,
	"BTCBUSD":  0.1,
}

var StepSizes = map[string]float64{
	"ETHBUSD":  0.001,
	"BNBBUSD":  0.01,
	"ADABUSD":  1,
	"XRPBUSD":  0.1,
	"DOGEBUSD": 1,
	"SOLBUSD":  1,
	"FTTBUSD":  0.1,
	"BTCBUSD":  0.001,
}

var MinSizes = map[string]float64{
	"BNBBUSD":  0.01,
	"ADABUSD":  1,
	"XRPBUSD":  0.1,
	"DOGEBUSD": 1,
	"SOLBUSD":  1,
	"FTTBUSD":  0.1,
	"BTCBUSD":  0.001,
	"ETHBUSD":  0.001,
}

var MinNotional = map[string]float64{
	"ADABUSD":  5,
	"XRPBUSD":  5,
	"DOGEBUSD": 5,
	"SOLBUSD":  5,
	"FTTBUSD":  5,
	"BTCBUSD":  5,
	"ETHBUSD":  5,
	"BNBBUSD":  5,
}

var MultiplierUps = map[string]float64{
	"BNBBUSD":  1.05,
	"ADABUSD":  1.05,
	"XRPBUSD":  1.05,
	"DOGEBUSD": 1.05,
	"SOLBUSD":  1.05,
	"FTTBUSD":  1.05,
	"BTCBUSD":  1.05,
	"ETHBUSD":  1.05,
}

var MultiplierDowns = map[string]float64{
	"FTTBUSD":  0.95,
	"BTCBUSD":  0.95,
	"ETHBUSD":  0.95,
	"BNBBUSD":  0.95,
	"ADABUSD":  0.95,
	"XRPBUSD":  0.95,
	"DOGEBUSD": 0.95,
	"SOLBUSD":  0.95,
}

var TickPrecisions = map[string]int{
	"BTCBUSD":  1,
	"ETHBUSD":  2,
	"BNBBUSD":  2,
	"ADABUSD":  4,
	"XRPBUSD":  4,
	"DOGEBUSD": 5,
	"SOLBUSD":  3,
	"FTTBUSD":  3,
}

var StepPrecisions = map[string]int{
	"BNBBUSD":  2,
	"ADABUSD":  0,
	"XRPBUSD":  1,
	"DOGEBUSD": 0,
	"SOLBUSD":  0,
	"FTTBUSD":  1,
	"BTCBUSD":  3,
	"ETHBUSD":  3,
}
