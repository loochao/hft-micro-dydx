package binance_tusdspot

var TickSizes = map[string]float64{
	"XRPTUSD":  0.0001,
	"ADATUSD":  0.001,
	"PHBTUSD":  0.000001,
	"BTTTUSD":  0.000001,
	"BCHTUSD":  0.1,
	"BTCTUSD":  0.01,
	"ETHTUSD":  0.01,
	"BNBTUSD":  0.1,
	"TRXTUSD":  0.00001,
	"LINKTUSD": 0.01,
	"LTCTUSD":  0.1,
}

var StepSizes = map[string]float64{
	"XRPTUSD":  1,
	"TRXTUSD":  0.1,
	"LINKTUSD": 0.01,
	"LTCTUSD":  0.001,
	"BTCTUSD":  0.00001,
	"ETHTUSD":  0.0001,
	"BNBTUSD":  0.001,
	"ADATUSD":  0.1,
	"BTTTUSD":  1,
	"PHBTUSD":  0.1,
	"BCHTUSD":  0.001,
}

var MinSizes = map[string]float64{
	"PHBTUSD":  0.1,
	"XRPTUSD":  1,
	"ADATUSD":  0.1,
	"TRXTUSD":  0.1,
	"LINKTUSD": 0.01,
	"LTCTUSD":  0.001,
	"BTTTUSD":  1,
	"BTCTUSD":  0.00001,
	"ETHTUSD":  0.0001,
	"BNBTUSD":  0.001,
	"BCHTUSD":  0.001,
}

var MinNotionals = map[string]float64{
	"ADATUSD":  10,
	"BCHTUSD":  10,
	"ETHTUSD":  10,
	"BNBTUSD":  10,
	"XRPTUSD":  10,
	"TRXTUSD":  10,
	"LINKTUSD": 10,
	"LTCTUSD":  10,
	"BTTTUSD":  10,
	"PHBTUSD":  10,
	"BTCTUSD":  10,
}

var MultiplierUps = map[string]float64{
	"TRXTUSD":  5,
	"LINKTUSD": 5,
	"BTTTUSD":  5,
	"BNBTUSD":  5,
	"ETHTUSD":  5,
	"XRPTUSD":  5,
	"ADATUSD":  5,
	"LTCTUSD":  5,
	"PHBTUSD":  5,
	"BCHTUSD":  5,
	"BTCTUSD":  5,
}

var MultiplierDowns = map[string]float64{
	"BTCTUSD":  0.2,
	"BNBTUSD":  0.2,
	"XRPTUSD":  0.2,
	"ADATUSD":  0.2,
	"LINKTUSD": 0.2,
	"PHBTUSD":  0.2,
	"ETHTUSD":  0.2,
	"TRXTUSD":  0.2,
	"LTCTUSD":  0.2,
	"BTTTUSD":  0.2,
	"BCHTUSD":  0.2,
}

var TickPrecisions = map[string]int{
	"TRXTUSD":  5,
	"LTCTUSD":  1,
	"PHBTUSD":  6,
	"BNBTUSD":  1,
	"XRPTUSD":  4,
	"ADATUSD":  3,
	"LINKTUSD": 2,
	"BTTTUSD":  6,
	"BCHTUSD":  1,
	"BTCTUSD":  2,
	"ETHTUSD":  2,
}

var StepPrecisions = map[string]int{
	"PHBTUSD":  1,
	"BCHTUSD":  3,
	"BNBTUSD":  3,
	"XRPTUSD":  0,
	"LTCTUSD":  3,
	"TRXTUSD":  1,
	"LINKTUSD": 2,
	"BTTTUSD":  0,
	"BTCTUSD":  5,
	"ETHTUSD":  4,
	"ADATUSD":  1,
}
