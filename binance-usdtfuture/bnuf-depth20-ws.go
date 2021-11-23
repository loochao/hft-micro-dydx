package binance_usdtfuture

import (
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/gorilla/websocket"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type Depth20WS struct {
	done        chan interface{}
	reconnectCh chan interface{}
	stopped     bool
	mu          sync.Mutex
}

func (w *Depth20WS) readLoop(conn *websocket.Conn, channels map[string]chan []byte) {
	logger.Debugf("START readLoop")
	defer logger.Debugf("EXIT readLoop")
	logSilentTime := time.Now()
	var ch chan []byte
	var ok bool
	var symbol string
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
		//{"stream":"btcusdt@depth20@100ms","data":{"e":"depthUpdate","E":1623494842640,"T":1623494842634,"s":"BTCUSDT","U":510756353509,"u":510756357084,"pu":510756353459,"b":[["35776.62","0.019"],["35776.34","0.049"],["35776.09","0.128"],["35776.08","0.559"],["35775.73","0.014"],["35775.36","0.006"],["35775.35","0.077"],["35775.34","0.089"],["35774.34","0.119"],["35774.10","0.015"],["35773.94","0.144"],["35773.93","0.092"],["35773.69","0.040"],["35773.55","0.015"],["35773.19","0.014"],["35773.16","0.761"],["35773.14","0.011"],["35773.08","0.140"],["35772.89","0.003"],["35772.88","0.056"]],"a":[["35778.12","0.250"],["35778.43","0.874"],["35778.49","2.100"],["35778.57","0.032"],["35779.00","0.997"],["35779.41","0.233"],["35779.42","0.001"],["35780.51","0.199"],["35780.95","0.001"],["35781.19","0.001"],["35781.20","0.162"],["35781.69","0.077"],["35781.70","0.015"],["35783.18","0.167"],["35783.19","0.432"],["35785.13","0.299"],["35787.14","0.115"],["35787.16","0.233"],["35788.69","0.003"],["35788.88","0.044"]]}}
		//{"stream":"linkusdt@depth20@100ms","data":{"e":"depthUpdate","E":1623494842601,"T":1623494842591,"s":"LINKUSDT","U":510756352078,"u":510756355671,"pu":510756351951,"b":[["21.038","0.76"],["21.033","2.38"],["21.032","241.49"],["21.031","11.47"],["21.030","6.00"],["21.029","8441.43"],["21.028","52.37"],["21.027","15.30"],["21.026","296.49"],["21.025","457.98"],["21.024","209.65"],["21.023","970.06"],["21.022","196.17"],["21.021","44.05"],["21.019","43.33"],["21.018","60.00"],["21.017","70.01"],["21.016","316.15"],["21.015","378.55"],["21.013","96.24"]],"a":[["21.045","184.00"],["21.047","49.81"],["21.048","261.43"],["21.049","24.53"],["21.050","257.44"],["21.051","216.80"],["21.052","72.61"],["21.053","319.43"],["21.054","597.00"],["21.056","11.88"],["21.057","87.96"],["21.059","276.63"],["21.060","217.37"],["21.061","371.19"],["21.062","467.53"],["21.063","720.86"],["21.064","306.14"],["21.065","144.71"],["21.066","166.19"],["21.067","157.86"]]}}
		//{"stream":"wavesusdt@depth20@100ms","data":{"e":"depthUpdate","E":1623494842673,"T":1623494842650,"s":"WAVESUSDT","U":510756354826,"u":510756357638,"pu":510756353771,"b":[["14.2940","21.7"],["14.2920","104.9"],["14.2910","26.9"],["14.2900","10.6"],["14.2890","266.8"],["14.2880","199.4"],["14.2870","190.7"],["14.2860","372.9"],["14.2850","187.9"],["14.2840","741.3"],["14.2830","47.2"],["14.2820","13.6"],["14.2800","77.3"],["14.2790","673.7"],["14.2780","328.6"],["14.2770","82.2"],["14.2760","15.7"],["14.2750","265.6"],["14.2740","694.7"],["14.2730","224.9"]],"a":[["14.2980","1.2"],["14.3020","12.0"],["14.3030","132.7"],["14.3040","85.0"],["14.3050","91.0"],["14.3060","65.1"],["14.3070","17.4"],["14.3090","125.9"],["14.3100","268.6"],["14.3110","410.8"],["14.3120","1005.2"],["14.3130","115.4"],["14.3140","87.4"],["14.3150","221.6"],["14.3160","5.0"],["14.3170","948.4"],["14.3180","694.1"],["14.3190","21.0"],["14.3200","144.9"],["14.3210","236.7"]]}}
		if len(msg) < 128 {
			continue
		}
		if msg[61] == 'E' {
			symbol = common.UnsafeBytesToString(msg[11:18])
		} else if msg[62] == 'E' {
			symbol = common.UnsafeBytesToString(msg[11:19])
		} else if msg[63] == 'E' {
			symbol = common.UnsafeBytesToString(msg[11:20])
		} else if msg[60] == 'E' {
			symbol = common.UnsafeBytesToString(msg[11:17])
		} else if msg[64] == 'E' {
			symbol = common.UnsafeBytesToString(msg[11:21])
		} else if msg[65] == 'E' {
			symbol = common.UnsafeBytesToString(msg[11:22])
		} else if msg[60] == 'E' {
			symbol = common.UnsafeBytesToString(msg[11:17])
		} else if msg[66] == 'E' {
			symbol = common.UnsafeBytesToString(msg[11:23])
		} else if msg[67] == 'E' {
			symbol = common.UnsafeBytesToString(msg[11:24])
		} else if msg[68] == 'E' {
			symbol = common.UnsafeBytesToString(msg[11:25])
		} else {
			if time.Now().Sub(logSilentTime) > 0 {
				logger.Debugf("bad msg, can't find symbol: %s", msg)
				logSilentTime = time.Now().Add(time.Minute)
			}
			continue
		}
		if ch, ok = channels[symbol]; ok {
			select {
			case ch <- msg:
			default:
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("ch <- msg failed %s len(ch) = %d", symbol, len(ch))
					logSilentTime = time.Now().Add(time.Minute)
				}
			}
		}
	}
}

