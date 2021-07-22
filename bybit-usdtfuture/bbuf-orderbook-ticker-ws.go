package bybit_usdtfuture

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

type OrderBookTickerWS struct {
	writeCh       chan interface{}
	done          chan interface{}
	reconnectCh   chan interface{}
	symbolCh      chan string
	symbolResetCh chan string
	stopped       int32
}

func (w *OrderBookTickerWS) writeLoop(ctx context.Context, conn *websocket.Conn) {
	logger.Debugf("START writeLoop")
	defer logger.Debugf("EXIT writeLoop")
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case msg := <-w.writeCh:
			var msgBytes []byte
			var err error
			switch d := msg.(type) {
			case []byte:
				msgBytes = d
			case string:
				msgBytes = ([]byte)(d)
			default:
				msgBytes, err = json.Marshal(msg)
				if err != nil {
					logger.Debugf("json.Marshal(msg) err %v", err)
					continue
				}
			}
			if len(msgBytes) != 14 {
				logger.Debugf("%s", msgBytes)
			}
			err = conn.SetWriteDeadline(time.Now().Add(time.Minute))
			if err != nil {
				logger.Debugf("conn.SetWriteDeadline error %v", err)
				w.restart()
				return
			}
			err = conn.WriteMessage(websocket.TextMessage, msgBytes)
			if err != nil {
				logger.Debugf("conn.WriteMessage error %v", err)
				w.restart()
				return
			}
		}
	}
}

func (w *OrderBookTickerWS) readLoop(conn *websocket.Conn, channels map[string]chan []byte) {
	logger.Debugf("START readLoop")
	defer logger.Debugf("EXIT readLoop")
	logSilentTime := time.Now()
	var symbol string
	var msg []byte
	var err error
	var ch chan []byte
	var ok bool
	for {
		err = conn.SetReadDeadline(time.Now().Add(time.Minute))
		if err != nil {
			logger.Debugf("conn.SetReadDeadline error %v", err)
			w.restart()
			return
		}

		_, msg, err = conn.ReadMessage()
		if err != nil {
			logger.Debugf("conn.NextReader error %v", err)
			w.restart()
			return
		}
		msgLen := len(msg)
		if msgLen > 128 && msg[2] == 't' {
			if msg[32] == '"' {
				symbol = common.UnsafeBytesToString(msg[25:32])
			} else if msg[31] == '"' {
				symbol = common.UnsafeBytesToString(msg[25:31])
			} else if msg[33] == '"' {
				symbol = common.UnsafeBytesToString(msg[25:33])
			} else if msg[34] == '"' {
				symbol = common.UnsafeBytesToString(msg[25:34])
			} else if msg[35] == '"' {
				symbol = common.UnsafeBytesToString(msg[25:35])
			} else {
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("other msg %s", msg)
					logSilentTime = time.Now().Add(time.Minute)
				}
				continue
			}
		} else {
			if time.Now().Sub(logSilentTime) > 0 && msgLen > 128 {
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
					logger.Debugf(" ch <- msg %s ch len %d", symbol, len(ch))
					logSilentTime = time.Now().Add(time.Minute)
				}
			}
		}
	}
}

