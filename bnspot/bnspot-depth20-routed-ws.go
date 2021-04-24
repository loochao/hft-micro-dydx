package bnspot

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
	"sync/atomic"
	"time"
	"unsafe"
)

type Depth20RoutedWebsocket struct {
	messageCh   chan []byte
	done        chan interface{}
	reconnectCh chan interface{}
	stopped     int32
}

func (w *Depth20RoutedWebsocket) readLoop(conn *websocket.Conn, channels map[string]chan *common.DepthRawMessage) {
	logger.Debugf("START readLoop")
	defer logger.Debugf("EXIT readLoop")
	logSilentTime := time.Now()
	var symbol string
	var symbolBytes []byte
	var ch chan *common.DepthRawMessage
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
			logger.Warnf("w.readAll error %v", err)
			w.restart()
			return
		}
		//{"stream":"flmusdt@depth20@100ms","data":{"lastUpdateId":165284515,"bids":[["0.48560000","2036.02000000"],["0.48550000","480.00000000"],["0.48520000","14257.67000000"],["0.48510000","1056.25000000"],["0.48500000","1894.32000000"],["0.48480000","2145.67000000"],["0.48460000","2196.59000000"],["0.48330000","3000.00000000"],["0.48320000","2531.26000000"],["0.48310000","21.18000000"],["0.48300000","4292.54000000"],["0.48270000","5042.00000000"],["0.48240000","5051.00000000"],["0.48230000","24.83000000"],["0.48220000","457.11000000"],["0.48200000","4142.12000000"],["0.48160000","31.15000000"],["0.48150000","71.96000000"],["0.48130000","1284.94000000"],["0.48110000","1098.85000000"]],"asks":[["0.48630000","5601.00000000"],["0.48650000","990.00000000"],["0.48670000","7816.00000000"],["0.48680000","7914.96000000"],["0.48690000","963.00000000"],["0.48720000","3640.00000000"],["0.48730000","814.24000000"],["0.48780000","3560.00000000"],["0.48800000","1029.00000000"],["0.48880000","13221.24000000"],["0.48940000","3000.00000000"],["0.48980000","62.75000000"],["0.49000000","1482.94000000"],["0.49040000","516.34000000"],["0.49110000","46.50000000"],["0.49120000","27.10000000"],["0.49130000","31.03000000"],["0.49150000","66.27000000"],["0.49160000","1291.65000000"],["0.49190000","159.76000000"]]}}
		if len(msg) > 128 {
			if msg[18] == '@' {
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
			}else{
				if time.Now().Sub(logSilentTime)> 0 {
					logger.Debugf("other msg %s", msg)
					logSilentTime = time.Now().Add(time.Minute)
				}
				continue
			}
		}
		if ch, ok = channels[symbol]; ok {
			select {
			case ch <- &common.DepthRawMessage{
				Symbol: symbol,
				Time:   time.Now(),
				Depth:  msg,
			}:
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
	b := make([]byte, 0, 2048)
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
		logger.Debugf("reconnect %s, %d retries", wsUrl, counter)
	}

	var dialer *websocket.Dialer

	if proxy != "" {
		proxyUrl, err := url.Parse(proxy)
		if err != nil {
			return nil, fmt.Errorf("url.Parse(proxy) error %v", err)
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
		logger.Warnf("dialer.DialContext error %v", err)
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

func (w *Depth20RoutedWebsocket) mainLoop(ctx context.Context, proxy string, channels map[string]chan *common.DepthRawMessage) {
	logger.Debugf("START mainLoop")
	urlStr := "wss://stream.binance.com:9443/stream?streams="
	for symbol := range channels {
		urlStr += fmt.Sprintf(
			"%s@depth20@100ms/",
			strings.ToLower(symbol),
		)
	}
	urlStr = urlStr[:len(urlStr)-1]

	ctx, cancel := context.WithCancel(ctx)
	var internalCtx context.Context
	var internalCancel context.CancelFunc

	defer func() {
		cancel()
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
		case <-w.done:
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
				return
			}
			go w.readLoop(conn, channels)
			go w.heartbeatLoop(internalCtx, conn)
		}
	}
}

func (w *Depth20RoutedWebsocket) heartbeatLoop(ctx context.Context, conn *websocket.Conn) {
	logger.Debugf("START heartbeatLoop")
	defer func() {
		logger.Debugf("EXIT heartbeatLoop")
		err := conn.Close()
		if err != nil {
			logger.Debugf("conn.Close() ERROR %v", err)
		}
	}()

	trafficCh := make(chan interface{}, 100)
	conn.SetPingHandler(func(msg string) error {
		select {
		case trafficCh <- nil:
		default:
		}
		err := conn.WriteControl(websocket.PongMessage, []byte(msg), time.Now().Add(time.Minute))
		if err != nil {
			w.restart()
			return err
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
	case w.reconnectCh <- nil:
		logger.Debugf("restart")
	default:
		logger.Debugf("w.reconnectCh <- nil failed, stop ws")
		w.Stop()
	}
}

func (w *Depth20RoutedWebsocket) Done() chan interface{} {
	return w.done
}


func NewDepth20RoutedWebsocket(
	ctx context.Context,
	proxy string,
	channels map[string]chan *common.DepthRawMessage,
) *Depth20RoutedWebsocket {
	ws := Depth20RoutedWebsocket{
		done:        make(chan interface{}),
		reconnectCh: make(chan interface{}, 100),
		messageCh:   make(chan []byte, 10*len(channels)),
		stopped:     0,
	}
	go ws.mainLoop(ctx, proxy, channels)
	ws.reconnectCh <- nil
	return &ws
}
