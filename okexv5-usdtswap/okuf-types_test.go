package okexv5_usdtswap

import (
	"encoding/json"
	"github.com/geometrybase/hft-micro/logger"
	"testing"
)

func TestParseAccount(t *testing.T) {
	msg := []byte(`{"arg":{"channel":"account","uid":"1483770842722304"},"data":[{"adjEq":"","details":[{"availBal":"","availEq":"200.01283421173144","cashBal":"211.14899579926876","ccy":"USDT","coinUsdPrice":"1.00018","crossLiab":"","disEq":"211.05060688104533","eq":"211.01262460861577","eqUsd":"211.05060688104533","frozenBal":"10.999790396884334","interest":"","isoEq":"0","isoLiab":"","isoUpl":"0","liab":"","maxLoan":"","mgnRatio":"620.8197866862248","notionalLever":"0.1563857672111321","ordFrozen":"0","stgyEq":"0","twap":"0","uTime":"1636819211491","upl":"-0.1363711906530014"},{"availBal":"","availEq":"0.499777","cashBal":"0.499777","ccy":"ATOM","coinUsdPrice":"33.015","crossLiab":"","disEq":"13.200110124","eq":"0.499777","eqUsd":"16.500137655","frozenBal":"0","interest":"","isoEq":"0","isoLiab":"","isoUpl":"0","liab":"","maxLoan":"","mgnRatio":"","notionalLever":"0","ordFrozen":"0","stgyEq":"0","twap":"0","uTime":"1636709429372","upl":"0"},{"availBal":"","availEq":"0.00006","cashBal":"0.00006","ccy":"BAL","coinUsdPrice":"24.278","crossLiab":"","disEq":"0.00072834","eq":"0.00006","eqUsd":"0.00145668","frozenBal":"0","interest":"","isoEq":"0","isoLiab":"","isoUpl":"0","liab":"","maxLoan":"","mgnRatio":"","notionalLever":"0","ordFrozen":"0","stgyEq":"0","twap":"0","uTime":"1625739325118","upl":"0"},{"availBal":"","availEq":"0.0004","cashBal":"0.0004","ccy":"XRP","coinUsdPrice":"1.20113","crossLiab":"","disEq":"0.0004083842","eq":"0.0004","eqUsd":"0.000480452","frozenBal":"0","interest":"","isoEq":"0","isoLiab":"","isoUpl":"0","liab":"","maxLoan":"","mgnRatio":"","notionalLever":"0","ordFrozen":"0","stgyEq":"0","twap":"0","uTime":"1625739325118","upl":"0"},{"availBal":"","availEq":"0.00044","cashBal":"0.00044","ccy":"ONT","coinUsdPrice":"1.0857","crossLiab":"","disEq":"0.000238854","eq":"0.00044","eqUsd":"0.000477708","frozenBal":"0","interest":"","isoEq":"0","isoLiab":"","isoUpl":"0","liab":"","maxLoan":"","mgnRatio":"","notionalLever":"0","ordFrozen":"0","stgyEq":"0","twap":"0","uTime":"1625739325118","upl":"0"},{"availBal":"","availEq":"0.00831","cashBal":"0.00831","ccy":"IOST","coinUsdPrice":"0.047957","crossLiab":"","disEq":"0.000199261335","eq":"0.00831","eqUsd":"0.00039852267","frozenBal":"0","interest":"","isoEq":"0","isoLiab":"","isoUpl":"0","liab":"","maxLoan":"","mgnRatio":"","notionalLever":"0","ordFrozen":"0","stgyEq":"0","twap":"0","uTime":"1625739325065","upl":"0"},{"availBal":"","availEq":"0.00006","cashBal":"0.00006","ccy":"XTZ","coinUsdPrice":"5.8648","crossLiab":"","disEq":"0.0002815104","eq":"0.00006","eqUsd":"0.000351888","frozenBal":"0","interest":"","isoEq":"0","isoLiab":"","isoUpl":"0","liab":"","maxLoan":"","mgnRatio":"","notionalLever":"0","ordFrozen":"0","stgyEq":"0","twap":"0","uTime":"1625739325118","upl":"0"},{"availBal":"","availEq":"0.003","cashBal":"0.003","ccy":"TRX","coinUsdPrice":"0.11257","crossLiab":"","disEq":"0.0002870535","eq":"0.003","eqUsd":"0.00033771","frozenBal":"0","interest":"","isoEq":"0","isoLiab":"","isoUpl":"0","liab":"","maxLoan":"","mgnRatio":"","notionalLever":"0","ordFrozen":"0","stgyEq":"0","twap":"0","uTime":"1625739325065","upl":"0"},{"availBal":"","availEq":"0.00000051","cashBal":"0.00000051","ccy":"LTC","coinUsdPrice":"260.84","crossLiab":"","disEq":"0.00012637698","eq":"0.00000051","eqUsd":"0.0001330284","frozenBal":"0","interest":"","isoEq":"0","isoLiab":"","isoUpl":"0","liab":"","maxLoan":"","mgnRatio":"","notionalLever":"0","ordFrozen":"0","stgyEq":"0","twap":"0","uTime":"1625739325118","upl":"0"},{"availBal":"","availEq":"0.00002656","cashBal":"0.00002656","ccy":"CRV","coinUsdPrice":"4.2824","crossLiab":"","disEq":"0.000056870272","eq":"0.00002656","eqUsd":"0.000113740544","frozenBal":"0","interest":"","isoEq":"0","isoLiab":"","isoUpl":"0","liab":"","maxLoan":"","mgnRatio":"","notionalLever":"0","ordFrozen":"0","stgyEq":"0","twap":"0","uTime":"1625739325065","upl":"0"},{"availBal":"","availEq":"0.000001","cashBal":"0.000001","ccy":"AVAX","coinUsdPrice":"93.525","crossLiab":"","disEq":"0.0000841725","eq":"0.000001","eqUsd":"0.000093525","frozenBal":"0","interest":"","isoEq":"0","isoLiab":"","isoUpl":"0","liab":"","maxLoan":"","mgnRatio":"","notionalLever":"0","ordFrozen":"0","stgyEq":"0","twap":"0","uTime":"1625739325118","upl":"0"},{"availBal":"","availEq":"0.0000004","cashBal":"0.0000004","ccy":"DASH","coinUsdPrice":"228.9","crossLiab":"","disEq":"0.000073248","eq":"0.0000004","eqUsd":"0.00009156","frozenBal":"0","interest":"","isoEq":"0","isoLiab":"","isoUpl":"0","liab":"","maxLoan":"","mgnRatio":"","notionalLever":"0","ordFrozen":"0","stgyEq":"0","twap":"0","uTime":"1625739325065","upl":"0"},{"availBal":"","availEq":"0.0000299","cashBal":"0.0000299","ccy":"ALGO","coinUsdPrice":"2.1067","crossLiab":"","disEq":"0.000050392264","eq":"0.0000299","eqUsd":"0.00006299033","frozenBal":"0","interest":"","isoEq":"0","isoLiab":"","isoUpl":"0","liab":"","maxLoan":"","mgnRatio":"","notionalLever":"0","ordFrozen":"0","stgyEq":"0","twap":"0","uTime":"1625739325065","upl":"0"},{"availBal":"","availEq":"0.00000013","cashBal":"0.00000013","ccy":"OKB","coinUsdPrice":"27.409","crossLiab":"","disEq":"0.000003206853","eq":"0.00000013","eqUsd":"0.00000356317","frozenBal":"0","interest":"","isoEq":"0","isoLiab":"","isoUpl":"0","liab":"","maxLoan":"","mgnRatio":"","notionalLever":"0","ordFrozen":"0","stgyEq":"0","twap":"0","uTime":"1625739547785","upl":"0"},{"availBal":"","availEq":"0.000001","cashBal":"0.000001","ccy":"ZRX","coinUsdPrice":"1.3046","crossLiab":"","disEq":"0.0000006523","eq":"0.000001","eqUsd":"0.0000013046","frozenBal":"0","interest":"","isoEq":"0","isoLiab":"","isoUpl":"0","liab":"","maxLoan":"","mgnRatio":"","notionalLever":"0","ordFrozen":"0","stgyEq":"0","twap":"0","uTime":"1625739325118","upl":"0"},{"availBal":"","availEq":"0.000001","cashBal":"0.000001","ccy":"FLM","coinUsdPrice":"0.674428117","crossLiab":"","disEq":"0.0000003372140585","eq":"0.000001","eqUsd":"0.000000674428117","frozenBal":"0","interest":"","isoEq":"0","isoLiab":"","isoUpl":"0","liab":"","maxLoan":"","mgnRatio":"","notionalLever":"0","ordFrozen":"0","stgyEq":"0","twap":"0","uTime":"1625739325118","upl":"0"},{"availBal":"","availEq":"0.000001","cashBal":"0.000001","ccy":"CVC","coinUsdPrice":"0.4680395532","crossLiab":"","disEq":"0.0000002340197766","eq":"0.000001","eqUsd":"0.0000004680395532","frozenBal":"0","interest":"","isoEq":"0","isoLiab":"","isoUpl":"0","liab":"","maxLoan":"","mgnRatio":"","notionalLever":"0","ordFrozen":"0","stgyEq":"0","twap":"0","uTime":"1625739325118","upl":"0"},{"availBal":"","availEq":"0.000074","cashBal":"0.000074","ccy":"WXT","coinUsdPrice":"0.00603302544","crossLiab":"","disEq":"0","eq":"0.000074","eqUsd":"0.0000004464438826","frozenBal":"0","interest":"","isoEq":"0","isoLiab":"","isoUpl":"0","liab":"","maxLoan":"","mgnRatio":"","notionalLever":"0","ordFrozen":"0","stgyEq":"0","twap":"0","uTime":"1625739325065","upl":"0"},{"availBal":"","availEq":"0.000001","cashBal":"0.000001","ccy":"DOGE","coinUsdPrice":"0.259786","crossLiab":"","disEq":"0.0000002338074","eq":"0.000001","eqUsd":"0.000000259786","frozenBal":"0","interest":"","isoEq":"0","isoLiab":"","isoUpl":"0","liab":"","maxLoan":"","mgnRatio":"","notionalLever":"0","ordFrozen":"0","stgyEq":"0","twap":"0","uTime":"1625739325065","upl":"0"},{"availBal":"","availEq":"0.0005745","cashBal":"0.0005745","ccy":"IPC","coinUsdPrice":"0.000300033","crossLiab":"","disEq":"0","eq":"0.0005745","eqUsd":"0.0000001723689585","frozenBal":"0","interest":"","isoEq":"0","isoLiab":"","isoUpl":"0","liab":"","maxLoan":"","mgnRatio":"","notionalLever":"0","ordFrozen":"0","stgyEq":"0","twap":"0","uTime":"1625739325118","upl":"0"}],"imr":"","isoEq":"0","mgnRatio":"","mmr":"","notionalUsd":"","ordFroz":"","totalEq":"227.55474922982586","uTime":"1636821759530"}]}`)
	ads := make([]AccountData, 0)
	cc := &CommonCapture{}
	err := json.Unmarshal(msg, cc)
	if err != nil {
		t.Fatal(err)
	}
	err = json.Unmarshal(cc.Data, &ads)
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", ads)
}

