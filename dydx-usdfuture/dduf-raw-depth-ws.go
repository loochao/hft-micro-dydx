package dydx_usdfuture

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/gorilla/websocket"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"
)

type RawDepthWS struct {
	writeCh     chan interface{}
	done        chan interface{}
	reconnectCh chan interface{}
	marketCh    chan string
	prefix      []byte
	stopped     int32
}

func (w *RawDepthWS) writeLoop(ctx context.Context, conn *websocket.Conn) {
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
func (w *RawDepthWS) findMarket(msg []byte) (string, error) {
	collectStart := 0
	collectEnd := 94
	msgLen := len(msg)
	for collectEnd < msgLen {
		if collectStart != 0 {
			if msg[collectEnd] == '"' {
				return common.UnsafeBytesToString(msg[collectStart:collectEnd]), nil
			}
		} else if msg[collectEnd] == '"' &&
			msg[collectEnd-1] == 'd' &&
			msg[collectEnd-2] == 'i' &&
			msg[collectEnd-3] == '"' {
			collectEnd += 3
			collectStart = collectEnd
		}
		collectEnd++
	}
	return "", fmt.Errorf("market not found")
}

func (w *RawDepthWS) readLoop(conn *websocket.Conn, channels map[string]chan *common.RawMessage) {
	logger.Debugf("START readLoop")
	defer logger.Debugf("EXIT readLoop")
	logSilentTime := time.Now()

	var market string
	var msg []byte
	var err error
	var ch chan *common.RawMessage
	var ok bool
	var message *common.RawMessage
	const bufferSize = 8192
	index := -1
	pool := [bufferSize]*common.RawMessage{}
	for i := 0; i < bufferSize; i++ {
		pool[i] = &common.RawMessage{
			Prefix: w.prefix,
		}
	}

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
		//logger.Debugf("%s", msg)
		msgLen := len(msg)
		if msgLen > 128 {
			//if msg[9] == 's' {
			//	logger.Debugf("%s", msg)
			//}
			market, err = w.findMarket(msg)
			if err != nil {
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("find market failed %s", msg)
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
		if ch, ok = channels[market]; ok {
			index++
			if index == 4096 {
				index = 0
			}
			message = pool[index]
			message.Time = time.Now().UnixNano()
			message.Data = msg
			select {
			case ch <- message:
			default:
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf(" ch <- message %s ch len %d", market, len(ch))
					logSilentTime = time.Now().Add(time.Minute)
				}
			}
			select {
			case w.marketCh <- market:
			default:
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("w.marketCh <- market failed, ch len %d", len(w.marketCh))
					logSilentTime = time.Now().Add(time.Minute)
				}
			}
		}
	}
}

