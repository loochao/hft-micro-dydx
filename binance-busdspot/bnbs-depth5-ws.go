package binance_busdspot

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
)

type Depth5WS struct {
	done        chan interface{}
	reconnectCh chan interface{}
	stopped     int32
}

func (w *Depth5WS) readLoop(conn *websocket.Conn, channels map[string]chan []byte) {
	logger.Debugf("START readLoop")
	defer logger.Debugf("EXIT readLoop")
	logSilentTime := time.Now()
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
			logger.Warnf("w.readAll error %v", err)
			w.restart()
			return
		}
		if len(msg) > 128 {
			if msg[18] == '@' {
				symbol = common.UnsafeBytesToString(msg[11:18])
			} else if msg[19] == '@' {
				symbol = common.UnsafeBytesToString(msg[11:19])
			} else if msg[20] == '@' {
				symbol = common.UnsafeBytesToString(msg[11:20])
			} else if msg[21] == '@' {
				symbol = common.UnsafeBytesToString(msg[11:21])
			} else if msg[17] == '@' {
				symbol = common.UnsafeBytesToString(msg[11:17])
			} else {
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("other msg %s", msg)
					logSilentTime = time.Now().Add(time.Minute)
				}
				continue
			}
			if ch, ok = channels[symbol]; ok {
				select {
				case ch <- msg:
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("ch <- msg failed %s len(ch) = %d", symbol, len(ch))
						logSilentTime = time.Now().Add(time.Minute)
					}
				}
			}
		}
	}
}

func (w *Depth5WS) readAll(r io.Reader) ([]byte, error) {
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

func (w *Depth5WS) reconnect(ctx context.Context, wsUrl string, proxy string, counter int64) (*websocket.Conn, error) {

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

func (w *Depth5WS) mainLoop(ctx context.Context, channels map[string]chan []byte, proxy string) {
	urlStr := "wss://stream.binance.com:9443/stream?streams="
	for symbol := range channels {
		urlStr += fmt.Sprintf(
			"%s@depth5@100ms/",
			strings.ToLower(symbol),
		)
	}
	urlStr = urlStr[:len(urlStr)-1]
	logger.Debugf("START mainLoop %s", urlStr)
	var internalCtx context.Context
	var internalCancel context.CancelFunc

	defer func() {
		if internalCancel != nil {
			internalCancel()
			internalCancel = nil
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
			reconnectTimer.Reset(time.Second * 5)
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
			reconnectTimer.Reset(time.Hour * 9999)
		}
	}
}

func (w *Depth5WS) heartbeatLoop(ctx context.Context, conn *websocket.Conn) {
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

func (w *Depth5WS) Stop() {
	if atomic.CompareAndSwapInt32(&w.stopped, 0, 1) {
		close(w.done)
		logger.Debugf("stopped")
	}
}

func (w *Depth5WS) restart() {
	select {
	case <-w.done:
	case w.reconnectCh <- nil:
		logger.Debugf("restart")
	default:
		logger.Debugf("w.reconnectCh <- nil failed, stop ws")
		w.Stop()
	}
}

func (w *Depth5WS) Done() chan interface{} {
	return w.done
}

func (w *Depth5WS) dataHandleLoop(ctx context.Context, symbol string, inputCh chan []byte, outputCh chan common.Depth) {
	logSilentTime := time.Now()
	index := -1
	pool := [common.BufferSizeFor100msData]*Depth5{}
	for i := 0; i < common.BufferSizeFor100msData; i++ {
		pool[i] = &Depth5{}
	}
	var depth5 *Depth5
	var err error
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case msg := <-inputCh:
			index++
			if index == common.BufferSizeFor100msData {
				index = 0
			}
			depth5 = pool[index]
			err = ParseDepth5(msg, depth5)
			if err != nil {
				logger.Debugf("%s ParseDepth5 error %v %s", symbol, err, msg)
				continue
			}
			select {
			case outputCh <- depth5:
			default:
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("outputCh <- depth5 failed, %s ch len %d", symbol, len(outputCh))
					logSilentTime = time.Now().Add(time.Minute)
				}
			}
		}
	}
}

func NewDepth5WS(
	ctx context.Context,
	proxy string,
	channels map[string]chan common.Depth,
) *Depth5WS {
	ws := Depth5WS{
		done:        make(chan interface{}),
		reconnectCh: make(chan interface{}, common.ChannelSizeLowLoad),
		stopped:     0,
	}
	messageChs := make(map[string]chan []byte)
	for symbol, ch := range channels {
		messageChs[strings.ToLower(symbol)] = make(chan []byte, common.ChannelSizeLowLoadLowLatency)
		go ws.dataHandleLoop(ctx, symbol, messageChs[strings.ToLower(symbol)], ch)
	}
	go ws.mainLoop(ctx, messageChs, proxy)
	ws.reconnectCh <- nil
	return &ws
}
