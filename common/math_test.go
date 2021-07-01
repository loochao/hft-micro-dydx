package common

import (
	"fmt"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
	"time"
)

func TestNewTimedSum(t *testing.T) {
	ts1 := NewTimedSum(time.Second*10)
	ts2 := NewTimedSum(time.Second*10)
	eventTime := time.Unix(0,0)
	for i := 0; i < 100; i++ {
		ts1.Insert(eventTime, rand.Float64())
		ts2.Insert(eventTime, rand.Float64())
		logger.Debugf("%f %d %f %d", ts1.Sum(), ts1.Len(), ts2.Sum(), ts2.Len())
		assert.Less(t, ts1.Sum()/float64(ts1.Len()), 1.0)
		assert.Less(t, ts2.Sum()/float64(ts2.Len()), 1.0)
		assert.Greater(t, ts1.Sum()/float64(ts1.Len()), 0.0)
		assert.Greater(t, ts2.Sum()/float64(ts2.Len()), 0.0)
		assert.Less(t, (ts1.Sum() - ts2.Sum())/(ts1.Sum() + ts2.Sum()), 1.0)
		assert.Greater(t, (ts1.Sum() - ts2.Sum())/(ts1.Sum() + ts2.Sum()), -1.0)
		eventTime = eventTime.Add(time.Second)
	}
}

func TestRollingSum_Insert(t *testing.T) {
	for l := 2; l < 100; l ++ {
		ts := NewRollingSum(l)
		for i := 0; i < 1000; i++ {
			s := 0.0
			if i >= l {
				for j := i - l + 1 ; j <= i; j++ {
					s += float64(j)
				}
			}else {
				for j := 0; j <= i; j++ {
					s += float64(j)
				}
			}
			ts.Insert(float64(i))
			assert.Equal(t, s, ts.Sum(), fmt.Sprintf("%d", i))
		}
	}
}
