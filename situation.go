package main

import (
	"fmt"
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

	showCount int
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

//r 行 c 列填入 n
//同时会执行同一互斥组的排除（修改 s.cellExclude）
func (s *Situation) Set(r, c, n int) bool {
	if s.cells[r][c] != -1 {
		panic(fmt.Errorf("reseting cell(%d,%d)=%d", r, c, n))
	}
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

func (s *Situation) exclude(r, c, n int) bool {
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
	ShowCells(&s.cells, fmt.Sprintf("<%d> %s", s.showCount, title), r, c)
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

// 从当前局势 s 创建一个"排除"实例 Excluding
// Excluding 用于找到可能的填充单元格选项
// 给定一系列（9个）选项， 每个选项调用一次 Test(r,c,n)
// 最后调用 Apply，如果 9 个选项有且只有 1 个合理，则填入到 s 中。
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
// Apply 返回三种可能：
// - A、所有选项都不合理，发生了矛盾。
// - B、唯一合理的选项，则填入到局势中。
// - C、唯一合理的选项，但单元格早前已经被填入该数字。
// - D、唯一合理的选项，但单元格早前已经被填入不同的数字，发生了矛盾。
// - E、超过一个可能选项，不能确定。
// 返回值：
// - done = true: 情况B。局势 s 已经填入数字。只有选项 B 会返回 done=true，
// - changed = true：情况B和部分的E，局势 s 发生了改变。和 done 不同的是，即使没有填入数字也可能发生了改变，因为排除了某些选项。
// - consistent = true：A、D 两种情况发生了矛盾，如果之前的所有推断没有错，这个局势显然无法完成。
// - applyCell ：对于B、C、D三种情况，返回该单元格的位置。
func (ex *Excluding) Apply() (done, changed, consistent bool, applyCell RowCol) {
	switch ex.matchedCount {
	case 0:
		//A
		return false, false, false, ex.selectedCell
	case 1:
		r, c, n := ex.selectedCell.Row, ex.selectedCell.Col, ex.selectedN
		switch ex.s.cells[r][c] {
		case -1:
			//B
			ex.s.Set(r, c, n)
			return true, true, true, ex.selectedCell
		case n:
			//C
			return false, false, true, ex.selectedCell //没有改变
		default:
			//D
			return false, false, false, ex.selectedCell
		}
	default:
		if !ex.sameNum {
			return false, false, true, ex.selectedCell //没有改变
		}

		//开始执行"复杂排除"。参见文件最后的解析。
		exChanged := false
		complexExclude := func(rc RowCol, n int) {
			if !ex.matchedCells[rc] {
				if excluded := &ex.s.cellExclude[rc.Row][rc.Col][n]; !*excluded {
					*excluded = true
					exChanged = true
				}
			}
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
		//E
		return false, exChanged, true, RowCol{-1, -1}
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

/*

=== 互斥组 ===

同一行、或同一列、或同一区块。

=== 复杂排除法 ===

虽然有多个可能填充选项，但这些选项是同一个数N，而且都在相同的互斥组中。
那么同一互斥组的其他单元格可以排除。

例如：

     1      |            | 5   #   4
     9   6  |         7  | #   #   #
            | 2       *  |     1
------------------------------------
            |            | 8       7
     8   5  |     6      |         2
         4  |            |
------------------------------------
     3      |            |     9
         9  |     3      |         5
            | 5   4      |     6

留意右上区块，根据其他行列排除法，标记"#"的单元格都不能是 6，所以 6 必定在同一区块剩余的两个单元格中。
刚好这两个选项同在另外一个互斥体（第 3 行），所以第 3 行其他单元格都不能是 6。
标记 "*" 的单元格因此排除6，你没有其他简单的方法可以作出这个断言。

*/
