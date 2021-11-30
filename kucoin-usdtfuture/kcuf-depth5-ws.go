package kucoin_usdtfuture

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

func (w *Depth5WS) readLoop(
	conn *websocket.Conn,
	channels map[string]chan []byte,
) {
	logger.Debugf("START readLoop")
	defer func() {
		logger.Debugf("EXIT readLoop")
	}()
	logSilentTime := time.Now()
	var symbol string
	var ch chan []byte
	var ok bool
	var msgLen int
	var readPool = [depth5TickerReadPoolSize][]byte{}
	var readIndex = -1
	var msg []byte
	var n int
	var r io.Reader
	var err error
	for i := range readPool {
		readPool[i] = make([]byte, depth5TickerReadMsgSize)
	}
mainLoop:
	for {
		err = conn.SetReadDeadline(time.Now().Add(time.Minute))
		if err != nil {
			logger.Debugf("conn.SetReadDeadline error %v", err)
			go w.restart()
			return
		}
		_, r, err = conn.NextReader()
		if err != nil {
			logger.Debugf("conn.NextReader error %v", err)
			go w.restart()
			return
		}
		readIndex += 1
		if readIndex == depth5TickerReadPoolSize {
			readIndex = 0
		}
		msg = readPool[readIndex]
		n, err = r.Read(msg)
		if err == nil {
			if n < depth5TickerReadPoolSize {
				msg = msg[:n]
			} else {
				for {
					if len(msg) == cap(msg) {
						// Add more capacity (let append pick how much).
						msg = append(msg, 0)[:len(msg)]
					}
					n, err = r.Read(msg[len(msg):cap(msg)])
					msg = msg[:len(msg)+n]
					if err != nil {
						if err == io.EOF {
							logger.Debugf("BAD BUFFER SIZE CAN'T READ %d INTO %d, MSG: %s", len(msg), depth5TickerReadMsgSize, msg)
						} else {
							logger.Debugf("r.Read error %v", err)
							continue mainLoop
						}
					} else {
						logger.Debugf("BAD BUFFER SIZE CAN'T READ %d INTO %d, MSG: %s", len(msg), depth5TickerReadMsgSize, msg)
					}
				}
			}
		} else {
			logger.Debugf("r.Read error %v", err)
			continue mainLoop
		}

		//中间有一次数据变更，可能两种格式
		//{"data":{"sequence":1616576945844,"asks":[[17.834,10],[18.019,10154],[18.082,11060]],"bids":[[17.797,701],[17.793,1061],[17.784,199],[17.781,881],[17.779,407]],"ts":1618717277315,
		//{"type":"message","topic":"/contractMarket/level2Depth5:GRTUSDTM","subject":"level2","data":{"sequence":1627365704601,"asks":[[0.62612,194],[0.62625,194],[0.62640,3230],[0.62655,6368],[0.62656,6300]],"bids":[[0.62580,1846],[0.62565,1087],[0.62555,1959],[0.62551,1038],[0.62550,601]],"ts":1627723139256,"timestamp":1627723139256}}
		msgLen = len(msg)
		if msgLen > 128 {
			if msg[2] == 't' {
				if msg[65] == ',' {
					symbol = common.UnsafeBytesToString(msg[56:64])
				} else if msg[66] == ',' {
					symbol = common.UnsafeBytesToString(msg[56:65])
				} else if msg[67] == ',' {
					symbol = common.UnsafeBytesToString(msg[56:66])
				} else if msg[64] == ',' {
					symbol = common.UnsafeBytesToString(msg[56:63])
				} else if msg[68] == ',' {
					symbol = common.UnsafeBytesToString(msg[56:67])
				} else {
					if time.Now().Sub(logSilentTime) > 0 {
						logSilentTime = time.Now().Add(time.Minute)
						logger.Debugf("OTHER MSG %s", msg)
					}
					continue
				}
			} else if msg[2] == 'd' {
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
						logSilentTime = time.Now().Add(time.Minute)
						logger.Debugf("OTHER MSG %s", msg)
					}
					continue
				}
			} else {
				if time.Now().Sub(logSilentTime) > 0 {
					logSilentTime = time.Now().Add(time.Minute)
					logger.Debugf("OTHER MSG %s", msg)
				}
				continue
			}
			if ch, ok = channels[symbol]; ok {
				select {
				case ch <- msg:
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logSilentTime = time.Now().Add(time.Minute)
						logger.Debugf("ch <- msg failed %s len(ch) %d", symbol, len(ch))
					}
				}
			}
		} else if msgLen > 3 && msg[2] == 'i' && msg[msgLen-3] == 'k' {
			logger.Debugf("%s", msg)
		} else {
			if time.Now().Sub(logSilentTime) > 0 {
				logSilentTime = time.Now().Add(common.LogInterval)
				logger.Debugf("UNKNOWN MSG %s", msg)
			}
		}

	}
}

func (w *Depth5WS) reconnect(ctx context.Context, wsUrl string, proxy string, counter int64) (*websocket.Conn, error) {

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

func (w *Depth5WS) mainLoop(
	ctx context.Context, api *API,
	proxy string,
	channels map[string]chan []byte,
) {

	logger.Debugf("START mainLoop")
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

func (w *Depth5WS) heartbeatLoop(ctx context.Context, conn *websocket.Conn, symbols []string, pingInterval time.Duration) {

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
			case w.writeCh <- Ping{
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
					case w.writeCh <- SubscribeMsg{
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

func (w *Depth5WS) Stop() {
	if atomic.CompareAndSwapInt32(&w.stopped, 0, 1) {
		close(w.done)
		logger.Debugf("stopped")
	}
}

func (w *Depth5WS) restart() {
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

func (w *Depth5WS) dataHandleLoop(ctx context.Context, symbol string, inputCh chan []byte, outputCh chan common.Depth) {
	logSilentTime := time.Now()
	var err error
	index := -1
	pool := [4]*Depth5{}
	for i := 0; i < 4; i++ {
		pool[i] = &Depth5{}
	}
	var depth5 *Depth5
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case msg := <-inputCh:
			index++
			if index == 4 {
				index = 0
			}
			depth5 = pool[index]
			err = ParseDepth5(msg, depth5)
			if err != nil {
				logger.Debugf("ParseDepth5(msg) error %v %s", err, msg)
				continue
			}
			select {
			case outputCh <- depth5:
				select {
				case w.symbolCh <- symbol:
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("w.symbolCh <- symbol failed, ch len %d", len(w.symbolCh))
						logSilentTime = time.Now().Add(time.Minute)
					}
				}
			default:
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("outputCh <- depth5 failed, ch len %d", len(outputCh))
					logSilentTime = time.Now().Add(time.Minute)
				}
			}
		}
	}
}

func (w *Depth5WS) Done() chan interface{} {
	return w.done
}

func NewDepth5WS(
	ctx context.Context,
	api *API,
	proxy string,
	channels map[string]chan common.Depth,
) *Depth5WS {
	ws := Depth5WS{
		done:        make(chan interface{}),
		reconnectCh: make(chan interface{}),
		RestartCh:   make(chan interface{}, 4),
		writeCh:     make(chan interface{}, 4*len(channels)),
		symbolCh:    make(chan string, 4*len(channels)),
		stopped:     0,
	}
	messageChs := make(map[string]chan []byte)
	for symbol, ch := range channels {
		messageChs[symbol] = make(chan []byte, 128)
		go ws.dataHandleLoop(ctx, symbol, messageChs[symbol], ch)
	}
	go ws.mainLoop(ctx, api, proxy, messageChs)
	ws.reconnectCh <- nil
	return &ws
}
