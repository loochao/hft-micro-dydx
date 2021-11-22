package okexv5_usdtspot

import (
	"encoding/json"
)

type WsArgs struct {
	Channel  string `json:"channel"`
	InstType string `json:"instType,omitempty"`
	Uly      string `json:"uly,omitempty"`
	InstId   string `json:"instId,omitempty"`
}

type WsSubUnsub struct {
	Op   string   `json:"op"`
	Args []WsArgs `json:"args"`
}

type WsLoginArgs struct {
	ApiKey     string `json:"apiKey"`
	Passphrase string `json:"passphrase"`
	Timestamp  string `json:"timestamp"`
	Sign       string `json:"sign"`
}

type WsLogin struct {
	Op   string        `json:"op"`
	Args []WsLoginArgs `json:"args"`
}

type CommonCapture struct {
	Table  string `json:"table,omitempty"`
	Action string `json:"action,omitempty"`

	Event string `json:"event,omitempty"`
	Msg   string `json:"msg,omitempty"`
	Code  string `json:"code,omitempty"`
	Arg   struct {
		Channel string `json:"channel,omitempty"`
		//UID int64 `json:"uid"`
	} `json:"arg,omitempty"`
	Data json.RawMessage `json:"data,omitempty"`
}


