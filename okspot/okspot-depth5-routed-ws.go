package okspot

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/gorilla/websocket"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"
)

type Depth5RoutedWebsocket struct {
	writeCh     chan interface{}
	done        chan interface{}
	reconnectCh chan interface{}
	symbolCh    chan string
	pingCh      chan []byte
	stopped     int32
}

func (w *Depth5RoutedWebsocket) writeLoop(ctx context.Context, conn *websocket.Conn) {
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
					logger.Debugf("json.Marshal(msg) err %v", err)
					continue
				}
			}
			logger.Debugf("%s", bytes)
			err = conn.SetWriteDeadline(time.Now().Add(time.Minute))
			if err != nil {
				logger.Debugf("conn.SetWriteDeadline error %v", err)
				w.restart()
				return
			}

			err = conn.WriteMessage(websocket.TextMessage, bytes)
			if err != nil {
				logger.Debugf("conn.WriteMessage error %v", err)
				w.restart()
				return
			}
		}
	}
}
func (w *Depth5RoutedWebsocket) parseBinaryResponse(resp []byte) ([]byte, error) {
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

func (w *Depth5RoutedWebsocket) readLoop(conn *websocket.Conn, channels map[string]chan *common.DepthRawMessage) {
	logger.Debugf("START readLoop")
	defer logger.Debugf("EXIT readLoop")
	totalCount := 0
	totalLen := 0
	logSilentTime := time.Now()
	var symbolBytes []byte
	var symbol string
	var timeBytes []byte
	var eventTime time.Time
	var ch chan *common.DepthRawMessage
	var ok bool
	var msg []byte
	var msgLen int
	var mType int
	var resp []byte
	var err error
	var subscribeEvent SubscribeEvent
	var segs []string
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
		totalCount += 1
		totalLen += msgLen
		if totalCount > 1000000 {
			logger.Debugf("AVERAGE MESSAGE LENGTH %d/%d = %d", totalLen, totalCount, totalLen/totalCount)
			totalLen = 0
			totalCount = 0
		}
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
		} else if msg[2] == 't' && len(msg) > 128{
			//{"table":"spot/depth5","data":[{"asks":[["31.605","4.32464","1"],["31.607","85","1"],["31.61","2","1"],["31.612","0.1","1"],["31.614","1.405511","1"]],"bids":[["31.583","302.09312","3"],["31.582","0.9","1"],["31.58","111.30127","1"],["31.579","76","1"],["31.576","31.83446","1"]],"instrument_id":"LINK-USDT","timestamp":"2021-04-25T08:24:33.352Z"}]}
			timeBytes = msg[msgLen-28:msgLen-4]
			eventTime, err = time.Parse(okspotTimeLayout, *(*string)(unsafe.Pointer(&timeBytes)))
			if err != nil {
				logger.Debugf("time.Parse %s error %v", timeBytes, err)
				continue
			}
			if msg[msgLen-53] == ':' {
				symbolBytes = msg[msgLen-51:msgLen-43]
				symbol = *(*string)(unsafe.Pointer(&symbolBytes))
			}else if msg[msgLen-54] == ':' {
				symbolBytes = msg[msgLen-52:msgLen-43]
				symbol = *(*string)(unsafe.Pointer(&symbolBytes))
			}else if msg[msgLen-55] == ':' {
				symbolBytes = msg[msgLen-53:msgLen-43]
				symbol = *(*string)(unsafe.Pointer(&symbolBytes))
			}else if msg[msgLen-56] == ':' {
				symbolBytes = msg[msgLen-54:msgLen-43]
				symbol = *(*string)(unsafe.Pointer(&symbolBytes))
			}else{
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("other msg %s", msg)
					logSilentTime = time.Now().Add(time.Minute)
				}
				continue
			}
		}else {
			if time.Now().Sub(logSilentTime) > 0 {
				logger.Debugf("other msg %s", msg)
				logSilentTime = time.Now().Add(time.Minute)
			}
			continue
		}
		//logger.Debugf("%s %v ",symbol, eventTime)
		if ch, ok = channels[symbol]; ok {
			select {
			case ch <- &common.DepthRawMessage{
				Depth:  msg,
				Symbol: symbol,
				Time:   eventTime,
			}:
			default:
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("ch <- &common.DepthRawMessage failed %s ch len %d", symbol, len(ch))
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

		////{"ch":"market.btcusdt.depth.step1","ts":1618475611868,"tick":{"bids":[[62753.2,2.140127],[62753.1,1.005768],[62751.4,0.01],[62750.2,1.588851],[62750.1,0.132173],[62747.1,1.04731],[62747.0,1.357035],[62746.0,0.001031],[62744.8,0.44207],[62744.6,0.064435],[62743.0,0.051222],[62741.8,8.0E-4],[62739.5,0.450211],[62739.0,0.026874],[62737.6,0.2],[62737.0,0.001401],[62736.9,0.1],[62736.7,0.047803],[62735.4,1.6E-4],[62733.6,0.135775]],"asks":[[62753.3,0.038953],[62754.3,0.03],[62758.0,0.09781],[62758.8,0.045154],[62759.5,0.01],[62760.8,0.134133],[62761.8,0.03],[62761.9,0.132173],[62763.4,8.95E-4],[62763.8,0.123199],[62764.2,0.06],[62765.0,0.010786],[62765.8,0.002],[62766.0,0.162855],[62767.2,0.001596],[62767.6,1.145],[62768.9,2.733775],[62769.1,0.159325],[62770.7,0.017235],[62771.5,0.04]],"version":125019588599,"ts":1618475611865}}
		//if msg[2] == 'c' && len(msg) > 128 {
		//	if msg[39] == ':' {
		//		timeInt, err = common.ParseInt(msg[40:53])
		//		if err != nil {
		//			logger.Debugf("common.ParseInt error %v %s", err, msg[40:53])
		//			continue
		//		}
		//		symbolBytes = msg[14:21]
		//		symbol = *(*string)(unsafe.Pointer(&symbolBytes))
		//	} else if msg[40] == ':' {
		//		timeInt, err = common.ParseInt(msg[41:54])
		//		if err != nil {
		//			logger.Debugf("common.ParseInt error %v %s", err, msg[41:54])
		//			continue
		//		}
		//		symbolBytes = msg[14:22]
		//		symbol = *(*string)(unsafe.Pointer(&symbolBytes))
		//	} else if msg[41] == ':' {
		//		timeInt, err = common.ParseInt(msg[42:55])
		//		if err != nil {
		//			logger.Debugf("common.ParseInt error %v %s", err, msg[42:55])
		//			continue
		//		}
		//		symbolBytes = msg[14:23]
		//		symbol = *(*string)(unsafe.Pointer(&symbolBytes))
		//	} else if msg[42] == ':' {
		//		timeInt, err = common.ParseInt(msg[43:56])
		//		if err != nil {
		//			logger.Debugf("common.ParseInt error %v %s", err, msg[43:56])
		//			continue
		//		}
		//		symbolBytes = msg[14:24]
		//		symbol = *(*string)(unsafe.Pointer(&symbolBytes))
		//	} else {
		//		if time.Now().Sub(logSilentTime) > 0 {
		//			logger.Debugf("bad msg, can't find timestamp: %s", msg)
		//			logSilentTime = time.Now().Add(time.Minute)
		//		}
		//	}
		//	if ch, ok = channels[symbol]; ok {
		//		select {
		//		case ch <- &common.DepthRawMessage{
		//			Depth:  msg,
		//			Symbol: symbol,
		//			Time:   time.Unix(0, timeInt*1000000),
		//		}:
		//		default:
		//			if time.Now().Sub(logSilentTime) > 0 {
		//				logger.Debugf("ch <- &common.DepthRawMessage failed %s ch len %d", symbol, len(ch))
		//				logSilentTime = time.Now().Add(time.Minute)
		//			}
		//		}
		//		select {
		//		case w.symbolCh <- symbol:
		//		default:
		//			if time.Now().Sub(logSilentTime) > 0 {
		//				logger.Debugf("w.symbolCh <- symbol failed %s ch len %d", symbol, len(w.symbolCh))
		//				logSilentTime = time.Now().Add(time.Minute)
		//			}
		//		}
		//	}
		//} else if msg[2] == 'p' {
		//	msg[3] = 'o'
		//	select {
		//	case w.pingCh <- msg:
		//	default:
		//		if time.Now().Sub(logSilentTime) > 0 {
		//			logger.Debugf("w.pingCh <- msg failed %s ch len %d", symbol, len(w.pingCh))
		//			logSilentTime = time.Now().Add(time.Minute)
		//		}
		//	}
		//}

		//err = gr.Close()
		//if err != nil {
		//	logger.Debugf("gr.Close() error %v", err)
		//	go w.restart()
		//	return
		//}
	}
}

