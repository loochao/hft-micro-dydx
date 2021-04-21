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
	"sync"
	"time"
)

type Depth20Websocket struct {
	messageCh   chan []byte
	DataCh      chan *Depth20
	done        chan interface{}
	reconnectCh chan interface{}
	mu          sync.Mutex
	stopped     bool
}

func (w *Depth20Websocket) startRead(conn *websocket.Conn) {
	defer func() {
		logger.Debugf("EXIT startRead")
	}()
	for {
		err := conn.SetReadDeadline(time.Now().Add(time.Minute))
		if err != nil {
			logger.Warnf("SetReadDeadline error %v", err)
			w.restart()
			return
		}
		_, r, err := conn.NextReader()
		if err != nil {
			logger.Warnf("NextReader error %v", err)
			w.restart()
			return
		}
		msg, err := w.readAll(r)
		if err != nil {
			logger.Warnf("readAll error %v", err)
			w.restart()
			return
		}
		//以最快的速度送走数据, 进行一下个LOOP, 盘口的数据是可以丢的
		select {
		case w.messageCh <- msg:
		default:
		}
	}
}

func (w *Depth20Websocket) readAll(r io.Reader) ([]byte, error) {
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

func (w *Depth20Websocket) reconnect(ctx context.Context, wsUrl string, proxy string, counter int64) (*websocket.Conn, error) {

	if counter != 0 {
		logger.Debugf("RECONNECT %s, %d RETRIES", wsUrl, counter)
	}

	var dialer *websocket.Dialer

	if proxy != "" {
		proxyUrl, err := url.Parse(proxy)
		if err != nil {
			return nil, fmt.Errorf("PARSE PROXY %v", err)
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

func (w *Depth20Websocket) start(ctx context.Context, symbols []string, proxy string) {
	urlStr := "wss://stream.binance.com:9443/stream?streams="
	for _, symbol := range symbols {
		urlStr += fmt.Sprintf(
			"%s@depth20@100ms/",
			strings.ToLower(symbol),
		)
	}
	urlStr = urlStr[:len(urlStr)-1]
	logger.Debugf("BNSPOT DEPTH20 WS %s", urlStr)

	ctx, cancel := context.WithCancel(ctx)
	var internalCtx context.Context
	var internalCancel context.CancelFunc

	defer func() {
		w.Stop()
		cancel()
		logger.Debugf("EXIT start")
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
				logger.Debugf("RECONNECT ERROR %v", err)
				if internalCancel != nil {
					internalCancel()
					internalCancel = nil
				}
				return
			}
			go w.startRead(conn)
			go w.maintainHeartbeat(internalCtx, conn)
		}
	}
}

func (w *Depth20Websocket) maintainHeartbeat(ctx context.Context, conn *websocket.Conn) {

	defer func() {
		logger.Debugf("EXIT maintainHeartbeat")
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
			logger.Warnf("BNSPOT DEPTH20 WS, NO TRAFFIC IN 15M, RESTART")
			w.restart()
			return
		case <-trafficCh:
			timer.Reset(time.Minute * 15)
		}
	}

}

func (w *Depth20Websocket) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.stopped {
		w.stopped = true
		close(w.done)
		logger.Infof("BNSPOT DEPTH20 WS STOPPED")
	}
}

func (w *Depth20Websocket) restart() {
	select {
	case <-w.done:
	case <-time.After(time.Second):
		logger.Debugf("BNSPOT NIL TO RECONNECT CH TIMEOUT IN 1S, EXIT WS")
		w.Stop()
	case w.reconnectCh <- nil:
		logger.Debugf("BNSPOT DEPTH20 WS RESTART")
	}
}

func (w *Depth20Websocket) Done() chan interface{} {
	return w.done
}

func (w *Depth20Websocket) startDataHandler(ctx context.Context) {
	logSilentTime := time.Now()
	totalLen := 0
	totalCount := 0
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
			depth20, err := ParseDepth20(msg)
			if err != nil {
				logger.Debugf("BNSPOT ParseDepth20 error %v %s", err, msg)
				continue
			}
			select {
			case <-ctx.Done():
				return
			case <-w.done:
				return
			case <-time.After(time.Millisecond):
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Warn("BNSPOT DEPTH20 TO OUTPUT CH TIME OUT IN 1MS")
					logSilentTime = time.Now().Add(time.Minute)
				}
			case w.DataCh <- depth20:
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("BNSPOT DEPTH20 DATA CH LEN %d", len(w.DataCh))
					logSilentTime = time.Now().Add(time.Minute)
				}
			}
		}
	}
}

func NewDepth20Websocket(
	ctx context.Context,
	symbols []string,
	proxy string,
) *Depth20Websocket {
	ws := Depth20Websocket{
		done:        make(chan interface{}),
		reconnectCh: make(chan interface{}),
		DataCh:      make(chan *Depth20, 100*len(symbols)),
		messageCh:   make(chan []byte, 400*len(symbols)),
		stopped:     false,
		mu:          sync.Mutex{},
	}
	go ws.start(ctx, symbols, proxy)
	go ws.startDataHandler(ctx)
	go ws.startDataHandler(ctx)
	go ws.startDataHandler(ctx)
	go ws.startDataHandler(ctx)
	ws.reconnectCh <- nil
	return &ws
}
