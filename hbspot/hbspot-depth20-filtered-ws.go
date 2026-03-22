package hbspot

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
	"sync/atomic"
	"time"
)

type Depth20FilteredWebsocket struct {
	messageCh   chan []byte
	DataCh      chan *Depth20
	writeCh     chan interface{}
	done        chan interface{}
	reconnectCh chan interface{}
	symbolCh    chan string
	pingCh      chan []byte
	stopped     int32
}

func (w *Depth20FilteredWebsocket) writeLoop(ctx context.Context, conn *websocket.Conn) {
	logger.Debugf("START writeLoop")
	defer logger.Debugf("EXIT writeLoop")
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
					logger.Debugf("json.Marshal(msg) err %v", err)
					continue
				}
			}
			err = conn.SetWriteDeadline(time.Now().Add(time.Minute))
			if err != nil {
				logger.Debugf("conn.SetWriteDeadline error %v", err)
				w.restart()
				return
			}

			err = conn.WriteMessage(websocket.TextMessage, bytes)
			if err != nil {
				logger.Debugf("conn.WriteMessage error %v", err)
				w.restart()
				return
			}
		}
	}
}

func (w *Depth20FilteredWebsocket) readLoop(conn *websocket.Conn) {
	logger.Debugf("START readLoop")
	defer logger.Debugf("EXIT readLoop")
	totalCount := 0
	totalLen := 0
	logSilentTime := time.Now()
	for {
		err := conn.SetReadDeadline(time.Now().Add(time.Minute))
		if err != nil {
			logger.Debugf("conn.SetReadDeadline error %v", err)
			w.restart()
			return
		}
		_, r, err := conn.NextReader()
		if err != nil {
			logger.Debugf("conn.NextReader error %v", err)
			w.restart()
			return
		}
		gr, err := gzip.NewReader(r)
		if err != nil {
			logger.Debugf("gzip.NewReader error %v", err)
			w.restart()
			return
		}
		msg, err := w.readAll(gr)
		if err != nil {
			logger.Debugf("w.readAll error %v", err)
			w.restart()
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
		case w.messageCh <- msg:
		default:
			if time.Now().Sub(logSilentTime) > 0 {
				logSilentTime = time.Now().Add(time.Minute)
				logger.Debugf("w.messageCh <- msg failed ch len %d", len(w.messageCh))
			}
		}
		err = gr.Close()
		if err != nil {
			logger.Debugf("gr.Close() error %v", err)
			go w.restart()
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

func (w *Depth20FilteredWebsocket) dataHandleLoop(ctx context.Context, id int, decay, bias float64) {
	logger.Debugf("START dataHandleLoop %d", id)
	defer logger.Debugf("EXIT dataHandleLoop %d", id)
	totalCount := 0
	filterCount := 0
	emaTimeDelta := 50.0
	timeDelta := 0.0
	decay1 := decay
	decay2 := 1.0 - decay
	logSilentTime := time.Now()
	bytesLen := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case msg := <-w.messageCh:
			if msg[2] == 'c' {
				bytesLen = len(msg)
				if bytesLen < 17 {
					continue
				}
				//{"ch":"market.btcusdt.depth.step1","ts":1618845880265,"tick":{"bids":[[55177.6,0.292371],[55176.8,0.084949],[55170.4,0.67],[55170.2,0.032873],[55169.0,0.01],[55165.8,0.1],[55163.1,0.32],[55159.6,1.28],[55156.5,0.99],[55155.9,0.003489],[55155.0,1.195819],[55154.3,0.014323],[55154.1,0.03],[55153.8,2.0E-4],[55153.1,0.017567],[55151.0,0.016979],[55150.4,0.132173],[55150.3,0.052],[55150.0,0.005958],[55146.2,2.0E-4]],"asks":[[55177.7,0.006907],[55182.3,0.004239],[55182.5,0.001],[55186.6,0.685273],[55187.0,0.006],[55188.7,0.006],[55189.7,0.006],[55190.0,0.003636],[55191.4,0.010475],[55192.0,0.2],[55192.7,0.054338],[55192.8,0.1],[55193.1,6.5E-4],[55193.4,0.136479],[55194.0,0.018871],[55195.5,0.01],[55196.5,0.01891],[55197.4,0.0057],[55198.0,0.006],[55198.3,2.0E-4]],"version":125371959389,"ts":1618845880259}}
				totalCount++
				if totalCount > 10000 {
					logger.Debugf("EMA TIME DELTA %f DROP RATIO %f", emaTimeDelta, float64(filterCount)/float64(totalCount))
					totalCount = 0
					filterCount = 0
				}
				if msg[bytesLen-16] == ':' {
					t, err := common.ParseInt(msg[bytesLen-15 : bytesLen-2])
					if err != nil {
						logger.Debugf("ParseDepth20 error %v %s", err, msg[bytesLen-15:bytesLen-2])
						continue
					}
					timeDelta = float64(time.Now().UnixNano()/1000000 - t)
					if timeDelta > 1000 {
						timeDelta = 1000
					}
					//logger.Debugf("%d %s %v %f", bytesLen-16, msg[bytesLen-15:bytesLen-2], t, timeDelta)
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
				case w.DataCh <- depth20:
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("w.DataCh <- depth20 failed, ch len %d", len(w.DataCh))
						logSilentTime = time.Now().Add(time.Minute)
					}
				}
				select {
				case w.symbolCh <- depth20.Symbol:
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("w.symbolCh <- depth20.Market failed, ch len %d", len(w.symbolCh))
						logSilentTime = time.Now().Add(time.Minute)
					}
				}
			} else if msg[2] == 'p' {
				msg[3] = 'o'
				select {
				case w.pingCh <- msg:
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("w.pingCh <- msg failed, ch len %d", len(w.pingCh))
						logSilentTime = time.Now().Add(time.Minute)
					}
				}
			} else {
				if msg[2] != 'i' {
					logger.Debugf("other msg %s", msg)
				}
			}
		}
	}
}

