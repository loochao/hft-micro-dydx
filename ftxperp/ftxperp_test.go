package ftxperp

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"testing"
)

func TestFtxperp(t *testing.T) {
	ctx := context.Background()
	settings := common.ExchangeSettings{
		ApiKey:    os.Getenv("FTX_TEST_KEY"),
		ApiSecret: os.Getenv("FTX_TEST_SECRET"),
		Proxy:     os.Getenv("FTX_TEST_PROXY"),
	}
	var exchange = common.Exchange(&Ftxperp{})
	err := exchange.Setup(ctx, settings)
	if err != nil {
		t.Fatal(err)
	}
	statusCh := make(chan common.SystemStatus, 100)
	accountCh := make(chan common.Account, 100)
	positionsCh := make(map[string]chan common.Position)
	ordersCh := make(map[string]chan common.Order)

	positionCh := make(chan common.Position, 100)
	positionsCh["BTC-PERP"] = positionCh
	positionsCh["ETH-PERP"] = positionCh
	orderCh := make(chan common.Order, 100)
	ordersCh["BTC-PERP"] = orderCh
	ordersCh["ETH-PERP"] = orderCh
	go exchange.StreamBasic(
		ctx,
		statusCh,
		accountCh,
		positionsCh,
		ordersCh,
	)
	for {
		select {
		case status := <-statusCh:
			logger.Debugf("SYSTEM STATUS %s", status)
		case account := <-accountCh:
			logger.Debugf("ACCOUNT %v", account)
		case position := <-positionCh:
			logger.Debugf("POSITION %v", position)
		case order := <-orderCh:
			logger.Debugf("ORDER %v", order)
		}
	}
}
