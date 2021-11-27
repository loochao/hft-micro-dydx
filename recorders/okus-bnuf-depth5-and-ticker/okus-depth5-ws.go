package main

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	//okex_usdtspot "github.com/geometrybase/hft-micro/okex-usdtspot"
	"github.com/gorilla/websocket"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"
)

type OkusDepth5WS struct {
	writeCh     chan interface{}
	done        chan interface{}
	reconnectCh chan interface{}
	symbolCh    chan string
	pingCh      chan []byte
	stopped     int32
}

func (w *OkusDepth5WS) writeLoop(ctx context.Context, conn *websocket.Conn) {
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
func (w *OkusDepth5WS) parseBinaryResponse(resp []byte) ([]byte, error) {
	var standardMessage []byte
	var err error
	// Detect GZIP
	if resp[0] == 31 && resp[1] == 139 {
		b := bytes.NewReader(resp)
		var gReader *gzip.Reader
		gReader, err = gzip.NewReader(b)
		if err != nil {
			return standardMessage, err
		}
		standardMessage, err = w.readAll(gReader)
		if err != nil {
			return standardMessage, err
		}
		err = gReader.Close()
		if err != nil {
			return standardMessage, err
		}
	} else {
		reader := flate.NewReader(bytes.NewReader(resp))
		standardMessage, err = w.readAll(reader)
		if err != nil {
			return standardMessage, err
		}
		err = reader.Close()
		if err != nil {
			return standardMessage, err
		}
	}
	return standardMessage, nil
}

func (w *OkusDepth5WS) readLoop(conn *websocket.Conn, channels map[string]chan *Message) {
	logger.Debugf("START readLoop")
	defer logger.Debugf("EXIT readLoop")
	logSilentTime := time.Now()
	var msg []byte
	var msgLen int
	var mType int
	var resp []byte
	var err error
	var subscribeEvent okex_usdtspot.SubscribeEvent
	var segs []string

	var ch chan *Message
	var ok bool
	var symbol string
	var message *Message
	index := -1
	pool := [1024]*Message{}
	for i := 0; i < 1024; i++ {
		pool[i] = &Message{
			Source: []byte{'O', 'D'},
		}
	}
	for {
		err = conn.SetReadDeadline(time.Now().Add(time.Minute))
		if err != nil {
			logger.Debugf("conn.SetReadDeadline error %v", err)
			w.restart()
			return
		}

		mType, resp, err = conn.ReadMessage()
		if err != nil {
			logger.Debugf("conn.NextReader error %v", err)
			w.restart()
			return
		}

		switch mType {
		case websocket.TextMessage:
			msg = resp
		case websocket.BinaryMessage:
			msg, err = w.parseBinaryResponse(resp)
			if err != nil {
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("w.parseBinaryResponse error %v", err)
					logSilentTime = time.Now().Add(time.Minute)
				}
				continue
			}
		}
		msgLen = len(msg)
		if msg[2] == 'e' {
			err = json.Unmarshal(msg, &subscribeEvent)
			if err != nil {
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("json.Unmarshal(msg, &subscribeEvent) error %v", err)
					logSilentTime = time.Now().Add(time.Minute)
				}
			}
			segs = strings.Split(subscribeEvent.Channel, ":")
			if len(segs) == 2 {
				select {
				case w.symbolCh <- segs[1]:
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("w.symbolCh <- symbol failed %s ch len %d", segs[2], len(w.symbolCh))
						logSilentTime = time.Now().Add(time.Minute)
					}
				}
			}
			continue
		} else if msg[2] == 't' && len(msg) > 128 {
			//{"table":"spot/depth5","data":[{"asks":[["31.605","4.32464","1"],["31.607","85","1"],["31.61","2","1"],["31.612","0.1","1"],["31.614","1.405511","1"]],"bids":[["31.583","302.09312","3"],["31.582","0.9","1"],["31.58","111.30127","1"],["31.579","76","1"],["31.576","31.83446","1"]],"instrument_id":"LINK-USDT","timestamp":"2021-04-25T08:24:33.352Z"}]}
			if msg[msgLen-53] == ':' {
				symbol = common.UnsafeBytesToString(msg[msgLen-51 : msgLen-43])
			} else if msg[msgLen-54] == ':' {
				symbol = common.UnsafeBytesToString(msg[msgLen-52 : msgLen-43])
			} else if msg[msgLen-55] == ':' {
				symbol = common.UnsafeBytesToString(msg[msgLen-53 : msgLen-43])
			} else if msg[msgLen-56] == ':' {
				symbol = common.UnsafeBytesToString(msg[msgLen-54 : msgLen-43])
			} else if msg[msgLen-52] == ':' {
				symbol = common.UnsafeBytesToString(msg[msgLen-49 : msgLen-43])
			} else {
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("other msg %s", msg)
					logSilentTime = time.Now().Add(time.Minute)
				}
				continue
			}
		} else if msgLen == 4 && msg[2] == 'p' {
			select {
			case w.pingCh <- msg:
			default:
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("w.pongCh <- msg failed %s ch len %d", symbol, len(w.pingCh))
					logSilentTime = time.Now().Add(time.Minute)
				}
			}
			continue
		//} else if msgLen == 4 {
		//	if time.Now().Sub(logSilentTime) > 0 {
		//		logger.Debugf("other msg %s", msg)
		//		logSilentTime = time.Now().Add(time.Minute)
		//	}
		//	continue
		}else{
			continue
		}
		//logger.Debugf("%s %v", symbol, channels)
		if ch, ok = channels[symbol]; ok {
			index++
			if index == 1024 {
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
			select {
			case w.symbolCh <- symbol:
			default:
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("w.symbolCh <- symbol failed %s ch len %d", symbol, len(w.symbolCh))
					logSilentTime = time.Now().Add(time.Minute)
				}
			}
		}
	}
}

