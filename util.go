package main

import "fmt"

type Queue struct {
	values []RowColNumExclude
	bits   int
	mask   int
	head   int
	tail   int
}

func NewQueueCapacity(initCapacity int) *Queue {
	initCapacityBits := 2
	for 1<<initCapacityBits < initCapacity+1 {
		initCapacityBits++
	}
	return NewQueueBits(initCapacityBits)
}

func NewQueueBits(bits int) *Queue {
	return &Queue{
		values: make([]RowColNumExclude, 1<<bits),
		bits:   bits,
		mask:   (1 << bits) - 1,
	}
}

func (q *Queue) Enqueue(item RowColNumExclude) {
	next := (q.tail + 1) & q.mask
	if next == q.head {
		newQueue := NewQueueBits(q.bits + 1)
		newQueue.copyFrom(q)
		*q = *newQueue
		next = q.tail + 1
	}
	q.values[q.tail] = item
	q.tail = next
}

func (q *Queue) Size() int {
	if q.tail < q.head {
		return q.tail + len(q.values) - q.head
	} else {
		return q.tail - q.head
	}
}

func (q *Queue) copyFrom(x *Queue) {
	if len(q.values) < x.Size()+1 {
		panic(fmt.Errorf("insufficient capacity x.size()=%d len(q.values)=%d", x.Size(), len(q.values)))
	}
	if x.tail >= x.head {
		q.head = 0
		q.tail = copy(q.values, x.values[x.head:x.tail])
	} else {
		q.head = 0
		n1 := copy(q.values, x.values[x.head:])
		n2 := copy(q.values[n1:], x.values[0:x.tail])
		q.tail = n1 + n2
	}
}

func (q *Queue) CopyFrom(x *Queue) {
	if len(q.values) < x.Size()+1 {
		newQueue := NewQueueBits(x.bits)
		*q = *newQueue
	}
	q.copyFrom(x)
}

func (q *Queue) Dequeue() (item RowColNumExclude, ok bool) {
	if q.head == q.tail {
		return
	}
	item = q.values[q.head]
	q.head = (q.head + 1) & q.mask
	ok = true
	return
}

func (q *Queue) DiscardAll() {
	q.head = q.tail
}
