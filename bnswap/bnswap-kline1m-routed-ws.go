package bnswap

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
	"strings"
	"sync/atomic"
	"time"
)

type Kline1MRoutedWebsocket struct {
	done        chan interface{}
	reconnectCh chan interface{}
	api         *API
	stopped     int32
	messageCh   chan []byte
}

func (w *Kline1MRoutedWebsocket) readLoop(conn *websocket.Conn, symbols []string) {
	logger.Debugf("START readLoop %s", symbols)
	defer logger.Debugf("EXIT readLoop %s", symbols)
	logSilentTime := time.Now()
	for {
		err := conn.SetReadDeadline(time.Now().Add(time.Minute))
		if err != nil {
			logger.Warnf("conn.SetReadDeadline error %v", err)
			w.restart()
			return
		}
		_, r, err := conn.NextReader()
		if err != nil {
			logger.Warnf("conn.NextReader error %v", err)
			w.restart()
			return
		}
		msg, err := w.readAll(r)
		if err != nil {
			logger.Warnf("w.readAl error %v", err)
			w.restart()
			return
		}
		select {
		case w.messageCh <- msg:
		default:
			if time.Now().Sub(logSilentTime) > 0 {
				logger.Debugf("w.messageCh <- msg timeout, ch len %d", len(w.messageCh))
				logSilentTime = time.Now().Add(time.Minute)
			}
		}
	}
}

func (w *Kline1MRoutedWebsocket) readAll(r io.Reader) ([]byte, error) {
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

func (w *Kline1MRoutedWebsocket) reconnect(ctx context.Context, wsUrl string, proxy string, counter int64) (*websocket.Conn, error) {

	if counter != 0 {
		logger.Debugf("RECONNECT %s, %d RETRIES", wsUrl, counter)
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
		logger.Warnf("dialer.DialContext ERROR %v", err)
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

func (w *Kline1MRoutedWebsocket) mainLoop(ctx context.Context, proxy string, channels map[string]chan common.KLine) {
	urlStr := "wss://fstream.binance.com/stream?streams="
	symbols := make([]string, 0)
	for symbol := range channels {
		symbols = append(symbols, symbol)
		urlStr += fmt.Sprintf(
			"%s@kline_1m/",
			strings.ToLower(symbol),
		)
	}
	urlStr = urlStr[:len(urlStr)-1]
	logger.Debugf("START mainLoop")

	ctx, cancel := context.WithCancel(ctx)
	var internalCtx context.Context
	var internalCancel context.CancelFunc

	defer func() {
		w.Stop()
		cancel()
		logger.Debugf("EXIT mainLoop %s", symbols)
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
				internalCancel = nil
			}
			internalCtx, internalCancel = context.WithCancel(ctx)
			conn, err := w.reconnect(internalCtx, urlStr, proxy, 0)
			if err != nil {
				logger.Debugf("w.reconnect error %v, stop ws", err)
				if internalCancel != nil {
					internalCancel()
					internalCancel = nil
				}
				w.Stop()
				return
			}
			go w.readLoop(conn, symbols)
			go w.heartbeatLoop(internalCtx, conn, symbols)

		}
	}
}

func (w *Kline1MRoutedWebsocket) heartbeatLoop(ctx context.Context, conn *websocket.Conn, symbols []string) {
	logger.Debugf("START heartbeatLoop %s", symbols)
	defer func() {
		logger.Debugf("EXIT heartbeatLoop %s", symbols)
		err := conn.Close()
		if err != nil {
			logger.Debugf("conn.Close() error %v", err)
		}
	}()

	trafficCh := make(chan interface{})

	conn.SetPingHandler(func(msg string) error {
		select {
		case trafficCh <- nil:
		default:
		}
		err := conn.WriteControl(websocket.PongMessage, []byte(msg), time.Now().Add(time.Minute))
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				w.restart()
			}
			return nil
		}
		return nil
	})

	timer := time.NewTimer(time.Minute * 15)

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case <-timer.C:
			logger.Warnf("no traffic in %v, restart ws", time.Minute*15)
			w.restart()
			return
		case <-trafficCh:
			timer.Reset(time.Minute * 15)
		}
	}

}

