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

	batchSize := flag.Int("batch", 40, "symbols group batch size")

	proxyAddress := flag.String("proxy", "", "symbols group batch size")
	symbolsStr := flag.String("symbols", "EOSUSDC,LINKUSDC,NEOUSDC,WINUSDC,BNBUSDC,ETHUSDC,XRPUSDC,BTTUSDC,ATOMUSDC,TRXUSDC,BCHUSDC,BTCUSDC,LTCUSDC,ZECUSDC,ADAUSDC", "symbols, separate by comma")
	savePath := flag.String("path", "/root/bncs-depth5", "data save folder")

	//proxyAddress := flag.String("proxy", "socks5://127.0.0.1:1080", "symbols group batch size")
	//symbolsStr := flag.String("symbols", "BTCUSDC,ETHUSDC", "symbols, separate by comma")
	//savePath := flag.String("path", "/Users/chenjilin/Downloads", "data save folder")
	flag.Parse()
	symbols := strings.Split(*symbolsStr, ",")
	ctx, cancel := context.WithCancel(context.Background())
	fileSavedCh := make(chan string, len(symbols))
	for start := 0; start < len(symbols); start += *batchSize {
		end := start + *batchSize
		if end > len(symbols) {
			end = len(symbols)
		}
		bMessageChMap := make(map[string]chan Message)
		for _, bSymbol := range symbols[start:end] {
			bMessageChMap[strings.ToLower(bSymbol)] = make(chan Message, 10000)
			go saveLoop(ctx, cancel, *savePath, bSymbol, bMessageChMap[strings.ToLower(bSymbol)], fileSavedCh)
		}
		go func(ctx context.Context, cancel context.CancelFunc, proxy string, outputChMap map[string]chan Message) {
			ws := NewSpotDepth5WS(ctx, proxy, outputChMap)
			select {
			case <-ctx.Done():
			case <-ws.Done():
				cancel()
			}
		}(ctx, cancel, *proxyAddress, bMessageChMap)
	}
	go archiveFiles(context.Background(), *savePath)
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
	counter := 0
	for {
		select {
		case symbol := <-fileSavedCh:
			logger.Debugf("%s saved", symbol)
			counter++
			if counter == len(symbols) {
				logger.Debugf("all symbols' file saved")
				return
			}
		case <-time.After(time.Second * 88):
			logger.Debugf("save timeout in 88s")
		}
	}
}
