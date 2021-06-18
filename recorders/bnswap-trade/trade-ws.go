package main

import (
	"compress/gzip"
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/gorilla/websocket"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"
)

type Depth20RoutedWebsocket struct {
	done        chan interface{}
	reconnectCh chan interface{}
	stopped     int32
}

func (w *Depth20RoutedWebsocket) readLoop(conn *websocket.Conn, symbols []string, channels map[string]chan []byte) {
	logger.Debugf("START readLoop %s", symbols)
	defer logger.Debugf("EXIT readLoop %s", symbols)
	logSilentTime := time.Now()
	var symbolBytes []byte
	var symbol string
	var ch chan []byte
	var ok bool
	for {
		err := conn.SetReadDeadline(time.Now().Add(time.Minute))
		if err != nil {
			logger.Warnf("conn.SetReadDeadline error %v", err)
			w.restart()
			return
		}
		_, r, err := conn.NextReader()
		if err != nil {
			logger.Warnf("conn.NextReader error %v", err)
			w.restart()
			return
		}
		msg, err := w.readAll(r)
		if err != nil {
			logger.Warnf("w.readAl error %v", err)
			w.restart()
			return
		}
		//{"stream":"btcusdt@trade","data":{"e":"trade","E":1620011599983,"s":"BTCUSDT","t":804764023,"p":"58095.95000000","q":"0.02123000","b":5757184069,"a":5757184083,"T":1620011599982,"m":true,"M":true}}
		//{"stream":"scusdt@trade","data":{"e":"trade","E":1623166064549,"s":"SCUSDT","t":17318895,"p":"0.01451000","q":"1614.00000000","b":174967945,"a":174967958,"T":1623166064548,"m":true,"M":true}}
		if len(msg) < 128 {
			continue
		}
		if msg[17] == '@' {
			symbolBytes = msg[11:17]
			symbol = strings.ToUpper(*(*string)(unsafe.Pointer(&symbolBytes)))
		}else if msg[18] == '@' {
			symbolBytes = msg[11:18]
			symbol = strings.ToUpper(*(*string)(unsafe.Pointer(&symbolBytes)))
		} else if msg[19] == '@' {
			symbolBytes = msg[11:19]
			symbol = strings.ToUpper(*(*string)(unsafe.Pointer(&symbolBytes)))
		} else if msg[20] == '@' {
			symbolBytes = msg[11:20]
			symbol = strings.ToUpper(*(*string)(unsafe.Pointer(&symbolBytes)))
		} else if msg[21] == '@' {
			symbolBytes = msg[11:21]
			symbol = strings.ToUpper(*(*string)(unsafe.Pointer(&symbolBytes)))
		} else {
			if time.Now().Sub(logSilentTime) > 0 {
				logger.Debugf("bad msg, can't find symbol: %s", msg)
				logSilentTime = time.Now().Add(time.Minute)
			}
			continue
		}
		if ch, ok = channels[symbol]; ok {
			select {
			case ch <- msg:
			default:
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("ch <- &common.DepthRawMessage failed %s len(ch) = %d", symbol, len(ch))
					logSilentTime = time.Now().Add(time.Minute)
				}
			}
		}
	}
}

