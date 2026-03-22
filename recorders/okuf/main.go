package main

import (
	"context"
	"flag"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	okexv5_usdtswap "github.com/geometrybase/hft-micro/okexv5-usdtswap"
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
	savePath := flag.String("path", "/root/okuf", "data save folder")

	//savePath := flag.String("path", "/home/clu/Downloads", "data save folder")
	//proxyAddress := flag.String("proxy", "socks5://127.0.0.1:1084", "symbols group batch size")
	flag.Parse()

	ctx := context.Background()
	api, err := okexv5_usdtswap.NewAPI(&okexv5_usdtswap.Credentials{}, *proxyAddress)
	if err != nil {
		logger.Fatal(err)
	}
	var instruments []okexv5_usdtswap.Instrument
	instruments, err = api.GetInstruments(context.Background())
	if err != nil {
		logger.Fatal(err)
	}

	symbols := make([]string, 0)
	for _, instrument := range instruments {
		if instrument.State != "live" {
			continue
		}
		if instrument.InstType != "SWAP" {
			continue
		}
		if len(instrument.InstId) < 5 {
			continue
		}
		if instrument.InstId[len(instrument.InstId)-5:] != "-SWAP" {
			continue
		}
		symbols = append(symbols, instrument.InstId)
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
	for start := 0; start < len(symbols); start += *batchSize {
		end := start + *batchSize
		if end > len(symbols) {
			end = len(symbols)
		}
		okufChMap := make(map[string]chan *common.RawMessage)
		for _, xSymbol := range symbols[start:end] {
			okufChMap[xSymbol] = make(chan *common.RawMessage, 1024)
			go common.RawWSMessageSaveLoopForSingleSymbol(ctx, cancel, *savePath, xSymbol, okufChMap[xSymbol], fileSavedCh)
		}
		go func(ctx context.Context, cancel context.CancelFunc, proxy string, outputChMap map[string]chan *common.RawMessage) {
			ws1 := okexv5_usdtswap.NewRawDepth5WS(ctx, proxy, []byte{'D'}, outputChMap)
			ws2 := okexv5_usdtswap.NewRawTickerWS(ctx, proxy, []byte{'B'}, outputChMap)
			ws3 := okexv5_usdtswap.NewRawTradeWS(ctx, proxy, []byte{'T'}, outputChMap)
			ws4 := okexv5_usdtswap.NewRawFundingRateWS(ctx, proxy, []byte{'F'}, outputChMap)
			select {
			case <-ctx.Done():
			case <-ws1.Done():
				cancel()
			case <-ws2.Done():
				cancel()
			case <-ws3.Done():
				cancel()
			case <-ws4.Done():
				cancel()
			}
		}(ctx, cancel, *proxyAddress, okufChMap)
	}
	go common.ArchiveDailyJlGzFiles(ctx, *savePath)
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
