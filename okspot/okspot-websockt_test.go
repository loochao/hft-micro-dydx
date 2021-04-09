package okspot

import (
	"context"
	"github.com/geometrybase/hft-micro/logger"
	"testing"
)

func TestNewMarketDataWebsocket(t *testing.T) {

	ws := NewWebsocket(
		context.Background(),
		WsUrl,
		&Credentials{
			//Key:        "297624eb-1abc-4c45-a13a-250a757bac1d",
			//Secret:     "9589D50C9D0A478C193BEFEBB1912B63",
			//Passphrase: "maodaye",
		},
		[]string{
			//"spot/depth5:ETH-USDT",
			//"spot/trade:ETH-USDT",
			"spot/ticker:BTC-USDT",
			"spot/account:BTC",
			"spot/order:BTC-USDT",
		},
		"socks5://127.0.0.1:1080",
		1000,
	)
	for {
		select {
		case data := <-ws.DataCh:
			switch data.(type) {
			case []WSDepth5:
				logger.Debugf("depth %v", data)
			case []WSTrade:
				logger.Debugf("trade %v", data)
			default:
				logger.Debugf("%v", data)
			}
			break
		}
	}
}
