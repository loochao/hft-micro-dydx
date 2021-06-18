package main

import (
	"sync"
	"time"
)

func main() {
	read := sync.Mutex{}
	wg := sync.WaitGroup{}
	func (){
		for {
			select {
			case <- time.After(time.Second):
				wg.Done()
			}
		}
	}()

	runtime.

}
