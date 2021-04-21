package bnswap

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

type Depth20FilteredWebsocket struct {
	messageCh   chan []byte
	DataCh      chan *Depth20
	done        chan interface{}
	reconnectCh chan interface{}
	api         *API
	mu          sync.Mutex
	stopped     bool
}

func (w *Depth20FilteredWebsocket) startRead(conn *websocket.Conn) {
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
		select {
		case w.messageCh <- msg:
		default:
		}
	}

}

func (w *Depth20FilteredWebsocket) readAll(r io.Reader) ([]byte, error) {
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

func (w *Depth20FilteredWebsocket) startDataHandler(ctx context.Context, id int, decay, bias float64) {
	defer func() {
		logger.Debugf("EXIT startDataHandler %d", id)
	}()
	totalCount := 0
	filterCount := 0
	emaTimeDelta := bias
	timeDelta := 0.0
	decay1 := decay
	decay2 := 1.0 - decay
	logSilentTime := time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case msg := <-w.messageCh:
			totalCount++
			if totalCount > 10000 {
				logger.Debugf("EMA TIME DELTA %f DROP RATIO %f", emaTimeDelta, float64(filterCount)/float64(totalCount))
				totalCount = 0
				filterCount = 0
			}
			if len(msg) < 79 {
				continue
			}
			if msg[61] == 'E' {
				t, err := common.ParseInt(msg[64:77])
				if err != nil {
					logger.Debugf("ParseDepth20 error %v %s", err, msg[64:77])
					continue
				}
				timeDelta = float64(time.Now().UnixNano()/1000000 - t)
				if timeDelta > 1000 {
					timeDelta = 1000
				}
				emaTimeDelta = emaTimeDelta*decay1 + timeDelta*decay2
				if timeDelta > emaTimeDelta+bias {
					filterCount++
					continue
				}
			} else if msg[62] == 'E' {
				t, err := common.ParseInt(msg[65:78])
				if err != nil {
					logger.Debugf("ParseDepth20 error %v %s", err, msg[65:78])
					continue
				}
				timeDelta = float64(time.Now().UnixNano()/1000000 - t)
				if timeDelta > 1000 {
					timeDelta = 1000
				}
				emaTimeDelta = emaTimeDelta*decay1 + timeDelta*decay2
				if timeDelta > emaTimeDelta+bias {
					filterCount++
					continue
				}
			} else if msg[63] == 'E' {
				t, err := common.ParseInt(msg[66:79])
				if err != nil {
					logger.Debugf("ParseDepth20 error %v %s", err, msg[66:79])
					continue
				}
				timeDelta = float64(time.Now().UnixNano()/1000000 - t)
				if timeDelta > 1000 {
					timeDelta = 1000
				}
				emaTimeDelta = emaTimeDelta*decay1 + timeDelta*decay2
				if timeDelta > emaTimeDelta+bias {
					filterCount++
					continue
				}
			} else {
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("bad msg, can't find timestamp: %s", msg)
					logSilentTime = time.Now().Add(time.Minute)
				}
			}
			depth20, err := ParseDepth20(msg)
			if err != nil {
				logger.Debugf("ParseDepth20 error %v %s", err, msg)
				continue
			}
			select {
			case <-ctx.Done():
				return
			case <-w.done:
				return
			case <-time.After(time.Millisecond):
				logger.Warn("BNSWAP DEPTH20 TO OUTPUT CH TIME OUT IN 1MS, CH LEN %d", len(w.DataCh))
			case w.DataCh <- depth20:
				//if time.Now().Sub(logSilentTime) > 0 {
				//	logger.Debugf("BNSWAP DEPTH20 DATA CH LEN %d", len(w.DataCh))
				//	logSilentTime = time.Now().Add(time.Minute)
				//}
			}
		}
	}
}

func (w *Depth20FilteredWebsocket) reconnect(ctx context.Context, wsUrl string, proxy string, counter int64) (*websocket.Conn, error) {

	if counter != 0 {
		logger.Debugf("RECONNECT %s, %d RETRIES", wsUrl, counter)
	}

	var dialer *websocket.Dialer

	if proxy != "" {
		proxyUrl, err := url.Parse(proxy)
		if err != nil {
			logger.Debugf("PARSE PROXY %v", err)
			return nil, err
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

func (w *Depth20FilteredWebsocket) start(ctx context.Context, symbols []string, proxy string) {
	urlStr := "wss://fstream.binance.com/stream?streams="
	for _, symbol := range symbols {
		urlStr += fmt.Sprintf(
			"%s@depth20@100ms/",
			strings.ToLower(symbol),
		)
	}
	urlStr = urlStr[:len(urlStr)-1]
	logger.Debugf("BNSWAP DEPTH20 WS %s", urlStr)

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
				logger.Debugf("RECONNECT ERROR %v, STOP WS", err)
				if internalCancel != nil {
					internalCancel()
					internalCancel = nil
				}
				w.Stop()
				return
			}
			go w.startRead(conn)
			go w.maintainHeartbeat(internalCtx, conn)

		}
	}
}

func (w *Depth20FilteredWebsocket) maintainHeartbeat(ctx context.Context, conn *websocket.Conn) {

	defer func() {
		logger.Debugf("EXIT maintainHeartbeat")
		err := conn.Close()
		if err != nil {
			logger.Debugf("conn.Close() ERROR %v", err)
		}
	}()

	conn.SetPingHandler(func(msg string) error {
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

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		}
	}

}

func (w *Depth20FilteredWebsocket) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.stopped {
		w.stopped = true
		close(w.done)
	}
}

func (w *Depth20FilteredWebsocket) restart() {
	select {
	case <-w.done:
		return
	default:
	}
	select {
	case <-time.After(time.Second):
		w.Stop()
		logger.Debugf("BNSWAP NIL TO RECONNECT CH TIMEOUT IN 1S, STOP WS")
	case w.reconnectCh <- nil:
		logger.Infof("BNSWAP WS RESTART")
		return
	}
}

func (w *Depth20FilteredWebsocket) Done() chan interface{} {
	return w.done
}

func NewDepth20FilteredWebsocket(
	ctx context.Context,
	decay, bias float64,
	symbols []string,
	proxy string,
) *Depth20FilteredWebsocket {
	ws := Depth20FilteredWebsocket{
		done:        make(chan interface{}),
		reconnectCh: make(chan interface{}),
		DataCh:      make(chan *Depth20, 10*len(symbols)),
		messageCh:   make(chan []byte, 10*len(symbols)),
		mu:          sync.Mutex{},
		stopped:     false,
	}
	go ws.start(ctx, symbols, proxy)
	go ws.startDataHandler(ctx, 0, decay, bias)
	go ws.startDataHandler(ctx, 1, decay, bias)
	go ws.startDataHandler(ctx, 2, decay, bias)
	go ws.startDataHandler(ctx, 3, decay, bias)
	ws.reconnectCh <- nil
	return &ws
}
