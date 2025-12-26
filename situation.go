package main

import (
	"fmt"
	"hash/crc64"
	"strconv"
	"strings"
	"sync"
)

// Situation 代表9*9数独的一个局势
// 所有行、列、单元格值都是0~8，显示的时候加1处理。
type Situation struct {
	//cell[r][c] = n
	//n >=0 : 单元格(r,c) 填入了数字n
	//n == -1 : 单元格(r,c) 还没填入数字
	cells [9][9]int8

	//已填单元格总数
	setCount int

	//numSetCount[n] = x ： n 已填充 x 次
	numSetCount [9]int8
	//rowSetCount[r] = x ： r 行已填充 x 个数
	rowSetCount [9]int8
	//colSetCount[c] = x ： c 列已填充 x 个数
	colSetCount [9]int8
	//blockSetCount[b] = x ： 宫 b 已填充 x 个数
	blockSetCount [9]int8

	//cellExclude[n][r][c] = 1 ： 单元格(r,c)排除 n
	cellExclude [9][9][9]int8

	//numExcludeMask[r][c] 的每一位代表该单元格(r,c)排除了哪些数字
	numExcludeMask [9][9]int16

	//rowExcludeMask[n][r] 的每一位代表 r 行排除了哪些单元格
	rowExcludeMask [9][9]int16

	//colExcludeMask[n][c] 的每一位代表 c 列排除了哪些单元格
	colExcludeMask [9][9]int16

	//blockExcludeMask[n][b] 的每一位代表 宫 b 排除了哪些单元格
	blockExcludeMask [9][9]int16

	//分支代数，每执行一次Copy就加1
	branchGeneration int
}

// 初始化一个数独谜题
// puzzle 是一个9行的字符串（前后空行会自动去除），以点和数字代表单元格
// 其中空格代表未填入的单元格
func ParseSituation(puzzle string) (*Situation, *Trigger) {
	s := NewSituation()
	lines := strings.Split(strings.Trim(puzzle, "\n"), "\n")
	t := NewTrigger()
	for r, line := range lines {
		if r > len(s.cells) {
			panic(fmt.Errorf("row exceed"))
		}
		for c, n := range line {
			if n >= '1' && n <= '9' {
				s.Set(t, RCN(int8(r), int8(c), int8(n-'1')))
			}
		}
	}
	return s, t
}

// 初始化一个数独谜题，不换行的81个字符
func ParseSituationFromLine(line []byte) (*Situation, *Trigger) {
	if len(line) != 81 {
		panic("invalid puzzle from line")
	}

	s := NewSituation()
	t := NewTrigger()
	for i, n := range line {
		if n >= '1' && n <= '9' {
			s.Set(t, RCN(int8(i/9), int8(i%9), int8(n-'1')))
		}
	}
	return s, t
}

var situationPool = sync.Pool{
	New: func() any {
		return new(Situation)
	},
}

var EmptySituation = func() *Situation {
	s := new(Situation)
	for r := range loop9 {
		for c := range loop9 {
			s.cells[r][c] = -1
		}
	}
	return s
}()

func NewSituation() *Situation {
	s := situationPool.Get().(*Situation)
	*s = *EmptySituation
	return s
}

func ReleaseSituation(s *Situation) {
	situationPool.Put(s)
}

func DuplicateSituation(s *Situation) *Situation {
	s2 := situationPool.Get().(*Situation)
	s2.CopyFrom(s)
	return s2
}

func (s *Situation) CopyFrom(x *Situation) {
	*s = *x
}

func (s *Situation) Count() int {
	return s.setCount
}

