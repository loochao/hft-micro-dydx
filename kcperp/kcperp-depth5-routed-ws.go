package kcperp

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
	"unsafe"
)

type Depth5RoutedWebsocket struct {
	RestartCh   chan interface{}
	writeCh     chan interface{}
	done        chan interface{}
	reconnectCh chan interface{}
	symbolCh    chan string
	stopped     int32
}

func (w *Depth5RoutedWebsocket) writeLoop(ctx context.Context, conn *websocket.Conn) {
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

func (w *Depth5RoutedWebsocket) readLoop(
	conn *websocket.Conn,
	decay, bias float64,
	reportCount int,
	reportCh chan common.DepthReport,
	channels map[string]chan *common.DepthRawMessage,
) {
	logger.Debugf("START readLoop")
	defer func() {
		logger.Debugf("EXIT readLoop")
	}()
	totalCount := 0
	totalLen := 0
	filterCount := 0
	emaTimeDelta := bias
	timeDelta := 0.0
	decay1 := decay
	decay2 := 1.0 - decay
	logSilentTime := time.Now()
	var symbolBytes []byte
	var symbol string
	var ch chan *common.DepthRawMessage
	var ok bool
	var t int64
	var report = common.DepthReport{
		Exchange: "kcperp",
		Decay:    decay,
		Bias:     bias,
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
		msgLen = len(msg)
		totalCount += 1
		totalLen += msgLen
		if totalCount > reportCount {
			if reportCh != nil {
				report.DropRatio = float64(filterCount) / float64(totalCount)
				report.AvgLen = totalLen / totalCount
				report.EmaTimeDelta = emaTimeDelta
				select {
				case reportCh <- report:
				default:
				}
			}
			totalLen = 0
			totalCount = 0
			filterCount = 0
		}
		//{"data":{"sequence":1616576945844,"asks":[[17.834,10],[18.019,10154],[18.082,11060]],"bids":[[17.797,701],[17.793,1061],[17.784,199],[17.781,881],[17.779,407]],"ts":1618717277315,
		//"timestamp":1618717277315},"subject":"level2","topic":"/contractMarket/level2Depth5:ATOMUSDTM","type":"message"}
		if msgLen > 128 {
			if msg[msgLen-28] == ':' {
				symbolBytes = msg[msgLen-27 : msgLen-19]
				symbol = *(*string)(unsafe.Pointer(&symbolBytes))
				t, err = common.ParseInt(msg[msgLen-99 : msgLen-86])
				if err != nil {
					logger.Debugf("common.ParseInt error %v %s", err, msg[msgLen-99:msgLen-86])
					continue
				}
				timeDelta = float64(time.Now().UnixNano()/1000000 - t)
				if timeDelta > 1000 {
					timeDelta = 1000
				}
				emaTimeDelta = emaTimeDelta*decay1 + timeDelta*decay2
				if timeDelta > emaTimeDelta+bias {
					filterCount++
					continue
				}
			} else if msg[msgLen-29] == ':' {
				symbolBytes = msg[msgLen-28 : msgLen-19]
				symbol = *(*string)(unsafe.Pointer(&symbolBytes))
				t, err = common.ParseInt(msg[msgLen-100 : msgLen-87])
				if err != nil {
					logger.Debugf("common.ParseInt error %v %s", err, msg[msgLen-100:msgLen-87])
					continue
				}
				timeDelta = float64(time.Now().UnixNano()/1000000 - t)
				if timeDelta > 1000 {
					timeDelta = 1000
				}
				emaTimeDelta = emaTimeDelta*decay1 + timeDelta*decay2
				if timeDelta > emaTimeDelta+bias {
					filterCount++
					continue
				}
			} else if msg[msgLen-30] == ':' {
				symbolBytes = msg[msgLen-29 : msgLen-19]
				symbol = *(*string)(unsafe.Pointer(&symbolBytes))
				t, err = common.ParseInt(msg[msgLen-101 : msgLen-88])
				if err != nil {
					logger.Debugf("common.ParseInt error %v %s", err, msg[msgLen-101:msgLen-88])
					continue
				}
				timeDelta = float64(time.Now().UnixNano()/1000000 - t)
				if timeDelta > 1000 {
					timeDelta = 1000
				}
				emaTimeDelta = emaTimeDelta*decay1 + timeDelta*decay2
				if timeDelta > emaTimeDelta+bias {
					filterCount++
					continue
				}
			} else if msg[msgLen-31] == ':' {
				symbolBytes = msg[msgLen-30 : msgLen-19]
				symbol = *(*string)(unsafe.Pointer(&symbolBytes))
				t, err = common.ParseInt(msg[msgLen-102 : msgLen-89])
				if err != nil {
					logger.Debugf("common.ParseInt error %v %s", err, msg[msgLen-102:msgLen-89])
					continue
				}
				timeDelta = float64(time.Now().UnixNano()/1000000 - t)
				if timeDelta > 1000 {
					timeDelta = 1000
				}
				emaTimeDelta = emaTimeDelta*decay1 + timeDelta*decay2
				if timeDelta > emaTimeDelta+bias {
					filterCount++
					continue
				}
			} else {
				logger.Debugf("OTHER MSG %s", msg)
				continue
			}

			//logger.Debugf("SYMBOL %s %v", symbol, time.Unix(0, t*1000000))

			if ch, ok = channels[symbol]; ok {
				select {
				case ch <- &common.DepthRawMessage{
					Symbol: symbol,
					Time:   time.Unix(0, t*1000000),
					Depth:  msg,
				}:
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("ch <- &common.DepthRawMessage failed %s len(ch) %d", symbol, len(ch))
						logSilentTime = time.Now().Add(time.Minute)
					}
				}
				select {
				case w.symbolCh <- symbol:
				default:
				}
			}
		}
	}
}

