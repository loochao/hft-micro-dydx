package stream_stats

import (
	"fmt"
	"github.com/montanaflynn/stats"
	"testing"
)

func TestCovariance(t *testing.T) {
	data1 := []float64{1.0, 2.1, 3.2, 4.823, 4.1, 5.8}
	data2 := []float64{1.0, 2.1, 3.2, 4.823, 4.1, 5.8}
	//v, _ := stats.Covariance(data1, data2)
	v, _ := stats.Correlation(data1, data2)
	stats.Variance(data1)
	fmt.Println(v)
}
