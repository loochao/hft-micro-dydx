package ftx_usdfuture

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

type TickerWS struct {
	writeCh       chan interface{}
	done          chan interface{}
	reconnectCh   chan interface{}
	marketCh      chan string
	marketResetCh chan string
	stopped       int32
}

func (w *TickerWS) writeLoop(ctx context.Context, conn *websocket.Conn) {
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

func (w *TickerWS) readLoop(conn *websocket.Conn, channels map[string]chan []byte) {
	logger.Debugf("START readLoop")
	defer logger.Debugf("EXIT readLoop")
	logSilentTime := time.Now()
	var symbol string
	var ch chan []byte
	var ok bool
	var readPool = [bookTickerReadPoolSize][]byte{}
	var readIndex = -1
	var msg []byte
	var n int
	var r io.Reader
	var err error
	for i := range readPool {
		readPool[i] = make([]byte, bookTickerReadMsgSize)
	}
	readCounter := 0
	partialReadCounter := 0
	allocateCounter := 0
mainLoop:
	for {
		err = conn.SetReadDeadline(time.Now().Add(time.Minute))
		if err != nil {
			logger.Warnf("conn.SetReadDeadline error %v", err)
			w.restart()
			return
		}
		_, r, err = conn.NextReader()
		if err != nil {
			logger.Warnf("conn.NextReader error %v", err)
			w.restart()
			return
		}
		readIndex += 1
		if readIndex == bookTickerReadPoolSize {
			readIndex = 0
		}
		msg = readPool[readIndex]
		n, err = r.Read(msg)
		if err == nil {
			readCounter++
			msg = msg[:n]
			if n < 2 || msg[n-1] != '}' || msg[n-2] != '}' {
				partialReadCounter++
				for {
					if len(msg) == cap(msg) {
						// Add more capacity (let append pick how much).
						msg = append(msg, 0)[:len(msg)]
						logger.Debugf("BAD BUFFER SIZE CAN'T READ %d INTO %d, MSG: %s", len(msg), bookTickerReadMsgSize, msg)
						allocateCounter++
					}
					n, err = r.Read(msg[len(msg):cap(msg)])
					msg = msg[:len(msg)+n]
					if err != nil {
						if err == io.EOF {
							break
						} else {
							logger.Debugf("r.Read error %v", err)
							continue mainLoop
						}
					}
				}
			}
		} else {
			logger.Debugf("r.Read error %v", err)
			continue mainLoop
		}
		if readCounter%1000000 == 0 {
			logger.Debugf("FTXUF BOOK TICKER READ SIZE %d TOTAL %d PARTIAL %d ALLOCATE %d", bookTickerReadMsgSize, readCounter, partialReadCounter, allocateCounter)
		}
		if len(msg) < 128 {
			continue mainLoop
		}
		//{"channel": "ticker", "market": "DOGE-PERP", "type": "update", "data": {"bid": 0.278362, "ask": 0.2784135, "bidSize": 107.0, "askSize": 5600.0, "last": 0.2783695, "time": 1624183024.08771}} 189
		if msg[31] == ' ' && msg[32] == '"' {
			if msg[41] == '"' {
				symbol = common.UnsafeBytesToString(msg[33:41])
			} else if msg[40] == '"' {
				symbol = common.UnsafeBytesToString(msg[33:40])
			} else if msg[42] == '"' {
				symbol = common.UnsafeBytesToString(msg[33:42])
			} else if msg[43] == '"' {
				symbol = common.UnsafeBytesToString(msg[33:43])
			} else if msg[44] == '"' {
				symbol = common.UnsafeBytesToString(msg[33:44])
			} else if msg[45] == ',' {
				symbol = common.UnsafeBytesToString(msg[33:45])
			} else {
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("other msg %s", msg)
					logSilentTime = time.Now().Add(time.Minute)
				}
				continue
			}
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
					logger.Debugf(" ch <- msg %s ch len %d", symbol, len(ch))
					logSilentTime = time.Now().Add(time.Minute)
				}
			}
		}
	}
}

//func (w *TickerWS) readAll(r io.Reader) ([]byte, error) {
//	b := make([]byte, 0, 256)
//	for {
//		if len(b) == cap(b) {
//			// Add more capacity (let append pick how much).
//			b = append(b, 0)[:len(b)]
//		}
//		n, err := r.Read(b[len(b):cap(b)])
//		b = b[:len(b)+n]
//		if err != nil {
//			if err == io.EOF {
//				err = nil
//			}
//			return b, err
//		}
//	}
//}

