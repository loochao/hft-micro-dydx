package main

import (
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestProcess(t *testing.T) {
	proxyUrl, err := url.Parse("socks5://127.0.0.1:10800")
	if err != nil {
		t.Fatal(err)
	}
	client := http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			Proxy:                 http.ProxyURL(proxyUrl),
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   60 * time.Second,
			ExpectContinueTimeout: 10 * time.Second,
			DialContext: (&net.Dialer{
				Timeout:   60 * time.Second,
				KeepAlive: 90 * time.Second,
			}).DialContext,
		},
	}
	req, err := http.NewRequest(http.MethodGet, "http://baidu.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		_, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		time.Sleep(time.Second)
	}
}
