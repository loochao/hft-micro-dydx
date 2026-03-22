package main

import (
	"context"
	"flag"
	"github.com/geometrybase/hft-micro/common"
	dydx_usdfuture "github.com/geometrybase/hft-micro/dydx-usdfuture"
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
	savePath := flag.String("path", "/root/dydx", "data save folder")

	//savePath := flag.String("path", "/home/clu/Downloads", "data save folder")
	//proxyAddress := flag.String("proxy", "socks5://127.0.0.1:1083", "symbols group batch size")
	flag.Parse()

	api, err := dydx_usdfuture.NewAPI(dydx_usdfuture.Credentials{}, *proxyAddress)
	if err != nil {
		logger.Fatal(err)
	}
	ctx := context.Background()
	markets, err := api.GetMarkets(ctx)
	if err != nil {
		logger.Fatal(err)
	}

	symbols := make([]string, 0)
	for _, market := range markets {
		if market.Type != "PERPETUAL" || market.Status != "ONLINE" {
			continue
		}
		symbols = append(symbols, market.Market)
	}

	sort.Strings(symbols)
	logger.Debugf("SYMBOLS %s", symbols)
	//symbols = symbols[:1]
	//symbols = []string {"BTC-USD"}

	err = os.MkdirAll(path.Join(*savePath, "/archive"), 0777)
	if err != nil {
		logger.Debugf("os.MkdirAll error %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	fileSavedCh := make(chan string, len(symbols))
	dydxAllChMap := make(map[string]chan *common.RawMessage)
	for start := 0; start < len(symbols); start += *batchSize {
		end := start + *batchSize
		if end > len(symbols) {
			end = len(symbols)
		}
		dydxChMap := make(map[string]chan *common.RawMessage)
		for _, xSymbol := range symbols[start:end] {
			dydxChMap[xSymbol] = make(chan *common.RawMessage, 1024)
			dydxAllChMap[xSymbol] = dydxChMap[xSymbol]
			go common.RawWSMessageSaveLoopForSingleSymbol(ctx, cancel, *savePath, xSymbol, dydxChMap[xSymbol], fileSavedCh)
		}
		go func(ctx context.Context, cancel context.CancelFunc, proxy string, outputChMap map[string]chan *common.RawMessage) {
			ws2 := dydx_usdfuture.NewRawDepthWS(ctx, proxy, []byte{'D'}, outputChMap)
			ws3 := dydx_usdfuture.NewRawTradeWS(ctx, proxy, []byte{'T'}, outputChMap)
			select {
			case <-ctx.Done():
			case <-ws2.Done():
				cancel()
			case <-ws3.Done():
				cancel()
			}
		}(ctx, cancel, *proxyAddress, dydxChMap)
	}
	go common.ArchiveDailyJlGzFiles(ctx, *savePath)
	go dydx_usdfuture.StreamRawFundingRate(ctx, *proxyAddress, []byte{'F'}, dydxAllChMap)
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
