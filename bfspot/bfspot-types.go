package bfspot

import (
	"fmt"
	"net/http"
)

type Response struct {
	Response *http.Response
	Body     []byte
}

type ErrorResponse struct {
	Response *Response
	Message  string `json:"message"`
	Code     int    `json:"code"`
}

func (r *ErrorResponse) Error() string {
	return fmt.Sprintf("%v %v: %d %v (%d)",
		r.Response.Response.Request.Method,
		r.Response.Response.Request.URL,
		r.Response.Response.StatusCode,
		r.Message,
		r.Code,
	)
}
