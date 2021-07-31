package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	kucoin_usdtspot "github.com/geometrybase/hft-micro/kucoin-usdtspot"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/gorilla/websocket"
	"io"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"
)

type Depth5WS struct {
	RestartCh   chan interface{}
	writeCh     chan interface{}
	done        chan interface{}
	reconnectCh chan interface{}
	symbolCh    chan string
	stopped     int32
}

func (w *Depth5WS) writeLoop(ctx context.Context, conn *websocket.Conn) {
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
					logger.Debugf("json.Marshal err %v", err)
					continue
				}
			}
			err = conn.SetWriteDeadline(time.Now().Add(300 * time.Millisecond))
			if err != nil {
				w.restart()
				return
			}

			err = conn.WriteMessage(websocket.TextMessage, bytes)

			if err != nil {
				logger.Debugf("conn.WriteMessage %s error %v", string(bytes), err)
				w.restart()
				return
			}
		}
	}
}

func (w *Depth5WS) readLoop(conn *websocket.Conn, channels map[string]chan *Message) {
	logger.Debugf("START readLoop")
	defer logger.Debugf("EXIT readLoop")
	var symbol string
	var msgLen int
	var ch chan *Message
	var ok bool
	var message *Message
	var logSilentTime time.Time
	index := -1
	pool := [4096]*Message{}
	for i := 0; i < 4096; i++ {
		pool[i] = &Message{
			Source: []byte{'K', 'D'},
		}
	}
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
			logger.Debugf("readAll error %v", err)
			go w.restart()
			return
		}

		msgLen = len(msg)

		//{"data":{"asks":[["55447.5","0.00128653"],["55447.6","0.0040067"],["55447.7","5.26962769"],["55449","0.00016278"],["55451.5","0.00013396"]],"bids":[["55403.1","0.01254575"],["55402.5","0.00005319"],["55279.9","0.201"],["55268.3","0.02406837"],["55233.5","0.0004668"]],"timestamp":1618724853172},"subject":"level2","topic":"/spotMarket/level2Depth5:BTC-USDT","type":"message"}`)
		//{"type":"message","topic":"/spotMarket/level2Depth5:ENJ-USDT","subject":"level2","data":{"asks":[["1.421","291.4019"],["1.4211","257.9855"],["1.4214","17.2666"],["1.4215","538.358"],["1.4217","2111.2333"]],"bids":[["1.4195","507.9287"],["1.4193","538.358"],["1.4191","308.6314"],["1.419","2110.5551"],["1.4188","4320.975"]],"timestamp":1627748904217}}

		if msgLen > 128 {
			if msg[2] == 'd' {
				if msg[msgLen-28] == ':' {
					symbol = common.UnsafeBytesToString(msg[msgLen-27 : msgLen-19])
				} else if msg[msgLen-29] == ':' {
					symbol = common.UnsafeBytesToString(msg[msgLen-28 : msgLen-19])
				} else if msg[msgLen-30] == ':' {
					symbol = common.UnsafeBytesToString(msg[msgLen-29 : msgLen-19])
				} else if msg[msgLen-31] == ':' {
					symbol = common.UnsafeBytesToString(msg[msgLen-30 : msgLen-19])
				} else {
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("OTHER MSG %s", msg)
						logSilentTime = time.Now().Add(time.Minute)
					}
					continue
				}
			} else if msg[2] == 't' && msg[51] == ':' {
				if msg[60] == '"' {
					symbol = common.UnsafeBytesToString(msg[52:60])
				} else if msg[61] == '"' {
					symbol = common.UnsafeBytesToString(msg[52:61])
				} else if msg[62] == '"' {
					symbol = common.UnsafeBytesToString(msg[52:62])
				} else if msg[63] == '"' {
					symbol = common.UnsafeBytesToString(msg[52:63])
				} else if msg[59] == '"' {
					symbol = common.UnsafeBytesToString(msg[52:59])
				} else if msg[64] == '"' {
					symbol = common.UnsafeBytesToString(msg[52:64])
				} else {
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("OTHER MSG %s", msg)
						logSilentTime = time.Now().Add(time.Minute)
					}
					continue
				}
			}else{
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("OTHER MSG %s", msg)
					logSilentTime = time.Now().Add(time.Minute)
				}
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
		}
	}
}

