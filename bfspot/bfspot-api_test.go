package bfspot

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"testing"
)

func TestAPI_GetTradingParis(t *testing.T) {
	api, err := NewAPI(&common.Credentials{}, "socks5://127.0.0.1:1080")
	if err != nil {
		t.Fatal(err)
	}
	_, err = api.GetTradingParis(context.Background())
	if err != nil {
		t.Fatal(err)
	}
}
