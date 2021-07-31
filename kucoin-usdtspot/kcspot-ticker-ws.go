package kucoin_usdtspot

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
		msgLen = len(msg)
		//{"data":{"sequence":"1618200194453","bestAsk":"32704.5","size":"0.00058862","bestBidSize":"0.06704767","price":"32704.5","time":1626290937603,"bestAskSize":"0.01955972","bestBid":"32704.4"},"subject":"trade.ticker","topic":"/market/ticker:BTC-USDT","type":"message"}
		//{"type":"message","topic":"/market/ticker:BTC-USDT","subject":"trade.ticker","data":{"bestAsk":"41217.7","bestAskSize":"0.21545096","bestBid":"41217.6","bestBidSize":"0.0265","price":"41217.7","sequence":"1618607525224","size":"0.00043659","time":1627752855836}}
		if msgLen > 128 {
			if msg[2] == 't' && msg[9] == 'm' {
				if msg[50] == '"' {
					symbol = common.UnsafeBytesToString(msg[42:50])
				} else if msg[51] == '"' {
					symbol = common.UnsafeBytesToString(msg[42:51])
				} else if msg[52] == '"' {
					symbol = common.UnsafeBytesToString(msg[42:52])
				} else if msg[53] == '"' {
					symbol = common.UnsafeBytesToString(msg[42:53])
				} else if msg[49] == '"' {
					symbol = common.UnsafeBytesToString(msg[42:49])
				} else {
					if time.Now().Sub(logSilentTime) > 0 {
						logSilentTime = time.Now().Add(time.Minute)
						logger.Debugf("OTHER MSG %s", msg)
					}
					continue
				}
				logger.Debugf("%s %s", symbol,msg)
			} else if msg[2] == 'd' {
				if msg[msgLen-27] == ':' {
					symbol = common.UnsafeBytesToString(msg[msgLen-26 : msgLen-19])
				} else if msg[msgLen-28] == ':' {
					symbol = common.UnsafeBytesToString(msg[msgLen-27 : msgLen-19])
				} else if msg[msgLen-29] == ':' {
					symbol = common.UnsafeBytesToString(msg[msgLen-29 : msgLen-19])
				} else if msg[msgLen-30] == ':' {
					symbol = common.UnsafeBytesToString(msg[msgLen-30 : msgLen-19])
				} else if msg[msgLen-31] == ':' {
					symbol = common.UnsafeBytesToString(msg[msgLen-31 : msgLen-19])
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
						logger.Debugf("ch <- msg failed %s len(ch) %d", symbol, len(ch))
						logSilentTime = time.Now().Add(time.Minute)
					}
				}
			}
		} else {
			if msgLen > 3 && msg[2] == 'i' && msg[msgLen-3] == 'k' {
				logger.Debugf("%s", msg)
			}
			continue
		}

	}
}

func (w *TickerWS) readAll(r io.Reader) ([]byte, error) {
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
					logger.Debugf("subscribe %s", fmt.Sprintf("/market/ticker:%s", symbol))
					select {
					case <-ctx.Done():
						return
					case <-time.After(time.Millisecond):
						logger.Debugf("send msg to writeCh timeout in 1m, %s", fmt.Sprintf("/market/ticker:%s", symbol))
					case w.writeCh <- SubscribeMsg{
						ID:             fmt.Sprintf("/market/ticker:%s", symbol),
						Type:           "subscribe",
						Topic:          fmt.Sprintf("/market/ticker:%s", symbol),
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
	pool := [4]*Ticker{}
	for i := 0; i < 4; i++ {
		pool[i] = &Ticker{}
	}
	var ticker *Ticker
	var msg []byte
	var parseTimer = time.NewTimer(time.Hour * 9999)
	defer parseTimer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case <-parseTimer.C:
			index++
			if index == 4 {
				index = 0
			}
			ticker = pool[index]
			err = ParseTicker(msg, ticker)
			if err != nil {
				if time.Now().Sub(logSilentTime) > 0 {
					logSilentTime = time.Now().Add(time.Minute)
					logger.Debugf("ParseTicker(msg) error %v %s", err, msg)
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
						logSilentTime = time.Now().Add(time.Minute)
					}
				}
			default:
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("outputCh <- depth5 failed, ch len %d", len(outputCh))
					logSilentTime = time.Now().Add(time.Minute)
				}
			}
			break
		case msg = <-inputCh:
			parseTimer.Reset(time.Millisecond)
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
		writeCh:     make(chan interface{}, 4*len(channels)),
		symbolCh:    make(chan string, 4*len(channels)),
		stopped:     0,
	}
	messageChs := make(map[string]chan []byte)
	for symbol, ch := range channels {
		messageChs[symbol] = make(chan []byte, 64)
		go ws.dataHandleLoop(ctx, symbol, messageChs[symbol], ch)
	}
	go ws.mainLoop(ctx, api, proxy, messageChs)
	ws.reconnectCh <- nil
	return &ws
}
