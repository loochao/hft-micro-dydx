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

type TickerWS struct {
	writeCh       chan interface{}
	done          chan interface{}
	reconnectCh   chan interface{}
	marketCh      chan string
	marketResetCh chan string
	stopped       int32
}

func (w *TickerWS) writeLoop(ctx context.Context, conn *websocket.Conn) {
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
			//logger.Debugf("%s", msgBytes)
			err = conn.WriteMessage(websocket.TextMessage, msgBytes)
			if err != nil {
				logger.Debugf("conn.WriteMessage error %v", err)
				w.restart()
				return
			}
		}
	}
}

func (w *TickerWS) findMarket(msg []byte) (string, error) {
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

func (w *TickerWS) readLoop(conn *websocket.Conn, channels map[string]chan []byte) {
	logger.Debugf("START readLoop")
	defer logger.Debugf("EXIT readLoop")
	logSilentTime := time.Now()
	var market string
	var ch chan []byte
	var ok bool
	var msgLen int
	var readPool = [depthReadPoolSize][]byte{}
	var readIndex = -1
	var msg []byte
	var n int
	var r io.Reader
	var err error
	for i := range readPool {
		readPool[i] = make([]byte, depthReadMsgSize)
	}
	//readCounter := 0
	//partialReadCounter := 0
	//allocateCounter := 0
mainLoop:
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
		readIndex += 1
		if readIndex == depthReadPoolSize {
			readIndex = 0
		}
		msg = readPool[readIndex]
		n, err = r.Read(msg)
		if err == nil {
			//readCounter++
			msg = msg[:n]
			if n < 2 || msg[n-1] != '}' || msg[n-2] != '}' {
				//partialReadCounter++
			readLoop:
				for {
					if len(msg) == cap(msg) {
						// Add more capacity (let append pick how much).
						msg = append(msg, 0)[:len(msg)]
						//allocateCounter++
						if msg[9] != 's' || msg[18] != 'd' {
							logger.Debugf("BAD BUFFER SIZE CAN'T READ INTO %d, MSG: %s", depthReadMsgSize, msg)
						}
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

		//if readCounter%1000000 == 0 {
		//	logger.Debugf("DYDX DEPTH TICKER READ SIZE %d TOTAL READ %d PARTIAL READ %d EXPAND ALLOCATE %d", depthReadMsgSize, readCounter, partialReadCounter, allocateCounter)
		//}

		msgLen = len(msg)
		if msgLen > 128 {
			market, err = w.findMarket(msg)
			if err != nil {
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("find market failed %s", msg)
					logSilentTime = time.Now().Add(time.Minute)
				}
				continue
			}
			//} else {
			//	if time.Now().Sub(logSilentTime) > 0 && msgLen > 128 {
			//		logger.Debugf("other msg %s", msg)
			//		logSilentTime = time.Now().Add(time.Minute)
			//	}
			//	continue
		}
		if ch, ok = channels[market]; ok {
			select {
			case ch <- msg:
			default:
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf(" ch <- msg %s ch len %d", market, len(ch))
					logSilentTime = time.Now().Add(time.Minute)
				}
			}
		}
	}
}

//func (w *TickerWS) readAll(r io.Reader) ([]byte, error) {
//	b := make([]byte, 0, 1024)
//	for {
//		if len(b) == cap(b) {
//			// Add more capacity (let append pick how much).
//			b = append(b, 0)[:len(b)]
//		}
//		n, err := r.Read(b[len(b):cap(b)])
//		b = b[:len(b)+n]
//		if err != nil {
//			if err == io.EOF {
//				err = nil
//			}
//			return b, err
//		}
//	}
//}

func (w *TickerWS) reconnect(ctx context.Context, wsUrl string, proxy string, counter int64) (*websocket.Conn, error) {

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

func (w *TickerWS) mainLoop(ctx context.Context, proxy string, channels map[string]chan []byte) {
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
			conn, err := w.reconnect(internalCtx, "wss://api.dydx.exchange/v3/ws", proxy, 0)
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

func (w *TickerWS) heartbeatLoop(ctx context.Context, conn *websocket.Conn, markets []string) {
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
	marketResetInterval := time.Minute * 30
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
		case market := <-w.marketResetCh:
			marketResetTimes[market] = time.Now()
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

func (w *TickerWS) Stop() {
	if atomic.LoadInt32(&w.stopped) == 0 {
		atomic.StoreInt32(&w.stopped, 1)
		close(w.done)
		logger.Debugf("stopped")
	}
}

func (w *TickerWS) restart() {
	select {
	case w.reconnectCh <- nil:
	default:
		logger.Debugf("w.reconnectCh <- nil failed, ch len %d", len(w.reconnectCh))
	}
}

func (w *TickerWS) Done() chan interface{} {
	return w.done
}

func (w *TickerWS) dataHandleLoop(ctx context.Context, market string, inputCh chan []byte, outputCh chan common.Ticker) {
	logger.Debugf("START dataHandleLoop %s", market)
	defer logger.Debugf("EXIT dataHandleLoop %s", market)
	logSilentTime := time.Now()
	var err error
	hour999 := time.Hour * 999
	resubDelay := time.Second * 15
	resubTimer := time.NewTimer(hour999)

	var depth = &Depth{}
	var outputDepth *Depth
	msgCounter := 0

	index := -1
	pool := [common.BufferSizeForRealTimeData]*Depth{}
	for i := 0; i < common.BufferSizeForRealTimeData; i++ {
		pool[i] = &Depth{}
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case <-resubTimer.C:
			select {
			case w.marketResetCh <- market:
			default:
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("w.marketResetCh <- market failed, ch len %d", len(w.marketResetCh))
					logSilentTime = time.Now().Add(time.Minute)
				}
			}
			logger.Debugf("%s RESUB SENT", market)
			resubTimer.Reset(hour999)
			continue
		case msg := <-inputCh:
			if msg[9] == 's' && msg[18] == 'd' {
				err = ParseDepth(msg, depth)
				if err != nil {
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("ParseDepth(msg, depth) error %v", err)
						logSilentTime = time.Now().Add(time.Minute)
					}
					continue
				}
				msgCounter = 0
			} else if msg[9] == 'c' || msg[20] == 'a' {
				err = UpdateDepth(msg, depth)
				if err != nil {
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("UpdateDepth(msg, depth) error %v", err)
						logSilentTime = time.Now().Add(time.Minute)
					}
					continue
				}
			} else {
				//if time.Now().Sub(logSilentTime) > 0 {
				//	logger.Debugf("other msg %s", msg)
				//	logSilentTime = time.Now().Add(time.Minute)
				//}
				continue
			}

			if !depth.IsValid() {
				if msgCounter > 0 {
					resubTimer.Reset(resubDelay)
					msgCounter = -4
					if len(depth.Bids) > 0 {
						depth.Bids = depth.Bids.Update([2]float64{depth.Bids[0][0], 0.0})
					}
					if len(depth.Asks) > 0 {
						depth.Asks = depth.Asks.Update([2]float64{depth.Asks[0][0], 0.0})
					}
				} else {
					msgCounter -= 4
					if len(depth.Bids) > 0 {
						depth.Bids = depth.Bids.Update([2]float64{depth.Bids[0][0], 0.0})
					}
					if len(depth.Asks) > 0 {
						depth.Asks = depth.Asks.Update([2]float64{depth.Asks[0][0], 0.0})
					}
				}
			} else {
				msgCounter++

				if msgCounter >= 0 {
					index++
					if index == common.BufferSizeForRealTimeData {
						index = 0
					}
					outputDepth = pool[index]
					*outputDepth = *depth
					select {
					case outputCh <- outputDepth:
					default:
						if time.Now().Sub(logSilentTime) > 0 {
							logger.Debugf("outputCh <- outputDepth failed, ch len %d", len(outputCh))
							logSilentTime = time.Now().Add(time.Minute)
						}
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
				resubTimer.Reset(hour999)
			}
			break
		}
	}
}

func NewTickerWS(
	ctx context.Context,
	proxy string,
	channels map[string]chan common.Ticker,
) *TickerWS {
	ws := TickerWS{
		done:          make(chan interface{}),
		reconnectCh:   make(chan interface{}, common.ChannelSizeLowLoad),
		writeCh:       make(chan interface{}, len(channels)*common.ChannelSizeLowLoad),
		marketCh:      make(chan string, len(channels)*common.ChannelSizeLowLoad),
		marketResetCh: make(chan string, len(channels)*common.ChannelSizeLowLoad),
		stopped:       0,
	}
	messagesCh := make(map[string]chan []byte)
	for market, ch := range channels {
		messagesCh[market] = make(chan []byte, common.ChannelSizeLowDropRatio)
		go ws.dataHandleLoop(ctx, market, messagesCh[market], ch)
	}
	go ws.mainLoop(ctx, proxy, messagesCh)
	ws.reconnectCh <- nil
	return &ws
}
