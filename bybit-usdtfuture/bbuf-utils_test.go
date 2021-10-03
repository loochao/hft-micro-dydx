package bybit_usdtfuture

import (
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
	"time"
)

func BenchmarkUpdateOrderBook(b *testing.B) {
	messages := `{"topic":"orderBookL2_25.XTZUSDT","type":"snapshot","data":{"order_book":[{"price":"2.393","symbol":"XTZUSDT","id":"23930","side":"Buy","size":21442.3},{"price":"2.394","symbol":"XTZUSDT","id":"23940","side":"Buy","size":994.89996},{"price":"2.395","symbol":"XTZUSDT","id":"23950","side":"Buy","size":1808.5},{"price":"2.396","symbol":"XTZUSDT","id":"23960","side":"Buy","size":20826.3},{"price":"2.397","symbol":"XTZUSDT","id":"23970","side":"Buy","size":951.7},{"price":"2.398","symbol":"XTZUSDT","id":"23980","side":"Buy","size":21589.2},{"price":"2.399","symbol":"XTZUSDT","id":"23990","side":"Buy","size":536.2},{"price":"2.400","symbol":"XTZUSDT","id":"24000","side":"Buy","size":1116.6},{"price":"2.401","symbol":"XTZUSDT","id":"24010","side":"Buy","size":21816.201},{"price":"2.402","symbol":"XTZUSDT","id":"24020","side":"Buy","size":228.50002},{"price":"2.403","symbol":"XTZUSDT","id":"24030","side":"Buy","size":771.3},{"price":"2.404","symbol":"XTZUSDT","id":"24040","side":"Buy","size":22017.898},{"price":"2.405","symbol":"XTZUSDT","id":"24050","side":"Buy","size":4574.6997},{"price":"2.406","symbol":"XTZUSDT","id":"24060","side":"Buy","size":579.2},{"price":"2.407","symbol":"XTZUSDT","id":"24070","side":"Buy","size":1842.3},{"price":"2.408","symbol":"XTZUSDT","id":"24080","side":"Buy","size":6936.1},{"price":"2.409","symbol":"XTZUSDT","id":"24090","side":"Buy","size":1956.9},{"price":"2.410","symbol":"XTZUSDT","id":"24100","side":"Buy","size":1408.7},{"price":"2.411","symbol":"XTZUSDT","id":"24110","side":"Buy","size":870.3},{"price":"2.412","symbol":"XTZUSDT","id":"24120","side":"Buy","size":1330.4},{"price":"2.413","symbol":"XTZUSDT","id":"24130","side":"Buy","size":938.2},{"price":"2.414","symbol":"XTZUSDT","id":"24140","side":"Buy","size":2278.8},{"price":"2.415","symbol":"XTZUSDT","id":"24150","side":"Buy","size":7831.3},{"price":"2.416","symbol":"XTZUSDT","id":"24160","side":"Buy","size":6020.6},{"price":"2.417","symbol":"XTZUSDT","id":"24170","side":"Buy","size":1851.2},{"price":"2.418","symbol":"XTZUSDT","id":"24180","side":"Sell","size":4622.8},{"price":"2.419","symbol":"XTZUSDT","id":"24190","side":"Sell","size":3106.2},{"price":"2.420","symbol":"XTZUSDT","id":"24200","side":"Sell","size":6803.1},{"price":"2.421","symbol":"XTZUSDT","id":"24210","side":"Sell","size":10018.2},{"price":"2.422","symbol":"XTZUSDT","id":"24220","side":"Sell","size":3350.4},{"price":"2.423","symbol":"XTZUSDT","id":"24230","side":"Sell","size":1992},{"price":"2.424","symbol":"XTZUSDT","id":"24240","side":"Sell","size":1870.7001},{"price":"2.425","symbol":"XTZUSDT","id":"24250","side":"Sell","size":1350.5},{"price":"2.426","symbol":"XTZUSDT","id":"24260","side":"Sell","size":1589.4},{"price":"2.427","symbol":"XTZUSDT","id":"24270","side":"Sell","size":700.49994},{"price":"2.428","symbol":"XTZUSDT","id":"24280","side":"Sell","size":726.9},{"price":"2.429","symbol":"XTZUSDT","id":"24290","side":"Sell","size":654.39996},{"price":"2.430","symbol":"XTZUSDT","id":"24300","side":"Sell","size":2431.3},{"price":"2.431","symbol":"XTZUSDT","id":"24310","side":"Sell","size":3012.7002},{"price":"2.432","symbol":"XTZUSDT","id":"24320","side":"Sell","size":21965.201},{"price":"2.433","symbol":"XTZUSDT","id":"24330","side":"Sell","size":651.7},{"price":"2.434","symbol":"XTZUSDT","id":"24340","side":"Sell","size":1721.1001},{"price":"2.435","symbol":"XTZUSDT","id":"24350","side":"Sell","size":21622.3},{"price":"2.436","symbol":"XTZUSDT","id":"24360","side":"Sell","size":593.5},{"price":"2.437","symbol":"XTZUSDT","id":"24370","side":"Sell","size":2055.1},{"price":"2.438","symbol":"XTZUSDT","id":"24380","side":"Sell","size":22218.5},{"price":"2.439","symbol":"XTZUSDT","id":"24390","side":"Sell","size":1596.9999},{"price":"2.440","symbol":"XTZUSDT","id":"24400","side":"Sell","size":1069.8},{"price":"2.441","symbol":"XTZUSDT","id":"24410","side":"Sell","size":21004.2},{"price":"2.442","symbol":"XTZUSDT","id":"24420","side":"Sell","size":20491.299}]},"cross_seq":"1168956791","timestamp_e6":"1626652978974501"}
{"topic":"orderBookL2_25.XTZUSDT","type":"delta","data":{"delete":[],"update":[{"price":"2.415","symbol":"XTZUSDT","id":"24150","side":"Buy","size":7960.1},{"price":"2.420","symbol":"XTZUSDT","id":"24200","side":"Sell","size":6929.4}],"insert":[]},"cross_seq":"1168956793","timestamp_e6":"1626652979254438"}
{"topic":"orderBookL2_25.XTZUSDT","type":"delta","data":{"delete":[],"update":[{"price":"2.421","symbol":"XTZUSDT","id":"24210","side":"Sell","size":9891.9}],"insert":[]},"cross_seq":"1168956794","timestamp_e6":"1626652979334447"}
{"topic":"orderBookL2_25.XTZUSDT","type":"delta","data":{"delete":[],"update":[{"price":"2.420","symbol":"XTZUSDT","id":"24200","side":"Sell","size":7184.1}],"insert":[]},"cross_seq":"1168956795","timestamp_e6":"1626652979494846"}
{"topic":"orderBookL2_25.XTZUSDT","type":"delta","data":{"delete":[],"update":[{"price":"2.415","symbol":"XTZUSDT","id":"24150","side":"Buy","size":7831.3}],"insert":[]},"cross_seq":"1168956796","timestamp_e6":"1626652979574391"}
{"topic":"orderBookL2_25.XTZUSDT","type":"delta","data":{"delete":[],"update":[{"price":"2.416","symbol":"XTZUSDT","id":"24160","side":"Buy","size":6487.8},{"price":"2.417","symbol":"XTZUSDT","id":"24170","side":"Buy","size":1591.2999}],"insert":[]},"cross_seq":"1168956799","timestamp_e6":"1626652979635777"}
{"topic":"orderBookL2_25.XTZUSDT","type":"delta","data":{"delete":[],"update":[{"price":"2.415","symbol":"XTZUSDT","id":"24150","side":"Buy","size":7631.3003},{"price":"2.419","symbol":"XTZUSDT","id":"24190","side":"Sell","size":3310.1},{"price":"2.414","symbol":"XTZUSDT","id":"24140","side":"Buy","size":2078.8},{"price":"2.418","symbol":"XTZUSDT","id":"24180","side":"Sell","size":4877.5},{"price":"2.416","symbol":"XTZUSDT","id":"24160","side":"Buy","size":6284},{"price":"2.420","symbol":"XTZUSDT","id":"24200","side":"Sell","size":7129.4}],"insert":[]},"cross_seq":"1168956806","timestamp_e6":"1626652979675472"}
{"topic":"orderBookL2_25.XTZUSDT","type":"delta","data":{"delete":[],"update":[{"price":"2.419","symbol":"XTZUSDT","id":"24190","side":"Sell","size":3510.0999},{"price":"2.417","symbol":"XTZUSDT","id":"24170","side":"Buy","size":1071.5},{"price":"2.416","symbol":"XTZUSDT","id":"24160","side":"Buy","size":2935.7}],"insert":[]},"cross_seq":"1168956811","timestamp_e6":"1626652979715783"}
{"topic":"orderBookL2_25.XTZUSDT","type":"delta","data":{"delete":[],"update":[{"price":"2.415","symbol":"XTZUSDT","id":"24150","side":"Buy","size":4598.8003},{"price":"2.411","symbol":"XTZUSDT","id":"24110","side":"Buy","size":1056},{"price":"2.423","symbol":"XTZUSDT","id":"24230","side":"Sell","size":1513},{"price":"2.414","symbol":"XTZUSDT","id":"24140","side":"Buy","size":2360.7},{"price":"2.421","symbol":"XTZUSDT","id":"24210","side":"Sell","size":9007},{"price":"2.418","symbol":"XTZUSDT","id":"24180","side":"Sell","size":4445.3},{"price":"2.422","symbol":"XTZUSDT","id":"24220","side":"Sell","size":2818.9},{"price":"2.412","symbol":"XTZUSDT","id":"24120","side":"Buy","size":1665.1}],"insert":[]},"cross_seq":"1168956833","timestamp_e6":"1626652979735570"}`
	lines := strings.Split(messages, "\n")
	ob := OrderBook{
		Symbol: "XTZUSDT",
		Bids:   common.Bids{},
		Asks:   common.Asks{},
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, l := range lines[:1] {
			_ = UpdateOrderBook([]byte(l), &ob)
		}
	}
}

