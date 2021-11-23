package stream_stats

import (
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"testing"
)

func TestNewXYMakerTakerStatsParams(t *testing.T) {
	v1 := NewXYMakerTakerStatsParams{}
	has, fields := common.DetectDefaultValues(v1,  []string{})
	logger.Debugf("%v %s", has, fields)
}
