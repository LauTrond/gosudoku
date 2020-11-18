package main

import (
	"fmt"
	"hash/crc64"
	"strconv"
	"strings"
)

var (
	loop9 [9]bool
	loop3 [3]bool
)

// Situation 代表9*9数独的一个局势
// 所有行、列、单元格值都是0~8，显示的时候加1处理。
type Situation struct {
	//cellExclude[r][c][n] = true 表示单元格(r,c)不可能是n
	cellExclude [9][9][9]bool

	//cell[r][c] = n
	//n >=0 : 单元格(r,c) 填入了数字n
	//n == -1 : 单元格(r,c) 还没填入数字
	cells [9][9]int

	setCount int
}

// 初始化一个数独谜题
// puzzle 是一个9行的字符串（前后空行会自动去除），以空格和数字代表单元格
// 其中空格代表未填入的单元格
func ParseSituation(puzzle string) *Situation {
	var s Situation
	for r := range loop9 {
		for c := range loop9 {
			s.cells[r][c] = -1
		}
	}
	lines := strings.Split(strings.Trim(puzzle, "\n"), "\n")
	for r, line := range lines {
		if r > len(s.cells) {
			panic(fmt.Errorf("row exceed"))
		}
		for c, n := range line {
			if n >= '1' && n <= '9' {
				s.Set(r, c, int(n-'1'))
			}
		}
	}
	return &s
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
func (s *Situation) Set(r, c, n int) bool {
	if s.cells[r][c] != -1 {
		panic(fmt.Errorf("reseting cell(%d,%d)=%d", r, c, n))
	}
	s.setCount++
	s.cells[r][c] = n
	R := r / 3
	C := c / 3

	for n0 := range loop9 {
		if n0 != n {
			s.cellExclude[r][c][n0] = true
		}
	}
	for r0 := range loop9 {
		if r0 != r {
			s.cellExclude[r0][c][n] = true
		}
	}
	for c0 := range loop9 {
		if c0 != c {
			s.cellExclude[r][c0][n] = true
		}
	}

	for r0 := range loop3 {
		for c0 := range loop3 {
			rr := R*3 + r0
			cc := C*3 + c0
			if rr != r || cc != c {
				s.cellExclude[rr][cc][n] = true
			}
		}
	}
	return true
}

func (s *Situation) Exclude(r, c, n int) bool {
	if s.cellExclude[r][c][n] {
		return false
	}
	s.cellExclude[r][c][n] = true
	return true
}

type GuessItem struct {
	RowCol
	Nums []int
}

func compareGuestItem(c1, c2 *GuessItem) bool {
	//Nums数量少的优先
	if len(c1.Nums) != len(c2.Nums) {
		return len(c1.Nums) < len(c2.Nums)
	}
	//随便一个吧
	return c1.Hash() < c2.Hash()
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
				if !s.cellExclude[r][c][n] {
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
	ShowCells(&s.cells, fmt.Sprintf("<%d> %s", s.setCount, title), r, c)
}

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
	raw := make([]byte, 9 * 9)
	for r := range loop9 {
		for c := range loop9 {
			raw[r * 9 + c] = byte(s.cells[r][c]+1)
		}
	}

	return crc64.Checksum(raw, crc64.MakeTable(crc64.ISO))
}

func (s *Situation) Completed() bool {
	return s.setCount == 9 * 9
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
	return (rc.Row * 277 + rc.Col * 659) % 997
}
