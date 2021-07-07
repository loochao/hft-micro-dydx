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
	symbolsStr := flag.String("symbols", "1INCH-USDT,AAC-USDT,AAVE-USDT,ABT-USDT,ACT-USDT,ADA-USDT,AE-USDT,AERGO-USDT,AKITA-USDT,ALGO-USDT,ALPHA-USDT,ALV-USDT,ANC-USDT,ANT-USDT,ANW-USDT,API3-USDT,APIX-USDT,APM-USDT,ARK-USDT,AST-USDT,ATOM-USDT,AUCTION-USDT,AVAX-USDT,BADGER-USDT,BAL-USDT,BAND-USDT,BAT-USDT,BCD-USDT,BCH-USDT,BCHA-USDT,BETH-USDT,BHP-USDT,BLOC-USDT,BNT-USDT,BOX-USDT,BSV-USDT,BTC-USDT,BTG-USDT,BTM-USDT,BTT-USDT,BZZ-USDT,CEL-USDT,CELO-USDT,CELR-USDT,CFX-USDT,CHAT-USDT,CHZ-USDT,CMT-USDT,CNTM-USDT,COMP-USDT,CONV-USDT,COVER-USDT,CQT-USDT,CRO-USDT,CRV-USDT,CSPR-USDT,CTC-USDT,CTXC-USDT,CVC-USDT,CVP-USDT,CVT-USDT,DAI-USDT,DAO-USDT,DASH-USDT,DCR-USDT,DEP-USDT,DGB-USDT,DHT-USDT,DIA-USDT,DMD-USDT,DMG-USDT,DNA-USDT,DOGE-USDT,DORA-USDT,DOT-USDT,EC-USDT,EGLD-USDT,EGT-USDT,ELF-USDT,EM-USDT,ENJ-USDT,EOS-USDT,ETC-USDT,ETH-USDT,ETM-USDT,EXE-USDT,FAIR-USDT,FIL-USDT,FLM-USDT,FLOW-USDT,FORTH-USDT,FRONT-USDT,FSN-USDT,FTM-USDT,GAL-USDT,GAS-USDT,GHST-USDT,GLM-USDT,GRT-USDT,GTO-USDT,GUSD-USDT,HBAR-USDT,HC-USDT,HDAO-USDT,HEGIC-USDT,HYC-USDT,ICP-USDT,ICX-USDT,INT-USDT,INX-USDT,IOST-USDT,IOTA-USDT,IQ-USDT,ITC-USDT,JFI-USDT,JST-USDT,KAN-USDT,KCASH-USDT,KINE-USDT,KISHU-USDT,KLAY-USDT,KNC-USDT,KONO-USDT,KP3R-USDT,KSM-USDT,LAMB-USDT,LAT-USDT,LBA-USDT,LEO-USDT,LET-USDT,LINK-USDT,LMCH-USDT,LON-USDT,LOON-USDT,LPT-USDT,LRC-USDT,LSK-USDT,LTC-USDT,LUNA-USDT,MANA-USDT,MASK-USDT,MATIC-USDT,MCO-USDT,MDA-USDT,MDT-USDT,MEME-USDT,MINA-USDT,MIR-USDT,MITH-USDT,MKR-USDT,MLN-USDT,MOF-USDT,MXC-USDT,MXT-USDT,NANO-USDT,NAS-USDT,NDN-USDT,NEAR-USDT,NEO-USDT,NMR-USDT,NU-USDT,NULS-USDT,OKB-USDT,OKT-USDT,OM-USDT,OMG-USDT,ONT-USDT,ORBS-USDT,ORS-USDT,OXT-USDT,PAX-USDT,PAY-USDT,PERP-USDT,PHA-USDT,PICKLE-USDT,PLG-USDT,PNK-USDT,POLS-USDT,PPT-USDT,PROPS-USDT,PRQ-USDT,PST-USDT,QTUM-USDT,QUN-USDT,REN-USDT,REP-USDT,RFUEL-USDT,RIO-USDT,RNT-USDT,ROAD-USDT,RSR-USDT,RVN-USDT,SAND-USDT,SC-USDT,SFG-USDT,SHIB-USDT,SKL-USDT,SNT-USDT,SNX-USDT,SOC-USDT,SOL-USDT,SRM-USDT,STORJ-USDT,STRK-USDT,STX-USDT,SUSHI-USDT,SWFTC-USDT,SWRV-USDT,TAI-USDT,TCT-USDT,THETA-USDT,TMTG-USDT,TOPC-USDT,TORN-USDT,TRA-USDT,TRADE-USDT,TRB-USDT,TRIO-USDT,TRUE-USDT,TRX-USDT,TUSD-USDT,UBTC-USDT,UMA-USDT,UNI-USDT,USDC-USDT,UTK-USDT,VALUE-USDT,VELO-USDT,VIB-USDT,VRA-USDT,VSYS-USDT,WAVES-USDT,WBTC-USDT,WGRT-USDT,WING-USDT,WNXM-USDT,WTC-USDT,WXT-USDT,XCH-USDT,XEM-USDT,XLM-USDT,XMR-USDT,XPO-USDT,XPR-USDT,XRP-USDT,XSR-USDT,XTZ-USDT,XUC-USDT,YEE-USDT,YFI-USDT,YFII-USDT,YOU-USDT,YOYO-USDT,ZEC-USDT,ZEN-USDT,ZIL-USDT,ZKS-USDT,ZRX-USDT,ZYRO-USDT", "symbols, separate by comma")
	savePath := flag.String("path", "/root/okuf-ticker", "data save folder")

	//proxyAddress := flag.String("proxy", "socks5://127.0.0.1:1083", "symbols group batch size")
	//symbolsStr := flag.String("symbols", "BTC-USDT,AAVE-USDT,WAVES-USDT", "symbols, separate by comma")
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
		messageChMap := make(map[string]chan []byte)
		for _, symbol := range symbols[start:end] {
			messageChMap[symbol] = make(chan []byte, 10000)
			go saveLoop(ctx, cancel, *savePath, symbol, messageChMap[symbol], fileSavedCh)
		}
		go func(ctx context.Context, cancel context.CancelFunc, proxy string, outputChMap map[string]chan []byte) {
			ws := NewTickerWS(ctx, proxy, outputChMap)
			select {
			case <-ctx.Done():
			case <-ws.Done():
				cancel()
			}
		}(ctx, cancel, *proxyAddress, messageChMap)
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
