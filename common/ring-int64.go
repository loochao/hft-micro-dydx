package common

import "encoding/json"

type Int64Ring struct {
	head int
	tail int
	buff []int64
}

func (r *Int64Ring) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Head int     `json:"head"`
		Tail int     `json:"tail"`
		Buff []int64 `json:"buff"`
	}{
		Head: r.head,
		Tail: r.tail,
		Buff: r.buff,
	})
}

func (r *Int64Ring) UnmarshalJSON(data []byte) error {
	aux := &struct {
		Head int     `json:"head"`
		Tail int     `json:"tail"`
		Buff []int64 `json:"buff"`
	}{}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	} else {
		r.head = aux.Head
		r.tail = aux.Tail
		r.buff = aux.Buff
	}
	return nil
}

// returns the modified index of an unmodified index
func (r *Int64Ring) mod(p int) int {
	v := p % len(r.buff)
	for v < 0 { // this bit fixes negative indices
		v += len(r.buff)
	}
	return v
}

// gets a value based at a given unmodified index
func (r *Int64Ring) get(p int) int64 {
	return r.buff[r.mod(p)]
}

// sets a value at the given unmodified index and returns the modified index of the value
func (r *Int64Ring) set(p int, v int64) {
	r.buff[r.mod(p)] = v
}

func (r *Int64Ring) Capacity() int {
	return len(r.buff)
}

/*
ContentSize returns the current number of elements inside the ring buffer.
*/
func (r *Int64Ring) ContentSize() int {
	if r.head == -1 {
		return 0
	} else {
		difference := r.head - r.tail
		if difference < 0 {
			difference += r.Capacity()
		}
		return difference + 1
	}
}

/*
Enqueue a value into the Ring buffer.
*/
func (r *Int64Ring) Enqueue(i int64) {
	if r.Capacity() == r.ContentSize() {
		r.setCapacity(r.Capacity() * 2)
	}
	r.set(r.head+1, i)
	old := r.head
	r.head = r.mod(r.head + 1)
	if old != -1 && r.head == r.tail {
		r.tail = r.mod(r.tail + 1)
	}
}

/*
Dequeue a value from the Ring buffer.
Returns nil if the ring buffer is empty.
*/
func (r *Int64Ring) Dequeue() *int64 {
	if r.head == -1 {
		return nil
	}
	v := r.get(r.tail)
	if r.tail == r.head {
		r.head = -1
		r.tail = 0
	} else {
		r.tail = r.mod(r.tail + 1)
	}
	return &v
}

/*
Read the value that Dequeue would have dequeued without actually dequeuing it.
Returns nil if the ring buffer is empty.
*/
func (r *Int64Ring) Peek() *int64 {
	if r.head == -1 {
		return nil
	}
	v := r.get(r.tail)
	return &v
}

func (r *Int64Ring) setCapacity(size int) {
	if size == len(r.buff) {
		return
	}

	if size < len(r.buff) {
		// shrink the buffer
		if r.head == -1 {
			// nothing in the buffer, so just shrink it directly
			r.buff = r.buff[0:size]
		} else {
			newb := make([]int64, 0, size)
			// buffer has stuff in it, so save the most recent stuff...
			// start at HEAD-SIZE-1 and walk forwards
			for i := size - 1; i >= 0; i-- {
				idx := r.mod(r.head - i)
				newb = append(newb, r.buff[idx])
			}
			// reset head and tail to proper values
			r.head = len(newb) - 1
			r.tail = 0
			r.buff = newb
		}
		return
	}

	// grow the buffer
	newb := make([]int64, size-len(r.buff))
	if r.head == -1 {
		// nothing in the buffer
		r.buff = append(r.buff, newb...)
	} else if r.head >= r.tail {
		// growing at the end is safe
		r.buff = append(r.buff, newb...)
	} else {
		// buffer has stuff that wraps around the end
		// have to rearrange the buffer so the contents are still in order
		part1 := make([]int64, len(r.buff[:r.head+1]))
		copy(part1, r.buff[:r.head+1])
		part2 := make([]int64, len(r.buff[r.tail:]))
		copy(part2, r.buff[r.tail:])
		r.buff = append(r.buff, newb...)
		newTail := r.mod(r.tail + len(newb))
		r.tail = newTail
		copy(r.buff[:r.head+1], part1)
		copy(r.buff[r.head+1:r.tail], newb)
		copy(r.buff[r.tail:], part2)
	}
}

///*
//Values returns a slice of all the values in the circular buffer without modifying them at all.
//The returned slice can be modified independently of the circular buffer. However, the values inside the slice
//are shared between the slice and circular buffer.
//*/
//func (r *Int64Ring) Values() []int64 {
//	if r.head == -1 {
//		return []int64{}
//	}
//	arr := make([]int64, 0, r.Capacity())
//	for i := 0; i < r.Capacity(); i++ {
//		idx := r.mod(i + r.tail)
//		arr = append(arr, r.get(idx))
//		if idx == r.head {
//			break
//		}
//	}
//	return arr
//}

func NewInt64Ring(capacity int) *Int64Ring {
	return &Int64Ring{
		head: -1,
		tail: 0,
		buff: make([]int64, capacity),
	}
}
