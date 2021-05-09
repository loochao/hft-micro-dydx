package main

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/ftxperp"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/gorilla/websocket"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"sync/atomic"
	"time"
	"unsafe"
)

type TradeWS struct {
	writeCh     chan interface{}
	done        chan interface{}
	reconnectCh chan interface{}
	marketCh    chan string
	stopped     int32
}

func (w *TradeWS) writeLoop(ctx context.Context, conn *websocket.Conn) {
	logger.Debugf("START writeLoop")
	defer func() {
		logger.Debugf("EXIT writeLoop")
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case msg := <-w.writeCh:
			var msgBytes []byte
			var err error
			switch d := msg.(type) {
			case []byte:
				msgBytes = d
			case string:
				msgBytes = ([]byte)(d)
			default:
				msgBytes, err = json.Marshal(msg)
				if err != nil {
					logger.Debugf("Marshal err %v", err)
					continue
				}
			}
			err = conn.SetWriteDeadline(time.Now().Add(time.Minute))
			if err != nil {
				logger.Debugf("conn.SetWriteDeadline error %v", err)
				w.restart()
				return
			}
			err = conn.WriteMessage(websocket.TextMessage, msgBytes)
			if err != nil {
				logger.Debugf("conn.WriteMessage error %v, %s", err, string(msgBytes))
				w.restart()
				return
			}
		}
	}
}

func (w *TradeWS) readLoop(
	conn *websocket.Conn,
	channels map[string]chan []byte,
) {
	logger.Debugf("START readLoop")
	defer func() {
		logger.Debugf("EXIT readLoop")
	}()
	logSilentTime := time.Now()
	var symbol string
	var symbolBytes []byte
	var ok bool
	var ch chan []byte
	for {
		err := conn.SetReadDeadline(time.Now().Add(time.Minute))
		if err != nil {
			logger.Debugf("conn.SetReadDeadline error %v", err)
			go w.restart()
			return
		}
		_, r, err := conn.NextReader()
		if err != nil {
			logger.Debugf("conn.NextReader error %v", err)
			go w.restart()
			return
		}
		msg, err := w.readAll(r)
		if err != nil {
			logger.Debugf("w.readAll error %v", err)
			go w.restart()
			return
		}
		msgLen := len(msg)

		if msgLen > 128 && msg[13] == 't' {
			if msg[41] == '"' {
				symbolBytes = msg[33:41]
				symbol = *(*string)(unsafe.Pointer(&symbolBytes))
			} else if msg[42] == '"' {
				symbolBytes = msg[33:42]
				symbol = *(*string)(unsafe.Pointer(&symbolBytes))
			} else if msg[43] == '"' {
				symbolBytes = msg[33:43]
				symbol = *(*string)(unsafe.Pointer(&symbolBytes))
			} else if msg[44] == '"' {
				symbolBytes = msg[33:44]
				symbol = *(*string)(unsafe.Pointer(&symbolBytes))
			} else if msg[45] == '"' {
				symbolBytes = msg[33:45]
				symbol = *(*string)(unsafe.Pointer(&symbolBytes))
			} else {
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("other msg %s", msg)
					logSilentTime = time.Now().Add(time.Minute)
				}
				continue
			}
		} else {
			if time.Now().Sub(logSilentTime) > 0 && msgLen > 64{
				logger.Debugf("other msg %s", msg)
				logSilentTime = time.Now().Add(time.Minute)
			}
			continue
		}
		if ch, ok = channels[symbol]; ok {
			select {
			case ch <- msg:
				select {
				case w.marketCh <- symbol:
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("w.marketCh <- symbol %s ch len %d", symbol, len(ch))
						logSilentTime = time.Now().Add(time.Minute)
					}
				}
			default:
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("ch <- msg failed, ch len %d", len(ch))
					logSilentTime = time.Now().Add(time.Minute)
				}
			}
		}
	}
}

func (w *TradeWS) readAll(r io.Reader) ([]byte, error) {
	b := make([]byte, 0, 1024)
	for {
		if len(b) == cap(b) {
			// Add more capacity (let append pick how much).
			b = append(b, 0)[:len(b)]
		}
		n, err := r.Read(b[len(b):cap(b)])
		b = b[:len(b)+n]
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return b, err
		}
	}
}

