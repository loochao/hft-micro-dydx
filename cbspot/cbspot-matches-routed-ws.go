package cbspot

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

type MatchesRoutedWS struct {
	writeCh     chan interface{}
	done        chan interface{}
	reconnectCh chan interface{}
	messageCh   chan []byte
	productIDCh chan string
	stopped     int32
}

func (w *MatchesRoutedWS) writeLoop(ctx context.Context, conn *websocket.Conn) {
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

func (w *MatchesRoutedWS) readLoop(conn *websocket.Conn) {
	logger.Debugf("START readLoop")
	defer func() {
		logger.Debugf("EXIT readLoop")
	}()
	logSilentTime := time.Now()
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

		select {
		case w.messageCh <- msg:
		default:
			if time.Now().Sub(logSilentTime) > 0 {
				logger.Debugf("w.messageCh <- msg failed ch len %d", len(w.messageCh))
				logSilentTime = time.Now().Add(time.Minute)
			}
		}
	}
}

func (w *MatchesRoutedWS) readAll(r io.Reader) ([]byte, error) {
	b := make([]byte, 0, 256)
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

func (w *MatchesRoutedWS) reconnect(ctx context.Context, wsUrl string, proxy string, counter int64) (*websocket.Conn, error) {

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

func (w *MatchesRoutedWS) mainLoop(
	ctx context.Context,
	proxy string,
	channels map[string]chan common.Trade,
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
			conn, err := w.reconnect(internalCtx, "wss://ws-feed.pro.coinbase.com", proxy, 0)
			if err != nil {
				if internalCancel != nil {
					internalCancel()
				}
				logger.Debugf("w.reconnect error %v, stop ws", err)
				return
			}
			go w.readLoop(conn)
			go w.writeLoop(internalCtx, conn)
			go w.heartbeatLoop(internalCtx, conn, symbols, time.Second*15)
		}
	}
}

func (w *MatchesRoutedWS) heartbeatLoop(ctx context.Context, conn *websocket.Conn, symbols []string, pingInterval time.Duration) {

	logger.Debugf("START heartbeatLoop")

	defer func() {
		err := conn.Close()
		if err != nil {
			logger.Debugf("conn.Close() ERROR %v", err)
		}
	}()

	symbolTimeout := time.Minute
	symbolCheckInterval := time.Second
	//pingTimer := time.NewTimer(time.Second)
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
		case symbol := <-w.productIDCh:
			symbolUpdatedTimes[symbol] = time.Now()
		//case <-pingTimer.C:
		//	pingTimer.Reset(pingInterval)
		//	select {
		//	case <-ctx.Done():
		//		return
		//	case <-time.After(time.Millisecond):
		//		logger.Debug("send ping to writeCh timeout in 1ms")
		//	case w.writeCh <- Ping{
		//		ID:   fmt.Sprintf("%d", time.Now().Nanosecond()/1000000),
		//		Type: "ping",
		//	}:
		//	}
		//	break
		case <-symbolCheckTimer.C:
			productIds := make([]string, 0)
			for symbol, updateTime := range symbolUpdatedTimes {
				if time.Now().Sub(updateTime) > symbolTimeout {
					productIds = append(productIds, symbol)
					symbolUpdatedTimes[symbol] = time.Now().Add(symbolCheckInterval)
				}
			}
			if len(productIds) > 0 {
				logger.Debugf("SUBSCRIBE %s", productIds)
				select {
				case w.writeCh <- Request{
					Type: "subscribe",
					Channels: []Channel{
						{ProductIDs: productIds, Name: "matches"},
					},
				}:
				default:
					logger.Debugf("w.writeCh <- Request failed, ch len %d", len(w.writeCh))
				}
			}
			symbolCheckTimer.Reset(symbolCheckInterval)
			break
		}
	}

}

func (w *MatchesRoutedWS) Stop() {
	if atomic.LoadInt32(&w.stopped) == 0 {
		atomic.StoreInt32(&w.stopped, 1)
		close(w.done)
		logger.Infof("stopped")
	}
}

func (w *MatchesRoutedWS) restart() {
	select {
	case <-w.done:
		return
	case w.reconnectCh <- nil:
		logger.Debugf("restart")
	default:
		logger.Debugf("w.reconnectCh <- nil failed, ch len %d", len(w.reconnectCh))
	}
}

func (w *MatchesRoutedWS) dataHandleLoop(ctx context.Context, id int, channels map[string]chan common.Trade) {
	logger.Debugf("START dataHandleLoop %d", id)
	defer logger.Debugf("EXIT dataHandleLoop %d", id)
	logSilentTime := time.Now()
	var ch chan common.Trade
	var err error
	var ok bool
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case msg := <-w.messageCh:
			switch msg[9] {
			case 'm':
				var match Match
				err = json.Unmarshal(msg, &match)
				if err != nil {
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("json.Unmarshal(msg, &match) error %v %s", err, msg)
						logSilentTime = time.Now().Add(time.Minute)
					}
					continue
				}
				if ch, ok = channels[match.ProductId]; ok {
					select {
					case ch <- &match:
					default:
						if time.Now().Sub(logSilentTime) > 0 {
							logger.Debugf("ch <- &match failed ch len %d", len(ch))
							logSilentTime = time.Now().Add(time.Minute)
						}
					}
					select {
					case w.productIDCh <- match.ProductId:
					default:
					}
				}
			case 's':
				break
			case 'l':
				break
			default:
				logger.Debugf("other msg %s", msg)
			}
		}
	}
}

func (w *MatchesRoutedWS) Done() chan interface{} {
	return w.done
}

func NewMatchRoutedWS(
	ctx context.Context,
	proxy string,
	channels map[string]chan common.Trade,
) *MatchesRoutedWS {
	ws := MatchesRoutedWS{
		done:        make(chan interface{}),
		reconnectCh: make(chan interface{}),
		writeCh:     make(chan interface{}, 100*len(channels)),
		productIDCh: make(chan string, 100*len(channels)),
		messageCh:   make(chan []byte, 100*len(channels)),
		stopped:     0,
	}
	go ws.mainLoop(ctx, proxy, channels)
	for i := 0; i < 4; i++ {
		cs := make(map[string]chan common.Trade)
		for symbol, ch := range channels {
			cs[symbol] = ch
		}
		go ws.dataHandleLoop(ctx, i, cs)
	}
	ws.reconnectCh <- nil
	return &ws
}