func (w *Depth20WS) readAll(r io.Reader) ([]byte, error) {
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

func (w *Depth20WS) reconnect(ctx context.Context, wsUrl string, proxy string, counter int64) (*websocket.Conn, error) {

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

func (w *Depth20WS) mainLoop(ctx context.Context, proxy string, channels map[string]chan []byte) {
	urlStr := "wss://fstream.binance.com/stream?streams="
	symbols := make([]string, 0)
	for symbol := range channels {
		symbols = append(symbols, symbol)
		urlStr += fmt.Sprintf(
			"%s@depth20@100ms/",
			strings.ToLower(symbol),
		)
	}
	urlStr = urlStr[:len(urlStr)-1]
	logger.Debugf("START mainLoop %s", symbols)

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
			go w.readLoop(conn, channels)
			go w.heartbeatLoop(internalCtx, conn, symbols)

		}
	}
}

func (w *Depth20WS) heartbeatLoop(ctx context.Context, conn *websocket.Conn, symbols []string) {
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

func (w *Depth20WS) Stop() {
	w.mu.Lock()
	if !w.stopped {
		w.stopped = true
		close(w.done)
	}
	w.mu.Unlock()
}

func (w *Depth20WS) restart() {
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

func (w *Depth20WS) Done() chan interface{} {
	return w.done
}

func (w *Depth20WS) dataHandleLoop(ctx context.Context, symbol string, inputCh chan []byte, outputCh chan common.Depth) {
	logSilentTime := time.Now()
	var err error
	var msg []byte
	var depth20 *Depth20
	index := -1
	pool := [common.BufferSizeFor100msData]*Depth20{}
	for i := 0; i < common.BufferSizeFor100msData; i++ {
		pool[i] = &Depth20{}
	}

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
			depth20 = pool[index]
			err = ParseDepth20(msg, depth20)
			if err != nil {
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("ParseDepth20 error %v %s", err, msg)
					logSilentTime = time.Now().Add(time.Minute)
				}
				break
			}
			select {
			case outputCh <- depth20:
			default:
				if time.Now().Sub(logSilentTime) > 0 {
					logger.Debugf("ch <- depth20 failed ch len %d", len(outputCh))
					logSilentTime = time.Now().Add(time.Minute)
				}
			}
			break
		}
	}
}

func NewDepth20WS(
	ctx context.Context,
	proxy string,
	channels map[string]chan common.Depth,
) *Depth20WS {
	ws := Depth20WS{
		done:        make(chan interface{}),
		reconnectCh: make(chan interface{}),
		stopped:     false,
		mu:          sync.Mutex{},
	}
	messageChs := make(map[string]chan []byte)
	for symbol, ch := range channels {
		messageChs[strings.ToLower(symbol)] = make(chan []byte, common.ChannelSizeLowLoadLowLatency)
		go ws.dataHandleLoop(ctx, symbol, messageChs[strings.ToLower(symbol)], ch)
	}
	go ws.mainLoop(ctx, proxy, messageChs)
	ws.reconnectCh <- nil
	return &ws
}
