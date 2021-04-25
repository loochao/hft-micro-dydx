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
	"unsafe"
)

type Depth20RoutedWebsocket struct {
	writeCh     chan interface{}
	done        chan interface{}
	reconnectCh chan interface{}
	symbolCh    chan string
	pingCh      chan []byte
	stopped     int32
}

func (w *Depth20RoutedWebsocket) writeLoop(ctx context.Context, conn *websocket.Conn) {
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

func (w *Depth20RoutedWebsocket) readLoop(conn *websocket.Conn, channels map[string]chan *common.DepthRawMessage) {
	logger.Debugf("START readLoop")
	defer logger.Debugf("EXIT readLoop")
	totalCount := 0
	totalLen := 0
	logSilentTime := time.Now()
	var symbolBytes []byte
	var symbol string
	var ch chan *common.DepthRawMessage
	var ok bool
	var timeInt int64
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

		//{"ch":"market.btcusdt.depth.step1","ts":1618475611868,"tick":{"bids":[[62753.2,2.140127],[62753.1,1.005768],[62751.4,0.01],[62750.2,1.588851],[62750.1,0.132173],[62747.1,1.04731],[62747.0,1.357035],[62746.0,0.001031],[62744.8,0.44207],[62744.6,0.064435],[62743.0,0.051222],[62741.8,8.0E-4],[62739.5,0.450211],[62739.0,0.026874],[62737.6,0.2],[62737.0,0.001401],[62736.9,0.1],[62736.7,0.047803],[62735.4,1.6E-4],[62733.6,0.135775]],"asks":[[62753.3,0.038953],[62754.3,0.03],[62758.0,0.09781],[62758.8,0.045154],[62759.5,0.01],[62760.8,0.134133],[62761.8,0.03],[62761.9,0.132173],[62763.4,8.95E-4],[62763.8,0.123199],[62764.2,0.06],[62765.0,0.010786],[62765.8,0.002],[62766.0,0.162855],[62767.2,0.001596],[62767.6,1.145],[62768.9,2.733775],[62769.1,0.159325],[62770.7,0.017235],[62771.5,0.04]],"version":125019588599,"ts":1618475611865}}
		if msg[2] == 'c' && len(msg) > 128 {
			if msg[39] == ':' {
				timeInt, err = common.ParseInt(msg[40:53])
				if err != nil {
					logger.Debugf("common.ParseInt error %v %s", err, msg[40:53])
					continue
				}
				symbolBytes = msg[14:21]
				symbol = *(*string)(unsafe.Pointer(&symbolBytes))
			} else if msg[40] == ':' {
				timeInt, err = common.ParseInt(msg[41:54])
				if err != nil {
					logger.Debugf("common.ParseInt error %v %s", err, msg[41:54])
					continue
				}
				symbolBytes = msg[14:22]
				symbol = *(*string)(unsafe.Pointer(&symbolBytes))
			} else if msg[41] == ':' {
				timeInt, err = common.ParseInt(msg[42:55])
				if err != nil {
					logger.Debugf("common.ParseInt error %v %s", err, msg[42:55])
					continue
				}
				symbolBytes = msg[14:23]
				symbol = *(*string)(unsafe.Pointer(&symbolBytes))
			} else if msg[42] == ':' {
				timeInt, err = common.ParseInt(msg[43:56])
				if err != nil {
					logger.Debugf("common.ParseInt error %v %s", err, msg[43:56])
					continue
				}
				symbolBytes = msg[14:24]
				symbol = *(*string)(unsafe.Pointer(&symbolBytes))
			} else {
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("bad msg, can't find timestamp: %s", msg)
					logSilentTime = time.Now().Add(time.Minute)
				}
			}
			if ch, ok = channels[symbol]; ok {
				select {
				case ch <- &common.DepthRawMessage{
					Depth:  msg,
					Symbol: symbol,
					Time:   time.Unix(0, timeInt*1000000),
				}:
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("ch <- &common.DepthRawMessage failed %s ch len %d", symbol, len(ch))
						logSilentTime = time.Now().Add(time.Minute)
					}
				}
				select {
				case w.symbolCh <- symbol:
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("w.symbolCh <- symbol failed %s ch len %d", symbol, len(w.symbolCh))
						logSilentTime = time.Now().Add(time.Minute)
					}
				}
			}
		} else if msg[2] == 'p' {
			msg[3] = 'o'
			select {
			case w.pingCh <- msg:
			default:
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("w.pingCh <- msg failed %s ch len %d", symbol, len(w.pingCh))
					logSilentTime = time.Now().Add(time.Minute)
				}
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

func (w *Depth20RoutedWebsocket) mainLoop(ctx context.Context, proxy string, channels map[string]chan *common.DepthRawMessage) {
	logger.Debugf("START mainLoop")
	defer logger.Debugf("EXIT mainLoop")
	ctx, cancel := context.WithCancel(ctx)
	var internalCtx context.Context
	var internalCancel context.CancelFunc
	symbols := make([]string, 0)
	for symbol := range channels {
		symbols = append(symbols, symbol)
	}

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
			go w.readLoop(conn, channels)
			go w.writeLoop(internalCtx, conn)
			go w.heartbeatLoop(internalCtx, conn, symbols)
		}
	}
}

func (w *Depth20RoutedWebsocket) heartbeatLoop(ctx context.Context, conn *websocket.Conn, symbols []string) {
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

func (w *Depth20RoutedWebsocket) Stop() {
	if atomic.LoadInt32(&w.stopped) == 0 {
		atomic.StoreInt32(&w.stopped, 1)
		close(w.done)
		logger.Debugf("stopped")
	}
}

func (w *Depth20RoutedWebsocket) restart() {
	select {
	case w.reconnectCh <- nil:
	default:
		logger.Debugf("w.reconnectCh <- nil failed, ch len %d", len(w.reconnectCh))
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
		writeCh:     make(chan interface{}, 100*len(channels)),
		symbolCh:    make(chan string, 100*len(channels)),
		pingCh:      make(chan []byte, 100),
		stopped:     0,
	}
	go ws.mainLoop(ctx, proxy, channels)
	ws.reconnectCh <- nil
	return &ws
}