func (w *OrderBookTickerWS) readAll(r io.Reader) ([]byte, error) {
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

func (w *OrderBookTickerWS) reconnect(ctx context.Context, wsUrl string, proxy string, counter int64) (*websocket.Conn, error) {

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

func (w *OrderBookTickerWS) mainLoop(ctx context.Context, proxy string, channels map[string]chan []byte) {
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
			conn, err := w.reconnect(internalCtx, "wss://stream.bybit.com/realtime_public", proxy, 0)
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

func (w *OrderBookTickerWS) heartbeatLoop(ctx context.Context, conn *websocket.Conn, symbols []string) {
	logger.Debugf("START heartbeatLoop")
	defer func() {
		logger.Debugf("Exit heartbeatLoop")
		err := conn.Close()
		if err != nil {
			logger.Debugf("conn.Close() ERROR %v", err)
		}
	}()
	marketTimeout := time.Minute
	marketCheckInterval := time.Second
	marketCheckTimer := time.NewTimer(time.Second)
	defer marketCheckTimer.Stop()
	marketUpdatedTimes := make(map[string]time.Time)
	for _, symbol := range symbols {
		marketUpdatedTimes[symbol] = time.Unix(0, 0)
	}
	trafficTimeout := time.NewTimer(time.Minute * 5)
	pingTimer := time.NewTimer(time.Second * 15)
	defer trafficTimeout.Stop()
	defer pingTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case <-trafficTimeout.C:
			logger.Debugf("traffic timeout in 30s, restart ws")
			w.restart()
			return
		case <-pingTimer.C:
			pingTimer.Reset(time.Second * 15)
			select {
			case w.writeCh <- []byte("{\"op\": \"ping\"}"):
				break
			default:
				logger.Debugf("w.writeCh <- ping failed, ch len %d", len(w.writeCh))
			}
			break
		case symbol := <-w.symbolCh:
			trafficTimeout.Reset(time.Second * 30)
			marketUpdatedTimes[symbol] = time.Now()
			break
		case symbol := <-w.symbolResetCh:
			logger.Debugf("RESET %s", symbol)
			trafficTimeout.Reset(time.Second * 30)
			marketUpdatedTimes[symbol] = time.Now().Add(-marketTimeout)
			break
		case <-marketCheckTimer.C:
			args := make([]string, 0)
			for market, updateTime := range marketUpdatedTimes {
				if time.Now().Sub(updateTime) > marketTimeout {
					args = append(args, fmt.Sprintf("orderBookL2_25.%s", market))
				}
			}
			select {
			case w.writeCh <- SubscribeParam{
				Op:   "unsubscribe",
				Args: args,
			}:
			default:
				logger.Debugf("w.writeCh <- Subscription failed, ch len %d", len(w.writeCh))
			}
			select {
			case w.writeCh <- SubscribeParam{
				Op:   "subscribe",
				Args: args,
			}:
			default:
				logger.Debugf("w.writeCh <- Subscription failed, ch len %d", len(w.writeCh))
			}
			marketCheckTimer.Reset(marketCheckInterval)
			break
		}
	}
}

func (w *OrderBookTickerWS) Stop() {
	if atomic.LoadInt32(&w.stopped) == 0 {
		atomic.StoreInt32(&w.stopped, 1)
		close(w.done)
		logger.Debugf("stopped")
	}
}

func (w *OrderBookTickerWS) restart() {
	select {
	case w.reconnectCh <- nil:
	default:
		logger.Debugf("w.reconnectCh <- nil failed, ch len %d", len(w.reconnectCh))
	}
}

func (w *OrderBookTickerWS) Done() chan interface{} {
	return w.done
}

func (w *OrderBookTickerWS) dataHandleLoop(ctx context.Context, symbol string, inputCh chan []byte, outputCh chan common.Ticker) {
	logger.Debugf("START dataHandleLoop %s", symbol)
	defer logger.Debugf("EXIT dataHandleLoop %s", symbol)
	logSilentTime := time.Now()
	var err error
	var orderBook = OrderBook{
		Bids:   common.Bids{},
		Asks:   common.Asks{},
		Symbol: symbol,
	}
	var symbolLen = len(symbol)
	var hasPartial = false
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case msg := <-inputCh:
			//logger.Debugf("%s", msg[35+symbolLen:])
			if msg[35+symbolLen] == 's' {
				hasPartial = true
				err = UpdateOrderBook(msg, &orderBook)
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("UpdateOrderBook error %v", err)
					logSilentTime = time.Now().Add(time.Minute)
					continue
				}
			} else if hasPartial {
				err = UpdateOrderBook(msg, &orderBook)
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("UpdateOrderBook error %v", err)
					logSilentTime = time.Now().Add(time.Minute)
					continue
				}
			} else {
				continue
			}

			if orderBook.IsValidate() {
				outOrderbook := orderBook
				//logger.Debugf("%v", outOrderbook.IsValidate())
				select {
				case outputCh <- &outOrderbook:
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("outputCh <- &outOrderbook failed, ch len %d", len(outputCh))
						logSilentTime = time.Now().Add(time.Minute)
					}
				}
				select {
				case w.symbolCh <- symbol:
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("w.symbolCh <- symbol failed, ch len %d", len(w.symbolCh))
						logSilentTime = time.Now().Add(time.Minute)
					}
				}
			} else {
				hasPartial = false
				select {
				case w.symbolResetCh <- symbol:
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("w.symbolResetCh <- symbol failed ch len %d", len(w.symbolResetCh))
						logSilentTime = time.Now().Add(time.Minute)
						continue
					}
				}
			}
			break
		}
	}
}

func NewOrderBookTickerWS(
	ctx context.Context,
	proxy string,
	channels map[string]chan common.Ticker,
) *OrderBookTickerWS {
	ws := OrderBookTickerWS{
		done:          make(chan interface{}),
		reconnectCh:   make(chan interface{}, 16),
		writeCh:       make(chan interface{}, 16*len(channels)),
		symbolCh:      make(chan string, 64*len(channels)),
		symbolResetCh: make(chan string, 64*len(channels)),
		stopped:       0,
	}
	messagesCh := make(map[string]chan []byte)
	for market, ch := range channels {
		messagesCh[market] = make(chan []byte, 256)
		go ws.dataHandleLoop(ctx, market, messagesCh[market], ch)
	}
	go ws.mainLoop(ctx, proxy, messagesCh)
	ws.reconnectCh <- nil
	return &ws
}
