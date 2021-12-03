package common

import (
	"testing"
)

func TestFloat64Ring_SetCapacity(t *testing.T) {
	r := NewFloat64Ring(100)
	r.setCapacity(10)
	if r.Capacity() != 10 {
		t.Fatal("Size of ring was not 10", r.Capacity())
	}
}

func TestFloat64Ring_SavesSomeData(t *testing.T) {
	r := NewFloat64Ring(10)
	for i := 0.0; i < 70.0; i++ {
		r.Enqueue(i)
	}
	for i := 0.0; i < 70.0; i++ {
		x := r.Dequeue()
		if x == nil || *x != i {
			t.Fatal("Unexpected response", x, "wanted", i)
		}
	}
	x := r.Dequeue()
	if x != nil {
		t.Fatal("Unexpected response", x, "wanted", nil)
	}
}

func TestFloat64Ring_Peeks(t *testing.T) {
	r := NewFloat64Ring(10)
	for i := 0.0; i < 100; i++ {
		r.Enqueue(i)
	}
	for i := 0.0; i < 100; i++ {
		r.Peek()
		r.Peek()
		x1 := r.Peek()
		if x1 == nil {
			t.Fatal("Unexpected response", x1, "wanted", i)
		}
		x := r.Dequeue()
		if x == nil {
			t.Fatal("Unexpected response", x, "wanted", i)
		}
		if *x != i {
			t.Fatal("Unexpected response", *x, "wanted", i)
		}
		if *x1 != *x {
			t.Fatal("Unexpected response", *x1, "wanted", *x)
		}
	}
}

//func TestFloat64Ring_ConstructArr(t *testing.T) {
//	r := NewFloat64Ring(10)
//	v := r.Values()
//	if len(v) != 0 {
//		t.Fatal("Unexpected values", v, "wanted len of", 0)
//	}
//	for i := 1.0; i < 21000; i++ {
//		r.Enqueue(i)
//		l := int(i)
//		v = r.Values()
//		if len(v) != l {
//			t.Fatal("Unexpected values", v, "wanted len of", l, "index", i)
//		}
//	}
//}

func TestFloat64Ring_ContentSize(t *testing.T) {
	r := NewFloat64Ring(10)

	for i := 1; i < 101; i++ {
		r.Enqueue(float64(i))
		s := r.ContentSize()
		if s != i {
			t.Fatal("Unexpected content size", s, "wanted", i)
		}
	}

	for i := 99; i > 0; i-- {
		r.Dequeue()
		s := r.ContentSize()
		if s != i {
			t.Fatal("Unexpected content size", s, "wanted", i)
		}
	}
}

func BenchmarkSliceAppend(b *testing.B) {
	s := make([]float64, 10000000)
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		for i := 0; i < 1000000; i++ {
			s = append(s, 0)
			s = s[1:]
		}
	}
}

func BenchmarkRingFloat64Append(b *testing.B) {
	s := NewFloat64Ring(1000000)
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		for i := 0; i < 1000000; i++ {
			s.Enqueue(0)
		}
	}
}
