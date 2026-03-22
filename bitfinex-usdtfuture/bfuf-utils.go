package bitfinex_usdtfuture

import (
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

// checkResponse checks response status code and response
// for errors.
func checkResponse(r *Response) error {
	if c := r.Response.StatusCode; c >= 200 && c <= 299 {
		return nil
	}

	var raw []interface{}
	// Try to decode error message
	errorResponse := &ErrorResponse{Response: r}
	err := json.Unmarshal(r.Body, &raw)
	if err != nil {
		errorResponse.Message = "Error decoding response error message. " +
			"Please see response body for more information."
		return errorResponse
	}

	if len(raw) < 3 {
		errorResponse.Message = fmt.Sprintf("Expected response to have three elements but got %#v", raw)
		return errorResponse
	}

	if str, ok := raw[0].(string); !ok || str != "error" {
		errorResponse.Message = fmt.Sprintf("Expected first element to be \"error\" but got %#v", raw)
		return errorResponse
	}

	code, ok := raw[1].(float64)
	if !ok {
		errorResponse.Message = fmt.Sprintf("Expected second element to be error code but got %#v", raw)
		return errorResponse
	}
	errorResponse.Code = int(code)

	msg, ok := raw[2].(string)
	if !ok {
		errorResponse.Message = fmt.Sprintf("Expected third element to be error message but got %#v", raw)
		return errorResponse
	}
	errorResponse.Message = msg

	return errorResponse
}

// newResponse creates new wrapper.
func newResponse(r *http.Response) *Response {
	// Use a LimitReader of arbitrary size (here ~8.39MB) to prevent us from
	// reading overly large response bodies.
	lr := io.LimitReader(r.Body, 8388608)
	body, err := ioutil.ReadAll(lr)
	if err != nil {
		body = []byte(`Error reading body:` + err.Error())
	}

	return &Response{r, body}
}

//[[PAIR,[PLACEHOLDER,PLACEHOLDER,PLACEHOLDER,MIN_ORDER_SIZE,MAX_ORDER_SIZE,PLACEHOLDER,PLACEHOLDER,PLACEHOLDER,INITIAL_MARGIN,MIN_MARGIN]]...]
func ParsePairs(msg []byte) ([]Pair, error) {
	pSegs := strings.Split(string(msg), "],[")
	pairs := make([]Pair, 0)
	var err error
	for _, seg := range pSegs {
		seg = strings.Replace(seg, "[", "", -1)
		seg = strings.Replace(seg, "]", "", -1)
		segs := strings.Split(seg, ",")
		pair := Pair{}
		if len(segs) != 11 {
			continue
		}
		pair.Symbol = segs[0][1:len(segs[0])-1]
		if len(segs[4]) <= 2 {
			continue
		}
		pair.MinOrderSize, err = common.ParseDecimal([]byte(segs[4][2 : len(segs[4])-1]))
		if err != nil {
			return nil, err
		}
		if len(segs[5]) <= 2 {
			continue
		}
		pair.MaxOrderSize, err = common.ParseDecimal([]byte(segs[5][2 : len(segs[5])-1]))
		if err != nil {
			return nil, err
		}

		pair.initialMargin, err = common.ParseDecimal([]byte(segs[9]))
		if err != nil {
			return nil, err
		}
		pair.minMargin, err = common.ParseDecimal([]byte(segs[10]))
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, pair)
	}
	return pairs, nil
}
