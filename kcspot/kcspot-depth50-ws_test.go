package kcspot


func init() {
}

//func TestNewDepth50Websocket(t *testing.T) {
//	var api *API
//	var ctx = context.Background()
//	var err error
//	api, err = NewAPI(&common.Credentials{}, "socks5://127.0.0.1:1081")
//	if err != nil {
//		log.Fatal(err)
//	}
//	ws := NewDepth50Websocket(ctx, api, []string{"BTC-USDT"}, time.Minute, "socks5://127.0.0.1:1081" )
//	for {
//		select {
//		case d := <- ws.DataCh:
//			logger.Debugf("%v", d)
//		}
//	}
//}

