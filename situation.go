package main

import (
	"fmt"
	"hash/crc64"
	"sort"
	"strconv"
	"strings"
	"sync"
)

var (
	loop9     [9]int
	loop3     [3]int
	loop3skip = [3][2]int{{1, 2}, {0, 2}, {0, 1}}
	loop9skip = [3][6]int{
		{3, 4, 5, 6, 7, 8},
		{0, 1, 2, 6, 7, 8},
		{0, 1, 2, 3, 4, 5},
	}
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

	//cellExclude[n][r][c] = 1 ： 单元格(r,c)排除n
	cellExclude [9][9][9]int8

	//numExcludes[n] ： n 的总排除次数
	numExcludes [9]int8

	//cellNumExcludes[r][c] = x ： r 行 c 列 排除了 x 个数
	//等于 sum(cellExclude[...][r][c])
	cellNumExcludes [9][9]int8

	//rowExcludes[n][r] = x ： r 行的 n 排除了 x 个单元格
	//等于 sum(cellExclude[n][r][...])
	rowExcludes [9][9]int8

	//rowExcludes[n][r][C] = x ： 第 r 行 C 宫的 n 排除了 x 个单元格
	//rowExcludes[n][r][C] = sum(cellExclude[n][r][C*3..C*3+2])
	rowSliceExcludes [9][9][3]int8

	//colExcludes[n][c] = x ：c 列的 n 排除了 x 个单元格
	//colExcludes[n][c] = sum(cellExclude[n][...][c])
	colExcludes [9][9]int8

	//colSliceExcludes[n][R][c] = x ： c 列 R 宫的 n 排除了 x 个单元格
	//colSliceExcludes[n][R][c] = sum(cellExclude[n][R*3..R*3+2][c])
	colSliceExcludes [9][3][9]int8

	//blockExcludes[n][R][C] = x ：宫 (R,C) 的 n 排除了 x 个单元格
	blockExcludes [9][3][3]int8

	//以下是策略排除可能用到的参数

	//分支代数，每执行一次Copy就加1
	branchGeneration int
}

