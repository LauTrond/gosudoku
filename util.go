package main

import (
	"fmt"
	"sync"
)

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

type RowCol struct {
	Row, Col int
}

type RowColNum struct {
	RowCol
	Num int
}

func RCN(r, c, n int) RowColNum {
	return RowColNum{
		RowCol: RowCol{
			Row: r,
			Col: c,
		},
		Num: n,
	}
}

func (rcn RowColNum) Extract() (r, c, n int) {
	return rcn.Row, rcn.Col, rcn.Num
}

type RowColNumExclude struct {
	RowColNum
	CheckFlag int
}

func RCNE(r, c, n, e int) RowColNumExclude {
	return RowColNumExclude{
		RowColNum: RowColNum{
			RowCol: RowCol{
				Row: r,
				Col: c,
			},
			Num: n,
		},
		CheckFlag: e,
	}
}

func (rcne RowColNumExclude) RowToColCheck() bool {
	return rcne.CheckFlag&NoRowToColCheck == 0
}
func (rcne RowColNumExclude) RowToBlockCheck() bool {
	return rcne.CheckFlag&NoRowToBlockCheck == 0
}
func (rcne RowColNumExclude) RowToNumCheck() bool {
	return rcne.CheckFlag&NoRowToNumCheck == 0
}
func (rcne RowColNumExclude) ColToRowCheck() bool {
	return rcne.CheckFlag&NoColToRowCheck == 0
}
func (rcne RowColNumExclude) ColToBlockCheck() bool {
	return rcne.CheckFlag&NoColToBlockCheck == 0
}
func (rcne RowColNumExclude) ColToNumCheck() bool {
	return rcne.CheckFlag&NoColToNumCheck == 0
}
func (rcne RowColNumExclude) BlockToRowCheck() bool {
	return rcne.CheckFlag&NoBlockToRowCheck == 0
}
func (rcne RowColNumExclude) BlockToColCheck() bool {
	return rcne.CheckFlag&NoBlockToColCheck == 0
}
func (rcne RowColNumExclude) BlockToNumCheck() bool {
	return rcne.CheckFlag&NoBlockToNumCheck == 0
}
func (rcne RowColNumExclude) NumToRowCheck() bool {
	return rcne.CheckFlag&NoNumToRowCheck == 0
}
func (rcne RowColNumExclude) NumToColCheck() bool {
	return rcne.CheckFlag&NoNumToColCheck == 0
}
func (rcne RowColNumExclude) NumToBlockCheck() bool {
	return rcne.CheckFlag&NoNumToBlockCheck == 0
}

func add(p *int8, n int8) int8 {
	*p += n
	return *p
}

func setBit(p *int16, bitOffset int) int16 {
	*p |= 1 << bitOffset
	return *p
}

type BranchChoices struct {
	tmpArray [9]RowColNum
	Choices  []RowColNum
}

func (c *BranchChoices) Init() {
	c.Choices = c.tmpArray[:0]
}

func (c *BranchChoices) Size() int {
	if c == nil {
		return 0
	}
	return len(c.Choices)
}

func (c *BranchChoices) Add(rcn RowColNum) {
	c.Choices = append(c.Choices, rcn)
}

var branchChoicesPool = sync.Pool{
	New: func() any {
		return new(BranchChoices)
	},
}

func NewBranchChoices() *BranchChoices {
	c := branchChoicesPool.Get().(*BranchChoices)
	c.Init()
	return c
}

func ReleaseBranchChoices(c *BranchChoices) {
	branchChoicesPool.Put(c)
}
