package main

import (
	"context"
	"flag"
	binance_busdfuture "github.com/geometrybase/hft-micro/binance-busdfuture"
	"github.com/geometrybase/hft-micro/common"
	ftx_usdspot "github.com/geometrybase/hft-micro/ftx-usdspot"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"
)

func main() {

	batchSize := flag.Int("batch", 30, "symbols group batch size")

	proxyAddress := flag.String("proxy", "", "proxy address")
	savePath := flag.String("path", "/root/ftxus-bnbf-ticker", "data save folder")

	//savePath := flag.String("path", "/home/clu/Downloads", "data save folder")
	//proxyAddress := flag.String("proxy", "socks5://127.0.0.1:1083", "proxy address")

	flag.Parse()
	symbols := make([]string, 0)
	for xSymbol := range ftx_usdspot.PriceIncrements {
		if _, ok := binance_busdfuture.TickSizes[strings.Replace(xSymbol, "/USD", "BUSD", -1)]; ok {
			symbols = append(symbols, xSymbol)
		}
	}
	sort.Strings(symbols)
	//symbols = symbols[:1]
	logger.Debugf("SYMBOLS %s", symbols)

	bnbfApi, err := binance_busdfuture.NewAPI(&common.Credentials{}, *proxyAddress)
	if err != nil {
		logger.Fatal(err)
	}

	bnbfAllChMap := make(map[string]chan *Message)


	ctx, cancel := context.WithCancel(context.Background())
	fileSavedCh := make(chan string, len(symbols))
	for start := 0; start < len(symbols); start += *batchSize {
		end := start + *batchSize
		if end > len(symbols) {
			end = len(symbols)
		}
		ftxusChMap := make(map[string]chan *Message)
		bnbfChMap := make(map[string]chan *Message)
		for _, xSymbol := range symbols[start:end] {
			ySymbol := strings.Replace(xSymbol, "/USD", "BUSD", -1)
			ftxusChMap[xSymbol] = make(chan *Message, 1024)
			bnbfChMap[strings.ToLower(ySymbol)] = ftxusChMap[xSymbol]
			bnbfAllChMap[strings.ToLower(ySymbol)] = ftxusChMap[xSymbol]
			go saveLoop(ctx, cancel, *savePath, xSymbol, ySymbol, ftxusChMap[xSymbol], fileSavedCh)
		}
		go func(ctx context.Context, cancel context.CancelFunc, proxy string, outputChMap map[string]chan *Message) {
			//ws1 := NewBnbfDepth5WS(ctx, proxy, outputChMap)
			ws2 := NewBnbfBookTickerWS(ctx, proxy, outputChMap)
			select {
			case <-ctx.Done():
			//case <-ws1.Done():
			//	cancel()
			case <-ws2.Done():
				cancel()
			}
		}(ctx, cancel, *proxyAddress, bnbfChMap)
		go func(ctx context.Context, cancel context.CancelFunc, proxy string, outputChMap map[string]chan *Message) {
			ws1 := NewFtxusTickerWS(ctx, proxy, outputChMap)
			select {
			case <-ctx.Done():
			case <-ws1.Done():
				cancel()
			}
		}(ctx, cancel, *proxyAddress, ftxusChMap)
	}
	go archiveFiles(context.Background(), *savePath)
	go streamBnbfFundingRate(ctx, bnbfApi, bnbfAllChMap)
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
