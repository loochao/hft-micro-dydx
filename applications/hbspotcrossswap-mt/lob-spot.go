package main

import (
	"context"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/hbspot"
)

func makerDepthWSLoop(
	ctx context.Context,
	cancel context.CancelFunc,
	proxyAddress string,
	channels map[string]chan *common.DepthRawMessage,
) {
	ws := hbspot.NewDepth20RoutedWebsocket(ctx,  proxyAddress, channels)
	defer ws.Stop()
	for {
		select {
		case <-ws.Done():
			cancel()
			return
		case <-ctx.Done():
			return
		}
	}
}

