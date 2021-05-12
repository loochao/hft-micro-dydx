package main

import "time"

type TimedVolume struct {
	lookback  time.Duration
	times     []time.Time
	volumes   []float64
	volumeSum float64
}

func (tm *TimedVolume) Insert(timestamp time.Time, volume float64) float64 {
	tm.times = append(tm.times, timestamp)
	tm.volumes = append(tm.volumes, volume)
	tm.volumeSum += volume
	cutIndex := 0
	for i, t := range tm.times {
		if timestamp.Sub(t) > tm.lookback {
			cutIndex = i
		} else {
			break
		}
	}
	if cutIndex > 0 {
		for _, value := range tm.volumes[:cutIndex] {
			tm.volumeSum -= value
		}
		tm.volumes = tm.volumes[cutIndex:]
		tm.times = tm.times[cutIndex:]
	}
	return tm.volumeSum
}

func (tm *TimedVolume) Sum() float64 {
	return tm.volumeSum
}

func (tm *TimedVolume) Len() int {
	return len(tm.times)
}

func NewTimedVolume(lookback time.Duration) *TimedVolume {
	return &TimedVolume{
		lookback: lookback,
		times:    make([]time.Time, 0),
		volumes:  make([]float64, 0),
	}
}
