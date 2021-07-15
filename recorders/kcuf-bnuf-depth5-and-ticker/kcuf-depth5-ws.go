package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	kucoin_usdtfuture "github.com/geometrybase/hft-micro/kucoin-usdtfuture"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/gorilla/websocket"
	"io"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"
)

type KcufDepth5WS struct {
	RestartCh   chan interface{}
	writeCh     chan interface{}
	done        chan interface{}
	reconnectCh chan interface{}
	symbolCh    chan string
	stopped     int32
}

func (w *KcufDepth5WS) writeLoop(ctx context.Context, conn *websocket.Conn) {
	logger.Debugf("START writeLoop")
	defer func() {
		logger.Debugf("EXIT writeLoop")
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
					logger.Debugf("Marshal err %v", err)
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
				logger.Debugf("conn.WriteMessage error %v, %s", err, string(bytes))
				w.restart()
				return
			}
		}
	}
}

func (w *KcufDepth5WS) readLoop(
	conn *websocket.Conn,
	channels map[string]chan  *Message,
) {
	logger.Debugf("START readLoop")
	defer func() {
		logger.Debugf("EXIT readLoop")
	}()
	var symbol string
	var ch chan *Message
	var ok bool
	var message *Message
	index := -1
	pool := [4096]*Message{}
	var logSilentTime time.Time
	for i := 0; i < 4096; i++ {
		pool[i] = &Message{
			Source: []byte{'K', 'D'},
		}
	}
	var msgLen int
	for {
		err := conn.SetReadDeadline(time.Now().Add(time.Minute))
		if err != nil {
			logger.Debugf("conn.SetReadDeadline error %v", err)
			go w.restart()
			return
		}
		_, r, err := conn.NextReader()
		if err != nil {
			logger.Debugf("conn.NextReader error %v", err)
			go w.restart()
			return
		}
		msg, err := w.readAll(r)
		if err != nil {
			logger.Debugf("w.readAll error %v", err)
			go w.restart()
			return
		}
		//{"data":{"sequence":1616576945844,"asks":[[17.834,10],[18.019,10154],[18.082,11060]],"bids":[[17.797,701],[17.793,1061],[17.784,199],[17.781,881],[17.779,407]],"ts":1618717277315,
		msgLen = len(msg)
		if msgLen > 128 {
			if msg[msgLen-28] == ':' {
				symbol = common.UnsafeBytesToString(msg[msgLen-27 : msgLen-19])
			} else if msg[msgLen-29] == ':' {
				symbol = common.UnsafeBytesToString(msg[msgLen-28 : msgLen-19])
			} else if msg[msgLen-30] == ':' {
				symbol = common.UnsafeBytesToString(msg[msgLen-29 : msgLen-19])
			} else if msg[msgLen-31] == ':' {
				symbol = common.UnsafeBytesToString(msg[msgLen-30 : msgLen-19])
			} else {
				logger.Debugf("OTHER MSG %s", msg)
				continue
			}
			if ch, ok = channels[symbol]; ok {
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
						logger.Debugf("ch <- message failed %s len(ch) = %d", symbol, len(ch))
						logSilentTime = time.Now().Add(time.Minute)
					}
				}
			}
		} else {
			if len(msg) > 3 && msg[2] == 'i' && msg[len(msg)-3] == 'k' {
				logger.Debugf("%s", msg)
			}
		}
	}
}

