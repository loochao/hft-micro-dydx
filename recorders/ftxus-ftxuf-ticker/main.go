package main

import (
	"context"
	"flag"
	"github.com/geometrybase/hft-micro/common"
	ftx_usdfuture "github.com/geometrybase/hft-micro/ftx-usdfuture"
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

	//proxyAddress := flag.String("proxy", "", "proxy address")
	//savePath := flag.String("path", "/root/ftxus-ftxuf-ticker", "data save folder")

	savePath := flag.String("path", "/Users/chenjilin/Downloads", "data save folder")
	proxyAddress := flag.String("proxy", "socks5://127.0.0.1:1083", "symbols group batch size")
	flag.Parse()

	symbols := make([]string, 0)
	for key := range ftx_usdspot.PriceIncrements {
		if _, ok := ftx_usdfuture.PriceIncrements[strings.Replace(key, "/USD", "-PERP", -1)]; ok {
			symbols = append(symbols, key)
		}
	}
	sort.Strings(symbols)
	//symbols = symbols[:1]
	symbols = []string{"HT/USD"}
	logger.Debugf("SYMBOLS %s", symbols)

	ctx, cancel := context.WithCancel(context.Background())
	fileSavedCh := make(chan string, len(symbols))
	ftxufAllChMap := make(map[string]chan *common.RawMessage)
	for start := 0; start < len(symbols); start += *batchSize {
		end := start + *batchSize
		if end > len(symbols) {
			end = len(symbols)
		}
		ftxusChMap := make(map[string]chan *common.RawMessage)
		ftxufChMap := make(map[string]chan *common.RawMessage)
		for _, xSymbol := range symbols[start:end] {
			ySymbol := strings.Replace(xSymbol, "/USD", "-PERP", -1)
			ftxusChMap[xSymbol] = make(chan *common.RawMessage, 1024)
			ftxufChMap[ySymbol] = ftxusChMap[xSymbol]
			ftxufAllChMap[ySymbol] = ftxusChMap[xSymbol]
			go common.RawWSMessageSaveLoop(ctx, cancel, *savePath, xSymbol, ySymbol, ftxusChMap[xSymbol], fileSavedCh)
		}
		go func(ctx context.Context, cancel context.CancelFunc, proxy string, outputChMap map[string]chan *common.RawMessage) {
			ws2 := ftx_usdspot.NewRawTickerWS(ctx, proxy, []byte{'X', 'T'}, outputChMap)
			select {
			case <-ctx.Done():
			case <-ws2.Done():
				cancel()
			}
		}(ctx, cancel, *proxyAddress, ftxusChMap)
		go func(ctx context.Context, cancel context.CancelFunc, proxy string, outputChMap map[string]chan *common.RawMessage) {
			ws1 := ftx_usdfuture.NewRawTickerWS(ctx, proxy, []byte{'Y', 'T'}, outputChMap)
			select {
			case <-ctx.Done():
			case <-ws1.Done():
				cancel()
			}
		}(ctx, cancel, *proxyAddress, ftxufChMap)
	}
	go common.ArchiveDailyJlGzFiles(ctx, *savePath)
	go ftx_usdfuture.StreamRawFundingRate(ctx, *proxyAddress, []byte{'Y', 'F'}, ftxufAllChMap)
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
