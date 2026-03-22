package main

import "sync/atomic"

func main() {
	flag := uint32(0)
	go func() {
		for {
			atomic.CompareAndSwapUint32(&flag, 0, 1)
		}
	}()

	for {
		for {
			if atomic.CompareAndSwapUint32(&flag, 1, 0) {
				break
			}
		}
	}
}
