package bnspot

import (
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/gorilla/websocket"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"
)

type Depth5Websocket struct {
	messageCh   chan []byte
	DataCh      chan *Depth5
	done        chan interface{}
	reconnectCh chan interface{}
	stopped     int32
}

func (w *Depth5Websocket) readLoop(conn *websocket.Conn) {
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
			logger.Warnf("w.readAll error %v", err)
			w.restart()
			return
		}
		//以最快的速度送走数据, 进行一下个LOOP, 盘口的数据是可以丢的
		select {
		case w.messageCh <- msg:
		default:
			if time.Now().Sub(logSilentTime) > 0 {
				logger.Debugf("w.messageCh <- msg failed, len(w.messageCh) = %d, msg %s", len(w.messageCh), msg)
				logSilentTime = time.Now().Add(time.Minute)
			}
		}
	}
}

func (w *Depth5Websocket) readAll(r io.Reader) ([]byte, error) {
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

func (w *Depth5Websocket) reconnect(ctx context.Context, wsUrl string, proxy string, counter int64) (*websocket.Conn, error) {

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

func (w *Depth5Websocket) mainLoop(ctx context.Context, symbols []string, proxy string) {
	logger.Debugf("START mainLoop")
	urlStr := "wss://stream.binance.com:9443/stream?streams="
	for _, symbol := range symbols {
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
			go w.readLoop(conn)
			go w.heartbeatLoop(internalCtx, conn)
		}
	}
}

func (w *Depth5Websocket) heartbeatLoop(ctx context.Context, conn *websocket.Conn) {
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

func (w *Depth5Websocket) Stop() {
	if atomic.LoadInt32(&w.stopped) == 0 {
		atomic.StoreInt32(&w.stopped, 1)
		close(w.done)
		logger.Debugf("stopped")
	}
}

func (w *Depth5Websocket) restart() {
	select {
	case <-w.done:
	case w.reconnectCh <- nil:
		logger.Debugf("restart")
	default:
		logger.Debugf("w.reconnectCh <- nil failed, stop ws")
		w.Stop()
	}
}

func (w *Depth5Websocket) Done() chan interface{} {
	return w.done
}

func (w *Depth5Websocket) dataHandleLoop(ctx context.Context) {
	logger.Debugf("START dataHandleLoop")
	defer logger.Debugf("EXIT dataHandleLoop")
	logSilentTime := time.Now()
	totalLen := 0
	totalCount := 0
	logSilentTime = time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case msg := <-w.messageCh:
			totalCount += 1
			totalLen += len(msg)
			if totalCount > 1000000 {
				logger.Debugf("AVERAGE MESSAGE LENGTH %d/%d = %d", totalLen, totalCount, totalLen/totalCount)
				totalLen = 0
				totalCount = 0
			}
			if msg[2] != 's' {
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("BNSPOT OTHER MSG %s", msg)
					logSilentTime = time.Now().Add(time.Minute)
				}
				continue
			}
			depth20, err := ParseDepth5(msg)
			if err != nil {
				logger.Debugf("ParseDepth5 error %v %s", err, msg)
				continue
			}
			select {
			case w.DataCh <- depth20:
			default:
				logger.Debugf("w.DataCh <- depth20 failed, len(w.DataCh) = %d", len(w.DataCh))
			}
		}
	}
}

func NewDepth5Websocket(
	ctx context.Context,
	symbols []string,
	proxy string,
) *Depth5Websocket {
	ws := Depth5Websocket{
		done:        make(chan interface{}),
		reconnectCh: make(chan interface{}, 100),
		DataCh:      make(chan *Depth5, 10*len(symbols)),
		messageCh:   make(chan []byte, 10*len(symbols)),
		stopped:     0,
	}
	go ws.mainLoop(ctx, symbols, proxy)
	go ws.dataHandleLoop(ctx)
	go ws.dataHandleLoop(ctx)
	go ws.dataHandleLoop(ctx)
	go ws.dataHandleLoop(ctx)
	ws.reconnectCh <- nil
	return &ws
}
