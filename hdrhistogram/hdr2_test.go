package hdrhistogram

import (
	"fmt"
	"math"
	"math/bits"
	"testing"
)

func TestDebug(t *testing.T) {
	hh := New(1, 2000, 1)
	fmt.Printf("bucketCount %b %d\n", hh.bucketCount, hh.bucketCount)
	fmt.Printf("subBucketCount %b %d\n", hh.subBucketCount, hh.subBucketCount)
	fmt.Printf("unitMagnitude %d\n", hh.unitMagnitude)
	fmt.Printf("subBucketMask %d %b\n", hh.subBucketMask, hh.subBucketMask)
	fmt.Printf("countsLen %d\n", hh.countsLen)
	fmt.Printf("%b\n", hh.subBucketMask)

	value := int64(20000)
	fmt.Printf("value %b %b %b\n", value, hh.subBucketMask, value|hh.subBucketMask)
	fmt.Printf("LeadingZeros64 %d\n", bits.LeadingZeros64(uint64(value|hh.subBucketMask)))
	fmt.Printf("bucketIndex %d\n", hh.getBucketIndex(value))
	fmt.Printf("subBucketIdx value %b -> %b\n", value, value>>uint32(hh.getBucketIndex(value)+int32(hh.unitMagnitude)))
	fmt.Printf("%d\n", hh.getSubBucketIdx(value, hh.getBucketIndex(value)))
	fmt.Printf("\n\n")
	value = 10019
	fmt.Printf("%b\n", value|hh.subBucketMask)
	fmt.Printf("LeadingZeros64 %d\n", bits.LeadingZeros64(uint64(value|hh.subBucketMask)))
	fmt.Printf("%d\n", hh.getBucketIndex(value))
	fmt.Printf("%d\n", value>>uint(int32(hh.getBucketIndex(value))+int32(hh.unitMagnitude)))
	fmt.Printf("%d\n", hh.getSubBucketIdx(value, hh.getBucketIndex(value)))

	fmt.Printf("%d\n", int32(math.Floor(math.Log2(float64(1<<63)))))

	fmt.Printf("%b %b\n", 1024, 1024>>8)
}
