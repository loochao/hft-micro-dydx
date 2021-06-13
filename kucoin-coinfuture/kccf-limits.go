package kucoin_coinfuture

var Multipliers = map[string]float64{
	"DOTUSDM": -1,
	"ETHUSDM": -1,
	"XBTMM21": -1,
	"XBTMU21": -1,
	"XBTUSDM": -1,
	"XRPUSDM": -1,
}

var TickSizes = map[string]float64{
	"DOTUSDM": 0.001,
	"ETHUSDM": 0.05,
	"XBTMM21": 1,
	"XBTMU21": 1,
	"XBTUSDM": 1,
	"XRPUSDM": 0.0001,
}

var LotSizes = map[string]float64{
	"DOTUSDM": 1,
	"ETHUSDM": 1,
	"XBTMM21": 1,
	"XBTMU21": 1,
	"XBTUSDM": 1,
	"XRPUSDM": 1,
}

var MaxPrices = map[string]float64{
	"DOTUSDM": 1000000,
	"ETHUSDM": 1000000,
	"XBTMM21": 1000000,
	"XBTMU21": 1000000,
	"XBTUSDM": 1000000,
	"XRPUSDM": 1000000,
}

var MaxOrderSizes = map[string]float64{
	"DOTUSDM": 1000000,
	"ETHUSDM": 1000000,
	"XBTMM21": 10000000,
	"XBTMU21": 10000000,
	"XBTUSDM": 10000000,
	"XRPUSDM": 1000000,
}
