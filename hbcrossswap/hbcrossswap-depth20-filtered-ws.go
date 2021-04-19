package hbcrossswap

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/gorilla/websocket"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type Depth20FilteredWebsocket struct {
	messageCh   chan []byte
	DataCh      chan *Depth20
	writeCh     chan interface{}
	done        chan interface{}
	reconnectCh chan interface{}
	RestartCh   chan interface{}
	symbolCh    chan string
	pingCh      chan []byte
	mu          sync.Mutex
	stopped     bool
}

func (w *Depth20FilteredWebsocket) startWrite(ctx context.Context, conn *websocket.Conn) {
	defer func() {
		logger.Debugf("EXIT startWrite")
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case msg := <-w.writeCh:
			var bytes []byte
			var err error
			switch d := msg.(type) {
			case []byte:
				bytes = d
			case string:
				bytes = ([]byte)(d)
			default:
				bytes, err = json.Marshal(msg)
				if err != nil {
					logger.Warnf("Marshal err %v", err)
					continue
				}
			}
			err = conn.SetWriteDeadline(time.Now().Add(300 * time.Millisecond))
			if err != nil {
				select {
				case <-ctx.Done():
					break
				default:
					w.restart()
				}
				return
			}

			err = conn.WriteMessage(websocket.TextMessage, bytes)
			if err != nil {
				logger.Warnf("WriteMessage %s error %v", string(bytes), err)
				select {
				case <-ctx.Done():
					break
				default:
					w.restart()
				}
				return
			}
		}
	}
}

