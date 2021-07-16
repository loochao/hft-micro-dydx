package huobi_usdtfuture

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

func (w *Depth20RoutedWebsocket) writeLoop(ctx context.Context, conn *websocket.Conn, symbols []string) {
	logger.Debugf("START writeLoop %s", symbols)
	defer logger.Debugf("EXIT writeLoop %s", symbols)
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

func (w *Depth20RoutedWebsocket) readLoop(ctx context.Context, conn *websocket.Conn, symbols []string, channels map[string]chan *common.DepthRawMessage) {
	logger.Debugf("START readLoop %s", symbols)
	defer logger.Debugf("EXIT readLoop %s", symbols)
	//totalLen := 0
	//totalCount := 0
	logSilentTime := time.Now()
	var symbolBytes []byte
	var symbol string
	var ch chan *common.DepthRawMessage
	var ok bool
	var timeInt int64
	for {
		err := conn.SetReadDeadline(time.Now().Add(time.Minute))
		if err != nil {
			logger.Warnf("conn.SetReadDeadline error %v", err)
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
			logger.Warnf("conn.NextReader error %v", err)
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
			logger.Warnf("gzip.NewReader error %v", err)
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
			logger.Warnf("w.readAll error %v", err)
			select {
			case <-ctx.Done():
				break
			default:
				w.restart()
			}
			return
		}
		//totalCount += 1
		//totalLen += len(msg)
		//if totalCount > 10000 {
		//	logger.Debugf(
		//		"AVERAGE MESSAGE LENGTH %d/%d = %d",
		//		totalLen, totalCount, totalLen/totalCount,
		//	)
		//	totalLen = 0
		//	totalCount = 0
		//}
		//{"ch":"market.FIL-USDT.depth.step6","ts":1618845641135,"tick":{"mrid":18528726394,"id":1618845641,"bids":[[154.423,36],[154.419,214],[154.414,380],[154.407,421],[154.398,64],[154.388,73],[154.386,8],[154.361,171],[154.36,300],[154.359,1],[154.354,175],[154.34,171],[154.339,48],[154.329,283],[154.327,243],[154.323,13],[154.315,50],[154.303,200],[154.302,48],[154.285,806]],"asks":[[154.436,154],[154.459,441],[154.46,58],[154.472,154],[154.473,134],[154.475,380],[154.497,163],[154.499,666],[154.511,88],[154.514,30],[154.515,283],[154.516,715],[154.517,70],[154.52,2],[154.53,222],[154.532,50],[154.557,1297],[154.565,3],[154.609,48],[154.61,4]],"ts":1618845641132,"version":1618845641,"ch":"market.FIL-USDT.depth.step6"}}
		if msg[2] == 'c' && len(msg) > 57 {
			if msg[40] == ':' {
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
			} else if msg[43] == ':' {
				timeInt, err = common.ParseInt(msg[44:57])
				if err != nil {
					logger.Debugf("common.ParseInt error %v %s", err, msg[44:57])
					continue
				}
				symbolBytes = msg[14:25]
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
					//logger.Debugf("SEND %s", symbol)
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("ch <- &common.DepthRawMessage failed %s ch len %d", symbol, len(ch))
						logSilentTime = time.Now().Add(time.Minute)
					}
				}
				select {
				case w.symbolCh <- symbol:
				default:
				}
			}
		} else if msg[2] == 'p' {
			msg[3] = 'o'
			select {
			case w.pingCh <- msg:
			default:
			}
		}
		err = gr.Close()
		if err != nil {
			logger.Warnf("gr.Close() error %v", err)
			w.restart()
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

func (w *Depth20RoutedWebsocket) mainLoop(ctx context.Context, proxy string, channels map[string]chan *common.DepthRawMessage) {
	logger.Debugf("START mainLoop")

	ctx, cancel := context.WithCancel(ctx)
	var internalCtx context.Context
	var internalCancel context.CancelFunc

	symbols := make([]string, 0)
	for symbol := range channels {
		symbols = append(symbols, symbol)
	}

	defer func() {
		logger.Debugf("EXIT mainLoop")
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
				internalCancel = nil
			}
			internalCtx, internalCancel = context.WithCancel(ctx)
			conn, err := w.reconnect(internalCtx, "wss://api.hbdm.vn/linear-swap-ws", proxy, 0)
			if err != nil {
				logger.Debugf("w.reconnect error %v, stop ws", err)
				w.Stop()
				if internalCancel != nil {
					internalCancel()
					internalCancel = nil
				}
				return
			}
			go w.readLoop(internalCtx, conn, symbols, channels)
			go w.writeLoop(internalCtx, conn, symbols)
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
			case w.writeCh <- msg:
				break
			default:
				logger.Debugf("w.writeCh <- ping msg failed, ch len %d", len(w.writeCh))
			}
			break
		case <-symbolCheckTimer.C:
		loop:
			for symbol, updateTime := range symbolUpdatedTimes {
				if time.Now().Sub(updateTime) > symbolTimeout {
					logger.Debugf("SWAP SUBSCRIBE %s", fmt.Sprintf("market.%s.depth.step6", symbol))
					select {
					case w.writeCh <- SubParam{
						ID:  fmt.Sprintf("%d", time.Now().UnixNano()),
						Sub: fmt.Sprintf("market.%s.depth.step6", symbol),
					}:
						symbolUpdatedTimes[symbol] = time.Now().Add(symbolCheckInterval * time.Duration(len(symbols)*2))
						break loop
					default:
						logger.Debugf("w.writeCh <- SubParam failed %s ch len %d", symbol, len(w.writeCh))
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

func (w *Depth20RoutedWebsocket) Stop() {
	if atomic.LoadInt32(&w.stopped) == 0 {
		atomic.StoreInt32(&w.stopped, 1)
		close(w.done)
	}
}

func (w *Depth20RoutedWebsocket) restart() {
	select {
	case <- w.done:
		return
	case w.reconnectCh <- nil:
		logger.Debugf("restart ws")
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
		reconnectCh: make(chan interface{}),
		writeCh:     make(chan interface{}, len(channels)),
		symbolCh:    make(chan string, 100*len(channels)),
		pingCh:      make(chan []byte, 100),
		stopped:     0,
	}
	go ws.mainLoop(ctx, proxy, channels)
	ws.reconnectCh <- nil
	return &ws
}