func (w *Depth5WS) readAll(r io.Reader) ([]byte, error) {
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

func (w *Depth5WS) reconnect(ctx context.Context, wsUrl string, proxy string, counter int64) (*websocket.Conn, error) {

	if counter != 0 {
		logger.Debugf("reconnect %s, %d retries", wsUrl, counter)
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

func (w *Depth5WS) mainLoop(ctx context.Context,  proxy string, channels map[string]chan *Message) {
	logger.Debugf("START mainLoop")

	api, err := kucoin_usdtspot.NewAPI("", "", "", proxy)
	if err != nil {
		logger.Debugf("NewAPI error %v", err)
		return
	}

	symbols := make([]string, 0)
	for symbol := range channels {
		symbols = append(symbols, symbol)
	}
	ctx, cancel := context.WithCancel(ctx)
	var internalCtx context.Context
	var internalCancel context.CancelFunc

	defer func() {
		if internalCancel != nil {
			internalCancel()
		}
		cancel()
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
		case <-w.done:
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
			break
		case <-reconnectTimer.C:
			if internalCancel != nil {
				internalCancel()
			}
			internalCtx, internalCancel = context.WithCancel(ctx)
			connectToken, err := api.GetPublicConnectToken(internalCtx)
			if err != nil {
				logger.Debugf("api.GetPublicConnectToken error %v", err)
				internalCancel()
				w.Stop()
				return
			}
			if len(connectToken.InstanceServers) == 0 {
				logger.Debugf("no InstanceServers %v", connectToken)
				internalCancel()
				w.Stop()
				return
			}
			urlStr := connectToken.InstanceServers[0].Endpoint + "?token=" + connectToken.Token
			conn, err := w.reconnect(internalCtx, urlStr, proxy, 0)
			if err != nil {
				internalCancel()
				logger.Debugf("w.reconnect error %v", err)
				return
			}
			go w.readLoop(conn, channels)
			go w.writeLoop(internalCtx, conn)
			go w.heartbeatLoop(internalCtx, conn, symbols)
			break
		}
	}
}

func (w *Depth5WS) heartbeatLoop(ctx context.Context, conn *websocket.Conn, symbols []string) {
	logger.Debugf("START heartbeatLoop")
	defer func() {
		logger.Debugf("EXIT heartbeatLoop")
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
		case <-pingTimer.C:
			pingTimer.Reset(time.Second * 15)
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Millisecond):
				logger.Debug("w.writeCh <- Ping timeout in 1ms")
			case w.writeCh <- kucoin_usdtspot.Ping{
				ID:   fmt.Sprintf("%d", time.Now().Nanosecond()/1000000),
				Type: "ping",
			}:
			}
			break
		case <-symbolCheckTimer.C:
		loop:
			for symbol, updateTime := range symbolUpdatedTimes {
				if time.Now().Sub(updateTime) > symbolTimeout {
					logger.Debugf("SUBSCRIBE %s", fmt.Sprintf("/spotMarket/level2Depth5:%s", symbol))
					select {
					case <-ctx.Done():
					case <-time.After(time.Millisecond):
						logger.Debugf("w.writeCh <- SubscribeMsg timeout in 1ms, %s", fmt.Sprintf("/spotMarket/level2Depth5:%s", symbol))
					case w.writeCh <- kucoin_usdtspot.SubscribeMsg{
						ID:             fmt.Sprintf("/spotMarket/level2Depth5:%s", symbol),
						Type:           "subscribe",
						Topic:          fmt.Sprintf("/spotMarket/level2Depth5:%s", symbol),
						PrivateChannel: false,
						Response:       false,
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

func (w *Depth5WS) Stop() {
	if atomic.LoadInt32(&w.stopped) == 0 {
		atomic.StoreInt32(&w.stopped, 1)
		close(w.done)
		logger.Debugf("stopped")
	}
}

func (w *Depth5WS) restart() {
	select {
	case w.RestartCh <- nil:
	default:
		logger.Debugf("w.RestartCh <- nil failed")
	}
	select {
	case <-w.done:
		return
	case <-time.After(time.Millisecond):
		logger.Debugf("w.reconnectCh <- nil time out in 1m, stop ws")
		w.Stop()
	case w.reconnectCh <- nil:
	}
}

func (w *Depth5WS) Done() chan interface{} {
	return w.done
}

func NewKcusDepth5WS(
	ctx context.Context,
	proxy string,
	channels map[string]chan *Message,
) *Depth5WS {
	ws := Depth5WS{
		done:        make(chan interface{}),
		reconnectCh: make(chan interface{}),
		RestartCh:   make(chan interface{}, 4),
		writeCh:     make(chan interface{}, 4*len(channels)),
		symbolCh:    make(chan string, 4*len(channels)),
		stopped:     0,
	}
	go ws.mainLoop(ctx,  proxy, channels)
	ws.reconnectCh <- nil
	return &ws
}
