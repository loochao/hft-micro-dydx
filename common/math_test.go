package common

import (
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
	"time"
)

func TestNewTimedSum(t *testing.T) {
	ts1 := NewTimedSum(time.Second*100)
	ts2 := NewTimedSum(time.Second*100)
	eventTime := time.Unix(0,0)
	for i := 0; i < 100000; i++ {
		ts1.Insert(eventTime, rand.Float64())
		ts2.Insert(eventTime, rand.Float64())
		assert.Less(t, ts1.Sum()/float64(ts1.Len()), 1.0)
		assert.Less(t, ts2.Sum()/float64(ts2.Len()), 1.0)
		assert.Greater(t, ts1.Sum()/float64(ts1.Len()), 0.0)
		assert.Greater(t, ts2.Sum()/float64(ts2.Len()), 0.0)
		assert.Less(t, (ts1.Sum() - ts2.Sum())/(ts1.Sum() + ts2.Sum()), 1.0)
		assert.Greater(t, (ts1.Sum() - ts2.Sum())/(ts1.Sum() + ts2.Sum()), -1.0)
		eventTime.Add(time.Duration(rand.Intn(10)-10))
	}
}