// 填数
func (s *Situation) Set(t *Trigger, rcn RowColNum) bool {
	r, c, n := rcn.Extract()
	if s.cells[r][c] != -1 {
		return false
	}
	s.cells[r][c] = n

	var (
		R, C         = r / 3, c / 3
		rr, cc       = r % 3, c % 3
		b, p         = R*3 + C, rr*3 + cc
		nm     int16 = 1 << n
		rm     int16 = 1 << r
		cm     int16 = 1 << c
		pm     int16 = 1 << p
		Rm     int16 = 07 << (R * 3)
		Cm     int16 = 07 << (C * 3)
		ccm    int16 = 0111 << cc
		rrm    int16 = 07 << (rr * 3)
	)

	s.setCount++
	s.numSetCount[n]++
	s.rowSetCount[r]++
	s.colSetCount[c]++
	s.blockSetCount[b]++

	if !setInt16(&s.numExcludeMask[r][c], skip9mask[n]) {
		for _, n0 := range loop9skip[n] {
			if setInt8(&s.cellExclude[n0][r][c], 1) {
				continue
			}
			if rcn, confirm := s.applyRowMask(n0, r, cm); confirm {
				s.confirmRow(t, rcn)
			}
			if rcn, confirm := s.applyColMask(n0, c, rm); confirm {
				s.confirmCol(t, rcn)
			}
			if bpn, confirm := s.applyBlockMask(n0, b, pm); confirm {
				s.confirmBlock(t, bpn)
			}
		}
	}
	if !setInt16(&s.rowExcludeMask[n][r], skip9mask[c]) {
		for _, c0 := range loop9skip3[C] {
			if setInt8(&s.cellExclude[n][r][c0], 1) {
				continue
			}
			if rcn, confirm := s.applyNumMask(r, c0, nm); confirm {
				s.confirmNum(t, rcn)
			}
			if rcn, confirm := s.applyColMask(n, c0, rm); confirm {
				s.confirmCol(t, rcn)
			}
		}
		for _, C0 := range loop3skip[C] {
			if bpn, confirm := s.applyBlockMask(n, R*3+C0, rrm); confirm {
				s.confirmBlock(t, bpn)
			}
		}
	}
	if !setInt16(&s.colExcludeMask[n][c], skip9mask[r]) {
		for _, r0 := range loop9skip3[R] {
			if setInt8(&s.cellExclude[n][r0][c], 1) {
				continue
			}
			if rcn, confirm := s.applyNumMask(r0, c, nm); confirm {
				s.confirmNum(t, rcn)
			}
			if rcn, confirm := s.applyRowMask(n, r0, cm); confirm {
				s.confirmRow(t, rcn)
			}
		}
		for _, R0 := range loop3skip[R] {
			if bpn, confirm := s.applyBlockMask(n, R0*3+C, ccm); confirm {
				s.confirmBlock(t, bpn)
			}
		}
	}
	if !setInt16(&s.blockExcludeMask[n][b], skip9mask[p]) {
		for _, p0 := range loop9skip[p] {
			r0, c0 := rcbp(b, p0)
			if setInt8(&s.cellExclude[n][r0][c0], 1) {
				continue
			}
			if rcn, confirm := s.applyNumMask(r0, c0, nm); confirm {
				s.confirmNum(t, rcn)
			}
		}
		for _, rr0 := range loop3skip[rr] {
			r0 := R*3 + rr0
			if rcn, confirm := s.applyRowMask(n, r0, Cm); confirm {
				s.confirmRow(t, rcn)
			}
		}
		for _, cc0 := range loop3skip[cc] {
			c0 := C*3 + cc0
			if rcn, confirm := s.applyColMask(n, c0, Rm); confirm {
				s.confirmCol(t, rcn)
			}
		}
	}

	return true
}

func (s *Situation) applyNumMask(r, c int8, mask int16) (RowColNum, bool) {
	n0 := pos0(bitwiseOr(&s.numExcludeMask[r][c], mask))
	return RCN(r, c, n0), n0 != -2
}

func (s *Situation) confirmNum(t *Trigger, rcn RowColNum) {
	if rcn.Num >= 0 {
		s.confirm(t, rcn)
	} else {
		t.Conflict(ConflictCell, rcn)
	}
}

func (s *Situation) applyRowMask(n, r int8, mask int16) (RowColNum, bool) {
	c0 := pos0(bitwiseOr(&s.rowExcludeMask[n][r], mask))
	return RCN(r, c0, n), c0 != -2
}

func (s *Situation) confirmRow(t *Trigger, rcn RowColNum) {
	if rcn.Col >= 0 {
		s.confirm(t, rcn)
	} else {
		t.Conflict(ConflictRow, rcn)
	}
}

func (s *Situation) applyColMask(n, c int8, mask int16) (RowColNum, bool) {
	r0 := pos0(bitwiseOr(&s.colExcludeMask[n][c], mask))
	return RCN(r0, c, n), r0 != -2
}

