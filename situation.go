package main

import (
	"fmt"
	"hash/crc64"
	"strconv"
	"strings"
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
	cells [9][9]int

	//cellExclude[r][c][n] = 1 表示单元格(r,c)排除n
	cellExclude [9][9][9]int

	//numExcludes[n] 表示 n 的总排除次数
	numExcludes [9]int

	//cellNumExcludes[r][c] = x 表示 r 行 c 列 排除了 x 个数
	//等于 sum(cellExclude[r][c][...])
	cellNumExcludes [9][9]int

	//rowExcludes[r][n] = x 表示第 r 行的 n 排除了 x 个单元格
	//等于 sum(cellExclude[r][...][n])
	rowExcludes [9][9]int

	//colExcludes[c][n] = x 表示第 c 列的 n 排除了 x 个单元格
	//等于 sum(cellExclude[...][c][n])
	colExcludes [9][9]int

	//blockExcludes[R][C][n] = x 表示宫 (R,C) 的 n 排除了 x 个单元格
	blockExcludes [3][3][9]int

	setCount int
}

// 初始化一个数独谜题
// puzzle 是一个9行的字符串（前后空行会自动去除），以空格和数字代表单元格
// 其中空格代表未填入的单元格
func ParseSituation(puzzle string) (*Situation, *Trigger) {
	var s Situation
	for r := range loop9 {
		for c := range loop9 {
			s.cells[r][c] = -1
		}
	}
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
	return &s, t
}

func (s *Situation) Copy() *Situation {
	s2 := *s
	return &s2
}

func (s *Situation) Count() int {
	return s.setCount
}

//r 行 c 列填入 n
//同时会执行同一互斥组的排除（修改 s.cellExclude）
func (s *Situation) Set(t *Trigger, rcn RowColNum) bool {
	r, c, n := rcn.Row, rcn.Col, rcn.Num
	if s.cells[r][c] != -1 {
		return false
	}
	s.setCount++
	s.cells[r][c] = n

	for n0 := range loop9 {
		if n0 != n {
			s.Exclude(t, RCN(r, c, n0))
		}
	}
	for r0 := range loop9 {
		if r0 != r {
			s.Exclude(t, RCN(r0, c, n))
		}
	}
	for c0 := range loop9 {
		if c0 != c {
			s.Exclude(t, RCN(r, c0, n))
		}
	}
	R := r / 3
	C := c / 3
	for r0 := range loop3 {
		for c0 := range loop3 {
			rr := R*3 + r0
			cc := C*3 + c0
			if rr != r || cc != c {
				s.Exclude(t, RCN(rr, cc, n))
			}
		}
	}

	return true
}

func (s *Situation) Exclude(t *Trigger, rcn RowColNum) bool {
	r, c, n := rcn.Row, rcn.Col, rcn.Num
	if s.cellExclude[r][c][n] != 0 {
		return false
	}
	s.cellExclude[r][c][n] = 1
	s.numExcludes[n]++

	switch add(&s.cellNumExcludes[r][c], 1) {
	case 8:
		n1 := 0
		for n0 := range loop9 {
			n1 += n0 * (1 - s.cellExclude[r][c][n0])
		}
		t.Confirm(RCN(r, c, n1))
	case 9:
		t.Conflict(fmt.Sprintf("单元格(%d,%d)没有可填充数字", r+1, c+1))
	}

	switch add(&s.rowExcludes[r][n], 1) {
	case 8:
		c1 := 0
		for c0 := range loop9 {
			c1 += c0 * (1 - s.cellExclude[r][c0][n])
		}
		t.Confirm(RCN(r, c1, n))
	case 9:
		t.Conflict(fmt.Sprintf("第 %d 行没有单元格可填充 %d", r+1, n+1))
	}

	switch add(&s.colExcludes[c][n], 1) {
	case 8:
		r1 := 0
		for r0 := range loop9 {
			r1 += r0 * (1 - s.cellExclude[r0][c][n])
		}
		t.Confirm(RCN(r1, c, n))
	case 9:
		t.Conflict(fmt.Sprintf("第 %d 列没有单元格可填充 %d", c+1, n+1))
	}

	switch R,C := r/3,c/3; add(&s.blockExcludes[R][C][n], 1) {
	case 8:
		r1 := 0
		c1 := 0
		for r0 := range loop3 {
			for c0 := range loop3 {
				cellMatched := 1 - s.cellExclude[R*3+r0][C*3+c0][n]
				r1 += r0 * cellMatched
				c1 += c0 * cellMatched
			}
		}
		t.Confirm(RCN(R*3+r1, C*3+c1, n))
	case 9:
		t.Conflict(fmt.Sprintf("宫(%d,%d)没有单元格可填充 %d", R+1, C+1, n+1))
	}

	return true
}