func (w *OkusDepth5WS) readAll(r io.Reader) ([]byte, error) {
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

func (w *OkusDepth5WS) reconnect(ctx context.Context, wsUrl string, proxy string, counter int64) (*websocket.Conn, error) {

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

func (w *OkusDepth5WS) mainLoop(ctx context.Context, proxy string, channels map[string]chan *Message) {
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
			conn, err := w.reconnect(internalCtx, "wss://real.okex.com:8443/ws/v3", proxy, 0)
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

func (w *OkusDepth5WS) heartbeatLoop(ctx context.Context, conn *websocket.Conn, symbols []string) {
	logger.Debugf("START heartbeatLoop")
	defer func() {
		logger.Debugf("Exit heartbeatLoop")
		err := conn.Close()
		if err != nil {
			logger.Debugf("conn.Close() ERROR %v", err)
		}
	}()
	symbolTimeout := time.Minute
	symbolCheckInterval := time.Second
	symbolCheckTimer := time.NewTimer(time.Second)
	defer symbolCheckTimer.Stop()
	symbolUpdatedTimes := make(map[string]time.Time)
	for _, symbol := range symbols {
		symbolUpdatedTimes[symbol] = time.Unix(0, 0)
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
			case w.writeCh <- []byte("ping"):
				break
			default:
				logger.Debugf("w.writeCh <- ping failed, ch len %d", len(w.writeCh))
			}
			break
		case symbol := <-w.symbolCh:
			pingTimer.Reset(time.Second * 15)
			trafficTimeout.Reset(time.Second * 30)
			symbolUpdatedTimes[symbol] = time.Now()
			break
		case <-w.pingCh:
			logger.Debugf("PING MSG")
			pingTimer.Reset(time.Second * 15)
			trafficTimeout.Reset(time.Second * 30)
			break
		case <-symbolCheckTimer.C:
			args := make([]string, 0)
			for symbol, updateTime := range symbolUpdatedTimes {
				if time.Now().Sub(updateTime) > symbolTimeout {
					args = append(args, fmt.Sprintf("spot/depth5:%s", symbol))
					symbolUpdatedTimes[symbol] = time.Now().Add(symbolTimeout)
				}
			}
			if len(args) > 0 {
				logger.Debugf("SUBSCRIBE %s", args)
				for start := 0; start < len(args); start += 50 {
					end := start + 50
					if end > len(args) {
						end = len(args)
					}
					select {
					case w.writeCh <- okex_usdtspot.Subscription{
						Op:   "subscribe",
						Args: args[start:end],
					}:
					default:
						logger.Debugf("w.writeCh <- Subscription failed, ch len %d", len(w.writeCh))
					}
				}
			}
			symbolCheckTimer.Reset(symbolCheckInterval)
			break
		}
	}

}

func (w *OkusDepth5WS) Stop() {
	if atomic.CompareAndSwapInt32(&w.stopped, 0, 1) {
		close(w.done)
		logger.Debugf("stopped")
	}
}

func (w *OkusDepth5WS) restart() {
	select {
	case w.reconnectCh <- nil:
	default:
		logger.Debugf("w.reconnectCh <- nil failed, ch len %d", len(w.reconnectCh))
	}
}

func (w *OkusDepth5WS) Done() chan interface{} {
	return w.done
}

func NewOkusDepth5WS(
	ctx context.Context,
	proxy string,
	channels map[string]chan *Message,
) *OkusDepth5WS {
	ws := OkusDepth5WS{
		done:        make(chan interface{}),
		reconnectCh: make(chan interface{}, 4),
		writeCh:     make(chan interface{}, len(channels)),
		symbolCh:    make(chan string, len(channels)),
		pingCh:      make(chan []byte, 4),
		stopped:     0,
	}
	go ws.mainLoop(ctx, proxy, channels)
	ws.reconnectCh <- nil
	return &ws
}
