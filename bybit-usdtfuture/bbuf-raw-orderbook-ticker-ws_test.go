package bybit_usdtfuture

//func TestNewRawOrderBookTickerWS(t *testing.T) {
//	var ctx = context.Background()
//	symbols := []string{"ETHUSDT"}
//	channels := make(map[string]chan *common.RawMessage)
//	for _, symbol := range symbols {
//		channels[symbol] = make(chan *common.RawMessage, 100)
//	}
//	_ = NewOrderBookTickerWS(ctx, os.Getenv("BYBIT_TEST_PROXY"), channels)
//	for {
//		select {
//		case d := <-channels[symbols[0]]:
//			logger.Debugf("%v", d)
//		}
//	}
//}
