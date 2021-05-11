package main

import (
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {

	defer trExchange.Stop()

	//订阅帐号基本信息
	positionsCh := make(map[string]chan common.Position)
	ordersCh := make(map[string]chan common.Order)
	for _, symbol := range trConfig.ExchangeSettings.Symbols {
		positionsCh[symbol] = trPositionCh
		ordersCh[symbol] = trOrderCh
	}
	go trExchange.StreamBasic(
		trGlobalContext,
		trStatusCh,
		trAccountCh,
		positionsCh,
		ordersCh,
	)

	depthChannels := make(map[string]chan common.Depth)
	tradeChannels := make(map[string]chan common.Trade)
	for _, symbol := range trConfig.ExchangeSettings.Symbols {
		depthChannels[symbol] = make(chan common.Depth, 100)
		tradeChannels[symbol] = make(chan common.Trade, 100)
		go streamSignal(
			trGlobalContext, symbol, trConfig.UpdateInterval,
			trConfig.TradeLookback, trConfig.DepthLevel,
			depthChannels[symbol],
			tradeChannels[symbol],
			trSignalCh,
		)
	}

	go trExchange.StreamDepth(trGlobalContext, depthChannels, trConfig.BatchSize)
	go trExchange.StreamTrade(trGlobalContext, tradeChannels, trConfig.BatchSize)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		logger.Debugf("CATCH EXIT SIGNAL %v", sig)
		trGlobalCancel()
	}()

mainLoop:
	for {
		select {
		case <-trGlobalContext.Done():
			break mainLoop
		case <-trExchange.Done():
			break mainLoop
		case trSystemStatus = <-trStatusCh:
			logger.Debugf("SYSTEM STATUS %s", trSystemStatus)
			if trSystemStatus != common.SystemStatusReady {
				trGlobalSilent = time.Now().Add(trConfig.GlobalSilent)
			}
			break
		case trAccount = <-trAccountCh:
			break
		case pos := <-trPositionCh:
			trPositions[pos.GetSymbol()] = pos
			break
		case order := <-trOrderCh:
			trOrders[order.GetSymbol()] = order
			break
		case s := <-trSignalCh:
			logger.Debugf("signal %v", s)
		}
	}

	logger.Debugf("waiting 15s for exit.")
	select {
	case <-time.After(time.Second*15):
	}
	logger.Debugf("exit main loop")
}
