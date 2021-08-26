package binance_tusdspot

var TickSizes = map[string]float64{
	"BTCTUSD": 0.01,
	"BNBTUSD": 0.1,
	"XRPTUSD": 0.0001,
	"LINKTUSD": 0.01,
	"LTCTUSD": 0.1,
	"PHBTUSD": 0.000001,
	"ETHTUSD": 0.01,
	"ADATUSD": 0.001,
	"TRXTUSD": 0.00001,
	"BTTTUSD": 0.000001,
	"BCHTUSD": 0.1,
}

var StepSizes = map[string]float64{
	"ADATUSD": 0.1,
	"LTCTUSD": 0.001,
	"BTTTUSD": 1,
	"BNBTUSD": 0.001,
	"XRPTUSD": 1,
	"TRXTUSD": 0.1,
	"LINKTUSD": 0.01,
	"PHBTUSD": 0.1,
	"BCHTUSD": 0.001,
	"BTCTUSD": 0.00001,
	"ETHTUSD": 0.0001,
}

var MinSizes = map[string]float64{
	"ADATUSD": 0.1,
	"TRXTUSD": 0.1,
	"PHBTUSD": 0.1,
	"BTCTUSD": 0.00001,
	"ETHTUSD": 0.0001,
	"BNBTUSD": 0.001,
	"XRPTUSD": 1,
	"LINKTUSD": 0.01,
	"LTCTUSD": 0.001,
	"BTTTUSD": 1,
	"BCHTUSD": 0.001,
}

var MinNotionals = map[string]float64{
	"BTCTUSD": 10,
	"BNBTUSD": 10,
	"XRPTUSD": 10,
	"LINKTUSD": 10,
	"LTCTUSD": 10,
	"BTTTUSD": 10,
	"PHBTUSD": 10,
	"ETHTUSD": 10,
	"ADATUSD": 10,
	"TRXTUSD": 10,
	"BCHTUSD": 10,
}

var MultiplierUps = map[string]float64{
	"BNBTUSD": 5,
	"XRPTUSD": 5,
	"ADATUSD": 5,
	"TRXTUSD": 5,
	"LTCTUSD": 5,
	"BTCTUSD": 5,
	"ETHTUSD": 5,
	"LINKTUSD": 5,
	"BTTTUSD": 5,
	"PHBTUSD": 5,
	"BCHTUSD": 5,
}

var MultiplierDowns = map[string]float64{
	"LTCTUSD": 0.2,
	"PHBTUSD": 0.2,
	"BTCTUSD": 0.2,
	"BNBTUSD": 0.2,
	"TRXTUSD": 0.2,
	"LINKTUSD": 0.2,
	"BTTTUSD": 0.2,
	"BCHTUSD": 0.2,
	"ETHTUSD": 0.2,
	"XRPTUSD": 0.2,
	"ADATUSD": 0.2,
}

var TickPrecisions = map[string]int{
	"BTCTUSD": 2,
	"ETHTUSD": 2,
	"BNBTUSD": 1,
	"XRPTUSD": 4,
	"ADATUSD": 3,
	"LINKTUSD": 2,
	"LTCTUSD": 1,
	"BCHTUSD": 1,
	"TRXTUSD": 5,
	"BTTTUSD": 6,
	"PHBTUSD": 6,
}

var StepPrecisions = map[string]int{
	"BTCTUSD": 5,
	"ETHTUSD": 4,
	"XRPTUSD": 0,
	"ADATUSD": 1,
	"LINKTUSD": 2,
	"LTCTUSD": 3,
	"BTTTUSD": 0,
	"BNBTUSD": 3,
	"TRXTUSD": 1,
	"PHBTUSD": 1,
	"BCHTUSD": 3,
}
