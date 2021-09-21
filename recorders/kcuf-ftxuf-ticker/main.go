package main

import (
	"context"
	"flag"
	"github.com/geometrybase/hft-micro/common"
	ftx_usdfuture "github.com/geometrybase/hft-micro/ftx-usdfuture"
	kucoin_usdtfuture "github.com/geometrybase/hft-micro/kucoin-usdtfuture"
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
	savePath := flag.String("path", "/root/kcuf-ftxuf-ticker", "data save folder")

	//savePath := flag.String("path", "/Users/chenjilin/Downloads", "data save folder")
	//proxyAddress := flag.String("proxy", "socks5://127.0.0.1:1083", "symbols group batch size")
	flag.Parse()

	symbolsMap := map[string]string{
		"XBTUSDTM": "BTC-PERP",
	}
	symbols := []string{
		"XBTUSDTM",
	}
	for xSymbol := range kucoin_usdtfuture.TickSizes {
		ySymbol := strings.Replace(xSymbol, "USDTM", "-PERP", -1)
		if _, ok := ftx_usdfuture.PriceIncrements[ySymbol]; ok {
			symbols = append(symbols, xSymbol)
			symbolsMap[xSymbol] = ySymbol
		}
	}
	sort.Strings(symbols)
	symbols = symbols[:1]
	logger.Debugf("SYMBOLS %s", symbols)

	ctx, cancel := context.WithCancel(context.Background())
	fileSavedCh := make(chan string, len(symbols))
	ftxufAllChMap := make(map[string]chan *common.RawMessage)
	kcufAllChMap := make(map[string]chan *common.RawMessage)
	for start := 0; start < len(symbols); start += *batchSize {
		end := start + *batchSize
		if end > len(symbols) {
			end = len(symbols)
		}
		kcufChMap := make(map[string]chan *common.RawMessage)
		ftxufChMap := make(map[string]chan *common.RawMessage)
		for _, xSymbol := range symbols[start:end] {
			ySymbol := symbolsMap[xSymbol]
			kcufChMap[xSymbol] = make(chan *common.RawMessage, 1024)
			ftxufChMap[ySymbol] = kcufChMap[xSymbol]
			kcufAllChMap[xSymbol] = kcufChMap[xSymbol]
			ftxufAllChMap[ySymbol] = kcufChMap[xSymbol]
			go common.RawWSMessageSaveLoop(ctx, cancel, *savePath, xSymbol, ySymbol, kcufChMap[xSymbol], fileSavedCh)
		}
		go func(ctx context.Context, cancel context.CancelFunc, proxy string, outputChMap map[string]chan *common.RawMessage) {
			ws1 := kucoin_usdtfuture.NewRawTickerWS(ctx, proxy, []byte{'X', 'T'}, outputChMap)
			select {
			case <-ctx.Done():
			case <-ws1.Done():
				cancel()
			}
		}(ctx, cancel, *proxyAddress, kcufChMap)
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
	go kucoin_usdtfuture.StreamRawFundingRate(ctx, *proxyAddress, []byte{'X', 'F'}, kcufAllChMap)
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
