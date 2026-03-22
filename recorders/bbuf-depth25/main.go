package main

import (
	"context"
	"flag"
	bybit_usdtfuture "github.com/geometrybase/hft-micro/bybit-usdtfuture"
	"github.com/geometrybase/hft-micro/common"
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
	savePath := flag.String("path", "/root/bbuf-depth25", "data save folder")

	//savePath := flag.String("path", "/home/clu/Downloads", "data save folder")
	//proxyAddress := flag.String("proxy", "socks5://127.0.0.1:1083", "symbols group batch size")
	flag.Parse()

	symbolsMap := map[string]string{}
	symbols := make([]string, 0)
	for xSymbol := range bybit_usdtfuture.TickSizes {
		ySymbol := strings.Replace(xSymbol, "USDT", "USDT", -1)
		if _, ok := bybit_usdtfuture.TickSizes[ySymbol]; ok {
			symbols = append(symbols, xSymbol)
			symbolsMap[xSymbol] = ySymbol
		}
	}
	sort.Strings(symbols)
	//symbols = symbols[:1]

	logger.Debugf("SYMBOLS %s", symbols)
	ctx, cancel := context.WithCancel(context.Background())
	fileSavedCh := make(chan string, len(symbols))
	bybitAllChMap := make(map[string]chan *common.RawMessage)
	for start := 0; start < len(symbols); start += *batchSize {
		end := start + *batchSize
		if end > len(symbols) {
			end = len(symbols)
		}
		bybitChMap := make(map[string]chan *common.RawMessage)
		for _, xSymbol := range symbols[start:end] {
			ySymbol := symbolsMap[xSymbol]
			bybitChMap[xSymbol] = make(chan *common.RawMessage, 1024)
			bybitAllChMap[ySymbol] = bybitChMap[xSymbol]
			go common.RawWSMessageSaveLoop(ctx, cancel, *savePath, xSymbol, ySymbol, bybitChMap[xSymbol], fileSavedCh)
		}
		go func(ctx context.Context, cancel context.CancelFunc, proxy string, outputChMap map[string]chan *common.RawMessage) {
			ws1 := bybit_usdtfuture.NewRawDepth25WS(ctx, proxy, []byte{'X', 'D'}, outputChMap)
			select {
			case <-ctx.Done():
			case <-ws1.Done():
				cancel()
			}
		}(ctx, cancel, *proxyAddress, bybitChMap)
	}
	go common.ArchiveDailyJlGzFiles(ctx, *savePath)
	go bybit_usdtfuture.StreamRawFundingRate(ctx, *proxyAddress, []byte{'X', 'F'}, bybitAllChMap)
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
