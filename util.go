package main

type Queue struct {
	values []any
	bits   int
	head   int
	tail   int
}

func NewQueue(initCapacity int) *Queue {
	initCapacityBits := 2
	for 1<<initCapacityBits < initCapacity {
		initCapacityBits++
	}
	return &Queue{
		values: make([]any, 1<<initCapacityBits),
		bits:   initCapacityBits,
	}
}

func (q *Queue) Enqueue(item any) {
	next := (q.tail + 1) & ((1 << q.bits) - 1)
	if next == q.head {
		newBits := q.bits + 1
		newValues := make([]any, 1<<newBits)
		var newTail int
		if q.tail > q.head {
			copy(newValues, q.values[q.head:q.tail])
			newTail = q.tail - q.head
		} else {
			n := copy(newValues, q.values[q.head:])
			copy(newValues[n:], q.values[0:q.tail])
			newTail = q.tail + n
		}
		q.values = newValues
		q.bits = newBits
		q.head = 0
		q.tail = newTail
		next = newTail + 1
	}
	q.values[q.tail] = item
	q.tail = next
}

func (q *Queue) Dequeue() (item any, ok bool) {
	if q.head == q.tail {
		return
	}
	item = q.values[q.head]
	q.head = (q.head + 1) & ((1 << q.bits) - 1)
	ok = true
	return
}