func (s *Situation) ExcludeByRules(t *Trigger) {
	//占位排除法（列）
	s.ApplyRuleMultiStandCol(t)

	//占位排除法（行）
	s.ApplyRuleMultiStandRow(t)

	//X-Wing（行）
	s.ApplyRuleXWingRow(t)

	//X-Wing（列）
	s.ApplyRuleXWingCol(t)
}

// 获取当前无法排除的所有填充选项。
func (s *Situation) Choices() []*GuessItem {
	m := map[RowCol][]int{}
	for r := range loop9 {
		for c := range loop9 {
			if s.cells[r][c] >= 0 {
				continue
			}

			cell := RowCol{r, c}
			for n := range loop9 {
				if s.cellExclude[r][c][n] == 0 {
					m[cell] = append(m[cell], n)
				}
			}
		}
	}

	result := make([]*GuessItem, 0)
	for rc, nums := range m {
		result = append(result, &GuessItem{
			RowCol: rc,
			Nums:   nums,
		})
	}
	return result
}

func (s *Situation) Show(title string, r, c int) {
	lines := strings.Split(title, "\n")
	lines[0] = fmt.Sprintf("<%02d> %s", s.setCount, lines[0])
	for i := 1 ; i < len(lines); i++ {
		lines[i] = "     " + lines[i]
	}
	title = strings.Join(lines, "\n")
	ShowCells(&s.cells, title, r, c)
}

func (s *Situation) CompareGuestItem(c1, c2 *GuessItem) bool {
	//Nums数量少的优先
	if len(c1.Nums) != len(c2.Nums) {
		return len(c1.Nums) < len(c2.Nums)
	}
	//随便一个吧
	return c1.Hash() < c2.Hash()
}

func (s *Situation) CompareNumInCell(rc RowCol, n1, n2 int) bool {
	score1 := s.numExcludes[n1]
	score2 := s.numExcludes[n2]
	//score越高，说明这个数填得多，或占据更关键的位置，选这个数可能更快结束分支演算
	if score1 != score2 {
		return score1 > score2
	}
	base := rc.Row * 4 + rc.Col * 7
	return (base + n1) % 9 < (base + n2) % 9}

func ShowCells(cells *[9][9]int, title string, r, c int) {
	fmt.Println("=============================")
	fmt.Println(title)
	for r1 := range loop9 {
		for c1 := range loop9 {
			s := " "
			if n1 := cells[r1][c1]; n1 >= 0 {
				s = strconv.Itoa(n1 + 1)
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
		rc.Row + r,
		rc.Col + c,
	}
}

func (rc RowCol) Hash() int {
	return (rc.Row*277 + rc.Col*659) % 997
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

type Trigger struct {
	Confirms  []RowColNum
	Conflicts []string
}

func (t *Trigger) Confirm(rcn RowColNum) {
	if t == nil {
		return
	}
	t.Confirms = append(t.Confirms, rcn)
}

func (t *Trigger) Conflict(msg string) {
	if t == nil {
		return
	}
	t.Conflicts = append(t.Conflicts, msg)
}

type GuessItem struct {
	RowCol
	Nums []int
}

func add(p *int, n int) int {
	*p += n
	return *p
}