func (w *Depth20FilteredWebsocket) reconnect(ctx context.Context, wsUrl string, proxy string, counter int64) (*websocket.Conn, error) {

	if counter != 0 {
		logger.Debugf("reconnect %s, %d retires", wsUrl, counter)
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

func (w *Depth20FilteredWebsocket) mainLoop(ctx context.Context, symbols []string, proxy string, decay, bias float64) {
	logger.Debugf("START mainLoop")
	defer logger.Debugf("EXIT mainLoop")
	ctx, cancel := context.WithCancel(ctx)
	var internalCtx context.Context
	var internalCancel context.CancelFunc

	defer func() {
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
			conn, err := w.reconnect(internalCtx, "wss://api-aws.huobi.pro/ws", proxy, 0)
			if err != nil {
				logger.Debugf("w.reconnect error %v", err)
				internalCancel()
				w.Stop()
				return
			}
			go w.readLoop(conn)
			go w.writeLoop(internalCtx, conn)
			go w.heartbeatLoop(internalCtx, conn, symbols)
		}
	}
}

func (w *Depth20FilteredWebsocket) heartbeatLoop(ctx context.Context, conn *websocket.Conn, symbols []string) {
	logger.Debugf("START heartbeatLoop")
	defer func() {
		logger.Debugf("Exit heartbeatLoop")
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
		case <-w.done:
			return
		case symbol := <-w.symbolCh:
			symbolUpdatedTimes[symbol] = time.Now()
			break
		case msg := <-w.pingCh:
			select {
			case w.writeCh <- msg:
				break
			default:
				logger.Debugf("w.writeCh <- msg failed, ch len %d", len(w.writeCh))
			}
			break
		case <-symbolCheckTimer.C:
		loop:
			for symbol, updateTime := range symbolUpdatedTimes {
				if time.Now().Sub(updateTime) > symbolTimeout {
					logger.Debugf("SUBSCRIBE %s", fmt.Sprintf("market.%s.depth.step1", symbol))
					select {
					case w.writeCh <- SubParam{
						ID:  fmt.Sprintf("market.%s.depth.step1", symbol),
						Sub: fmt.Sprintf("market.%s.depth.step1", symbol),
					}:
						symbolUpdatedTimes[symbol] = time.Now().Add(symbolCheckInterval * time.Duration(len(symbols)*2))
						break loop
					default:
						logger.Debugf("w.writeCh <- SubParam failed, ch len %d", len(w.writeCh))
					}
				}
			}
			symbolCheckTimer.Reset(symbolCheckInterval)
			break
		}
	}

}

func (w *Depth20FilteredWebsocket) Stop() {
	if atomic.LoadInt32(&w.stopped) == 0 {
		atomic.StoreInt32(&w.stopped, 1)
		close(w.done)
		logger.Debugf("stopped")
	}
}

func (w *Depth20FilteredWebsocket) restart() {
	select {
	case w.reconnectCh <- nil:
	default:
		logger.Debugf("w.reconnectCh <- nil failed, ch len %d", len(w.reconnectCh))
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
		reconnectCh: make(chan interface{}, 100),
		DataCh:      make(chan *Depth20, 100*len(symbols)),
		messageCh:   make(chan []byte, 100*len(symbols)),
		writeCh:     make(chan interface{}, 100*len(symbols)),
		symbolCh:    make(chan string, 100*len(symbols)),
		pingCh:      make(chan []byte, 100),
		stopped:     0,
	}
	go ws.mainLoop(ctx, symbols, proxy, decay, bias)
	go ws.dataHandleLoop(ctx, 0, decay, bias)
	go ws.dataHandleLoop(ctx, 2, decay, bias)
	go ws.dataHandleLoop(ctx, 3, decay, bias)
	go ws.dataHandleLoop(ctx, 4, decay, bias)
	ws.reconnectCh <- nil
	return &ws
}
