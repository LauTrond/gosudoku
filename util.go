package main

import (
	"fmt"
	"sync"
)

var (
	loop9      = [9]int8{}
	loop9skip3 = [3][6]int8{
		{3, 4, 5, 6, 7, 8},
		{0, 1, 2, 6, 7, 8},
		{0, 1, 2, 3, 4, 5},
	}
	loop9skip3col = [3][6]int8{
		{1, 2, 4, 5, 7, 8},
		{0, 2, 3, 5, 6, 8},
		{0, 1, 3, 4, 6, 7},
	}
	loop9skip = [9][8]int8{
		{1, 2, 3, 4, 5, 6, 7, 8},
		{0, 2, 3, 4, 5, 6, 7, 8},
		{0, 1, 3, 4, 5, 6, 7, 8},
		{0, 1, 2, 4, 5, 6, 7, 8},
		{0, 1, 2, 3, 5, 6, 7, 8},
		{0, 1, 2, 3, 4, 6, 7, 8},
		{0, 1, 2, 3, 4, 5, 7, 8},
		{0, 1, 2, 3, 4, 5, 6, 8},
		{0, 1, 2, 3, 4, 5, 6, 7},
	}
	loop3        = [3]int8{}
	loop3skip    = [3][2]int8{{1, 2}, {0, 2}, {0, 1}}
	skip9mask    = [9]int16{}
	skip3mask    = [3]int16{0770, 0707, 077}
	skip3maskCol = [3]int16{0666, 0555, 0333}

	//返回有多少位为1
	countTrueBitsMap [1 << 9]int8

	// 从低位开始，返回bits第一个为0的位
	// xxxxxxxx1 -> -1
	// xxxxxxxx0 -> 0
	// xxxxxxx01 -> 1
	// xxxx01111 -> 4
	// 011111111 -> 8
	pos0map [1 << 9]int8
)

func init() {
	for i := range skip9mask {
		skip9mask[i] = ((1 << 9) - 1) ^ (1 << i)
	}
	for i := range countTrueBitsMap {
		for bit := range loop9 {
			countTrueBitsMap[i] += int8((i >> bit) & 1)
		}
	}
	for i := range pos0map {
		pos0map[i] = -2
	}
	for i := range loop9 {
		pos0map[skip9mask[i]] = int8(i)
	}
	pos0map[511] = -1
}

// 从低位开始，返回i唯一为0的位，最多支持9位
// 如果0不存在，返回-1
// 如果多于一个0，返回-2
// 101010101 -> -2
// 111111111 -> -1
// 111111110 -> 0
// 111111101 -> 1
// 111101111 -> 4
// 011111111 -> 8
func pos0(i int16) int8 {
	return pos0map[i]
}

func countTrueBits(i int16) int8 {
	return countTrueBitsMap[i]
}

func rcbp(r, c int8) (b int8, p int8) {
	return r/3*3 + c/3, r%3*3 + c%3
}

type RowCol struct {
	Row, Col int8
}

type RowColNum struct {
	RowCol
	Num int8
}

func RCN(r, c, n int8) RowColNum {
	return RowColNum{
		RowCol: RowCol{
			Row: r,
			Col: c,
		},
		Num: n,
	}
}

func NRC(n, r, c int8) RowColNum {
	return RCN(r, c, n)
}

func NCR(n, c, r int8) RowColNum {
	return RCN(r, c, n)
}

func RNC(r, n, c int8) RowColNum {
	return RCN(r, c, n)
}

func CNR(c, n, r int8) RowColNum {
	return RCN(r, c, n)
}

func CRN(c, r, n int8) RowColNum {
	return RCN(r, c, n)
}

func BNPtoRCN(b, n, p int8) RowColNum {
	r, c := rcbp(b, p)
	return RCN(r, c, n)
}

func BPNtoRCN(b, p, n int8) RowColNum {
	r, c := rcbp(b, p)
	return RCN(r, c, n)
}

func (rcn RowColNum) Extract() (r, c, n int8) {
	return rcn.Row, rcn.Col, rcn.Num
}

type BlockPos struct {
	Block, Pos int8
}

type BlockPosNum struct {
	BlockPos
	Num int8
}

func BPN(b, p, n int8) BlockPosNum {
	return BlockPosNum{
		BlockPos: BlockPos{
			Block: b,
			Pos:   p,
		},
		Num: n,
	}
}

func (bpn BlockPosNum) RCN() RowColNum {
	r, c := rcbp(bpn.Block, bpn.Pos)
	return RCN(r, c, bpn.Num)
}

type Queue struct {
	values []RowColNum
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
		values: make([]RowColNum, 1<<bits),
		bits:   bits,
		mask:   (1 << bits) - 1,
	}
}

func (q *Queue) Enqueue(item RowColNum) {
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

func (q *Queue) Dequeue() (item RowColNum, ok bool) {
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

func bitwiseOr(p *int16, mask int16) int16 {
	*p |= mask
	return *p
}

// 将 p 设置为 v，如果没改变则返回true
func setInt8(p *int8, v int8) bool {
	old := *p
	*p = v
	return old == v
}

// 将 p 设置为 v，如果没改变则返回true
func setInt16(p *int16, v int16) bool {
	old := *p
	*p = v
	return old == v
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

const (
	ConflictCell  = 1
	ConflictRow   = 2
	ConflictCol   = 3
	ConflictBlock = 4
)

type Conflict struct {
	ConflictType int
	RowColNum
}

func (c Conflict) String() string {
	switch c.ConflictType {
	case ConflictCell:
		return fmt.Sprintf("单元格 (%d,%d) 没有可以填的数字", c.Row+1, c.Col+1)
	case ConflictRow:
		return fmt.Sprintf("行 %d 没有单元格可以填 %d", c.Row+1, c.Num+1)
	case ConflictCol:
		return fmt.Sprintf("列 %d 没有单元格可以填 %d", c.Col+1, c.Num+1)
	case ConflictBlock:
		return fmt.Sprintf("宫 (%d,%d) 没有单元格可以填 %d", c.Row/3+1, c.Col/3+1, c.Num+1)
	default:
		return ""
	}
}
