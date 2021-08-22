package main

import (
	"context"
	"flag"
	binance_usdtfuture "github.com/geometrybase/hft-micro/binance-usdtfuture"
	"github.com/geometrybase/hft-micro/common"
	ftx_usdfuture "github.com/geometrybase/hft-micro/ftx-usdfuture"
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

	//proxyAddress := flag.String("proxy", "", "symbols group batch size")
	//savePath := flag.String("path", "/root/ftxuf-bnuf-depth5-and-ticker", "data save folder")

	savePath := flag.String("path", "/Users/chenjilin/Downloads", "data save folder")
	proxyAddress := flag.String("proxy", "socks5://127.0.0.1:1083", "symbols group batch size")
	flag.Parse()

	symbols := make([]string, 0)
	for key := range ftx_usdfuture.PriceIncrements {
		if _, ok := binance_usdtfuture.TickSizes[strings.Replace(key, "-PERP", "USDT", -1)]; ok {
			symbols = append(symbols, key)
		}
	}
	sort.Strings(symbols)
	symbols = symbols[:1]
	logger.Debugf("SYMBOLS %s", symbols)

	ftxufApi, err := ftx_usdfuture.NewAPI("", "", *proxyAddress)
	if err != nil {
		logger.Fatal(err)
	}
	bnufApi, err := binance_usdtfuture.NewAPI(&common.Credentials{}, *proxyAddress)
	if err != nil {
		logger.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	fileSavedCh := make(chan string, len(symbols))
	bnufAllChMap := make(map[string]chan *Message)
	ftxufAllChMap := make(map[string]chan *Message)
	for start := 0; start < len(symbols); start += *batchSize {
		end := start + *batchSize
		if end > len(symbols) {
			end = len(symbols)
		}
		ftxufChMap := make(map[string]chan *Message)
		bnufChMap := make(map[string]chan *Message)
		for _, xSymbol := range symbols[start:end] {
			ySymbol := strings.Replace(xSymbol, "-PERP", "USDT", -1)
			ftxufChMap[xSymbol] = make(chan *Message, 1024)
			bnufChMap[strings.ToLower(ySymbol)] = ftxufChMap[xSymbol]
			ftxufAllChMap[xSymbol] = ftxufChMap[xSymbol]
			bnufAllChMap[strings.ToLower(ySymbol)] = ftxufChMap[xSymbol]
			go saveLoop(ctx, cancel, *savePath, xSymbol, ySymbol, ftxufChMap[xSymbol], fileSavedCh)
		}
		go func(ctx context.Context, cancel context.CancelFunc, proxy string, outputChMap map[string]chan *Message) {
			ws1 := NewBnufDepth5WS(ctx, proxy, outputChMap)
			ws2 := NewBnufBookTickerWS(ctx, proxy, outputChMap)
			select {
			case <-ctx.Done():
			case <-ws1.Done():
				cancel()
			case <-ws2.Done():
				cancel()
			}
		}(ctx, cancel, *proxyAddress, bnufChMap)
		go func(ctx context.Context, cancel context.CancelFunc, proxy string, outputChMap map[string]chan *Message) {
			ws1 := NewFtxufTickerWS(ctx, proxy, outputChMap)
			select {
			case <-ctx.Done():
			case <-ws1.Done():
				cancel()
			}
		}(ctx, cancel, *proxyAddress, ftxufChMap)
	}
	go archiveFiles(context.Background(), *savePath)
	go streamFtxufFundingRate(ctx, ftxufApi, ftxufAllChMap)
	go streamBnufFundingRate(ctx, bnufApi, bnufAllChMap)
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
