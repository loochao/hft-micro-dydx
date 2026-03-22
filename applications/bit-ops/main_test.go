package bit_ops_test

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/geometrybase/hft-micro/logger"
)

func ExamplePrintABC() {

	a := 60
	b := 13
	fmt.Printf("OR  %6b | %b = %6b %d\n", a, b, a|b, a|b)
	fmt.Printf("AND %6b & %b = %6b %d\n", a, b, a&b, a&b)
	fmt.Printf("XOR %6b ^ %b = %6b %d\n", a, b, a^b, a^b)
	fmt.Printf("<< %6b ^ %b = %6b %d\n", a, b, a^b, a^b)
	a = a + 2
	b = b + 2
	//Check if the integer is even or odd.
	if a&1 == 0 {
		fmt.Printf("%d is even\n", a)
	} else {
		fmt.Printf("%d is odd\n", a)
	}
	if b&1 == 0 {
		fmt.Printf("%d is even\n", b)
	} else {
		fmt.Printf("%d is odd\n", b)
	}
	//Test if the n-th bit is set.
	for i := 0; i < 6; i++ {
		if (a & (1 << i)) > 0 {
			fmt.Printf("%b %6b %d-th bit is set\n", a, 1<<i, i)
		} else {
			fmt.Printf("%b %6b %d-th bit is not set\n", a, 1<<i, i)
		}
	}
	// Set the n-th bit
	for i := 0; i < 10; i++ {
		fmt.Printf("%b %6b set %d-th bit\n", a, a | (1<<i), i)
	}

	//Toggle the n-th bit.
	for i := 0; i < 10; i++ {
		fmt.Printf("%b %6b toggle %d-th bit\n", a, a ^ (1<<i), i)
	}
	//Turn off the rightmost 1-bit.
	fmt.Printf("%b %6b off the rightmost 1-bit\n", a, a & (a-1))
	c := a & (a-1)
	fmt.Printf("%b %6b off the rightmost 1-bit\n", c, c & (c-1))

	fmt.Printf("a %b -a %b\n", a, -a)
	// Isolate the rightmost 1-bit.
	fmt.Printf("%b %b a & (-a) %b\n", a, a, a & (-a))
	fmt.Printf("%b %b b & (-b) %b\n", b, b, b & (-b))

	//Right propagate the rightmost 1-bit.
	//01010000 -> 01011111
	fmt.Printf("%b -> %b\n", a,  a | (a - 1))
	fmt.Printf("%b -> %b\n", b,  b | (b - 1))

	var bitwisenot byte = 0x0F

	fmt.Printf("%b -> %b\n", bitwisenot,  ^bitwisenot)
	////Isolate the rightmost 0-bit.
	//fmt.Printf("%b -> %b\n", a,  (^a) & (a + 1))
	//fmt.Printf("%b -> %b\n", b,  (^b) & (b + 1))

	//Turn on the rightmost 0-bit.
	fmt.Printf("%b -> %b\n", a, a | (a+1))
	fmt.Printf("%b -> %b\n", a, b | (b+1))

	fmt.Printf("%b\n", 1<<63-1)
	//fmt.Printf("%b\n", math.MaxFloat64)

	//The &^ Operator
	//var a1 byte = 0xAB
	//fmt.Printf("%08b %08b\n", a1, &^(a1, 0x0F))

	smallestUnTrackableValue := int64(2)
	maxValue := int64(100000000)
	bucketsNeeded := int32(1)
	for smallestUnTrackableValue < maxValue {
		if smallestUnTrackableValue > (math.MaxInt64/2) {
			bucketsNeeded ++
			logger.Debugf("%d %d %b %d", maxValue, smallestUnTrackableValue, smallestUnTrackableValue, bucketsNeeded)
			break
		}
		smallestUnTrackableValue <<=1
		bucketsNeeded ++
		logger.Debugf("%d %d %b %d", maxValue, smallestUnTrackableValue, smallestUnTrackableValue, bucketsNeeded)
	}

	//hh := hdrhistogram.New(10, 20000, 3)
	//logger.Debugf()

	// Output:
	// 16 16 16
}
