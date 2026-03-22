package hbspot

import (
	"encoding/json"
	"testing"
)

func TestWSOrderEvent_UnmarshalJSON(t *testing.T) {
	bs := []byte(`{"orderSize":"0.1","remainAmt":"0.1","execAmt":"0","lastActTime":1618486619890,"orderSource":"spot-web","orderPrice":"165","symbol":"filusdt","type":"buy-limit","clientOrderId":"","orderStatus":"canceled","orderId":255726957080695,"eventType":"cancellation"}`)
	oe := WSOrder{}
	err := json.Unmarshal(bs, &oe)
	if err != nil {
		t.Fatal(err)
	}
}

func TestWSOrder_UnmarshalJSON(t *testing.T) {
	bs := []byte(`{"orderSize":"0.1","remainAmt":"0.1","execAmt":"0","lastActTime":1618486619890,"orderSource":"spot-web","orderPrice":"165","symbol":"filusdt","type":"buy-limit","clientOrderId":"","orderStatus":"canceled","orderId":255726957080695,"eventType":"cancellation"}}`)
	oe := WSOrderEvent{}
	err := json.Unmarshal(bs, &oe)
	if err != nil {
		t.Fatal(err)
	}
}