func BenchmarkUpdateOrderBookStdJson(b *testing.B) {
	messages := `{"topic":"orderBookL2_25.XTZUSDT","type":"snapshot","data":{"order_book":[{"price":"2.393","symbol":"XTZUSDT","id":"23930","side":"Buy","size":21442.3},{"price":"2.394","symbol":"XTZUSDT","id":"23940","side":"Buy","size":994.89996},{"price":"2.395","symbol":"XTZUSDT","id":"23950","side":"Buy","size":1808.5},{"price":"2.396","symbol":"XTZUSDT","id":"23960","side":"Buy","size":20826.3},{"price":"2.397","symbol":"XTZUSDT","id":"23970","side":"Buy","size":951.7},{"price":"2.398","symbol":"XTZUSDT","id":"23980","side":"Buy","size":21589.2},{"price":"2.399","symbol":"XTZUSDT","id":"23990","side":"Buy","size":536.2},{"price":"2.400","symbol":"XTZUSDT","id":"24000","side":"Buy","size":1116.6},{"price":"2.401","symbol":"XTZUSDT","id":"24010","side":"Buy","size":21816.201},{"price":"2.402","symbol":"XTZUSDT","id":"24020","side":"Buy","size":228.50002},{"price":"2.403","symbol":"XTZUSDT","id":"24030","side":"Buy","size":771.3},{"price":"2.404","symbol":"XTZUSDT","id":"24040","side":"Buy","size":22017.898},{"price":"2.405","symbol":"XTZUSDT","id":"24050","side":"Buy","size":4574.6997},{"price":"2.406","symbol":"XTZUSDT","id":"24060","side":"Buy","size":579.2},{"price":"2.407","symbol":"XTZUSDT","id":"24070","side":"Buy","size":1842.3},{"price":"2.408","symbol":"XTZUSDT","id":"24080","side":"Buy","size":6936.1},{"price":"2.409","symbol":"XTZUSDT","id":"24090","side":"Buy","size":1956.9},{"price":"2.410","symbol":"XTZUSDT","id":"24100","side":"Buy","size":1408.7},{"price":"2.411","symbol":"XTZUSDT","id":"24110","side":"Buy","size":870.3},{"price":"2.412","symbol":"XTZUSDT","id":"24120","side":"Buy","size":1330.4},{"price":"2.413","symbol":"XTZUSDT","id":"24130","side":"Buy","size":938.2},{"price":"2.414","symbol":"XTZUSDT","id":"24140","side":"Buy","size":2278.8},{"price":"2.415","symbol":"XTZUSDT","id":"24150","side":"Buy","size":7831.3},{"price":"2.416","symbol":"XTZUSDT","id":"24160","side":"Buy","size":6020.6},{"price":"2.417","symbol":"XTZUSDT","id":"24170","side":"Buy","size":1851.2},{"price":"2.418","symbol":"XTZUSDT","id":"24180","side":"Sell","size":4622.8},{"price":"2.419","symbol":"XTZUSDT","id":"24190","side":"Sell","size":3106.2},{"price":"2.420","symbol":"XTZUSDT","id":"24200","side":"Sell","size":6803.1},{"price":"2.421","symbol":"XTZUSDT","id":"24210","side":"Sell","size":10018.2},{"price":"2.422","symbol":"XTZUSDT","id":"24220","side":"Sell","size":3350.4},{"price":"2.423","symbol":"XTZUSDT","id":"24230","side":"Sell","size":1992},{"price":"2.424","symbol":"XTZUSDT","id":"24240","side":"Sell","size":1870.7001},{"price":"2.425","symbol":"XTZUSDT","id":"24250","side":"Sell","size":1350.5},{"price":"2.426","symbol":"XTZUSDT","id":"24260","side":"Sell","size":1589.4},{"price":"2.427","symbol":"XTZUSDT","id":"24270","side":"Sell","size":700.49994},{"price":"2.428","symbol":"XTZUSDT","id":"24280","side":"Sell","size":726.9},{"price":"2.429","symbol":"XTZUSDT","id":"24290","side":"Sell","size":654.39996},{"price":"2.430","symbol":"XTZUSDT","id":"24300","side":"Sell","size":2431.3},{"price":"2.431","symbol":"XTZUSDT","id":"24310","side":"Sell","size":3012.7002},{"price":"2.432","symbol":"XTZUSDT","id":"24320","side":"Sell","size":21965.201},{"price":"2.433","symbol":"XTZUSDT","id":"24330","side":"Sell","size":651.7},{"price":"2.434","symbol":"XTZUSDT","id":"24340","side":"Sell","size":1721.1001},{"price":"2.435","symbol":"XTZUSDT","id":"24350","side":"Sell","size":21622.3},{"price":"2.436","symbol":"XTZUSDT","id":"24360","side":"Sell","size":593.5},{"price":"2.437","symbol":"XTZUSDT","id":"24370","side":"Sell","size":2055.1},{"price":"2.438","symbol":"XTZUSDT","id":"24380","side":"Sell","size":22218.5},{"price":"2.439","symbol":"XTZUSDT","id":"24390","side":"Sell","size":1596.9999},{"price":"2.440","symbol":"XTZUSDT","id":"24400","side":"Sell","size":1069.8},{"price":"2.441","symbol":"XTZUSDT","id":"24410","side":"Sell","size":21004.2},{"price":"2.442","symbol":"XTZUSDT","id":"24420","side":"Sell","size":20491.299}]},"cross_seq":"1168956791","timestamp_e6":"1626652978974501"}
{"topic":"orderBookL2_25.XTZUSDT","type":"delta","data":{"delete":[],"update":[{"price":"2.415","symbol":"XTZUSDT","id":"24150","side":"Buy","size":7960.1},{"price":"2.420","symbol":"XTZUSDT","id":"24200","side":"Sell","size":6929.4}],"insert":[]},"cross_seq":"1168956793","timestamp_e6":"1626652979254438"}
{"topic":"orderBookL2_25.XTZUSDT","type":"delta","data":{"delete":[],"update":[{"price":"2.421","symbol":"XTZUSDT","id":"24210","side":"Sell","size":9891.9}],"insert":[]},"cross_seq":"1168956794","timestamp_e6":"1626652979334447"}
{"topic":"orderBookL2_25.XTZUSDT","type":"delta","data":{"delete":[],"update":[{"price":"2.420","symbol":"XTZUSDT","id":"24200","side":"Sell","size":7184.1}],"insert":[]},"cross_seq":"1168956795","timestamp_e6":"1626652979494846"}
{"topic":"orderBookL2_25.XTZUSDT","type":"delta","data":{"delete":[],"update":[{"price":"2.415","symbol":"XTZUSDT","id":"24150","side":"Buy","size":7831.3}],"insert":[]},"cross_seq":"1168956796","timestamp_e6":"1626652979574391"}
{"topic":"orderBookL2_25.XTZUSDT","type":"delta","data":{"delete":[],"update":[{"price":"2.416","symbol":"XTZUSDT","id":"24160","side":"Buy","size":6487.8},{"price":"2.417","symbol":"XTZUSDT","id":"24170","side":"Buy","size":1591.2999}],"insert":[]},"cross_seq":"1168956799","timestamp_e6":"1626652979635777"}
{"topic":"orderBookL2_25.XTZUSDT","type":"delta","data":{"delete":[],"update":[{"price":"2.415","symbol":"XTZUSDT","id":"24150","side":"Buy","size":7631.3003},{"price":"2.419","symbol":"XTZUSDT","id":"24190","side":"Sell","size":3310.1},{"price":"2.414","symbol":"XTZUSDT","id":"24140","side":"Buy","size":2078.8},{"price":"2.418","symbol":"XTZUSDT","id":"24180","side":"Sell","size":4877.5},{"price":"2.416","symbol":"XTZUSDT","id":"24160","side":"Buy","size":6284},{"price":"2.420","symbol":"XTZUSDT","id":"24200","side":"Sell","size":7129.4}],"insert":[]},"cross_seq":"1168956806","timestamp_e6":"1626652979675472"}
{"topic":"orderBookL2_25.XTZUSDT","type":"delta","data":{"delete":[],"update":[{"price":"2.419","symbol":"XTZUSDT","id":"24190","side":"Sell","size":3510.0999},{"price":"2.417","symbol":"XTZUSDT","id":"24170","side":"Buy","size":1071.5},{"price":"2.416","symbol":"XTZUSDT","id":"24160","side":"Buy","size":2935.7}],"insert":[]},"cross_seq":"1168956811","timestamp_e6":"1626652979715783"}
{"topic":"orderBookL2_25.XTZUSDT","type":"delta","data":{"delete":[],"update":[{"price":"2.415","symbol":"XTZUSDT","id":"24150","side":"Buy","size":4598.8003},{"price":"2.411","symbol":"XTZUSDT","id":"24110","side":"Buy","size":1056},{"price":"2.423","symbol":"XTZUSDT","id":"24230","side":"Sell","size":1513},{"price":"2.414","symbol":"XTZUSDT","id":"24140","side":"Buy","size":2360.7},{"price":"2.421","symbol":"XTZUSDT","id":"24210","side":"Sell","size":9007},{"price":"2.418","symbol":"XTZUSDT","id":"24180","side":"Sell","size":4445.3},{"price":"2.422","symbol":"XTZUSDT","id":"24220","side":"Sell","size":2818.9},{"price":"2.412","symbol":"XTZUSDT","id":"24120","side":"Buy","size":1665.1}],"insert":[]},"cross_seq":"1168956833","timestamp_e6":"1626652979735570"}`
	lines := strings.Split(messages, "\n")
	bids := common.Bids{}
	asks := common.Asks{}
	obm := OrderBookMsg{}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, l := range lines {
			_ = json.Unmarshal([]byte(l), &obm)
			if obm.Type == "snapshot" {
				for _, r := range obm.Data.OrderBook {
					if r.Side == "Buy" {
						bids = bids.Update([2]float64{r.Price, r.Size})
					} else if r.Side == "Sell" {
						asks = asks.Update([2]float64{r.Price, r.Size})
					}
				}
			} else {
				for _, r := range obm.Data.Delete {
					if r.Side == "Buy" {
						bids = bids.Update([2]float64{r.Price, 0})
					} else if r.Side == "Sell" {
						asks = asks.Update([2]float64{r.Price, 0})
					}
				}
				for _, r := range obm.Data.Update {
					if r.Side == "Buy" {
						bids = bids.Update([2]float64{r.Price, r.Size})
					} else if r.Side == "Sell" {
						asks = asks.Update([2]float64{r.Price, r.Size})
					}
				}
				for _, r := range obm.Data.Insert {
					if r.Side == "Buy" {
						bids = bids.Update([2]float64{r.Price, r.Size})
					} else if r.Side == "Sell" {
						asks = asks.Update([2]float64{r.Price, r.Size})
					}
				}
			}
		}
	}
}

