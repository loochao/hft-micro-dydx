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
)

type Depth20RoutedWebsocket struct {
	done        chan interface{}
	reconnectCh chan interface{}
	messageCh   chan []byte
	stopped     int32
}

func (w *Depth20RoutedWebsocket) readLoop(conn *websocket.Conn) {
	logger.Debugf("START readLoop")
	defer logger.Debugf("EXIT readLoop")
	logSilentTime := time.Now()
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
		//{"stream":"btcusdt@depth5@100ms","data":{"e":"depthUpdate","E":1616509191577,"T":1616509191571,"s":"CDEFH1INCHUSDT","U":276060537661,"u":276060540084,"pu":276060537525,"b":[["55302.93","1.203"],["55302.33","1.052"],["55302.32","0.036"],["55301.31","0.048"],["55301.30","1.936"],["55299.12","0.036"],["55299.11","0.240"],["55299.06","2.851"],["55299.01","0.124"],["55299.00","1.337"],["55298.52","0.100"],["55298.51","0.008"],["55298.41","0.110"],["55297.71","0.278"],["55297.31","0.292"],["55297.28","0.542"],["55297.18","0.362"],["55295.75","0.136"],["55295.68","0.160"],["55294.81","0.278"]],"a":[["55302.94","0.116"],["55305.98","0.202"],["55306.33","0.001"],["55306.58","0.054"],["55309.34","0.074"],["55309.36","0.090"],["55309.37","0.098"],["55309.52","0.116"],["55309.99","0.033"],["55310.62","0.181"],["55310.72","0.020"],["55311.04","0.217"],["55311.21","0.090"],["55311.41","0.181"],["55311.58","0.180"],["55311.59","0.519"],["55311.76","0.100"],["55311.86","0.243"],["55312.02","0.247"],["55312.42","0.090"]]}}
		if len(msg) < 128 {
			continue
		}
		select {
		case w.messageCh <- msg:
		default:
			if time.Now().Sub(logSilentTime) > 0 {
				logger.Debugf("w.messageCh <- msg failed len(w.messageCh) = %d", len(w.messageCh))
				logSilentTime = time.Now().Add(time.Minute)
			}
		}
	}
}

func (w *Depth20RoutedWebsocket) readAll(r io.Reader) ([]byte, error) {
	b := make([]byte, 0, 512)
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

func (w *Depth20RoutedWebsocket) mainLoop(ctx context.Context, proxy string, symbols []string) {
	urlStr := "wss://fstream.binance.com/stream?streams="
	for _, symbol := range symbols {
		urlStr += fmt.Sprintf(
			"%s@depth5@100ms/",
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
			go w.readLoop(conn)
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

func (w *Depth20RoutedWebsocket) saveLoop(ctx context.Context, savePath, symbols string, fileSavedCh chan string) {
	logger.Debugf("START saveLoop %s", symbols)
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
			_ = gw.Flush()
			_ = gw.Close()
		}
		if file != nil {
			_ = file.Close()
		}
		fileSavedCh <- symbols
		logger.Debugf("EXIT saveLoop %s", symbols)
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case <-hourUpdateTimer.C:
			if gw != nil {
				_ = gw.Flush()
				_ = gw.Close()
			}
			if file != nil {
				_ = file.Close()
			}
			dayTime = time.Now().Truncate(time.Hour * 24)
			outPath = fmt.Sprintf("%s/%s-%s.depth5.jl.gz", savePath, dayTime.Format("20060102"), symbols)
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
			hourUpdateTimer.Reset(
				time.Now().Truncate(
					time.Hour * 24,
				).Add(
					time.Hour * 24,
				).Add(
					time.Duration(rand.Intn(60)) * time.Second,
				).Sub(time.Now()),
			)
		case msg = <-w.messageCh:
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

func NewDepth5WS(
	ctx context.Context,
	proxy string,
	savePath string,
	symbols []string,
	fileSavedCh chan string,
) *Depth20RoutedWebsocket {
	ws := Depth20RoutedWebsocket{
		done:        make(chan interface{}),
		reconnectCh: make(chan interface{}),
		messageCh:   make(chan []byte, 128),
		stopped:     0,
	}
	go ws.saveLoop(ctx, savePath, strings.Join(symbols, ","), fileSavedCh)
	go ws.mainLoop(ctx, proxy, symbols)
	ws.reconnectCh <- nil
	return &ws
}