func (w *KcufDepth5WS) readAll(r io.Reader) ([]byte, error) {
	b := make([]byte, 0, 512)
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

func (w *KcufDepth5WS) reconnect(ctx context.Context, wsUrl string, proxy string, counter int64) (*websocket.Conn, error) {

	if counter != 0 {
		logger.Debugf("reconnect %d %s", counter, wsUrl)
	}

	var dialer *websocket.Dialer

	if proxy != "" {
		proxyUrl, err := url.Parse(proxy)
		if err != nil {
			return nil, fmt.Errorf("url.Parse error %v", err)
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

func (w *KcufDepth5WS) mainLoop(
	ctx context.Context,
	proxy string,
	channels map[string]chan *Message,
) {

	logger.Debugf("START mainLoop")


	api, err := kucoin_usdtfuture.NewAPI("", "", "", proxy)
	if err != nil {
		logger.Debugf("NewAPI error %v", err)
		return
	}


	symbols := make([]string, 0)
	for symbol := range channels {
		symbols = append(symbols, symbol)
	}
	var internalCtx context.Context
	var internalCancel context.CancelFunc

	defer func() {
		if internalCancel != nil {
			internalCancel()
		}
		w.Stop()
		logger.Debugf("EXIT mainLoop")
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
			connectToken, err := api.GetPublicConnectToken(internalCtx)
			if err != nil {
				if internalCancel != nil {
					internalCancel()
				}
				w.Stop()
				logger.Debugf("api.GetPublicConnectToken error %v, stop ws", err)
				return
			}
			if len(connectToken.InstanceServers) == 0 {
				if internalCancel != nil {
					internalCancel()
				}
				w.Stop()
				logger.Debugf("no InstanceServers %v, stop ws", connectToken)
				return
			}
			urlStr := connectToken.InstanceServers[0].Endpoint + "?token=" + connectToken.Token

			conn, err := w.reconnect(internalCtx, urlStr, proxy, 0)
			if err != nil {
				if internalCancel != nil {
					internalCancel()
				}
				logger.Debugf("w.reconnect error %v, stop ws", err)
				return
			}
			go w.readLoop(conn, channels)
			go w.writeLoop(internalCtx, conn)
			go w.heartbeatLoop(internalCtx, conn, symbols, time.Millisecond*time.Duration(connectToken.InstanceServers[0].PingInterval))
			reconnectTimer.Reset(time.Hour * 9999)
		}
	}
}

func (w *KcufDepth5WS) heartbeatLoop(ctx context.Context, conn *websocket.Conn, symbols []string, pingInterval time.Duration) {

	logger.Debugf("START heartbeatLoop")

	defer func() {
		err := conn.Close()
		if err != nil {
			logger.Debugf("conn.Close() ERROR %v", err)
		}
	}()

	symbolTimeout := time.Minute
	symbolCheckInterval := time.Second
	pingTimer := time.NewTimer(time.Second)
	symbolCheckTimer := time.NewTimer(time.Second)
	defer symbolCheckTimer.Stop()
	symbolUpdatedTimes := make(map[string]time.Time)
	resubCounters := make(map[string]int)
	for _, symbol := range symbols {
		resubCounters[symbol] = 0
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
			resubCounters[symbol] = 0
		case <-pingTimer.C:
			pingTimer.Reset(pingInterval)
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Millisecond):
				logger.Debug("send ping to writeCh timeout in 1ms")
			case w.writeCh <- kucoin_usdtfuture.Ping{
				ID:   fmt.Sprintf("%d", time.Now().UnixNano()/1000000),
				Type: "ping",
			}:
			}
			break
		case <-symbolCheckTimer.C:
		loop:
			for symbol, updateTime := range symbolUpdatedTimes {
				if time.Now().Sub(updateTime) > symbolTimeout {
					resubCounters[symbol] ++
					if resubCounters[symbol] > 10 {
						logger.Debugf("%s SUBSCRIBE 10 TIMES FAILED, RESTART WS", symbol)
						w.restart()
						return
					}
					logger.Debugf("SUBSCRIBE %s", fmt.Sprintf("/contractMarket/level2Depth5:%s", symbol))
					select {
					case <-ctx.Done():
						return
					case <-time.After(time.Millisecond):
						logger.Debugf("send msg to writeCh timeout in 1m, %s", fmt.Sprintf("/contractMarket/level2Depth5:%s", symbol))
					case w.writeCh <- kucoin_usdtfuture.SubscribeMsg{
						ID:             fmt.Sprintf("/contractMarket/level2Depth5:%s", symbol),
						Type:           "subscribe",
						Topic:          fmt.Sprintf("/contractMarket/level2Depth5:%s", symbol),
						PrivateChannel: false,
						Response:       true,
					}:
						symbolUpdatedTimes[symbol] = time.Now().Add(symbolCheckInterval * time.Duration(len(symbols)*2))
						break loop
					}
				}
			}
			symbolCheckTimer.Reset(symbolCheckInterval)
			break
		}
	}

}

func (w *KcufDepth5WS) Stop() {
	if atomic.CompareAndSwapInt32(&w.stopped, 0, 1) {
		close(w.done)
		logger.Debugf("stopped")
	}
}

func (w *KcufDepth5WS) restart() {
	select {
	case <-w.done:
	case <-time.After(time.Millisecond):
		w.Stop()
		logger.Debugf("KCPERP NIL TO RECONNECT CH TIMEOUT IN 1MS, STOP WS!")
	case w.reconnectCh <- nil:
		logger.Infof("KCPERP WS RESTART")
		select {
		case w.RestartCh <- nil:
		default:
			logger.Debugf("KCPERP NIL TO RESTART FAILED, STOP WS!")
		}
	}
}

func (w *KcufDepth5WS) Done() chan interface{} {
	return w.done
}

func NewKcufDepth5WS(
	ctx context.Context,
	proxy string,
	channels map[string]chan *Message,
) *KcufDepth5WS {
	ws := KcufDepth5WS{
		done:        make(chan interface{}),
		reconnectCh: make(chan interface{}),
		RestartCh:   make(chan interface{}, 4),
		writeCh:     make(chan interface{}, 4*len(channels)),
		symbolCh:    make(chan string, 4*len(channels)),
		stopped:     0,
	}
	go ws.mainLoop(ctx, proxy, channels)
	ws.reconnectCh <- nil
	return &ws
}
