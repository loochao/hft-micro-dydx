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

	batchSize := flag.Int("batch", 20, "symbols group batch size")

	proxyAddress := flag.String("proxy", "", "symbols group batch size")
	symbolsStr := flag.String("symbols", "ETCBUSD,BAKEBUSD,UNFIBUSD,CELRBUSD,FILBUSD,SANDBUSD,MKRBUSD,COMPBUSD,IOSTBUSD,ZRXBUSD,HOTBUSD,LINKBUSD,LUNABUSD,1INCHBUSD,BTCBUSD,ATOMBUSD,FLMBUSD,HBARBUSD,YFIIBUSD,LRCBUSD,TOMOBUSD,CHRBUSD,SRMBUSD,ONEBUSD,VETBUSD,CHZBUSD,FTMBUSD,RUNEBUSD,UNIBUSD,XMRBUSD,AAVEBUSD,TRBBUSD,SOLBUSD,XLMBUSD,BNBBUSD,ENJBUSD,THETABUSD,NEOBUSD,DASHBUSD,DODOBUSD,SCBUSD,SNXBUSD,EOSBUSD,RLCBUSD,SXPBUSD,GRTBUSD,OCEANBUSD,ETHBUSD,DGBBUSD,ALICEBUSD,YFIBUSD,BELBUSD,GTCBUSD,TRXBUSD,KSMBUSD,AXSBUSD,EGLDBUSD,ADABUSD,RSRBUSD,STMXBUSD,ALGOBUSD,ONTBUSD,LINABUSD,SFPBUSD,DOTBUSD,BATBUSD,BALBUSD,MATICBUSD,KNCBUSD,XEMBUSD,WAVESBUSD,RVNBUSD,BANDBUSD,ZILBUSD,AVAXBUSD,OMGBUSD,SUSHIBUSD,IOTABUSD,CRVBUSD,BCHBUSD,HNTBUSD,REEFBUSD,BTTBUSD,NEARBUSD,CTKBUSD,SKLBUSD,LTCBUSD,XTZBUSD,LITBUSD,BZRXBUSD,DOGEBUSD,ZECBUSD,XRPBUSD,MANABUSD,ICPBUSD,COTIBUSD,KEEPBUSD,ALPHABUSD,ICXBUSD,KAVABUSD,ZENBUSD,QTUMBUSD", "symbols, separate by comma")
	savePath := flag.String("path", "/root/bnbs-bnuf-depth5-and-ticker", "data save folder")

	//savePath := flag.String("path", "/Users/chenjilin/Downloads", "data save folder")
	//symbolsStr := flag.String("symbols", "BTCBUSD", "symbols, separate by comma")
	//proxyAddress := flag.String("proxy", "socks5://127.0.0.1:1080", "symbols group batch size")
	flag.Parse()
	symbols := strings.Split(*symbolsStr, ",")
	ctx, cancel := context.WithCancel(context.Background())
	fileSavedCh := make(chan string, len(symbols))
	for start := 0; start < len(symbols); start += *batchSize {
		end := start + *batchSize
		if end > len(symbols) {
			end = len(symbols)
		}
		bnbsChMap := make(map[string]chan *Message)
		bnufChMap := make(map[string]chan *Message)
		for _, xSymbol := range symbols[start:end] {
			ySymbol := strings.Replace(xSymbol, "BUSD", "USDT", -1)
			bnbsChMap[strings.ToLower(xSymbol)] = make(chan *Message, 1024)
			bnufChMap[strings.ToLower(ySymbol)] = bnbsChMap[strings.ToLower(xSymbol)]
			go saveLoop(ctx, cancel, *savePath, xSymbol, ySymbol, bnbsChMap[strings.ToLower(xSymbol)], fileSavedCh)
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
			ws1 := NewBnbsDepth5WS(ctx, proxy, outputChMap)
			ws2 := NewBnbsBookTickerWS(ctx, proxy, outputChMap)
			select {
			case <-ctx.Done():
			case <-ws1.Done():
				cancel()
			case <-ws2.Done():
				cancel()
			}
		}(ctx, cancel, *proxyAddress, bnbsChMap)
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