func (w *TradeWS) reconnect(ctx context.Context, wsUrl string, proxy string, counter int64) (*websocket.Conn, error) {

	if counter != 0 {
		logger.Debugf("reconnect %d %s", counter, wsUrl)
	}

	var dialer *websocket.Dialer

	if proxy != "" {
		proxyUrl, err := url.Parse(proxy)
		if err != nil {
			return nil, fmt.Errorf("url.Parse error %v", err)
		}
		dialer = &websocket.Dialer{
			Proxy:            http.ProxyURL(proxyUrl),
			HandshakeTimeout: 60 * time.Second,
		}
	} else {
		dialer = &websocket.Dialer{
			HandshakeTimeout: 10 * time.Second,
		}
	}
	conn, _, err := dialer.DialContext(
		ctx,
		wsUrl,
		http.Header{
			"User-Agent":      []string{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/76.0.3809.132 Safari/537.36"},
			"Accept-Encoding": []string{"gzip, deflate, br"},
			"Accept-Language": []string{"en-US,en;q=0.9,zh-CN;q=0.8,zh;q=0.7,fr;q=0.6,nl;q=0.5,zh-TW;q=0.4,vi;q=0.3"},
		},
	)
	if err != nil {
		logger.Debugf("dialer.DialContext ERROR %v", err)
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("reconnect error: context is done")
		case <-w.done:
			return nil, fmt.Errorf("reconnect error: ws is done")
		case <-time.After(time.Second * 10):
			return w.reconnect(ctx, wsUrl, proxy, counter+1)
		}
	}
	return conn, nil
}

func (w *TradeWS) mainLoop(
	ctx context.Context,
	proxy string,
	channels map[string]chan []byte,
) {

	logger.Debugf("START mainLoop")

	markets := make([]string, 0)
	for symbol := range channels {
		markets = append(markets, symbol)
	}
	ctx, cancel := context.WithCancel(ctx)
	var internalCtx context.Context
	var internalCancel context.CancelFunc

	defer func() {
		cancel()
		if internalCancel != nil {
			internalCancel()
		}
		w.Stop()
		logger.Debugf("EXIT mainLoop")
	}()
	reconnectTimer := time.NewTimer(time.Hour * 9999)
	defer reconnectTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			if internalCancel != nil {
				internalCancel()
				internalCancel = nil
			}
			return
		case <-w.reconnectCh:
			if internalCancel != nil {
				internalCancel()
				internalCancel = nil
			}
			reconnectTimer.Reset(time.Second * 15)
		case <-reconnectTimer.C:
			if internalCancel != nil {
				internalCancel()
			}
			internalCtx, internalCancel = context.WithCancel(ctx)
			conn, err := w.reconnect(internalCtx, "wss://ftx.com/ws/", proxy, 0)
			if err != nil {
				if internalCancel != nil {
					internalCancel()
				}
				logger.Debugf("w.reconnect error %v, stop ws", err)
				return
			}
			go w.readLoop(conn, channels)
			go w.writeLoop(internalCtx, conn)
			go w.heartbeatLoop(internalCtx, conn, markets)
		}
	}
}

func (w *TradeWS) heartbeatLoop(ctx context.Context, conn *websocket.Conn, markets []string) {

	logger.Debugf("START heartbeatLoop")

	defer func() {
		err := conn.Close()
		if err != nil {
			logger.Debugf("conn.Close() ERROR %v", err)
		}
	}()

	symbolTimeout := time.Minute
	marketCheckInterval := time.Second
	pingTimer := time.NewTimer(time.Second)
	marketCheckTimer := time.NewTimer(time.Second)
	defer marketCheckTimer.Stop()
	marketUpdatedTimes := make(map[string]time.Time)
	for _, symbol := range markets {
		marketUpdatedTimes[symbol] = time.Unix(0, 0)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case symbol := <-w.marketCh:
			marketUpdatedTimes[symbol] = time.Now()
		case <-pingTimer.C:
			pingTimer.Reset(time.Second * 15)
			select {
			case w.writeCh <- []byte("{\"op\": \"ping\"}"):
			default:
				logger.Debugf("w.writeCh <- ping failed ch len %d", len(w.writeCh))

			}
			break
		case <-marketCheckTimer.C:
		loop:
			for market, updateTime := range marketUpdatedTimes {
				if time.Now().Sub(updateTime) > symbolTimeout {
					select {
					case w.writeCh <- ftxperp.SubscribeParam{
						Operation: "subscribe",
						Channel:   "trades",
						Market:    market,
					}:
						marketUpdatedTimes[market] = time.Now().Add(marketCheckInterval * time.Duration(len(markets)*2))
						break loop
					default:
						logger.Debugf("w.writeCh <- ftxperp.SubscribeParam failed, ch len %d", len(w.writeCh))
					}
				}
			}
			marketCheckTimer.Reset(marketCheckInterval)
			break
		}
	}

}