func TestUpdateOrderBook(t *testing.T) {
	messages := `{"topic":"orderBookL2_25.XTZUSDT","type":"snapshot","data":{"order_book":[{"price":"2.393","symbol":"XTZUSDT","id":"23930","side":"Buy","size":21442.3},{"price":"2.394","symbol":"XTZUSDT","id":"23940","side":"Buy","size":994.89996},{"price":"2.395","symbol":"XTZUSDT","id":"23950","side":"Buy","size":1808.5},{"price":"2.396","symbol":"XTZUSDT","id":"23960","side":"Buy","size":20826.3},{"price":"2.397","symbol":"XTZUSDT","id":"23970","side":"Buy","size":951.7},{"price":"2.398","symbol":"XTZUSDT","id":"23980","side":"Buy","size":21589.2},{"price":"2.399","symbol":"XTZUSDT","id":"23990","side":"Buy","size":536.2},{"price":"2.400","symbol":"XTZUSDT","id":"24000","side":"Buy","size":1116.6},{"price":"2.401","symbol":"XTZUSDT","id":"24010","side":"Buy","size":21816.201},{"price":"2.402","symbol":"XTZUSDT","id":"24020","side":"Buy","size":228.50002},{"price":"2.403","symbol":"XTZUSDT","id":"24030","side":"Buy","size":771.3},{"price":"2.404","symbol":"XTZUSDT","id":"24040","side":"Buy","size":22017.898},{"price":"2.405","symbol":"XTZUSDT","id":"24050","side":"Buy","size":4574.6997},{"price":"2.406","symbol":"XTZUSDT","id":"24060","side":"Buy","size":579.2},{"price":"2.407","symbol":"XTZUSDT","id":"24070","side":"Buy","size":1842.3},{"price":"2.408","symbol":"XTZUSDT","id":"24080","side":"Buy","size":6936.1},{"price":"2.409","symbol":"XTZUSDT","id":"24090","side":"Buy","size":1956.9},{"price":"2.410","symbol":"XTZUSDT","id":"24100","side":"Buy","size":1408.7},{"price":"2.411","symbol":"XTZUSDT","id":"24110","side":"Buy","size":870.3},{"price":"2.412","symbol":"XTZUSDT","id":"24120","side":"Buy","size":1330.4},{"price":"2.413","symbol":"XTZUSDT","id":"24130","side":"Buy","size":938.2},{"price":"2.414","symbol":"XTZUSDT","id":"24140","side":"Buy","size":2278.8},{"price":"2.415","symbol":"XTZUSDT","id":"24150","side":"Buy","size":7831.3},{"price":"2.416","symbol":"XTZUSDT","id":"24160","side":"Buy","size":6020.6},{"price":"2.417","symbol":"XTZUSDT","id":"24170","side":"Buy","size":1851.2},{"price":"2.418","symbol":"XTZUSDT","id":"24180","side":"Sell","size":4622.8},{"price":"2.419","symbol":"XTZUSDT","id":"24190","side":"Sell","size":3106.2},{"price":"2.420","symbol":"XTZUSDT","id":"24200","side":"Sell","size":6803.1},{"price":"2.421","symbol":"XTZUSDT","id":"24210","side":"Sell","size":10018.2},{"price":"2.422","symbol":"XTZUSDT","id":"24220","side":"Sell","size":3350.4},{"price":"2.423","symbol":"XTZUSDT","id":"24230","side":"Sell","size":1992},{"price":"2.424","symbol":"XTZUSDT","id":"24240","side":"Sell","size":1870.7001},{"price":"2.425","symbol":"XTZUSDT","id":"24250","side":"Sell","size":1350.5},{"price":"2.426","symbol":"XTZUSDT","id":"24260","side":"Sell","size":1589.4},{"price":"2.427","symbol":"XTZUSDT","id":"24270","side":"Sell","size":700.49994},{"price":"2.428","symbol":"XTZUSDT","id":"24280","side":"Sell","size":726.9},{"price":"2.429","symbol":"XTZUSDT","id":"24290","side":"Sell","size":654.39996},{"price":"2.430","symbol":"XTZUSDT","id":"24300","side":"Sell","size":2431.3},{"price":"2.431","symbol":"XTZUSDT","id":"24310","side":"Sell","size":3012.7002},{"price":"2.432","symbol":"XTZUSDT","id":"24320","side":"Sell","size":21965.201},{"price":"2.433","symbol":"XTZUSDT","id":"24330","side":"Sell","size":651.7},{"price":"2.434","symbol":"XTZUSDT","id":"24340","side":"Sell","size":1721.1001},{"price":"2.435","symbol":"XTZUSDT","id":"24350","side":"Sell","size":21622.3},{"price":"2.436","symbol":"XTZUSDT","id":"24360","side":"Sell","size":593.5},{"price":"2.437","symbol":"XTZUSDT","id":"24370","side":"Sell","size":2055.1},{"price":"2.438","symbol":"XTZUSDT","id":"24380","side":"Sell","size":22218.5},{"price":"2.439","symbol":"XTZUSDT","id":"24390","side":"Sell","size":1596.9999},{"price":"2.440","symbol":"XTZUSDT","id":"24400","side":"Sell","size":1069.8},{"price":"2.441","symbol":"XTZUSDT","id":"24410","side":"Sell","size":21004.2},{"price":"2.442","symbol":"XTZUSDT","id":"24420","side":"Sell","size":20491.299}]},"cross_seq":"1168956791","timestamp_e6":"1626652978974501"}
{"topic":"orderBookL2_25.XTZUSDT","type":"delta","data":{"delete":[],"update":[{"price":"2.415","symbol":"XTZUSDT","id":"24150","side":"Buy","size":7960.1},{"price":"2.420","symbol":"XTZUSDT","id":"24200","side":"Sell","size":6929.4}],"insert":[]},"cross_seq":"1168956793","timestamp_e6":"1626652979254438"}
{"topic":"orderBookL2_25.XTZUSDT","type":"delta","data":{"delete":[],"update":[{"price":"2.421","symbol":"XTZUSDT","id":"24210","side":"Sell","size":9891.9}],"insert":[]},"cross_seq":"1168956794","timestamp_e6":"1626652979334447"}
{"topic":"orderBookL2_25.XTZUSDT","type":"delta","data":{"delete":[],"update":[{"price":"2.420","symbol":"XTZUSDT","id":"24200","side":"Sell","size":7184.1}],"insert":[]},"cross_seq":"1168956795","timestamp_e6":"1626652979494846"}
{"topic":"orderBookL2_25.XTZUSDT","type":"delta","data":{"delete":[],"update":[{"price":"2.415","symbol":"XTZUSDT","id":"24150","side":"Buy","size":7831.3}],"insert":[]},"cross_seq":"1168956796","timestamp_e6":"1626652979574391"}
{"topic":"orderBookL2_25.XTZUSDT","type":"delta","data":{"delete":[],"update":[{"price":"2.416","symbol":"XTZUSDT","id":"24160","side":"Buy","size":6487.8},{"price":"2.417","symbol":"XTZUSDT","id":"24170","side":"Buy","size":1591.2999}],"insert":[]},"cross_seq":"1168956799","timestamp_e6":"1626652979635777"}
{"topic":"orderBookL2_25.XTZUSDT","type":"delta","data":{"delete":[],"update":[{"price":"2.415","symbol":"XTZUSDT","id":"24150","side":"Buy","size":7631.3003},{"price":"2.419","symbol":"XTZUSDT","id":"24190","side":"Sell","size":3310.1},{"price":"2.414","symbol":"XTZUSDT","id":"24140","side":"Buy","size":2078.8},{"price":"2.418","symbol":"XTZUSDT","id":"24180","side":"Sell","size":4877.5},{"price":"2.416","symbol":"XTZUSDT","id":"24160","side":"Buy","size":6284},{"price":"2.420","symbol":"XTZUSDT","id":"24200","side":"Sell","size":7129.4}],"insert":[]},"cross_seq":"1168956806","timestamp_e6":"1626652979675472"}
{"topic":"orderBookL2_25.XTZUSDT","type":"delta","data":{"delete":[],"update":[{"price":"2.419","symbol":"XTZUSDT","id":"24190","side":"Sell","size":3510.0999},{"price":"2.417","symbol":"XTZUSDT","id":"24170","side":"Buy","size":1071.5},{"price":"2.416","symbol":"XTZUSDT","id":"24160","side":"Buy","size":2935.7}],"insert":[]},"cross_seq":"1168956811","timestamp_e6":"1626652979715783"}
{"topic":"orderBookL2_25.XTZUSDT","type":"delta","data":{"delete":[],"update":[{"price":"2.415","symbol":"XTZUSDT","id":"24150","side":"Buy","size":4598.8003},{"price":"2.411","symbol":"XTZUSDT","id":"24110","side":"Buy","size":1056},{"price":"2.423","symbol":"XTZUSDT","id":"24230","side":"Sell","size":1513},{"price":"2.414","symbol":"XTZUSDT","id":"24140","side":"Buy","size":2360.7},{"price":"2.421","symbol":"XTZUSDT","id":"24210","side":"Sell","size":9007},{"price":"2.418","symbol":"XTZUSDT","id":"24180","side":"Sell","size":4445.3},{"price":"2.422","symbol":"XTZUSDT","id":"24220","side":"Sell","size":2818.9},{"price":"2.412","symbol":"XTZUSDT","id":"24120","side":"Buy","size":1665.1}],"insert":[]},"cross_seq":"1168956833","timestamp_e6":"1626652979735570"}`
	lines := strings.Split(messages, "\n")
	ob := OrderBook{
		Symbol: "XTZUSDT",
		Bids:   common.Bids{},
		Asks:   common.Asks{},
	}
	bids := common.Bids{}
	asks := common.Asks{}
	obm := OrderBookMsg{}
	for ln, l := range lines {
		err := UpdateOrderBook([]byte(l), &ob)
		if err != nil {
			t.Fatal(err)
		}
		err = json.Unmarshal([]byte(l), &obm)
		if err != nil {
			t.Fatal(err)
		}
		if obm.Type == "snapshot" {
			assert.Equal(t, len(obm.Data.OrderBook), len(ob.Bids)+len(ob.Asks), fmt.Sprintf("snapshot"))
			for _, r := range obm.Data.OrderBook {
				if r.Side == "Buy" {
					bids = bids.Update([2]float64{r.Price, r.Size})
				} else if r.Side == "Sell" {
					asks = asks.Update([2]float64{r.Price, r.Size})
				}
			}
		} else {
			for _, r := range obm.Data.Delete {
				if r.Side == "Buy" {
					bids = bids.Update([2]float64{r.Price, 0})
				} else if r.Side == "Sell" {
					asks = asks.Update([2]float64{r.Price, 0})
				}
			}
			for _, r := range obm.Data.Update {
				if r.Side == "Buy" {
					bids = bids.Update([2]float64{r.Price, r.Size})
				} else if r.Side == "Sell" {
					asks = asks.Update([2]float64{r.Price, r.Size})
				}
			}
			for _, r := range obm.Data.Insert {
				if r.Side == "Buy" {
					bids = bids.Update([2]float64{r.Price, r.Size})
				} else if r.Side == "Sell" {
					asks = asks.Update([2]float64{r.Price, r.Size})
				}
			}
		}
		assert.Equal(t, "XTZUSDT", ob.Symbol)
		assert.Equal(t, time.Duration(0), ob.EventTime.Sub(time.Unix(0, obm.TimestampE6*1000)))
		assert.Equal(t, len(bids), len(ob.Bids), fmt.Sprintf("line number %d", ln))
		assert.Equal(t, len(asks), len(ob.Asks), fmt.Sprintf("line number %d", ln))
		for i, r := range ob.Bids {
			assert.Equal(t, bids[i][0], r[0], fmt.Sprintf("line number %d", ln))
			assert.Equal(t, bids[i][1], r[1], fmt.Sprintf("line number %d", ln))
		}
		for i, r := range ob.Asks {
			assert.Equal(t, asks[i][0], r[0], fmt.Sprintf("line number %d", ln))
			assert.Equal(t, asks[i][1], r[1], fmt.Sprintf("line number %d", ln))
		}
		assert.Greater(t, ob.Asks[0][0], ob.Bids[0][0])
		for i := 0; i < len(ob.Asks)-1; i++ {
			assert.Greater(t, ob.Asks[i+1][0], ob.Bids[i][0])
		}
		for i := 0; i < len(ob.Asks)-1; i++ {
			assert.Greater(t, ob.Asks[i+1][0], ob.Asks[i][0])
		}
		for i := 0; i < len(ob.Bids)-1; i++ {
			assert.Less(t, ob.Bids[i+1][0], ob.Bids[i][0])
		}
	}

	//
	//
	//msg := []byte(`{"topic":"orderBookL2_25.XTZUSDT","type":"snapshot","data":{"order_book":[{"price":"2.393","symbol":"XTZUSDT","id":"23930","side":"Buy","size":21442.3},{"price":"2.394","symbol":"XTZUSDT","id":"23940","side":"Buy","size":994.89996},{"price":"2.395","symbol":"XTZUSDT","id":"23950","side":"Buy","size":1808.5},{"price":"2.396","symbol":"XTZUSDT","id":"23960","side":"Buy","size":20826.3},{"price":"2.397","symbol":"XTZUSDT","id":"23970","side":"Buy","size":951.7},{"price":"2.398","symbol":"XTZUSDT","id":"23980","side":"Buy","size":21589.2},{"price":"2.399","symbol":"XTZUSDT","id":"23990","side":"Buy","size":536.2},{"price":"2.400","symbol":"XTZUSDT","id":"24000","side":"Buy","size":1116.6},{"price":"2.401","symbol":"XTZUSDT","id":"24010","side":"Buy","size":21816.201},{"price":"2.402","symbol":"XTZUSDT","id":"24020","side":"Buy","size":228.50002},{"price":"2.403","symbol":"XTZUSDT","id":"24030","side":"Buy","size":771.3},{"price":"2.404","symbol":"XTZUSDT","id":"24040","side":"Buy","size":22017.898},{"price":"2.405","symbol":"XTZUSDT","id":"24050","side":"Buy","size":4574.6997},{"price":"2.406","symbol":"XTZUSDT","id":"24060","side":"Buy","size":579.2},{"price":"2.407","symbol":"XTZUSDT","id":"24070","side":"Buy","size":1842.3},{"price":"2.408","symbol":"XTZUSDT","id":"24080","side":"Buy","size":6936.1},{"price":"2.409","symbol":"XTZUSDT","id":"24090","side":"Buy","size":1956.9},{"price":"2.410","symbol":"XTZUSDT","id":"24100","side":"Buy","size":1408.7},{"price":"2.411","symbol":"XTZUSDT","id":"24110","side":"Buy","size":870.3},{"price":"2.412","symbol":"XTZUSDT","id":"24120","side":"Buy","size":1330.4},{"price":"2.413","symbol":"XTZUSDT","id":"24130","side":"Buy","size":938.2},{"price":"2.414","symbol":"XTZUSDT","id":"24140","side":"Buy","size":2278.8},{"price":"2.415","symbol":"XTZUSDT","id":"24150","side":"Buy","size":7831.3},{"price":"2.416","symbol":"XTZUSDT","id":"24160","side":"Buy","size":6020.6},{"price":"2.417","symbol":"XTZUSDT","id":"24170","side":"Buy","size":1851.2},{"price":"2.418","symbol":"XTZUSDT","id":"24180","side":"Sell","size":4622.8},{"price":"2.419","symbol":"XTZUSDT","id":"24190","side":"Sell","size":3106.2},{"price":"2.420","symbol":"XTZUSDT","id":"24200","side":"Sell","size":6803.1},{"price":"2.421","symbol":"XTZUSDT","id":"24210","side":"Sell","size":10018.2},{"price":"2.422","symbol":"XTZUSDT","id":"24220","side":"Sell","size":3350.4},{"price":"2.423","symbol":"XTZUSDT","id":"24230","side":"Sell","size":1992},{"price":"2.424","symbol":"XTZUSDT","id":"24240","side":"Sell","size":1870.7001},{"price":"2.425","symbol":"XTZUSDT","id":"24250","side":"Sell","size":1350.5},{"price":"2.426","symbol":"XTZUSDT","id":"24260","side":"Sell","size":1589.4},{"price":"2.427","symbol":"XTZUSDT","id":"24270","side":"Sell","size":700.49994},{"price":"2.428","symbol":"XTZUSDT","id":"24280","side":"Sell","size":726.9},{"price":"2.429","symbol":"XTZUSDT","id":"24290","side":"Sell","size":654.39996},{"price":"2.430","symbol":"XTZUSDT","id":"24300","side":"Sell","size":2431.3},{"price":"2.431","symbol":"XTZUSDT","id":"24310","side":"Sell","size":3012.7002},{"price":"2.432","symbol":"XTZUSDT","id":"24320","side":"Sell","size":21965.201},{"price":"2.433","symbol":"XTZUSDT","id":"24330","side":"Sell","size":651.7},{"price":"2.434","symbol":"XTZUSDT","id":"24340","side":"Sell","size":1721.1001},{"price":"2.435","symbol":"XTZUSDT","id":"24350","side":"Sell","size":21622.3},{"price":"2.436","symbol":"XTZUSDT","id":"24360","side":"Sell","size":593.5},{"price":"2.437","symbol":"XTZUSDT","id":"24370","side":"Sell","size":2055.1},{"price":"2.438","symbol":"XTZUSDT","id":"24380","side":"Sell","size":22218.5},{"price":"2.439","symbol":"XTZUSDT","id":"24390","side":"Sell","size":1596.9999},{"price":"2.440","symbol":"XTZUSDT","id":"24400","side":"Sell","size":1069.8},{"price":"2.441","symbol":"XTZUSDT","id":"24410","side":"Sell","size":21004.2},{"price":"2.442","symbol":"XTZUSDT","id":"24420","side":"Sell","size":20491.299}]},"cross_seq":"1168956791","timestamp_e6":"1626652978974501"}`)
	//err := UpdateOrderBook(msg, &ob)
	//if err != nil {
	//	t.Fatal(err)
	//}
	//logger.Debugf("bids %v", ob.Bids)
	//logger.Debugf("asks %v", ob.Asks)
	//
	//msg = []byte(`{"topic":"orderBookL2_25.XTZUSDT","type":"delta","data":{"delete":[],"update":[{"price":"2.426","symbol":"XTZUSDT","id":"24260","side":"Buy","size":2902.7},{"price":"2.431","symbol":"XTZUSDT","id":"24310","side":"Sell","size":5981}],"insert":[]},"cross_seq":"1168950555","timestamp_e6":"1626652805434387"}`)
	//err = UpdateOrderBook(msg, &ob)
	//if err != nil {
	//	t.Fatal(err)
	//}
	//logger.Debugf("bids %v", ob.Bids)
	//logger.Debugf("asks %v", ob.Asks)
	//
	//msg = []byte(`{"topic":"orderBookL2_25.XTZUSDT","type":"delta","data":{"delete":[{"price":"2.429","symbol":"XTZUSDT","id":"24290","side":"Sell"}],"update":[{"price":"2.433","symbol":"XTZUSDT","id":"24330","side":"Sell","size":3672.8}],"insert":[{"price":"2.453","symbol":"XTZUSDT","id":"24530","side":"Sell","size":22041.799}]},"cross_seq":"1168950573","timestamp_e6":"1626652805714547"}`)
	//err = UpdateOrderBook(msg, &ob)
	//if err != nil {
	//	t.Fatal(err)
	//}
	//logger.Debugf("bids %v", ob.Bids)
	//logger.Debugf("asks %v", ob.Asks)
	//
	//msg = []byte(`{"topic":"orderBookL2_25.XTZUSDT","type":"delta","data":{"delete":[],"update":[{"price":"2.433","symbol":"XTZUSDT","id":"24330","side":"Sell","size":3672.8}],"insert":[{"price":"2.453","symbol":"XTZUSDT","id":"24530","side":"Sell","size":22041.799}]},"cross_seq":"1168950573","timestamp_e6":"1626652805714547"}`)
	//err = UpdateOrderBook(msg, &ob)
	//if err != nil {
	//	t.Fatal(err)
	//}
	//logger.Debugf("bids %v", ob.Bids)
	//logger.Debugf("asks %v", ob.Asks)
	//
	//msg = []byte(`{"topic":"orderBookL2_25.XTZUSDT","type":"delta","data":{"delete":[],"update":[],"insert":[{"price":"2.453","symbol":"XTZUSDT","id":"24530","side":"Sell","size":22041.799}]},"cross_seq":"1168950573","timestamp_e6":"1626652805714547"}`)
	//err = UpdateOrderBook(msg, &ob)
	//if err != nil {
	//	t.Fatal(err)
	//}
	//logger.Debugf("bids %v", ob.Bids)
	//logger.Debugf("asks %v", ob.Asks)
	//
	//msg = []byte(`{"topic":"orderBookL2_25.XTZUSDT","type":"delta","data":{"delete":[],"update":[],"insert":[]},"cross_seq":"1168950573","timestamp_e6":"1626652805714547"}`)
	//err = UpdateOrderBook(msg, &ob)
	//if err != nil {
	//	t.Fatal(err)
	//}
	//logger.Debugf("bids %v", ob.Bids)
	//logger.Debugf("asks %v", ob.Asks)
}