func (w *TickerWS) reconnect(ctx context.Context, wsUrl string, proxy string, counter int64) (*websocket.Conn, error) {

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
			//EnableCompression: true,
		}
	} else {
		dialer = &websocket.Dialer{
			HandshakeTimeout: 10 * time.Second,
			//EnableCompression: true,
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

func (w *TickerWS) mainLoop(ctx context.Context, proxy string, channels map[string]chan []byte) {
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
			conn, err := w.reconnect(internalCtx, "wss://ftx.com/ws/", proxy, 0)
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

func (w *TickerWS) heartbeatLoop(ctx context.Context, conn *websocket.Conn, symbols []string) {
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
		case symbol := <-w.marketCh:
			trafficTimeout.Reset(time.Second * 30)
			marketUpdatedTimes[symbol] = time.Now()
			break
		case symbol := <-w.marketResetCh:
			logger.Debugf("RESET %s", symbol)
			trafficTimeout.Reset(time.Second * 30)
			marketUpdatedTimes[symbol] = time.Now().Add(-marketTimeout)
			break
		case <-marketCheckTimer.C:
			for market, updateTime := range marketUpdatedTimes {
				if time.Now().Sub(updateTime) > marketTimeout {
					select {
					case w.writeCh <- SubscribeParam{
						Operation: "unsubscribe",
						Channel:   "ticker",
						Market:    market,
					}:
						marketUpdatedTimes[market] = time.Now().Add(marketTimeout)
					default:
						logger.Debugf("w.writeCh <- Subscription failed, ch len %d", len(w.writeCh))
					}
					select {
					case w.writeCh <- SubscribeParam{
						Operation: "subscribe",
						Channel:   "ticker",
						Market:    market,
					}:
						marketUpdatedTimes[market] = time.Now().Add(marketTimeout)
					default:
						logger.Debugf("w.writeCh <- Subscription failed, ch len %d", len(w.writeCh))
					}
				}
			}
			marketCheckTimer.Reset(marketCheckInterval)
			break
		}
	}
}

func (w *TickerWS) Stop() {
	if atomic.LoadInt32(&w.stopped) == 0 {
		atomic.StoreInt32(&w.stopped, 1)
		close(w.done)
		logger.Debugf("stopped")
	}
}

func (w *TickerWS) restart() {
	select {
	case w.reconnectCh <- nil:
	default:
		logger.Debugf("w.reconnectCh <- nil failed, ch len %d", len(w.reconnectCh))
	}
}

func (w *TickerWS) Done() chan interface{} {
	return w.done
}

func (w *TickerWS) dataHandleLoop(ctx context.Context, market string, inputCh chan []byte, outputCh chan common.Ticker) {
	logger.Debugf("START dataHandleLoop %s", market)
	defer logger.Debugf("EXIT dataHandleLoop %s", market)
	logSilentTime := time.Now()
	index := -1
	pool := [common.BufferSizeFor100msData]*Ticker{}
	for i := 0; i < common.BufferSizeFor100msData; i++ {
		pool[i] = &Ticker{}
	}
	var ticker *Ticker
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
			ticker = pool[index]
			err = ParseTicker(msg, ticker)
			if err != nil && time.Now().Sub(logSilentTime) > 0 {
				logger.Debugf("ParseTicker(msg, ticker) error %s %v", msg, err)
				logSilentTime = time.Now().Add(time.Minute)
			} else {
				select {
				case outputCh <- ticker:
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("outputCh <- ticker failed ch len %d", len(outputCh))
						logSilentTime = time.Now().Add(time.Minute)
						continue
					}
				}
			}
			select {
			case w.marketCh <- market:
			default:
				//if time.Now().Sub(logSilentTime) > 0 {
				//	logger.Debugf("w.marketCh <- market failed, ch len %d", len(w.marketCh))
				//	logSilentTime = time.Now().Add(time.Minute)
				//}
			}
			break
		}
	}
}

func NewTickerWS(
	ctx context.Context,
	proxy string,
	channels map[string]chan common.Ticker,
) *TickerWS {
	ws := TickerWS{
		done:          make(chan interface{}),
		reconnectCh:   make(chan interface{}, common.ChannelSizeLowLoad),
		writeCh:       make(chan interface{}, common.ChannelSizeLowLoad*len(channels)),
		marketCh:      make(chan string, len(channels)*common.ChannelSizeLowLoad),
		marketResetCh: make(chan string, len(channels)),
		stopped:       0,
	}
	messagesCh := make(map[string]chan []byte)
	for market, ch := range channels {
		messagesCh[market] = make(chan []byte, common.ChannelSizeLowLoadLowLatency)
		go ws.dataHandleLoop(ctx, market, messagesCh[market], ch)
	}
	go ws.mainLoop(ctx, proxy, messagesCh)
	ws.reconnectCh <- nil
	return &ws
}
