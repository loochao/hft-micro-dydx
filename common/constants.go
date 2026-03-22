package common

import "time"

const (
	DefaultBidAskOffset = 0.001

	BufferSizeForHighLoadRealTimeData = 4096
	BufferSizeForRealTimeData         = 2048
	BufferSizeForHighLoad1msData      = 1024
	BufferSizeFor1msData              = 512
	BufferSizeFor10msData             = 256
	BufferSizeFor50msData             = 128
	BufferSizeFor100msData            = 64
	BufferSizeFor1sData               = 32
	BufferSizeFor10sData              = 16
	BufferSizeFor30sData              = 8
	BufferSizeFor60sData              = 4

	ChannelSizeLowLoadLowLatency    = 4
	ChannelSizeMediumLoadLowLatency = 8
	ChannelSizeHighLoadLowLatency   = 16
	ChannelSizeLowLoad              = 32
	ChannelSizeMediumLoad           = 64
	ChannelSizeHighLoad             = 128
	ChannelSizeLowDropRatio         = 256

	LogInterval         = time.Minute
	LogSlowInterval     = time.Minute * 2
	LogVerySlowInterval = time.Minute * 5
	LogFastInterval     = time.Second * 15
	LogVeryFastInterval = time.Second * 5
)