func (w *RawDepthWS) readAll(r io.Reader) ([]byte, error) {
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

func (w *RawDepthWS) reconnect(ctx context.Context, wsUrl string, proxy string, counter int64) (*websocket.Conn, error) {

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
			EnableCompression: true,
		}
	} else {
		dialer = &websocket.Dialer{
			HandshakeTimeout: 10 * time.Second,
			EnableCompression: true,
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

func (w *RawDepthWS) mainLoop(ctx context.Context, proxy string, channels map[string]chan *common.RawMessage) {
	logger.Debugf("START mainLoop")
	defer logger.Debugf("EXIT mainLoop")
	ctx, cancel := context.WithCancel(ctx)
	var internalCtx context.Context
	var internalCancel context.CancelFunc
	markets := make([]string, 0)
	for market := range channels {
		markets = append(markets, market)
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
			conn, err := w.reconnect(internalCtx, "wss://api.dydx.exchange/v3/ws", proxy, 0)
			if err != nil {
				logger.Debugf("w.reconnect error %v", err)
				internalCancel()
				w.Stop()
				return
			}
			go w.readLoop(conn, channels)
			go w.writeLoop(internalCtx, conn)
			go w.heartbeatLoop(internalCtx, conn, markets)
		}
	}
}

func (w *RawDepthWS) heartbeatLoop(ctx context.Context, conn *websocket.Conn, markets []string) {
	logger.Debugf("START heartbeatLoop")
	defer func() {
		logger.Debugf("Exit heartbeatLoop")
		err := conn.Close()
		if err != nil {
			logger.Debugf("conn.Close() ERROR %v", err)
		}
	}()
	marketTimeout := time.Minute
	marketCheckInterval := time.Second * 5
	marketResetInterval := time.Minute * 5
	marketCheckTimer := time.NewTimer(time.Second)
	defer marketCheckTimer.Stop()

	resetCheckTimer := time.NewTimer(time.Second)
	defer resetCheckTimer.Stop()

	marketResetTimes := make(map[string]time.Time)
	marketUpdateTimes := make(map[string]time.Time)
	for _, market := range markets {
		marketResetTimes[market] = time.Now().Add(time.Duration(rand.Intn(int(marketResetInterval/time.Second)))*time.Second + marketCheckInterval)
		marketUpdateTimes[market] = time.Unix(0, 0)
	}
	trafficTimeout := time.NewTimer(time.Minute * 5)
	defer trafficTimeout.Stop()


	conn.SetPingHandler(func(msg string) error {
		trafficTimeout.Reset(time.Second * 30)
		err := conn.WriteControl(websocket.PongMessage, []byte(msg), time.Now().Add(time.Minute))
		if err != nil {
			select {
			case <-ctx.Done():
				break
			default:
				go w.restart()
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
		case <-trafficTimeout.C:
			logger.Debugf("traffic timeout in 30s, restart ws")
			w.restart()
			return
		case market := <-w.marketCh:
			trafficTimeout.Reset(time.Second * 30)
			marketUpdateTimes[market] = time.Now().Add(marketTimeout)
			break
		case <-marketCheckTimer.C:
			for market := range marketUpdateTimes {
				if time.Now().Sub(marketUpdateTimes[market]) > 0 ||
					time.Now().Sub(marketResetTimes[market]) > 0 {
					select {
					case w.writeCh <- WSOrderBookSubscribe{
						Id:      market,
						Type:    "unsubscribe",
						Channel: "v3_orderbook",
					}:
					default:
						logger.Debugf("w.writeCh <- Subscription failed, ch len %d", len(w.writeCh))
					}
					select {
					case w.writeCh <- WSOrderBookSubscribe{
						Id:      market,
						Type:    "subscribe",
						Channel: "v3_orderbook",
					}:
					default:
						logger.Debugf("w.writeCh <- Subscription failed, ch len %d", len(w.writeCh))
					}
					marketUpdateTimes[market] = time.Now().Add(marketTimeout)
					marketResetTimes[market] = time.Now().Add(time.Duration(rand.Intn(int(marketCheckInterval/time.Second)))*time.Second + marketResetInterval)
				}
			}
			marketCheckTimer.Reset(marketCheckInterval)
			break
		}
	}
}

func (w *RawDepthWS) Stop() {
	if atomic.LoadInt32(&w.stopped) == 0 {
		atomic.StoreInt32(&w.stopped, 1)
		close(w.done)
		logger.Debugf("stopped")
	}
}

func (w *RawDepthWS) restart() {
	select {
	case w.reconnectCh <- nil:
	default:
		logger.Debugf("w.reconnectCh <- nil failed, ch len %d", len(w.reconnectCh))
	}
}

func (w *RawDepthWS) Done() chan interface{} {
	return w.done
}

func NewRawDepthWS(
	ctx context.Context,
	proxy string,
	prefixes []byte,
	channels map[string]chan *common.RawMessage,
) *RawDepthWS {
	ws := RawDepthWS{
		done:        make(chan interface{}),
		reconnectCh: make(chan interface{}, 16),
		writeCh:     make(chan interface{}, 16*len(channels)),
		marketCh:    make(chan string, 64*len(channels)),
		prefix:      prefixes,
		stopped:     0,
	}
	go ws.mainLoop(ctx, proxy, channels)
	ws.reconnectCh <- nil
	return &ws
}
