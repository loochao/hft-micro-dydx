package ftxperp

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
)

type Ftxperp struct {
	api        *API
	orderCh    chan common.Order
	accountCh  chan common.Account
	positionCh chan common.Position
	markets    []string
	restartCh  chan interface{}
	statusCh   chan bool
	done chan interface{}
}

func (ftx *Ftxperp) Initial(ctx context.Context, settings common.ExchangeSettings) error {
	var err error
	ftx.api, err = NewAPI(*settings.ApiKey, *settings.ApiSecret, *settings.Proxy)
	if err != nil {
		return err
	}
	ftx.markets = settings.Symbols
	ftx.orderCh = make(chan common.Order, 100*len(settings.Symbols))
	ftx.accountCh = make(chan common.Account, 100*len(settings.Symbols))
	ftx.positionCh = make(chan common.Position, 100*len(settings.Symbols))
	ftx.statusCh = make(chan bool, 100)
	ftx.restartCh = make(chan interface{}, 100)
	return nil
}

func (ftx *Ftxperp) Start(ctx context.Context) {
}

func (ftx *Ftxperp) OrderCh() chan common.Order {
	return ftx.orderCh
}

func (ftx *Ftxperp) AccountCh() chan common.Account {
	return ftx.accountCh
}

func (ftx *Ftxperp) RestartCh() chan interface{} {
	return ftx.restartCh
}

func (ftx *Ftxperp) StatusCh() chan bool {
	return ftx.statusCh
}

func (ftx *Ftxperp) Done() chan common.Account {
	return ftx.accountCh
}


