package common

import (
	"testing"
)

func TestInt64Ring_SetCapacity(t *testing.T) {
	r := NewInt64Ring(100)
	r.setCapacity(10)
	if r.Capacity() != 10 {
		t.Fatal("Size of ring was not 10", r.Capacity())
	}
}

func TestInt64Ring_SavesSomeData(t *testing.T) {
	r := NewInt64Ring(10)
	for i := int64(0); i < 70; i++ {
		r.Enqueue(i)
	}
	for i := int64(0); i < 70; i++ {
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

func TestInt64Ring_Peeks(t *testing.T) {
	r := NewInt64Ring(10)
	for i := int64(0); i < 100; i++ {
		r.Enqueue(i)
	}
	for i := int64(0); i < 100; i++ {
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

//func TestInt64Ring_ConstructArr(t *testing.T) {
//	r := NewInt64Ring(10)
//	v := r.Values()
//	if len(v) != 0 {
//		t.Fatal("Unexpected values", v, "wanted len of", 0)
//	}
//	for i := int64(1); i < 21000; i++ {
//		r.Enqueue(i)
//		l := int(i)
//		v = r.Values()
//		if len(v) != l {
//			t.Fatal("Unexpected values", v, "wanted len of", l, "index", i)
//		}
//	}
//}

func TestInt64Ring_ContentSize(t *testing.T) {
	r := NewInt64Ring(10)

	for i := int64(1); i < 101; i++ {
		r.Enqueue(i)
		s := r.ContentSize()
		if s != int(i) {
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
