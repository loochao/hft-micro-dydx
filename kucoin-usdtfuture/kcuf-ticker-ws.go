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

type TickerWS struct {
	writeCh     chan interface{}
	done        chan interface{}
	reconnectCh chan interface{}
	symbolCh    chan string
	stopped     int32
}

func (w *TickerWS) writeLoop(ctx context.Context, conn *websocket.Conn) {
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

func (w *TickerWS) readLoop(
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
		if readIndex == bookTickerReadPoolSize {
			readIndex = 0
		}
		msg = readPool[readIndex]
		n, err = r.Read(msg)
		if err == nil {
			readCounter++
			msg = msg[:n]
			if n > bookTickerReadMsgSize || msg[n-1] != '}' {
				partialReadCounter++
			readLoop:
				for {
					if len(msg) == cap(msg) {
						// Add more capacity (let append pick how much).
						msg = append(msg, 0)[:len(msg)]
						logger.Debugf("BAD BUFFER SIZE CAN'T READ INTO %d, MSG: %s", bookTickerReadMsgSize, msg)
					}
					n, err = r.Read(msg[len(msg):cap(msg)])
					msg = msg[:len(msg)+n]
					if err != nil {
						if err == io.EOF {
							break readLoop
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

		if readCounter%10000 == 0 {
			logger.Debugf("KUCOIN BOOK TICKER TOTAL READ %d PARTIAL READ %d", readCounter, partialReadCounter)
		}
		msgLen = len(msg)
		//{"data":{"symbol":"XBTUSDTM","sequence":1624824090150,"side":"sell","size":2,"price":33590,"bestBidSize":47,"bestBidPrice":"33590.0","bestAskPrice":"33591.0","tradeId":"60e92c8c3c7feb289d2ab154","ts":1625894028299209614,"bestAskSize":463},"subject":"ticker","topic":"/contractMarket/ticker:XBTUSDTM","type":"message"}
		//{"type":"message","topic":"/contractMarket/ticker:1INCHUSDTM","subject":"ticker","data":{"symbol":"1INCHUSDTM","sequence":1627371661456,"side":"buy","size":21,"price":2.379,"bestBidSize":203,"bestBidPrice":"2.377","bestAskPrice":"2.38","tradeId":"6105178a991e1303211759d8","ts":1627723658671236584,"bestAskSize":251}}
		if msgLen > 128 {
			if msg[2] == 't' {
				if msg[59] == ',' {
					symbol = common.UnsafeBytesToString(msg[50:58])
				} else if msg[60] == ',' {
					symbol = common.UnsafeBytesToString(msg[50:59])
				} else if msg[61] == ',' {
					symbol = common.UnsafeBytesToString(msg[50:60])
				} else if msg[58] == ',' {
					symbol = common.UnsafeBytesToString(msg[50:57])
				} else if msg[62] == ',' {
					symbol = common.UnsafeBytesToString(msg[50:61])
				} else {
					if time.Now().Sub(logSilentTime) > 0 {
						logSilentTime = time.Now().Add(time.Minute)
						logger.Debugf("UNKNOWN MSG %s", msg)
					}
					continue
				}
			} else if msg[2] == 'd' {
				if msg[27] == '"' {
					symbol = common.UnsafeBytesToString(msg[19:27])
				} else if msg[28] == '"' {
					symbol = common.UnsafeBytesToString(msg[19:28])
				} else if msg[29] == '"' {
					symbol = common.UnsafeBytesToString(msg[19:29])
				} else if msg[30] == '"' {
					symbol = common.UnsafeBytesToString(msg[19:30])
				} else if msg[31] == '"' {
					symbol = common.UnsafeBytesToString(msg[19:31])
				} else {
					if time.Now().Sub(logSilentTime) > 0 {
						logSilentTime = time.Now().Add(time.Minute)
						logger.Debugf("UNKNOWN MSG %s", msg)
					}
					continue
				}
			}
			if ch, ok = channels[symbol]; ok {
				select {
				case ch <- msg:
				default:
					//if time.Now().Sub(logSilentTime) > 0 {
					//	logger.Debugf("ch <- msg failed %s len(ch) %d", symbol, len(ch))
					//	logSilentTime = time.Now().Add(time.Minute)
					//}
				}
			}
		} else if msgLen > 3 && msg[2] == 'i' && msg[msgLen-3] == 'k' {
			logger.Debugf("%s", msg)
		} else if msgLen > 3 && msg[2] == 'i' && msg[msgLen-3] == 'g' {
			//logger.Debugf("%s", msg)
		} else {
			if time.Now().Sub(logSilentTime) > 0 {
				logSilentTime = time.Now().Add(common.LogInterval)
				logger.Debugf("UNKNOWN MSG %s", msg)
			}
		}
	}
}

func (w *TickerWS) reconnect(ctx context.Context, wsUrl string, proxy string, counter int64) (*websocket.Conn, error) {

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

func (w *TickerWS) mainLoop(
	ctx context.Context, api *API,
	proxy string,
	channels map[string]chan []byte,
) {

	logger.Debugf("START mainLoop")
	symbols := make([]string, 0)
	for symbol := range channels {
		symbols = append(symbols, symbol)
	}
	ctx, cancel := context.WithCancel(ctx)
	var internalCtx context.Context
	var internalCancel context.CancelFunc

	defer func() {
		cancel()
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
		}
	}
}

func (w *TickerWS) heartbeatLoop(ctx context.Context, conn *websocket.Conn, symbols []string, pingInterval time.Duration) {

	logger.Debugf("START heartbeatLoop")

	defer func() {
		err := conn.Close()
		if err != nil {
			logger.Debugf("conn.Close() ERROR %v", err)
		}
	}()

	symbolTimeout := time.Minute * 5
	symbolCheckInterval := time.Second
	pingTimer := time.NewTimer(time.Second)
	symbolCheckTimer := time.NewTimer(time.Second)
	defer symbolCheckTimer.Stop()
	symbolUpdatedTimes := make(map[string]time.Time)
	reSubCounters := make(map[string]int)
	for _, symbol := range symbols {
		reSubCounters[symbol] = 0
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
			reSubCounters[symbol] = 0
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
					reSubCounters[symbol] ++
					if reSubCounters[symbol] > 10 {
						logger.Debugf("%s subscribe 10 times failed, restart ws", symbol)
						w.restart()
						return
					}
					logger.Debugf("subscribe %s", fmt.Sprintf("/contractMarket/ticker:%s", symbol))
					select {
					case <-ctx.Done():
						return
					case <-time.After(time.Millisecond):
						logger.Debugf("send msg to writeCh timeout in 1m, %s", fmt.Sprintf("/contractMarket/ticker:%s", symbol))
					case w.writeCh <- SubscribeMsg{
						ID:             fmt.Sprintf("/contractMarket/ticker:%s", symbol),
						Type:           "subscribe",
						Topic:          fmt.Sprintf("/contractMarket/ticker:%s", symbol),
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

func (w *TickerWS) Stop() {
	if atomic.CompareAndSwapInt32(&w.stopped, 0, 1) {
		close(w.done)
		logger.Debugf("stopped")
	}
}

func (w *TickerWS) restart() {
	select {
	case <-w.done:
		return
	default:
	}
	select {
	case w.reconnectCh <- nil:
		logger.Infof("KCPERP WS RESTART")
	default:
		w.Stop()
		logger.Debugf("w.reconnectCh <- nil timeout failed, ch len %d, stop ws", len(w.reconnectCh))
	}
}

func (w *TickerWS) dataHandleLoop(ctx context.Context, symbol string, inputCh chan []byte, outputCh chan common.Ticker) {
	logSilentTime := time.Now()
	var err error
	index := -1
	pool := [common.BufferSizeFor100msData]*Ticker{}
	for i := 0; i < common.BufferSizeFor100msData; i++ {
		pool[i] = &Ticker{}
	}
	var ticker *Ticker
	var msg []byte
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case msg = <-inputCh:
			index++
			if index == common.BufferSizeFor100msData {
				index = 0
			}
			ticker = pool[index]
			err = ParseTicker(msg, ticker)
			if err != nil {
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("ParseTicker(msg, ticker) error %v %s", err, msg)
					logSilentTime = time.Now().Add(common.LogInterval)
				}
				continue
			}
			select {
			case outputCh <- ticker:
				select {
				case w.symbolCh <- symbol:
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("w.symbolCh <- symbol failed, ch len %d", len(w.symbolCh))
						logSilentTime = time.Now().Add(common.LogInterval)
					}
				}
			default:
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("outputCh <- ticker failed, ch len %d", len(outputCh))
					logSilentTime = time.Now().Add(common.LogInterval)
				}
			}
			break
		}
	}
}

func (w *TickerWS) Done() chan interface{} {
	return w.done
}

func NewTickerWS(
	ctx context.Context,
	api *API,
	proxy string,
	channels map[string]chan common.Ticker,
) *TickerWS {
	ws := TickerWS{
		done:        make(chan interface{}),
		reconnectCh: make(chan interface{}),
		writeCh:     make(chan interface{}, common.ChannelSizeLowLoad*len(channels)),
		symbolCh:    make(chan string, common.ChannelSizeLowLoad*len(channels)),
		stopped:     0,
	}
	messageChs := make(map[string]chan []byte)
	for symbol, ch := range channels {
		messageChs[symbol] = make(chan []byte, common.ChannelSizeLowLoadLowLatency)
		go ws.dataHandleLoop(ctx, symbol, messageChs[symbol], ch)
	}
	go ws.mainLoop(ctx, api, proxy, messageChs)
	ws.reconnectCh <- nil
	return &ws
}