func (w *Depth20FilteredWebsocket) startRead(ctx context.Context, conn *websocket.Conn) {
	defer func() {
		logger.Debugf("EXIT startRead")
	}()
	totalCount := 0
	totalLen := 0
	for {
		err := conn.SetReadDeadline(time.Now().Add(time.Minute))
		if err != nil {
			logger.Warnf("SetReadDeadline error %v", err)
			select {
			case <-ctx.Done():
				break
			default:
				w.restart()
			}
			return
		}
		_, r, err := conn.NextReader()
		if err != nil {
			logger.Warnf("NextReader error %v", err)
			select {
			case <-ctx.Done():
				break
			default:
				w.restart()
			}
			return
		}
		gr, err := gzip.NewReader(r)
		if err != nil {
			logger.Warnf("NewReader error %v", err)
			select {
			case <-ctx.Done():
				break
			default:
				w.restart()
			}
			return
		}
		msg, err := w.readAll(gr)
		if err != nil {
			logger.Warnf("readAll error %v", err)
			select {
			case <-ctx.Done():
				break
			default:
				w.restart()
			}
			return
		}
		totalCount += 1
		totalLen += len(msg)
		if totalCount > 1000000 {
			logger.Debugf("AVERAGE MESSAGE LENGTH %d/%d = %d", totalLen, totalCount, totalLen/totalCount)
			totalLen = 0
			totalCount = 0
		}
		select {
		case <-time.After(time.Millisecond):
			logger.Debug("SWAP DEPTH20 MSG TO MESSAGE CH TIMEOUT IN 1MS")
		case w.messageCh <- msg:
		}
		err = gr.Close()
		if err != nil {
			logger.Warnf("gr.Close() error %v", err)
			select {
			case <-ctx.Done():
				break
			default:
				w.restart()
			}
			return
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

func (w *Depth20FilteredWebsocket) startDataHandler(ctx context.Context, id int, decay float64) {
	defer func() {
		logger.Debugf("EXIT startDataHandler %d", id)
	}()
	totalCount := 0
	slowCount := 0
	emaTimeDelta := 100.0
	timeDelta := 0.0
	decay1 := decay
	decay2 := 1.0 - decay
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case msg := <-w.messageCh:
			if msg[2] == 'c' && len(msg) > 56{
				totalCount++
				if totalCount > 10000 {
					if totalCount > 0 {
						logger.Debugf("EMA TIME DELTA %f SLOW RATIO %f", emaTimeDelta,float64(slowCount)/float64(totalCount))
					}
					totalCount = 0
					slowCount = 0
				}
				if msg[40] == ':' {
					t, err := common.ParseInt(msg[41:54])
					if err != nil {
						logger.Debugf("ParseDepth20 error %v %s", err, msg[41:54])
						continue
					}
					timeDelta = float64(time.Now().UnixNano()/1000000-t)
					emaTimeDelta = emaTimeDelta*decay1 + timeDelta*decay2
					if timeDelta > emaTimeDelta {
						slowCount++
						continue
					}
				} else if msg[41] == 'E' {
					t, err := common.ParseInt(msg[42:55])
					if err != nil {
						logger.Debugf("ParseDepth20 error %v %s", err, msg[42:55])
						continue
					}
					timeDelta = float64(time.Now().UnixNano()/1000000-t)
					emaTimeDelta = emaTimeDelta*decay1 + timeDelta*decay2
					if timeDelta > emaTimeDelta {
						slowCount++
						continue
					}
				} else if msg[42] == 'E' {
					t, err := common.ParseInt(msg[43:56])
					if err != nil {
						logger.Debugf("ParseDepth20 error %v %s", err, msg[43:56])
						continue
					}
					timeDelta = float64(time.Now().UnixNano()/1000000-t)
					emaTimeDelta = emaTimeDelta*decay1 + timeDelta*decay2
					if timeDelta > emaTimeDelta {
						slowCount++
						continue
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
					logger.Warn("SWAP DEPTH20 TO OUTPUT CH TIME OUT IN 1MS")
				case w.DataCh <- depth20:
				}
				select {
				case w.symbolCh <- depth20.Symbol:
				default:
				}
			} else if msg[2] == 'p' {
				msg[3] = 'o'
				select {
				case w.pingCh <- msg:
				default:
				}
			} else {
				//logger.Debugf("OTHER MSG %s", msg)
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
			logger.Debugf("HBSWAP PARSE PROXY %v", err)
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

func (w *Depth20FilteredWebsocket) start(ctx context.Context, decay float64, symbols []string, proxy string) {

	ctx, cancel := context.WithCancel(ctx)
	var internalCtx context.Context
	var internalCancel context.CancelFunc

	defer func() {
		logger.Debugf("EXIT start")
		cancel()
		if internalCancel != nil {
			internalCancel()
		}
		w.Stop()
	}()
	reconnectTimer := time.NewTimer(time.Hour * 9999)
	defer reconnectTimer.Stop()
	for {
		select {
		case <-ctx.Done():
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
			conn, err := w.reconnect(internalCtx, "wss://api.hbdm.vn/linear-swap-ws", proxy, 0)
			if err != nil {
				logger.Debugf("RECONNECT ERROR %v, STOP WS", err)
				w.Stop()
				return
			}
			go w.startRead(internalCtx, conn)
			go w.startWrite(internalCtx, conn)
			go w.maintainHeartbeat(internalCtx, conn, symbols)

			go w.startDataHandler(internalCtx, 0, decay)
			go w.startDataHandler(internalCtx, 1, decay)
			go w.startDataHandler(internalCtx, 2, decay)
			go w.startDataHandler(internalCtx, 3, decay)

			//go w.startDataHandler(internalCtx, 4)
			//go w.startDataHandler(internalCtx, 5)
			//go w.startDataHandler(internalCtx, 6)
			//go w.startDataHandler(internalCtx, 7)
		}
	}
}

func (w *Depth20FilteredWebsocket) maintainHeartbeat(ctx context.Context, conn *websocket.Conn, symbols []string) {

	defer func() {
		logger.Debugf("EXIT maintainHeartbeat")
		err := conn.Close()
		if err != nil {
			logger.Debugf("conn.Close() ERROR %v", err)
		}
	}()

	symbolTimeout := time.Minute
	symbolCheckInterval := time.Second
	symbolCheckTimer := time.NewTimer(time.Second)
	defer symbolCheckTimer.Stop()
	symbolUpdatedTimes := make(map[string]time.Time)
	for _, symbol := range symbols {
		symbolUpdatedTimes[symbol] = time.Unix(0, 0)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case symbol := <-w.symbolCh:
			symbolUpdatedTimes[symbol] = time.Now()
		case msg := <-w.pingCh:
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Millisecond):
				logger.Debug("SEND PONG TO WRITE TIMEOUT IN 1MS")
				break
			case w.writeCh <- msg:
				break
			}
			break
		case <-symbolCheckTimer.C:
		loop:
			for symbol, updateTime := range symbolUpdatedTimes {
				if time.Now().Sub(updateTime) > symbolTimeout {
					//logger.Debugf("SWAP SUBSCRIBE %s", fmt.Sprintf("market.%s.depth.step6", symbol))
					select {
					case <-ctx.Done():
						return
					case <-time.After(time.Millisecond):
						logger.Debugf("SEND SUBSCRIBE %s TO WRITE TIMEOUT IN 1MS", fmt.Sprintf("market.%s.depth.step6", symbol))
						break
					case w.writeCh <- SubParam{
						ID:  fmt.Sprintf("%d", time.Now().UnixNano()),
						Sub: fmt.Sprintf("market.%s.depth.step6", symbol),
					}:
						symbolUpdatedTimes[symbol] = time.Now().Add(symbolCheckInterval * time.Duration(len(symbols)*2))
						break loop
					}
				}
			}
			symbolCheckTimer.Reset(symbolCheckInterval)
			break
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
	logger.Infof("SWAP WS RESTART")
	select {
	case <-w.done:
		return
	default:
	}
	select {
	case <-time.After(time.Second):
		logger.Debugf("SWAP NIL TO RESTART CH TIMEOUT IN 1S, STOP WS")
		w.Stop()
		return
	case w.RestartCh <- nil:
	}
	select {
	case <-time.After(time.Second):
		logger.Debugf("NIL TO RECONNECT CH TIMEOUT IN 1S, STOP WS")
		w.Stop()
		return
	case w.reconnectCh <- nil:
	}
}

func (w *Depth20FilteredWebsocket) Done() chan interface{} {
	return w.done
}

func NewDepth20FilteredWebsocket(
	ctx context.Context,
	decay float64,
	symbols []string,
	proxy string,
) *Depth20FilteredWebsocket {
	ws := Depth20FilteredWebsocket{
		done:        make(chan interface{}),
		reconnectCh: make(chan interface{}),
		DataCh:      make(chan *Depth20, 100*len(symbols)),
		RestartCh:   make(chan interface{}, 100),
		messageCh:   make(chan []byte, 100*len(symbols)),
		writeCh:     make(chan interface{}, 100*len(symbols)),
		symbolCh:    make(chan string, 100*len(symbols)),
		pingCh:      make(chan []byte, 100),
		mu:          sync.Mutex{},
		stopped:     false,
	}
	go ws.start(ctx, decay, symbols, proxy)
	ws.reconnectCh <- nil
	return &ws
}