func (s *Situation) confirmCol(t *Trigger, rcn RowColNum) {
	if rcn.Row >= 0 {
		s.confirm(t, rcn)
	} else {
		t.Conflict(ConflictCol, rcn)
	}
}

func (s *Situation) applyBlockMask(n, b int8, mask int16) (BlockPosNum, bool) {
	p0 := pos0(bitwiseOr(&s.blockExcludeMask[n][b], mask))
	return BPN(b, p0, n), p0 != -2
}

func (s *Situation) confirmBlock(t *Trigger, bpn BlockPosNum) {
	if bpn.Pos >= 0 {
		s.confirm(t, bpn.RCN())
	} else {
		bpn.Pos = 0
		t.Conflict(ConflictBlock, bpn.RCN())
	}
}

func (s *Situation) ApplyExcludeRules(t *Trigger) (changed int) {
	changed += s.applyDimVariantRule(t, s.getMaskNumRow, s.getMaskNumCol, NRC)
	changed += s.applyDimVariantRule(t, s.getMaskNumCol, s.getMaskNumRow, NCR)
	changed += s.applyDimVariantRule(t, s.getMaskRowCol, s.getMaskRowNum, RCN)
	changed += s.applyDimVariantRule(t, s.getMaskRowNum, s.getMaskRowCol, RNC)
	changed += s.applyDimVariantRule(t, s.getMaskColNum, s.getMaskColRow, CNR)
	changed += s.applyDimVariantRule(t, s.getMaskColRow, s.getMaskColNum, CRN)
	changed += s.applyDimVariantRule(t, s.getMaskBlockNum, s.getMaskBlockPos, BNPtoRCN)
	changed += s.applyDimVariantRule(t, s.getMaskBlockPos, s.getMaskBlockNum, BPNtoRCN)
	changed += s.applyBlockRules(t)
	return
}

func (s *Situation) getMaskNumRow(n, r int8) *int16 {
	return &s.rowExcludeMask[n][r]
}

func (s *Situation) getMaskRowNum(r, n int8) *int16 {
	return &s.rowExcludeMask[n][r]
}

func (s *Situation) getMaskNumCol(n, c int8) *int16 {
	return &s.colExcludeMask[n][c]
}

func (s *Situation) getMaskColNum(c, n int8) *int16 {
	return &s.colExcludeMask[n][c]
}

func (s *Situation) getMaskRowCol(r, c int8) *int16 {
	return &s.numExcludeMask[r][c]
}

func (s *Situation) getMaskColRow(c, r int8) *int16 {
	return &s.numExcludeMask[r][c]
}

func (s *Situation) getMaskBlockNum(b, n int8) *int16 {
	return &s.blockExcludeMask[n][b]
}

func (s *Situation) getMaskBlockPos(b, p int8) *int16 {
	r, c := rcbp(b, p)
	return &s.numExcludeMask[r][c]
}

type getMaskFunc func(x, y int8) *int16
type getRowColNumFunc func(x, y, z int8) RowColNum

func (s *Situation) applyDimVariantRule(t *Trigger, getMaskDim1Dim2, getMaskDim1Dim3 getMaskFunc, getRCN getRowColNumFunc) (changed int) {
	for _dim1 := range loop9 {
		dim1 := int8(_dim1)
		var checkMap [1 << 9]int8
		for _dim2 := range loop9 {
			dim2b := int8(_dim2)
			dim3mask := *getMaskDim1Dim2(dim1, dim2b)
			if countTrueBits(dim3mask) != 7 {
				continue
			}
			if checkMap[dim3mask] == 0 {
				checkMap[dim3mask] = dim2b + 1
				continue
			}
			dim2a := checkMap[dim3mask] - 1

			dim2skipMask := skip9mask[dim2a] & skip9mask[dim2b]
			for _dim3 := range loop9 {
				dim3 := int8(_dim3)
				if dim3mask&(1<<dim3) > 0 {
					continue
				}
				dim2mask := *getMaskDim1Dim3(dim1, dim3)
				if dim2mask == dim2mask|dim2skipMask {
					continue
				}
				for _dim2c := range loop9 {
					dim2c := int8(_dim2c)
					if dim2c == dim2a || dim2c == dim2b {
						continue
					}
					changed += s.excludeOne(t, getRCN(dim1, dim2c, dim3))
				}
				if len(t.Conflicts) > 0 {
					return
				}
			}
		}

	}
	return
}