func (w *Depth5RoutedWebsocket) readAll(r io.Reader) ([]byte, error) {
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

func (w *Depth5RoutedWebsocket) reconnect(ctx context.Context, wsUrl string, proxy string, counter int64) (*websocket.Conn, error) {

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

func (w *Depth5RoutedWebsocket) mainLoop(ctx context.Context, proxy string, channels map[string]chan *common.DepthRawMessage) {
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

func (w *Depth5RoutedWebsocket) heartbeatLoop(ctx context.Context, conn *websocket.Conn, symbols []string) {
	logger.Debugf("START heartbeatLoop")
	defer func() {
		logger.Debugf("Exit heartbeatLoop")
		err := conn.Close()
		if err != nil {
			logger.Debugf("conn.Close() ERROR %v", err)
		}
	}()

	conn.SetPingHandler(func(msg string) error {
		logger.Debugf("get ping msg %s", msg)
		err := conn.WriteControl(websocket.PongMessage, []byte(msg), time.Now().Add(time.Second*10))
		if err != nil {
			w.restart()
			return nil
		}
		return nil
	})

	symbolTimeout := time.Minute
	symbolCheckInterval := time.Second
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
		case msg := <-w.pingCh:
			select {
			case w.writeCh <- msg:
				break
			default:
				logger.Debugf("w.writeCh <- msg failed, ch len %d", len(w.writeCh))
			}
			break
		case <-symbolCheckTimer.C:
			args := make([]string, 0)
			for symbol, updateTime := range symbolUpdatedTimes {
				if time.Now().Sub(updateTime) > symbolTimeout {
					args = append(args, fmt.Sprintf("spot/depth5:%s", symbol))
					symbolUpdatedTimes[symbol] = time.Now().Add(symbolCheckInterval)
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
					case w.writeCh <- Subscription{
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

func (w *Depth5RoutedWebsocket) Stop() {
	if atomic.LoadInt32(&w.stopped) == 0 {
		atomic.StoreInt32(&w.stopped, 1)
		close(w.done)
		logger.Debugf("stopped")
	}
}

func (w *Depth5RoutedWebsocket) restart() {
	select {
	case w.reconnectCh <- nil:
	default:
		logger.Debugf("w.reconnectCh <- nil failed, ch len %d", len(w.reconnectCh))
	}
}

func (w *Depth5RoutedWebsocket) Done() chan interface{} {
	return w.done
}

func NewDepth5RoutedWebsocket(
	ctx context.Context,
	proxy string,
	channels map[string]chan *common.DepthRawMessage,
) *Depth5RoutedWebsocket {
	ws := Depth5RoutedWebsocket{
		done:        make(chan interface{}),
		reconnectCh: make(chan interface{}, 100),
		writeCh:     make(chan interface{}, 100*len(channels)),
		symbolCh:    make(chan string, 100*len(channels)),
		pingCh:      make(chan []byte, 100),
		stopped:     0,
	}
	go ws.mainLoop(ctx, proxy, channels)
	ws.reconnectCh <- nil
	return &ws
}
