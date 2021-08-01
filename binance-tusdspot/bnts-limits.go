package binance_tusdspot

var TickSizes = map[string]float64{
	"ADATUSD": 0.0001,
	"LINKTUSD": 0.001,
	"LTCTUSD": 0.01,
	"BCHTUSD": 0.01,
	"ETHTUSD": 0.01,
	"XRPTUSD": 0.0001,
	"EOSTUSD": 0.0001,
	"TRXTUSD": 0.00001,
	"BTTTUSD": 0.0000001,
	"PHBTUSD": 0.00001,
	"BTCTUSD": 0.01,
	"BNBTUSD": 0.01,
}

var StepSizes = map[string]float64{
	"EOSTUSD": 0.01,
	"TRXTUSD": 0.1,
	"LTCTUSD": 0.00001,
	"PHBTUSD": 0.1,
	"BTCTUSD": 0.000001,
	"ETHTUSD": 0.00001,
	"XRPTUSD": 0.01,
	"BTTTUSD": 1,
	"BCHTUSD": 0.00001,
	"BNBTUSD": 0.0001,
	"ADATUSD": 0.01,
	"LINKTUSD": 0.001,
}

var MinSizes = map[string]float64{
	"LINKTUSD": 0.001,
	"LTCTUSD": 0.00001,
	"BTCTUSD": 0.000001,
	"ETHTUSD": 0.00001,
	"BNBTUSD": 0.0001,
	"XRPTUSD": 0.01,
	"TRXTUSD": 0.1,
	"EOSTUSD": 0.01,
	"ADATUSD": 0.01,
	"BTTTUSD": 1,
	"PHBTUSD": 0.1,
	"BCHTUSD": 0.00001,
}

var MinNotionals = map[string]float64{
	"ETHTUSD": 10,
	"XRPTUSD": 10,
	"ADATUSD": 10,
	"LTCTUSD": 10,
	"BCHTUSD": 10,
	"BTCTUSD": 10,
	"BNBTUSD": 10,
	"EOSTUSD": 10,
	"TRXTUSD": 10,
	"LINKTUSD": 10,
	"BTTTUSD": 10,
	"PHBTUSD": 10,
}

var MultiplierUps = map[string]float64{
	"EOSTUSD": 5,
	"TRXTUSD": 5,
	"BTTTUSD": 5,
	"PHBTUSD": 5,
	"BTCTUSD": 5,
	"ETHTUSD": 5,
	"XRPTUSD": 5,
	"LTCTUSD": 5,
	"BCHTUSD": 5,
	"BNBTUSD": 5,
	"ADATUSD": 5,
	"LINKTUSD": 5,
}

var MultiplierDowns = map[string]float64{
	"ADATUSD": 0.2,
	"LINKTUSD": 0.2,
	"BCHTUSD": 0.2,
	"BTCTUSD": 0.2,
	"ETHTUSD": 0.2,
	"BNBTUSD": 0.2,
	"XRPTUSD": 0.2,
	"EOSTUSD": 0.2,
	"TRXTUSD": 0.2,
	"LTCTUSD": 0.2,
	"BTTTUSD": 0.2,
	"PHBTUSD": 0.2,
}