func (s *Situation) applyBlockRules(t *Trigger) (changed int) {
	for _n := range loop9 {
		n := int8(_n)
		for _r := range loop9 {
			r := int8(_r)
			rr := r % 3
			for _C := range loop3 {
				C := int8(_C)
				b := r/3*3 + C
				if s.rowExcludeMask[n][r]&skip3mask[C] == skip3mask[C] && s.blockExcludeMask[n][b]&skip3mask[rr] != skip3mask[rr] {
					for _, p := range loop9skip3[rr] {
						changed += s.excludeOne(t, BPNtoRCN(b, p, n))
					}
				}
				if s.blockExcludeMask[n][b]&skip3mask[rr] == skip3mask[rr] && s.rowExcludeMask[n][r]&skip3mask[C] != skip3mask[C] {
					for _, c := range loop9skip3[C] {
						changed += s.excludeOne(t, RCN(r, c, n))
					}
				}
			}
		}
		for _c := range loop9 {
			c := int8(_c)
			cc := c % 3
			for _R := range loop3 {
				R := int8(_R)
				b := c/3 + R*3
				if s.colExcludeMask[n][c]&skip3mask[R] == skip3mask[R] && s.blockExcludeMask[n][b]&skip3maskCol[cc] != skip3maskCol[cc] {
					for _, p := range loop9skip3col[cc] {
						changed += s.excludeOne(t, BPNtoRCN(b, p, n))
					}
				}
				if s.blockExcludeMask[n][b]&skip3maskCol[cc] == skip3maskCol[cc] && s.colExcludeMask[n][c]&skip3mask[R] != skip3mask[R] {
					for _, r := range loop9skip3[R] {
						changed += s.excludeOne(t, RCN(r, c, n))
					}
				}
			}
		}
	}
	return
}

func (s *Situation) excludeOne(t *Trigger, rcn RowColNum) int {
	r, c, n := rcn.Extract()
	if setInt8(&s.cellExclude[r][c][n], 1) {
		return 0
	}
	if rcn0, confirm := s.applyNumMask(r, c, 1<<n); confirm {
		s.confirmNum(t, rcn0)
	}
	if rcn0, confirm := s.applyRowMask(n, r, 1<<c); confirm {
		s.confirmRow(t, rcn0)
	}
	if rcn0, confirm := s.applyColMask(n, c, 1<<r); confirm {
		s.confirmCol(t, rcn0)
	}
	b, p := rcbp(r, c)
	if bpn0, confirm := s.applyBlockMask(n, b, 1<<p); confirm {
		s.confirmBlock(t, bpn0)
	}
	return 1
}

func (s *Situation) confirm(t *Trigger, rcn RowColNum) {
	if s.cells[rcn.Row][rcn.Col] != -1 {
		return
	}
	t.Confirm(rcn)
}

func (s *Situation) Show(title string, r, c int) {
	lines := strings.Split(title, "\n")
	lines[0] = fmt.Sprintf("<%02d> %s", s.setCount, lines[0])
	for i := 1; i < len(lines); i++ {
		lines[i] = "     " + lines[i]
	}
	title = strings.Join(lines, "\n")
	ShowCells(&s.cells, title, r, c)
}

func (s *Situation) RowColHash(rc RowCol) int {
	return (int(rc.Row)*317 + int(rc.Col)*659 + s.setCount*531) % 997
}

func (s *Situation) ChooseBranchCell1() *BranchChoices {
	for nums := int8(2); nums <= 9; nums++ {
		result := s.ChooseBranchCell1Nums(nums)
		if result.Size() > 0 {
			return result
		}
	}
	return nil
}

