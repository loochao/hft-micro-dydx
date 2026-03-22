package benchmarks

import "testing"

type A struct{}

func (a A) Add() {
}

type B interface {
	Add()
}

var global_A *A
var global_B B

func BenchmarkMethodOnStruct(b *testing.B) {
	a := A{}
	for i := 0; i < b.N; i++ {
		a.Add()
	}
	global_A = &a
}

func BenchmarkMethodOnInterface(b *testing.B) {
	var c B
	c = &A{}
	for i := 0; i < b.N; i++ {
		c.Add()
	}
	global_B = c
}