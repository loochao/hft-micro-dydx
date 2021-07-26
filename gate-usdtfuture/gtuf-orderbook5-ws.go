package gate_usdtfuture

import (
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

type OrderBook5WS struct {
	writeCh     chan interface{}
	done        chan interface{}
	reconnectCh chan interface{}
	symbolCh    chan string
	pingCh      chan []byte
	stopped     int32
}

func (w *OrderBook5WS) writeLoop(ctx context.Context, conn *websocket.Conn) {
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

func (w *OrderBook5WS) readLoop(conn *websocket.Conn, channels map[string]chan []byte) {
	logger.Debugf("START readLoop")
	defer logger.Debugf("EXIT readLoop")
	//logSilentTime := time.Now()
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
		msg, err := w.readAll(r)
		if err != nil {
			logger.Debugf("w.readAll error %v", err)
			w.restart()
			return
		}
		logger.Debugf("%s %s", w.findSymbol(msg), msg)

	}
}

func (w *OrderBook5WS) readAll(r io.Reader) ([]byte, error) {
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

func (w *OrderBook5WS) reconnect(ctx context.Context, wsUrl string, proxy string, counter int64) (*websocket.Conn, error) {

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

func (w *OrderBook5WS) mainLoop(ctx context.Context, proxy string, channels map[string]chan []byte) {
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
			conn, err := w.reconnect(internalCtx, "wss://fx-ws.gateio.ws/v4/ws/usdt", proxy, 0)
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

func (w *OrderBook5WS) heartbeatLoop(ctx context.Context, conn *websocket.Conn, symbols []string) {
	logger.Debugf("START heartbeatLoop")
	defer func() {
		logger.Debugf("Exit heartbeatLoop")
		err := conn.Close()
		if err != nil {
			logger.Debugf("conn.Close() ERROR %v", err)
		}
	}()

	symbolTimeout := time.Minute
	symbolCheckInterval := time.Second*15
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
			for symbol, updateTime := range symbolUpdatedTimes {
				if time.Now().Sub(updateTime) > symbolTimeout {
					select {
					case w.writeCh <- WSRequest{
						Time:    time.Now().Unix(),
						Channel: "futures.order_book_update",
						Payload: []string{symbol, "100ms", "5"},
						Event:   "subscribe",
					}:
						symbolUpdatedTimes[symbol] = time.Now().Add(symbolTimeout)
					default:
						logger.Debugf("w.writeCh <- WSRequest failed, ch len %d", len(w.writeCh))
					}
				}
			}
			symbolCheckTimer.Reset(symbolCheckInterval)
			break
		}
	}

}
func (w *OrderBook5WS) findSymbol(msg []byte) string {
	msgLen := len(msg)
	if msgLen < 123 {
		return ""
	}
	start := 0
	end := 122
	for end < msgLen-1 {
		if start == 0 {
			if msg[end] == 's' && msg[end-1] == '"' && msg[end+1] == '"' {
				end += 4
				start = end
				end += 7
			}
		} else if msg[end] == '"' {
			return common.UnsafeBytesToString(msg[start:end])
		}
		end++
	}
	return ""
}

func (w *OrderBook5WS) Stop() {
	if atomic.LoadInt32(&w.stopped) == 0 {
		atomic.StoreInt32(&w.stopped, 1)
		close(w.done)
		logger.Debugf("stopped")
	}
}

func (w *OrderBook5WS) restart() {
	select {
	case w.reconnectCh <- nil:
	default:
		logger.Debugf("w.reconnectCh <- nil failed, ch len %d", len(w.reconnectCh))
	}
}

func (w *OrderBook5WS) Done() chan interface{} {
	return w.done
}

func (w *OrderBook5WS) dataHandleLoop(ctx context.Context, symbol string, inputCh chan []byte, outputCh chan common.Depth) {
	logger.Debugf("START dataHandleLoop %s", symbol)
	defer logger.Debugf("EXIT dataHandleLoop %s", symbol)
	////logSilentTime := time.Now()
	////var ch chan common.Depth
	////var err error
	////var ok bool
	////var wsTrade WSTrade
	////for {
	////	select {
	////	case <-ctx.Done():
	////		return
	////	case <-w.done:
	////		return
	////	case msg := <-w.messageCh:
	////		if len(msg) > 52 && msg[52] == 'u' {
	////			err = json.Unmarshal(msg, &wsTrade)
	////			if err != nil {
	////				if time.Now().Sub(logSilentTime) > 0 {
	////					logger.Debugf("json.Unmarshal(msg, &wsTrade) error %v", err)
	////					logSilentTime = time.Now().Add(time.Minute)
	////					continue
	////				}
	////			}
	////			if ch, ok = channels[wsTrade.Result.CurrencyPair]; ok {
	////				select {
	////				case ch <- &wsTrade.Result:
	////				default:
	////					if time.Now().Sub(logSilentTime) > 0 {
	////						logger.Debugf("ch <- &wsTrade.Result failed, ch len %d", len(ch))
	////						logSilentTime = time.Now().Add(time.Minute)
	////						continue
	////					}
	////				}
	////				select {
	////				case w.symbolCh <- wsTrade.Result.CurrencyPair:
	////				default:
	////					if time.Now().Sub(logSilentTime) > 0 {
	////						logger.Debugf("w.symbolCh <- wsTrade.Result.CurrencyPair failed, ch len %d", len(w.symbolCh))
	////						logSilentTime = time.Now().Add(time.Minute)
	////						continue
	////					}
	////				}
	////			}
	////		} else if msg[52] == 's'{
	////			//if time.Now().Sub(logSilentTime) > 0 {
	////			//	logger.Debugf("subscribe msg %s", msg)
	////			//	logSilentTime = time.Now().Add(time.Minute)
	////			//	continue
	////			//}
	////		} else {
	////			if time.Now().Sub(logSilentTime) > 0 {
	////				logger.Debugf("other msg %s", msg)
	////				logSilentTime = time.Now().Add(time.Minute)
	////				continue
	////			}
	////		}
	////	}
	//}
}

func NewOrderBook5WS(
	ctx context.Context,
	proxy string,
	channels map[string]chan common.Depth,
) *OrderBook5WS {
	ws := OrderBook5WS{
		done:        make(chan interface{}),
		reconnectCh: make(chan interface{}, 100),
		writeCh:     make(chan interface{}, 100*len(channels)),
		symbolCh:    make(chan string, 100*len(channels)),
		pingCh:      make(chan []byte, 100),
		stopped:     0,
	}
	messageChs := make(map[string]chan []byte)
	for symbol, ch := range channels {
		messageChs[symbol] = make(chan []byte, 4)
		go ws.dataHandleLoop(ctx, symbol, messageChs[symbol], ch)
	}
	go ws.mainLoop(ctx, proxy, messageChs)
	//for i := 0; i < 4; i++ {
	//	cs := make(map[string]chan common.Depth)
	//	for symbol, ch := range channels {
	//		cs[symbol] = ch
	//	}
	//	go ws.dataHandleLoop(ctx, i, cs)
	//}
	ws.reconnectCh <- nil
	return &ws
}
