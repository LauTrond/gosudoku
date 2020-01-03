package main

import (
	"fmt"
	"sort"
	"strings"
)

var (
	loop9 [9]bool
	loop3 [3]bool
)

type Situation struct {
	//cellExclude[r][c][n] = true
	//means cell(r,c) cannot be n
	cellExclude [9][9][9]bool

	//cell[r][c] = n
	//n >=0 : cell(r,c) is n
	//n == -1 : cell(r,c) not decided
	cells [9][9]int

	showCount int
}

func NewSituation(puzzle string) *Situation {
	var s Situation
	for r := range loop9 {
		for c := range loop9 {
			s.cells[r][c] = -1
		}
	}
	lines := strings.Split(strings.Trim(puzzle, "\n"), "\n")
	for r, line := range lines {
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

//设置r行c列是n
func (s *Situation) Set(r, c, n int) bool {
	if s.cells[r][c] == n {
		return false
	}
	s.cells[r][c] = n
	R := r / 3
	C := c / 3

	for n0 := range loop9 {
		if n0 != n {
			s.cellExclude[r][c][n0] = true
			//s.Exclude(r, c, n0)
		}
	}
	for r0 := range loop9 {
		if r0 != r {
			s.cellExclude[r0][c][n] = true
			//s.Exclude(r0, c, n)
		}
	}
	for c0 := range loop9 {
		if c0 != c {
			s.cellExclude[r][c0][n] = true
			//s.Exclude(r, c0, n)
		}
	}

	for r0 := range loop3 {
		for c0 := range loop3 {
			rr := R*3 + r0
			cc := C*3 + c0
			if rr != r || cc != c {
				s.cellExclude[rr][cc][n] = true
				//s.Exclude(rr, cc, n)
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

func (s *Situation) GuessChoices() []*GuessItem {
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
	sort.Slice(result, func(i, j int) bool {
		return len(result[i].Nums) < len(result[j].Nums)
	})
	return result
}

func (s *Situation) Show(title string, r, c int) {
	ShowCells(&s.cells, fmt.Sprintf("%d %s", s.showCount, title), r, c)
	s.showCount++
}

func ShowCells(cells *[9][9]int, title string, r, c int) {
	fmt.Println("====================================")
	fmt.Println(title)
	for r1 := range loop9 {
		for c1 := range loop9 {
			if n1 := cells[r1][c1]; n1 >= 0 {
				if r1 == r && c1 == c {
					fmt.Printf("[%d] ", n1+1)
				} else {
					fmt.Printf(" %d  ", n1+1)
				}
			} else {
				fmt.Printf("    ")
			}
			if c1 == 2 || c1 == 5 {
				fmt.Printf("|")
			}
		}
		fmt.Println()
		if r1 == 2 || r1 == 5 {
			fmt.Println("------------------------------------")
		}
	}
}

func (s *Situation) Completed() bool {
	for r := range loop9 {
		for c := range loop9 {
			if s.cells[r][c] < 0 {
				return false
			}
		}
	}
	return true
}

type Excluding struct {
	s            *Situation
	matchedCells map[RowCol]bool
	matchedCount int
	selectedCell RowCol
	selectedN    int
	sameBlock    bool
	sameRow      bool
	sameCol      bool
	sameNum      bool
}

func (s *Situation) NewExcluding() *Excluding {
	return &Excluding{
		s:            s,
		matchedCells: map[RowCol]bool{},
		matchedCount: 0,
		selectedCell: RowCol{-1, -1},
		selectedN:    -1,
		sameBlock:    true,
		sameRow:      true,
		sameCol:      true,
		sameNum:      true,
	}
}

func (ex *Excluding) Test(r, c, n int) {
	cell := RowCol{r, c}
	if !ex.s.cellExclude[cell.Row][cell.Col][n] {
		if ex.matchedCount > 0 {
			if ex.selectedCell.Row != cell.Row {
				ex.sameRow = false
			}
			if ex.selectedCell.Col != cell.Col {
				ex.sameCol = false
			}
			if ex.selectedCell.Block() != cell.Block() {
				ex.sameBlock = false
			}
			if ex.selectedN != n {
				ex.sameNum = false
			}
		}
		ex.selectedCell = cell
		ex.selectedN = n
		ex.matchedCells[cell] = true
		ex.matchedCount++
	}
}

func (ex *Excluding) Apply() (done, changed, consistent bool, applyCell RowCol) {
	switch ex.matchedCount {
	case 0:
		return false, false, false, ex.selectedCell
	case 1:
		r, c, n := ex.selectedCell.Row, ex.selectedCell.Col, ex.selectedN
		if ex.s.cells[r][c] < 0 {
			ex.s.Set(r, c, n)
			return true, true, true, ex.selectedCell
		}
		if ex.s.cells[r][c] != n {
			return false, false, false, ex.selectedCell
		}
		return false, false, true, ex.selectedCell //没有改变
	default:
		if !ex.sameNum {
			return false, false, true, ex.selectedCell //没有改变
		}

		changed := false

		complexExclude := func(rc RowCol, n int) {
			changed1 := false
			if !ex.matchedCells[rc] {
				changed1 = changed1 || ex.s.Exclude(rc.Row, rc.Col, n)
			}
			if changed1 {
				fmt.Printf("complexExclude(%d,%d):%d\n", rc.Row, rc.Col, n + 1)
			}
			changed = changed || changed1
		}

		if ex.sameRow {
			for c := range loop9 {
				complexExclude(RowCol{ex.selectedCell.Row, c}, ex.selectedN)
			}
		}
		if ex.sameCol {
			for r := range loop9 {
				complexExclude(RowCol{r, ex.selectedCell.Col}, ex.selectedN)
			}
		}
		if ex.sameBlock {
			for r := range loop3 {
				for c := range loop3 {
					complexExclude(ex.selectedCell.Block().LeftTop().Add(r, c), ex.selectedN)
				}
			}
		}
		return false, changed, true, RowCol{-1, -1}
	}
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