func (w *Kline1MRoutedWebsocket) Stop() {
	if atomic.LoadInt32(&w.stopped) == 0 {
		atomic.StoreInt32(&w.stopped, 1)
		close(w.done)
		logger.Debugf("stopped")
	}
}

func (w *Kline1MRoutedWebsocket) restart() {
	select {
	case <-w.done:
		return
	default:
	}
	select {
	case <-time.After(time.Second):
		w.Stop()
		logger.Debugf("w.reconnectCh <- nil timeout in 1s, stop ws")
	case w.reconnectCh <- nil:
		logger.Debugf("restart ws")
		return
	}
}

func (w *Kline1MRoutedWebsocket) dataHandleLoop(ctx context.Context, id int, channels map[string]chan common.KLine) {
	logger.Debugf("START dataHandleLoop")
	defer logger.Debugf("EXIT dataHandleLoop %d", id)
	wsk := WSKline{}
	var err error
	logSilentTime := time.Now()
	var ch chan common.KLine
	var ok bool
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case msg := <-w.messageCh:
			err = json.Unmarshal(msg, &wsk)
			if err != nil {
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("json.Unmarshal(msg, &wsk) error %v", err)
					logSilentTime = time.Now().Add(time.Minute)
				}
				continue
			}

			if ch, ok = channels[wsk.Data.K.Symbol]; ok && wsk.Data.K.Closed {
				select {
				case ch <- common.KLine{
					Close:     wsk.Data.K.Close,
					Open:      wsk.Data.K.Open,
					High:      wsk.Data.K.High,
					Low:       wsk.Data.K.Low,
					Volume:    wsk.Data.K.Volume,
					Timestamp: time.Unix(0, wsk.Data.K.CloseTime*1000000+1000000),
				}:
				default:
					if time.Now().Sub(logSilentTime) > 0 {
						logger.Debugf("ch <- common.KLine failed, ch len %d", len(ch))
						logSilentTime = time.Now().Add(time.Minute)
					}
				}
			}
		}
	}
}

func (w *Kline1MRoutedWebsocket) Done() chan interface{} {
	return w.done
}

func NewKline1MRoutedWebsocket(
	ctx context.Context,
	proxy string,
	channels map[string]chan common.KLine,
) *Kline1MRoutedWebsocket {
	ws := Kline1MRoutedWebsocket{
		done:        make(chan interface{}),
		reconnectCh: make(chan interface{}),
		stopped:     0,
		messageCh:   make(chan []byte, 1000),
	}
	go ws.mainLoop(ctx, proxy, channels)
	go ws.dataHandleLoop(ctx, 1, channels)
	go ws.dataHandleLoop(ctx, 2, channels)
	go ws.dataHandleLoop(ctx, 3, channels)
	go ws.dataHandleLoop(ctx, 4, channels)
	ws.reconnectCh <- nil
	return &ws
}

//{
//  "e": "kline",     // Event type
//  "E": 123456789,   // Event time
//  "s": "BTCUSDT",    // Market
//  "k": {
//    "t": 123400000, // Kline mainLoop time
//    "T": 123460000, // Kline close time
//    "s": "BTCUSDT",  // Market
//    "i": "1m",      // Interval
//    "f": 100,       // First trade ID
//    "L": 200,       // Last trade ID
//    "o": "0.0010",  // Open price
//    "c": "0.0020",  // Close price
//    "h": "0.0025",  // High price
//    "l": "0.0015",  // Low price
//    "v": "1000",    // Base asset volume
//    "n": 100,       // Number of trades
//    "x": false,     // Is this kline closed?
//    "q": "1.0000",  // Quote asset volume
//    "V": "500",     // Taker buy base asset volume
//    "Q": "0.500",   // Taker buy quote asset volume
//    "B": "123456"   // Ignore
//  }
//}

type WSKline struct {
	Data struct {
		K struct {
			LastTradeId int64 `json:"L"`
			StartTime int64   `json:"t"`
			CloseTime int64   `json:"T"`
			Symbol    string  `json:"s"`
			Open      float64 `json:"o,string"`
			High      float64 `json:"h,string"`
			Low       float64 `json:"l,string"`
			Close     float64 `json:"c,string"`
			Volume    float64 `json:"v,string"`
			Closed    bool    `json:"x"`
		} `json:"k"`
	} `json:"data"`
}