func (w *Depth5RoutedWebsocket) readAll(r io.Reader) ([]byte, error) {
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

func (w *Depth5RoutedWebsocket) reconnect(ctx context.Context, wsUrl string, proxy string, counter int64) (*websocket.Conn, error) {

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

func (w *Depth5RoutedWebsocket) mainLoop(
	ctx context.Context, api *API,
	proxy string,
	decay, bias float64,
	reportCount int,
	reportCh chan common.DepthReport,
	channels map[string]chan *common.DepthRawMessage,
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
			go w.readLoop(conn, decay, bias, reportCount, reportCh, channels)
			go w.writeLoop(internalCtx, conn)
			go w.heartbeatLoop(internalCtx, conn, symbols, time.Millisecond*time.Duration(connectToken.InstanceServers[0].PingInterval))
		}
	}
}

func (w *Depth5RoutedWebsocket) heartbeatLoop(ctx context.Context, conn *websocket.Conn, symbols []string, pingInterval time.Duration) {

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
		case <-pingTimer.C:
			pingTimer.Reset(pingInterval)
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Millisecond):
				logger.Debug("send ping to writeCh timeout in 1ms")
			case w.writeCh <- Ping{
				ID:   fmt.Sprintf("%d", time.Now().Nanosecond()/1000000),
				Type: "ping",
			}:
			}
			break
		case <-symbolCheckTimer.C:
		loop:
			for symbol, updateTime := range symbolUpdatedTimes {
				if time.Now().Sub(updateTime) > symbolTimeout {
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

func (w *Depth5RoutedWebsocket) Stop() {
	if atomic.LoadInt32(&w.stopped) == 0 {
		atomic.StoreInt32(&w.stopped, 1)
		close(w.done)
		logger.Infof("KCPERP DEPTH5 WS STOPPED")
	}
}

func (w *Depth5RoutedWebsocket) restart() {
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

func (w *Depth5RoutedWebsocket) Done() chan interface{} {
	return w.done
}

func NewDepth5RoutedWebsocket(
	ctx context.Context,
	api *API,
	proxy string,
	decay, bias float64,
	reportCount int,
	reportCh chan common.DepthReport,
	channels map[string]chan *common.DepthRawMessage,
) *Depth5RoutedWebsocket {
	ws := Depth5RoutedWebsocket{
		done:        make(chan interface{}),
		reconnectCh: make(chan interface{}),
		RestartCh:   make(chan interface{}, 100),
		writeCh:     make(chan interface{}, 100*len(channels)),
		symbolCh:    make(chan string, 100*len(channels)),
		stopped:     0,
	}
	go ws.mainLoop(ctx, api, proxy, decay, bias, reportCount, reportCh, channels)
	ws.reconnectCh <- nil
	return &ws
}