// 初始化一个数独谜题
// puzzle 是一个9行的字符串（前后空行会自动去除），以点和数字代表单元格
// 其中空格代表未填入的单元格
func ParseSituation(puzzle string) (*Situation, *Trigger) {
	s := NewSituation()
	lines := strings.Split(strings.Trim(puzzle, "\n"), "\n")
	t := &Trigger{}
	for r, line := range lines {
		if r > len(s.cells) {
			panic(fmt.Errorf("row exceed"))
		}
		for c, n := range line {
			if n >= '1' && n <= '9' {
				s.Set(t, RCN(r, c, int(n-'1')))
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
			s.Set(t, RCN(i/9, i%9, int(n-'1')))
		}
	}
	return s, t
}

var situationPool = sync.Pool{
	New: func() interface{} {
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

func (s *Situation) Release() {
	situationPool.Put(s)
}

func (s *Situation) Copy() *Situation {
	s2 := situationPool.Get().(*Situation)
	*s2 = *s
	s2.branchGeneration++
	return s2
}

func (s *Situation) Count() int {
	return s.setCount
}

//填数
func (s *Situation) Set(t *Trigger, rcn RowColNum) bool {
	r, c, n := rcn.Extract()
	if s.cells[r][c] != -1 {
		return false
	}

	R, C := r/3, c/3
	s.setCount++
	s.numSetCount[n]++
	s.cells[r][c] = int8(n)

	for n0 := range loop9 {
		if n0 != n {
			s.Exclude(t, RCN(r, c, n0))
		}
	}
	for _, r0 := range loop9skip[R] {
		s.Exclude(t, RCN(r0, c, n))
	}
	for _, c0 := range loop9skip[C] {
		s.Exclude(t, RCN(r, c0, n))
	}
	for rr0 := range loop3 {
		for cc0 := range loop3 {
			r0 := R*3 + rr0
			c0 := C*3 + cc0
			if r0 != r || c0 != c {
				s.Exclude(t, RCN(r0, c0, n))
			}
		}
	}
	return true
}

func (s *Situation) Exclude(t *Trigger, rcn RowColNum) bool {
	r, c, n := rcn.Extract()
	if s.cellExclude[n][r][c] != 0 {
		return false
	}
	s.cellExclude[n][r][c] = 1

	R, C := r/3, c/3
	rr, cc := r-R*3, c-C*3

	_ = add(&s.numExcludes[n], 1)
	cellNumExcludes := add(&s.cellNumExcludes[r][c], 1)
	rowExcludes := add(&s.rowExcludes[n][r], 1)
	colExcludes := add(&s.colExcludes[n][c], 1)
	blockExcludes := add(&s.blockExcludes[n][R][C], 1)
	rowSliceExcludes := add(&s.rowSliceExcludes[n][r][C], 1)
	colSliceExcludes := add(&s.colSliceExcludes[n][R][c], 1)

	switch cellNumExcludes {
	case 8:
		if s.cells[r][c] >= 0 {
			break
		}
		for n0 := range loop9 {
			if s.cellExclude[n0][r][c] == 0 {
				t.Confirm(RCN(r, c, n0))
				break
			}
		}
	case 9:
		reason := ""
		if !*flagShowOnlyResult {
			reason = fmt.Sprintf("单元格(%d,%d)没有可填充数字", r+1, c+1)
		}
		t.Conflict(reason)
	}

	switch rowExcludes {
	case 8:
		for c0 := range loop9 {
			if s.cellExclude[n][r][c0] == 0 && s.cells[r][c0] < 0 {
				t.Confirm(RCN(r, c0, n))
				break
			}
		}
	case 9:
		reason := ""
		if !*flagShowOnlyResult {
			reason = fmt.Sprintf("第 %d 行没有单元格可填充 %d", r+1, n+1)
		}
		t.Conflict(reason)
	}

	switch colExcludes {
	case 8:
		for r0 := range loop9 {
			if s.cellExclude[n][r0][c] == 0 && s.cells[r0][c] < 0 {
				t.Confirm(RCN(r0, c, n))
				break
			}
		}
	case 9:
		reason := ""
		if !*flagShowOnlyResult {
			reason = fmt.Sprintf("第 %d 列没有单元格可填充 %d", c+1, n+1)
		}
		t.Conflict(reason)
	}

	switch blockExcludes {
	case 8:
		loopRow: for r0 := range loop3 {
			for c0 := range loop3 {
				if s.cellExclude[n][R*3+r0][C*3+c0] == 0 && s.cells[R*3+r0][C*3+c0] < 0 {
					t.Confirm(RCN(R*3+r0, C*3+c0, n))
					break loopRow
				}
			}
		}
	case 9:
		reason := ""
		if !*flagShowOnlyResult {
			reason = fmt.Sprintf("宫(%d,%d)没有单元格可填充 %d", R+1, C+1, n+1)
		}
		t.Conflict(reason)
	}

	if rowExcludes == 6 || rowExcludes == 7 {
		for _, C0 := range loop3skip[C] {
			C1 := 3 - C - C0
			if rowSliceExcludes+s.rowSliceExcludes[n][r][C0] == 6 {
				for _, rr1 := range loop3skip[rr] {
					for cc1 := range loop3 {
						s.Exclude(t, RCN(R*3+rr1, C1*3+cc1, n))
					}
				}
			}
		}

	}

	if colExcludes == 6 || colExcludes == 7 {
		for _, R0 := range loop3skip[R] {
			R1 := 3 - R - R0
			if colSliceExcludes+s.colSliceExcludes[n][R0][c] == 6 {
				for rr1 := range loop3 {
					for _, cc1 := range loop3skip[cc] {
						s.Exclude(t, RCN(R1*3+rr1, C*3+cc1, n))
					}
				}
			}
		}
	}

	if blockExcludes == 6 || blockExcludes == 7 {
		for _, rr0 := range loop3skip[rr] {
			rr1 := 3 - rr - rr0
			if rowSliceExcludes+s.rowSliceExcludes[n][R*3+rr0][C] == 6 {
				for _, c0 := range loop9skip[C] {
					s.Exclude(t, RCN(R*3+rr1, c0, n))
				}
			}
		}
		for _, cc0 := range loop3skip[cc] {
			cc1 := 3 - cc - cc0
			if colSliceExcludes+s.colSliceExcludes[n][R][C*3+cc0] == 6 {
				for _, r0 := range loop9skip[R] {
					s.Exclude(t, RCN(r0, C*3+cc1, n))
				}
			}
		}
	}

	return true
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

func (s *Situation) ChooseGuessingCell() GuessItem {
	var (
		rSel, cSel int
		nums []int8
	)

	isBetter := func(r, c int) bool {
		if s.cellNumExcludes[r][c] != s.cellNumExcludes[rSel][cSel] {
			return s.cellNumExcludes[r][c] > s.cellNumExcludes[rSel][cSel]
		}
		return (r*277 + c*659) % 997 < (rSel*277 + cSel*659) % 997
	}

	for r := range loop9 {
		for c := range loop9 {
			if s.cellNumExcludes[r][c] >= 8 {
				continue
			}
			if nums == nil || isBetter(r, c) {
				rSel, cSel = r, c
				nums = nums[0:0]
				for n := range loop9 {
					if s.cellExclude[n][r][c] == 0 {
						nums = append(nums, int8(n))
					}
				}
			}
		}
	}

	sort.Slice(nums, func(i, j int) bool {
		return s.CompareNumInCell(rSel,cSel, int(nums[i]), int(nums[j]))
	})

	return GuessItem{
		RowCol: RowCol{Row: int(rSel), Col: int(cSel)},
		Nums: nums,
	}
}

//选择哪个号码开始猜测，返回true表示n1比较好
func (s *Situation) CompareNumInCell(r, c, n1, n2 int) bool {
	//我也不知道为啥这个指标会有效，只是测试结果表明，这样蒙对的概率更高
	score1 := int(s.numExcludes[n1])
	score2 := int(s.numExcludes[n2])
	if score1 != score2 {
		return score1 < score2
	}

	base := r*4 + c*7
	return (base+n1)%9 < (base+n2)%9
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

type RowCol struct {
	Row, Col int
}

func (rc RowCol) Block() RowCol {
	return RowCol{
		rc.Row / 3,
		rc.Col / 3,
	}
}

func (rc RowCol) LeftTop() RowCol {
	return RowCol{
		rc.Row * 3,
		rc.Col * 3,
	}
}

func (rc RowCol) Add(r, c int) RowCol {
	return RowCol{
		rc.Row + int(r),
		rc.Col + int(c),
	}
}

func (rc RowCol) Hash() int {
	return (int(rc.Row)*277 + int(rc.Col)*659) % 997
}

type RowColNum struct {
	RowCol
	Num int
}

func RCN(r, c, n int) RowColNum {
	return RowColNum{
		RowCol: RowCol{
			Row: int(r),
			Col: int(c),
		},
		Num: int(n),
	}
}

func (rcn RowColNum) Extract() (r, c, n int) {
	return int(rcn.Row), int(rcn.Col), int(rcn.Num)
}

type Trigger struct {
	Confirms  []RowColNum
	Conflicts []string
}

func NewTrigger() *Trigger {
	return &Trigger{
		Confirms: make([]RowColNum, 0, 8),
	}
}

func (t *Trigger) Init() {
	t.Confirms = t.Confirms[:0]
	t.Conflicts = nil
}

func (t *Trigger) Confirm(rcn RowColNum) {
	t.Confirms = append(t.Confirms, rcn)
}

func (t *Trigger) Conflict(msg string) {
	t.Conflicts = append(t.Conflicts, msg)
}

type GuessItem struct {
	RowCol
	Nums []int8
}

func add(p *int8, n int8) int8 {
	*p += n
	return *p
}