func (w *TradeWS) Stop() {
	if atomic.LoadInt32(&w.stopped) == 0 {
		atomic.StoreInt32(&w.stopped, 1)
		close(w.done)
		logger.Infof("KCPERP DEPTH5 WS STOPPED")
	}
}

func (w *TradeWS) restart() {
	select {
	case <-w.done:
	case <-time.After(time.Millisecond):
		w.Stop()
		logger.Debugf("KCPERP NIL TO RECONNECT CH TIMEOUT IN 1MS, STOP WS!")
	case w.reconnectCh <- nil:
		logger.Infof("KCPERP WS RESTART")
	}
}

func (w *TradeWS) Done() chan interface{} {
	return w.done
}

func (w *TradeWS) saveLoop(ctx context.Context, savePath, symbol string, inputCh chan []byte, fileSavedCh chan string) {
	logger.Debugf("START saveLoop %s", symbol)
	hourUpdateTimer := time.NewTimer(time.Second)
	var dayTime time.Time
	var outPath string
	var file *os.File
	var gw *gzip.Writer
	var msg []byte
	var err error
	var nextLine = []byte("\n")
	defer func() {
		if gw != nil {
			logger.Debugf("close gzip writer for %s", symbol)
			err = gw.Close()
			if err != nil {
				logger.Debugf("close gzip writer %s error %v, stop ws", outPath, err)
			}
		}
		if file != nil {
			logger.Debugf("close file %s", symbol)
			err = file.Close()
			if err != nil {
				logger.Debugf("close file %s error %v, stop ws", outPath, err)
			}
		}
		fileSavedCh <- symbol
		logger.Debugf("EXIT saveLoop %s", symbol)
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case <-hourUpdateTimer.C:
			if file != nil {
				err = file.Close()
				if err != nil {
					logger.Debugf("close file %s error %v, stop ws", outPath, err)
					w.Stop()
					return
				}
			}
			if gw != nil {
				err = gw.Close()
				if err != nil {
					logger.Debugf("close gzip writer %s error %v, stop ws", outPath, err)
					w.Stop()
					return
				}
			}
			dayTime = time.Now().Truncate(time.Hour * 24)
			outPath = fmt.Sprintf("%s/%s-%s.ftxperp.trade.jl.gz", savePath, dayTime.Format("20060102"), symbol)
			file, err = os.OpenFile(outPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0755)
			if err != nil {
				w.Stop()
				logger.Debugf("os.OpenFile error %v, stop ws", err)
				return
			}
			gw, err = gzip.NewWriterLevel(file, gzip.BestCompression)
			if err != nil {
				w.Stop()
				logger.Debugf("gzip.NewWriterLevel error %v, stop ws", err)
				return
			}
			gw.Name = fmt.Sprintf("%s-%s.depth20.jl", dayTime.Format("20060102"), symbol)
			gw.ModTime = time.Now()
			gw.Comment = fmt.Sprintf("depth20 raw json line for %s@%s", symbol, dayTime.Format("20060102"))
			hourUpdateTimer.Reset(
				time.Now().Truncate(
					time.Hour * 24,
				).Add(
					time.Hour * 24,
				).Add(
					time.Duration(rand.Intn(60)) * time.Second,
				).Sub(time.Now()),
			)
		case msg = <-inputCh:
			if gw != nil {
				_, err = gw.Write(msg)
				if err != nil {
					w.Stop()
					logger.Debugf("gw.Write error %v, stop ws", err)
					return
				}
				_, err = gw.Write(nextLine)
				if err != nil {
					w.Stop()
					logger.Debugf("gw.Write error %v, stop ws", err)
					return
				}
			}
		}
	}
}

func NewTradeWS(
	ctx context.Context,
	proxy string,
	savePath string,
	markets []string,
	fileSavedCh chan string,
) *TradeWS {
	ws := TradeWS{
		done:        make(chan interface{}),
		reconnectCh: make(chan interface{}),
		writeCh:     make(chan interface{}, 100*len(markets)),
		marketCh:    make(chan string, 100*len(markets)),
		stopped:     0,
	}
	channels := make(map[string]chan []byte)
	for _, symbol := range markets {
		channels[symbol] = make(chan []byte, 10000)
		go ws.saveLoop(ctx, savePath, symbol, channels[symbol], fileSavedCh)
	}
	go ws.mainLoop(ctx, proxy, channels)
	ws.reconnectCh <- nil
	return &ws
}
