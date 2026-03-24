package okexv5_usdtswap

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

type RawFundingRateWS struct {
	writeCh     chan interface{}
	done        chan interface{}
	reconnectCh chan interface{}
	symbolCh    chan string
	pingCh      chan []byte
	prefix      []byte
	stopped     int32
}

func (w *RawFundingRateWS) writeLoop(ctx context.Context, conn *websocket.Conn) {
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

func (w *RawFundingRateWS) readLoop(conn *websocket.Conn, channels map[string]chan *common.RawMessage) {
	logger.Debugf("START readLoop")
	defer logger.Debugf("EXIT readLoop")
	logSilentTime := time.Now()
	var symbol string
	var msg []byte
	var err error
	var ch chan *common.RawMessage
	var ok bool
	var message *common.RawMessage
	var r io.Reader
	const bufferSize = 8192
	index := -1
	pool := [bufferSize]*common.RawMessage{}
	for i := 0; i < bufferSize; i++ {
		pool[i] = &common.RawMessage{
			Prefix: w.prefix,
		}
	}
	var msgLen int
	for {
		err = conn.SetReadDeadline(time.Now().Add(time.Minute))
		if err != nil {
			logger.Debugf("conn.SetReadDeadline error %v", err)
			w.restart()
			return
		}
		_, r, err = conn.NextReader()
		if err != nil {
			logger.Warnf("conn.NextReader error %v", err)
			w.restart()
			return
		}
		msg, err = w.readAll(r)
		if err != nil {
			logger.Warnf("w.readAll error %v", err)
			w.restart()
			return
		}
		msgLen = len(msg)
		if msgLen > 72 && msg[2] == 'a' && msg[42] == '"' {
			//{"arg":{"channel":"funding-rate","instId":"DOGE-USDT-SWAP"},"data":[{"fundingRate":"0.00009327","fundingTime":"1636848000000","instId":"DOGE-USDT-SWAP","instType":"SWAP","nextFundingRate":"0.0003"}]}
			if msg[56] == '"' {
				symbol = common.UnsafeBytesToString(msg[43:56])
			} else if msg[55] == '"' {
				symbol = common.UnsafeBytesToString(msg[43:55])
			} else if msg[57] == '"' {
				symbol = common.UnsafeBytesToString(msg[43:57])
			} else if msg[58] == '"' {
				symbol = common.UnsafeBytesToString(msg[43:58])
			} else if msg[59] == '"' {
				symbol = common.UnsafeBytesToString(msg[43:59])
			} else if msg[60] == '"' {
				symbol = common.UnsafeBytesToString(msg[43:60])
			} else if msg[61] == '"' {
				symbol = common.UnsafeBytesToString(msg[43:61])
			} else if msg[62] == '"' {
				symbol = common.UnsafeBytesToString(msg[43:62])
			} else if msg[63] == '"' {
				symbol = common.UnsafeBytesToString(msg[43:63])
			} else if msg[64] == '"' {
				symbol = common.UnsafeBytesToString(msg[43:64])
			} else {
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("symbol not found for %s", msg)
					logSilentTime = time.Now().Add(time.Minute)
				}
				continue
			}
		} else if msgLen == 4 && msg[0] == 'p' {
			select {
			case w.pingCh <- msg:
			default:
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("w.pongCh <- msg failed %s ch len %d", symbol, len(w.pingCh))
					logSilentTime = time.Now().Add(time.Minute)
				}
			}
			continue
		} else {
			if time.Now().Sub(logSilentTime) > 0 {
				logger.Debugf("MSG %s", msg)
				logSilentTime = time.Now().Add(time.Minute)
			}
			continue
		}
		if ch, ok = channels[symbol]; ok {
			index++
			if index == bufferSize {
				index = 0
			}
			message = pool[index]
			message.Time = time.Now().UnixNano()
			message.Data = msg
			select {
			case ch <- message:
			default:
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf(" ch <- message %s ch len %d", symbol, len(ch))
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

func (w *RawFundingRateWS) readAll(r io.Reader) ([]byte, error) {
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

func (w *RawFundingRateWS) reconnect(ctx context.Context, wsUrl string, proxy string, counter int64) (*websocket.Conn, error) {

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
			Proxy:             http.ProxyURL(proxyUrl),
			HandshakeTimeout:  60 * time.Second,
			EnableCompression: true,
		}
	} else {
		dialer = &websocket.Dialer{
			HandshakeTimeout:  10 * time.Second,
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

func (w *RawFundingRateWS) mainLoop(ctx context.Context, proxy string, channels map[string]chan *common.RawMessage) {
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
			reconnectTimer.Reset(time.Second * 5)
		case <-reconnectTimer.C:
			if internalCancel != nil {
				internalCancel()
			}
			internalCtx, internalCancel = context.WithCancel(ctx)
			conn, err := w.reconnect(internalCtx, "wss://ws.okx.com:8443/ws/v5/public", proxy, 0)
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

func (w *RawFundingRateWS) heartbeatLoop(ctx context.Context, conn *websocket.Conn, symbols []string) {
	logger.Debugf("START heartbeatLoop")
	defer func() {
		logger.Debugf("Exit heartbeatLoop")
		err := conn.Close()
		if err != nil {
			logger.Debugf("conn.Close() ERROR %v", err)
		}
	}()
	symbolTimeout := time.Minute*5
	symbolCheckInterval := time.Second*5
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
			trafficTimeout.Reset(time.Minute*2)
			symbolUpdatedTimes[symbol] = time.Now()
			break
		case <-w.pingCh:
			//logger.Debugf("PING MSG")
			pingTimer.Reset(time.Second * 15)
			trafficTimeout.Reset(time.Minute*2)
			break
		case <-symbolCheckTimer.C:
			args := make([]WsArgs, 0)
			for symbol, updateTime := range symbolUpdatedTimes {
				if time.Now().Sub(updateTime) > symbolTimeout {
					args = append(args, WsArgs{
						Channel: "funding-rate",
						InstId:  symbol,
					})
					symbolUpdatedTimes[symbol] = time.Now().Add(symbolTimeout)
				}
			}
			if len(args) > 0 {
				logger.Debugf("SUB %s", args)
				for start := 0; start < len(args); start += 50 {
					end := start + 50
					if end > len(args) {
						end = len(args)
					}
					select {
					case w.writeCh <- WsSubUnsub{
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

func (w *RawFundingRateWS) Stop() {
	if atomic.CompareAndSwapInt32(&w.stopped, 0, 1) {
		close(w.done)
		logger.Debugf("stopped")
	}
}

func (w *RawFundingRateWS) restart() {
	select {
	case w.reconnectCh <- nil:
	default:
		logger.Debugf("w.reconnectCh <- nil failed, ch len %d", len(w.reconnectCh))
	}
}

func (w *RawFundingRateWS) Done() chan interface{} {
	return w.done
}

func (w *RawFundingRateWS) dataHandleLoop(ctx context.Context, symbol string, inputCh chan []byte, outputCh chan common.FundingRate) {
	logSilentTime := time.Now()
	const bufferLen = 4096
	var err error
	var fr *FundingRate
	index := -1
	pool := [bufferLen]*FundingRate{}
	for i := 0; i < bufferLen; i++ {
		pool[i] = &FundingRate{}
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case msg := <-inputCh:
			index++
			if index == bufferLen {
				index = 0
			}
			fr = pool[index]
			err = ParseFundingRate(msg, fr)
			if err != nil {
				logger.Debugf("%s ParseFundingRate error %v %s", symbol, err, msg)
				continue
			}
			select {
			case outputCh <- fr:
			default:
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("outputCh <- fr failed, %s ch len %d", symbol, len(outputCh))
					logSilentTime = time.Now().Add(time.Minute)
				}
			}
		}
	}
}

func NewRawFundingRateWS(
	ctx context.Context,
	proxy string,
	prefix []byte,
	channels map[string]chan *common.RawMessage,
) *RawFundingRateWS {
	ws := RawFundingRateWS{
		done:        make(chan interface{}),
		reconnectCh: make(chan interface{}, 16),
		writeCh:     make(chan interface{}, len(channels)*4),
		symbolCh:    make(chan string, len(channels)*16),
		pingCh:      make(chan []byte, 16),
		prefix:      prefix,
		stopped:     0,
	}
	go ws.mainLoop(ctx, proxy, channels)
	ws.reconnectCh <- nil
	return &ws
}
