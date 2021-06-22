package main

import (
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/gorilla/websocket"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type FutureDepth5WS struct {
	done        chan interface{}
	reconnectCh chan interface{}
	stopped     bool
	mu          sync.Mutex
}

func (w *FutureDepth5WS) readLoop(conn *websocket.Conn, channels map[string]chan Message) {
	logger.Debugf("START readLoop")
	defer logger.Debugf("EXIT readLoop")
	logSilentTime := time.Now()
	var ch chan Message
	var ok bool
	var symbol string
	var outMsg = Message{
		Source: []byte("B"),
	}
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
		if len(msg) < 128 {
			continue
		}
		//{"stream":"btcusdt@depth5@100ms","data":{"e":"depthUpdate","E":1623494540877,"T":1623494540870,"s":"BTCUSDT","U":510743908847,"u":510743911822,"pu":510743908726,"b":[["35701.24","2.079"],["35701.23","0.276"],["35701.22","0.001"],["35700.35","0.400"],["35699.59","0.147"]],"a":[["35701.25","0.134"],["35704.02","0.248"],["35704.03","0.272"],["35704.55","0.001"],["35704.56","0.003"]]}}
		//{"stream":"linkusdt@depth5@100ms","data":{"e":"depthUpdate","E":1623494540955,"T":1623494540947,"s":"LINKUSDT","U":510743911258,"u":510743914224,"pu":510743910356,"b":[["21.030","12.37"],["21.029","448.68"],["21.027","2.12"],["21.024","240.12"],["21.022","47.62"]],"a":[["21.031","4.66"],["21.034","20.68"],["21.036","7.17"],["21.038","20.53"],["21.039","251.82"]]}}
		//{"stream":"wavesusdt@depth5@100ms","data":{"e":"depthUpdate","E":1623494540937,"T":1623494540873,"s":"WAVESUSDT","U":510743910668,"u":510743911915,"pu":510743903045,"b":[["14.2300","0.4"],["14.2270","59.0"],["14.2260","112.0"],["14.2250","78.5"],["14.2240","195.9"]],"a":[["14.2310","11.0"],["14.2340","38.4"],["14.2350","105.0"],["14.2360","3.5"],["14.2370","193.0"]]}}
		if msg[60] == 'E' {
			symbol = common.UnsafeBytesToString(msg[11:18])
		} else if msg[61] == 'E' {
			symbol = common.UnsafeBytesToString(msg[11:19])
		} else if msg[62] == 'E' {
			symbol = common.UnsafeBytesToString(msg[11:20])
		} else if msg[59] == 'E' {
			symbol = common.UnsafeBytesToString(msg[11:17])
		} else if msg[63] == 'E' {
			symbol = common.UnsafeBytesToString(msg[11:21])
		} else if msg[64] == 'E' {
			symbol = common.UnsafeBytesToString(msg[11:22])
		} else if msg[65] == 'E' {
			symbol = common.UnsafeBytesToString(msg[11:23])
		} else if msg[59] == 'E' {
			symbol = common.UnsafeBytesToString(msg[11:17])
		} else if msg[66] == 'E' {
			symbol = common.UnsafeBytesToString(msg[11:24])
		} else if msg[67] == 'E' {
			symbol = common.UnsafeBytesToString(msg[11:25])
		} else {
			if time.Now().Sub(logSilentTime) > 0 {
				logger.Debugf("bad msg, can't find symbol: %s", msg)
				logSilentTime = time.Now().Add(time.Minute)
			}
			continue
		}
		//logger.Debugf("%s %v", symbol, channels)
		if ch, ok = channels[symbol]; ok {
			outMsg.Data = msg
			select {
			case ch <- outMsg:
			default:
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("ch <- msg failed %s len(ch) = %d", symbol, len(ch))
					logSilentTime = time.Now().Add(time.Minute)
				}
			}
		}
	}
}

func (w *FutureDepth5WS) readAll(r io.Reader) ([]byte, error) {
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

func (w *FutureDepth5WS) reconnect(ctx context.Context, wsUrl string, proxy string, counter int64) (*websocket.Conn, error) {

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

func (w *FutureDepth5WS) mainLoop(ctx context.Context, proxy string, channels map[string]chan Message) {
	urlStr := "wss://fstream.binance.com/stream?streams="
	symbols := make([]string, 0)
	for symbol := range channels {
		symbols = append(symbols, symbol)
		urlStr += fmt.Sprintf(
			"%s@depth5@100ms/",
			strings.ToLower(symbol),
		)
	}
	urlStr = urlStr[:len(urlStr)-1]
	logger.Debugf("START mainLoop %s", urlStr)

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
			go w.readLoop(conn, channels)
			go w.heartbeatLoop(internalCtx, conn)

		}
	}
}

func (w *FutureDepth5WS) heartbeatLoop(ctx context.Context, conn *websocket.Conn) {
	logger.Debugf("START heartbeatLoop")
	defer func() {
		logger.Debugf("EXIT heartbeatLoop")
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

func (w *FutureDepth5WS) Stop() {
	w.mu.Lock()
	if !w.stopped {
		w.stopped = true
		close(w.done)
	}
	w.mu.Unlock()
}

func (w *FutureDepth5WS) restart() {
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

func (w *FutureDepth5WS) Done() chan interface{} {
	return w.done
}

func NewFutureDepth5WS(
	ctx context.Context,
	proxy string,
	channels map[string]chan Message,
) *FutureDepth5WS {
	ws := FutureDepth5WS{
		done:        make(chan interface{}),
		reconnectCh: make(chan interface{}, 4),
		stopped:     false,
		mu:          sync.Mutex{},
	}
	go ws.mainLoop(ctx, proxy, channels)
	ws.reconnectCh <- nil
	return &ws
}
