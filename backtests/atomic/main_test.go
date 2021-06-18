package main

import (
	"sync"
	"sync/atomic"
	"testing"
)

var Global_Mu *sync.Mutex
var Global_MyMu *MyMutex



const (
	mutexFreed  = int32(0)
	mutexLocked = int32(1) // mutex is locked
)

type MyMutex struct {
	state int32
}

func (m *MyMutex) Lock() {
	//if atomic.CompareAndSwapInt32(&m.state, mutexLocked, mutexLocked) {
	//	panic("lock locked")
	//}
	for {
		if atomic.CompareAndSwapInt32(&m.state, mutexFreed, mutexLocked) {
			return
		}
	}
}

func (m *MyMutex) Unlock() {
	//if atomic.CompareAndSwapInt32(&m.state, mutexFreed, mutexFreed) {
	//	panic("unlock unlocked")
	//}
	for  {
		if atomic.CompareAndSwapInt32(&m.state, mutexLocked, mutexFreed) {
			return
		}
	}
}

func BenchmarkMyLockUnlock(b *testing.B) {
	mu := MyMutex{}
	for i := 0; i < b.N; i++ {
		mu.Lock()
		mu.Unlock()
	}
	Global_MyMu = &mu
}

func BenchmarkMutexLockUnlock(b *testing.B) {
	mu := sync.Mutex{}
	for i := 0; i < b.N; i++ {
		mu.Lock()
		mu.Unlock()
	}
	Global_Mu = &mu
}