func (w *Depth20RoutedWebsocket) readAll(r io.Reader) ([]byte, error) {
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

func (w *Depth20RoutedWebsocket) reconnect(ctx context.Context, wsUrl string, proxy string, counter int64) (*websocket.Conn, error) {

	if counter != 0 {
		logger.Debugf("RECONNECT %s, %d RETRIES", wsUrl, counter)
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
		logger.Warnf("dialer.DialContext ERROR %v", err)
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

func (w *Depth20RoutedWebsocket) mainLoop(ctx context.Context, proxy string, channels map[string]chan []byte) {
	urlStr := "wss://fstream.binance.com:9443/stream?streams="
	symbols := make([]string, 0)
	for symbol := range channels {
		symbols = append(symbols, symbol)
		urlStr += fmt.Sprintf(
			"%s@trade/",
			strings.ToLower(symbol),
		)
	}
	urlStr = urlStr[:len(urlStr)-1]
	logger.Debugf("START mainLoop %s", symbols)

	ctx, cancel := context.WithCancel(ctx)
	var internalCtx context.Context
	var internalCancel context.CancelFunc

	defer func() {
		w.Stop()
		cancel()
		logger.Debugf("EXIT mainLoop %s", symbols)
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
				internalCancel = nil
			}
			internalCtx, internalCancel = context.WithCancel(ctx)
			conn, err := w.reconnect(internalCtx, urlStr, proxy, 0)
			if err != nil {
				logger.Debugf("w.reconnect error %v, stop ws", err)
				if internalCancel != nil {
					internalCancel()
					internalCancel = nil
				}
				w.Stop()
				return
			}
			go w.readLoop(conn, symbols, channels)
			go w.heartbeatLoop(internalCtx, conn, symbols)

		}
	}
}

func (w *Depth20RoutedWebsocket) heartbeatLoop(ctx context.Context, conn *websocket.Conn, symbols []string) {
	logger.Debugf("START heartbeatLoop %s", symbols)
	defer func() {
		logger.Debugf("EXIT heartbeatLoop %s", symbols)
		err := conn.Close()
		if err != nil {
			logger.Debugf("conn.Close() error %v", err)
		}
	}()

	trafficCh := make(chan interface{})

	conn.SetPingHandler(func(msg string) error {
		select {
		case trafficCh <- nil:
		default:
		}
		err := conn.WriteControl(websocket.PongMessage, []byte(msg), time.Now().Add(time.Minute))
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				w.restart()
			}
			return nil
		}
		return nil
	})

	timer := time.NewTimer(time.Minute * 15)

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case <-timer.C:
			logger.Warnf("no traffic in %v, restart ws", time.Minute*15)
			w.restart()
			return
		case <-trafficCh:
			timer.Reset(time.Minute * 15)
		}
	}

}

func (w *Depth20RoutedWebsocket) Stop() {
	if atomic.LoadInt32(&w.stopped) == 0 {
		atomic.StoreInt32(&w.stopped, 1)
		close(w.done)
		logger.Debugf("stopped")
	}
}

func (w *Depth20RoutedWebsocket) restart() {
	select {
	case <-w.done:
		return
	default:
	}
	select {
	case <-time.After(time.Second):
		w.Stop()
		logger.Debugf("w.reconnectCh <- nil timeout in 1s, stop ws")
	case w.reconnectCh <- nil:
		logger.Debugf("restart ws")
		return
	}
}

func (w *Depth20RoutedWebsocket) Done() chan interface{} {
	return w.done
}

func (w *Depth20RoutedWebsocket) saveLoop(ctx context.Context, savePath, symbol string, inputCh chan []byte, fileSavedCh chan string) {
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
			outPath = fmt.Sprintf("%s/%s-%s.bnswap.trade.jl.gz", savePath, dayTime.Format("20060102"), symbol)
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
			gw.Name = fmt.Sprintf("%s-%s.trade.jl", dayTime.Format("20060102"), symbol)
			gw.ModTime = time.Now()
			gw.Comment = fmt.Sprintf("trade raw json line for %s@%s", symbol, dayTime.Format("20060102"))
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

func NewTradeRoutedWS(
	ctx context.Context,
	proxy string,
	savePath string,
	symbols []string,
	fileSavedCh chan string,
) *Depth20RoutedWebsocket {
	ws := Depth20RoutedWebsocket{
		done:        make(chan interface{}),
		reconnectCh: make(chan interface{}),
		stopped:     0,
	}
	channels := make(map[string]chan []byte)
	for _, symbol := range symbols {
		channels[symbol] = make(chan []byte, 10000)
		go ws.saveLoop(ctx, savePath, symbol, channels[symbol], fileSavedCh)
	}
	go ws.mainLoop(ctx, proxy, channels)
	ws.reconnectCh <- nil
	return &ws
}
