package binance_tusdspot

var TickSizes = map[string]float64{
	"ETHTUSD": 0.01,
	"BTTTUSD": 0.000001,
	"ADATUSD": 0.001,
	"TRXTUSD": 0.00001,
	"LTCTUSD": 0.1,
	"PHBTUSD": 0.000001,
	"BTCTUSD": 0.01,
	"BNBTUSD": 0.1,
	"XRPTUSD": 0.0001,
}

var StepSizes = map[string]float64{
	"BTCTUSD": 0.00001,
	"TRXTUSD": 0.1,
	"LTCTUSD": 0.001,
	"BTTTUSD": 1,
	"PHBTUSD": 0.1,
	"ETHTUSD": 0.0001,
	"BNBTUSD": 0.001,
	"XRPTUSD": 1,
	"ADATUSD": 0.1,
}

var MinSizes = map[string]float64{
	"XRPTUSD": 1,
	"PHBTUSD": 0.1,
	"BTCTUSD": 0.00001,
	"ETHTUSD": 0.0001,
	"BNBTUSD": 0.001,
	"ADATUSD": 0.1,
	"TRXTUSD": 0.1,
	"LTCTUSD": 0.001,
	"BTTTUSD": 1,
}

var MinNotionals = map[string]float64{
	"XRPTUSD": 10,
	"LTCTUSD": 10,
	"BTTTUSD": 10,
	"BTCTUSD": 10,
	"BNBTUSD": 10,
	"TRXTUSD": 10,
	"PHBTUSD": 10,
	"ETHTUSD": 10,
	"ADATUSD": 10,
}

var MultiplierUps = map[string]float64{
	"ETHTUSD": 5,
	"BTTTUSD": 5,
	"PHBTUSD": 5,
	"BTCTUSD": 5,
	"BNBTUSD": 5,
	"XRPTUSD": 5,
	"ADATUSD": 5,
	"TRXTUSD": 5,
	"LTCTUSD": 5,
}

var MultiplierDowns = map[string]float64{
	"ADATUSD": 0.2,
	"BNBTUSD": 0.2,
	"XRPTUSD": 0.2,
	"TRXTUSD": 0.2,
	"LTCTUSD": 0.2,
	"BTTTUSD": 0.2,
	"PHBTUSD": 0.2,
	"BTCTUSD": 0.2,
	"ETHTUSD": 0.2,
}

var TickPrecisions = map[string]int{
	"ADATUSD": 3,
	"TRXTUSD": 5,
	"BTTTUSD": 6,
	"BTCTUSD": 2,
	"ETHTUSD": 2,
	"BNBTUSD": 1,
	"XRPTUSD": 4,
	"LTCTUSD": 1,
	"PHBTUSD": 6,
}

var StepPrecisions = map[string]int{
	"BNBTUSD": 3,
	"XRPTUSD": 0,
	"ADATUSD": 1,
	"BTCTUSD": 5,
	"ETHTUSD": 4,
	"BTTTUSD": 0,
	"PHBTUSD": 1,
	"TRXTUSD": 1,
	"LTCTUSD": 3,
}
