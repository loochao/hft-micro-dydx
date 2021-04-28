package main

import (
	"context"
	"flag"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	symbolsStr := flag.String("symbols", "BTCUSDT", "symbols, separate by comma")
	savePath := flag.String("path", "", "data save folder")
	batchSize := flag.Int("batch", 20, "symbols group batch size")
	proxyAddress := flag.String("proxy", "", "symbols group batch size")
	flag.Parse()
	symbols := strings.Split(*symbolsStr, ",")
	ctx, cancel := context.WithCancel(context.Background())
	fileSavedCh := make(chan string, len(symbols))
	for start := 0; start < len(symbols); start += *batchSize {
		end := start + *batchSize
		if end > len(symbols) {
			end = len(symbols)
		}
		go func(ctx context.Context, cancel context.CancelFunc, proxy, savePath string, symbols []string, fileSavedCh chan string) {
			ws := NewDepth20RoutedWebsocket(ctx, proxy, savePath, symbols, fileSavedCh)
			select {
			case <-ctx.Done():
			case <-ws.Done():
				cancel()
			}
		}(ctx, cancel, *proxyAddress, *savePath, symbols[start:end], fileSavedCh)
	}
	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			sig := <-sigs
			logger.Debugf("CATCH SIG %v", sig)
			cancel()
		}()
	}()
	<-ctx.Done()
	logger.Debugf("exit waiting 88s to write files")
	counter := 0
	for {
		select {
		case symbol := <-fileSavedCh:
			logger.Debugf("%s SAVED", symbol)
			counter++
			if counter == len(symbols) {
				logger.Debugf("ALL FILES SAVED")
				return
			}
		case <-time.After(time.Second*88):
			logger.Debugf("ALL FILES SAVE FAILED, TIMEOUT IN 88s")
		}
	}
}