func TestParseOrder(t *testing.T) {
	msg := []byte(`
	{
		"accFillSz": "0",
		"amendResult": "",
		"avgPx": "0",
		"cTime": "1636718644502",
		"category": "normal",
		"ccy": "",
		"clOrdId": "",
		"code": "0",
		"execType": "",
		"fee": "0",
		"feeCcy": "USDT",
		"fillFee": "0",
		"fillFeeCcy": "",
		"fillNotionalUsd": "",
		"fillPx": "",
		"fillSz": "0",
		"fillTime": "",
		"instId": "ATOM-USDT",
		"instType": "SPOT",
		"lever": "0",
		"msg": "",
		"notionalUsd": "17.493419453650002",
		"ordId": "379360722823835652",
		"ordType": "limit",
		"pnl": "0",
		"posSide": "",
		"px": "35",
		"rebate": "0",
		"rebateCcy": "ATOM",
		"reqId": "",
		"side": "sell",
		"slOrdPx": "",
		"slTriggerPx": "",
		"slTriggerPxType": "last",
		"state": "live",
		"sz": "0.499777",
		"tag": "",
		"tdMode": "cash",
		"tgtCcy": "",
		"tpOrdPx": "",
		"tpTriggerPx": "",
		"tpTriggerPxType": "last",
		"tradeId": "",
		"uTime": "1636718644502"
	}`)
	order := &Order{}
	err := json.Unmarshal(msg, order)
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("%v", order)
}