func (s *Situation) ChooseBranchCell1Nums(nums int8) *BranchChoices {
	expectingExcludes := int8(9 - nums)
	type Candidate struct {
		RowCol
		// numCd, rowCd, colCd, blockCd int
		Score int
	}
	selected := Candidate{
		RowCol: RowCol{-1, -1},
		Score:  1 << 30,
	}
	isBetter := func(candidate Candidate) bool {
		if candidate.Score != selected.Score {
			return candidate.Score < selected.Score
		}
		return s.RowColHash(candidate.RowCol) < s.RowColHash(selected.RowCol)
	}
	for r, rowBits := range s.numExcludeMask {
		for c, cellBits := range rowBits {
			cellNumExcludes := countTrueBits(cellBits)
			if cellNumExcludes != expectingExcludes {
				continue
			}
			b, _ := rcbp(int8(r), int8(c))
			setRow := int(s.rowSetCount[r])
			setCol := int(s.colSetCount[c])
			setBlock := int(s.blockSetCount[b])

			candidate := Candidate{
				RowCol: RowCol{int8(r), int8(c)},
				Score:  setRow + setCol + setBlock,
			}
			if isBetter(candidate) {
				selected = candidate
			}
		}
	}
	if selected.Row == -1 {
		return nil
	}
	r, c := selected.Row, selected.Col
	numExcludeBits := s.numExcludeMask[r][c]
	var tmpArray [9]int8
	candidateNums := tmpArray[:0]
	for n := range loop9 {
		if numExcludeBits&(1<<n) == 0 {
			candidateNums = append(candidateNums, int8(n))
		}
	}
	if nums == 2 && s.CompareNumInCell(r, c, candidateNums[1], candidateNums[0]) {
		candidateNums[0], candidateNums[1] = candidateNums[1], candidateNums[0]
	}

	result := NewBranchChoices()
	for _, n := range candidateNums {
		result.Add(RCN(r, c, n))
	}
	return result
}

// 选择哪个号码开始猜测，返回true表示n1比较好
func (s *Situation) CompareNumInCell(r, c, n1, n2 int8) bool {
	//我也不知道为啥这个指标会有效，只是测试结果表明，这样蒙对的概率更高
	score1 := int(s.numSetCount[n1])
	score2 := int(s.numSetCount[n2])
	if score1 != score2 {
		return score1 > score2
	}

	base := int(r)*61 + int(c)*67 + s.setCount*71
	return base*int(n1)%41 < base*int(n2)%41
}

func ShowCells(cells *[9][9]int8, title string, r, c int) {
	fmt.Println("=============================")
	fmt.Println(title)
	for r1 := range loop9 {
		for c1 := range loop9 {
			s := " "
			if n1 := cells[r1][c1]; n1 >= 0 {
				s = strconv.Itoa(int(n1 + 1))
			}
			if r1 == r && c1 == c {
				fmt.Printf("[%s]", s)
			} else {
				fmt.Printf(" %s ", s)
			}
			if c1 == 2 || c1 == 5 {
				fmt.Printf("|")
			}
		}
		fmt.Println()
		if r1 == 2 || r1 == 5 {
			fmt.Println("-----------------------------")
		}
	}
}

func (s *Situation) Hash() uint64 {
	raw := make([]byte, 9*9)
	for r := range loop9 {
		for c := range loop9 {
			raw[r*9+c] = byte(s.cells[r][c] + 1)
		}
	}

	return crc64.Checksum(raw, crc64.MakeTable(crc64.ISO))
}

func (s *Situation) Completed() bool {
	return s.setCount == 9*9
}

var triggerPool = sync.Pool{
	New: func() any {
		return &Trigger{
			confirms: NewQueueCapacity(30),
		}
	},
}

func NewTrigger() *Trigger {
	t := triggerPool.Get().(*Trigger)
	t.Init()
	return t
}

func ReleaseTrigger(t *Trigger) {
	triggerPool.Put(t)
}

func DuplicateTrigger(t *Trigger) *Trigger {
	t2 := NewTrigger()
	t2.CopyFrom(t)
	return t2
}

type Trigger struct {
	confirms  *Queue
	Conflicts []Conflict
}

func (t *Trigger) Init() {
	t.confirms.DiscardAll()
	t.Conflicts = t.Conflicts[:0]
}

func (t *Trigger) Confirm(rcn RowColNum) {
	t.confirms.Enqueue(rcn)
}

func (t *Trigger) GetConfirm() (RowColNum, bool) {
	return t.confirms.Dequeue()
}

func (t *Trigger) Conflict(conflictType int, rcn RowColNum) {
	t.Conflicts = append(t.Conflicts, Conflict{
		ConflictType: conflictType,
		RowColNum:    rcn,
	})
}

func (t *Trigger) CopyFrom(x *Trigger) {
	t.confirms.CopyFrom(x.confirms)
	t.Conflicts = append(t.Conflicts, x.Conflicts...)
}
