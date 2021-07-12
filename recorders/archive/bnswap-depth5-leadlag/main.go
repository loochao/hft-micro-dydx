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
	symbolsStr := flag.String("symbols", "BTCUSDT,BTCBUSD", "symbols, separate by comma")
	savePath := flag.String("path", "", "data save folder")
	proxyAddress := flag.String("proxy", "", "symbols group batch size")
	flag.Parse()
	symbols := strings.Split(*symbolsStr, ",")
	ctx, cancel := context.WithCancel(context.Background())
	fileSavedCh := make(chan string, len(symbols))
	go func(ctx context.Context, cancel context.CancelFunc, proxy, savePath string, symbols []string, fileSavedCh chan string) {
		ws := NewDepth5WS(ctx, proxy, savePath, symbols, fileSavedCh)
		select {
		case <-ctx.Done():
		case <-ws.Done():
			cancel()
		}
	}(ctx, cancel, *proxyAddress, *savePath, symbols, fileSavedCh)
	go archiveFiles(context.Background(), symbols, *savePath)
	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			sig := <-sigs
			logger.Debugf("catch signal %v", sig)
			cancel()
		}()
	}()
	<-ctx.Done()
	logger.Debugf("waiting 88s to write files")
	for {
		select {
		case symbol := <-fileSavedCh:
			logger.Debugf("%s saved", symbol)
			return
		case <-time.After(time.Second * 88):
			logger.Debugf("save timeout in 88s")
		}
	}
}
