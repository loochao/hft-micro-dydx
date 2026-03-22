package main

import (
	"context"
	"flag"
	"github.com/geometrybase/hft-micro/common"
	ftx_usdfuture "github.com/geometrybase/hft-micro/ftx-usdfuture"
	"github.com/geometrybase/hft-micro/logger"
	"os"
	"os/signal"
	"path"
	"sort"
	"syscall"
	"time"
)

func main() {

	batchSize := flag.Int("batch", 30, "symbols group batch size")

	proxyAddress := flag.String("proxy", "", "proxy address")
	savePath := flag.String("path", "/root/ftxuf", "data save folder")

	//savePath := flag.String("path", "/home/clu/Downloads", "data save folder")
	//proxyAddress := flag.String("proxy", "socks5://127.0.0.1:1083", "symbols group batch size")
	flag.Parse()

	api, err := ftx_usdfuture.NewAPI("", "", "", *proxyAddress)
	if err != nil {
		logger.Fatal(err)
	}

	ctx := context.Background()
	futures, err := api.GetFutures(ctx)
	if err != nil {
		logger.Fatal(err)
	}

	symbols := make([]string, 0)
	for _, future := range futures {
		if future.Type == "perpetual" && future.Enabled{
			symbols = append(symbols, future.Name)
		}
	}

	sort.Strings(symbols)
	logger.Debugf("SYMBOLS %s", symbols)
	logger.Debugf("SYMBOLS LEN %d", len(symbols))
	//symbols = symbols[:1]

	err = os.MkdirAll(path.Join(*savePath, "/archive"), 0777)
	if err != nil {
		logger.Debugf("os.MkdirAll error %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	fileSavedCh := make(chan string, len(symbols))
	ftxufAllChMap := make(map[string]chan *common.RawMessage)
	for start := 0; start < len(symbols); start += *batchSize {
		end := start + *batchSize
		if end > len(symbols) {
			end = len(symbols)
		}
		ftxufChMap := make(map[string]chan *common.RawMessage)
		for _, xSymbol := range symbols[start:end] {
			ftxufChMap[xSymbol] = make(chan *common.RawMessage, 1024)
			ftxufAllChMap[xSymbol] = ftxufChMap[xSymbol]
			go common.RawWSMessageSaveLoopForSingleSymbol(ctx, cancel, *savePath, xSymbol, ftxufChMap[xSymbol], fileSavedCh)
		}
		go func(ctx context.Context, cancel context.CancelFunc, proxy string, outputChMap map[string]chan *common.RawMessage) {
			ws1 := ftx_usdfuture.NewRawOrderBookWS(ctx, proxy, []byte{'D'}, outputChMap)
			ws2 := ftx_usdfuture.NewRawBookTickerWS(ctx, proxy, []byte{'B'}, outputChMap)
			ws3 := ftx_usdfuture.NewRawTradeWS(ctx, proxy, []byte{'T'}, outputChMap)
			select {
			case <-ctx.Done():
			case <-ws1.Done():
				cancel()
			case <-ws2.Done():
				cancel()
			case <-ws3.Done():
				cancel()
			}
		}(ctx, cancel, *proxyAddress, ftxufChMap)
	}
	go common.ArchiveDailyJlGzFiles(ctx, *savePath)
	go ftx_usdfuture.StreamRawFundingRate(ctx, *proxyAddress, []byte{'F'}, ftxufAllChMap)
	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			sig := <-sigs
			logger.Debugf("catch signal %v", sig)
			cancel()
		}()
	}()
	select {
	case <-ctx.Done():
	case <-time.After(time.Hour * 48):
		logger.Debugf("48H Restart...")
		cancel()
	}
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
