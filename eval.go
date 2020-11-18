package main

import "fmt"

type SudokuContext struct {
	triedSituation map[uint64]bool
}

func newSudokuContext() *SudokuContext {
	return &SudokuContext{
		triedSituation: map[uint64]bool{},
	}
}

// recurseEval 开始推断局势 s，并返回所有可能的终局。
// 如果返回nil，表示这个局势有矛盾，不存在正确的解答。
func (c *SudokuContext) recurseEval(s *Situation) []*[9][9]int {
	if h := s.Hash(); c.triedSituation[h] {
		return nil
	} else {
		c.triedSituation[h] = true
	}

	if !c.logicalEval(s) {
		return nil
	}

	completed := s.Completed()
	if completed {
		cells := s.cells
		return []*[9][9]int{&cells}
	}

	//当前没有找到确定的填充选项，所以获取所有可能选项，然后在所有可能的选项里选一个单元格做尝试。

	result := make([]*[9][9]int, 0)
	//获取所有可能的选项
	choices := s.Choices()
	if len(choices) == 0 {
		return nil
	}

	try := choices[0]
	for _, c := range choices {
		if compareGuestItem(c, try) {
			try = c
		}
	}

	tryNumsForShow := make([]int, len(try.Nums))
	for i, n := range try.Nums {
		tryNumsForShow[i] = n + 1
	}

	for _, n := range try.Nums {
		s2 := s.Copy()
		s2.Set(try.Row, try.Col, n)
		if !*flagShowOnlyResult {
			s2.Show(fmt.Sprintf("在可能的选项里猜一个（全部选项：%v）", tryNumsForShow), try.Row, try.Col)
		}
		subResult := c.recurseEval(s2)
		result = append(result, subResult...)
		if len(result) > 0 && *flagShowStopAtFirst {
			break
		}
	}

	return result
}

// logicalEval 开始推断局势 s，直到没有找到确定的填充选项，不确保全部完成。
// 如果返回false，表示这个局势有矛盾。
func (c *SudokuContext) logicalEval(s *Situation) bool {
	for {
		changed := false
		for r := range loop9 {
			for n := range loop9 {
				ex := NewExcluding(s)
				for c := range loop9 {
					ex.Test(r, c, n)
				}
				done, changed2, consistency, cell := ex.Apply()
				if !consistency {
					if !*flagShowOnlyResult {
						fmt.Printf("发生矛盾：第 %d 行没有单元格可以填入 %d\n", r+1, n+1)
					}
					return false
				}
				changed = changed || changed2
				if done {
					if !*flagShowOnlyResult {
						s.Show(fmt.Sprintf("该行唯一可以填入 %d 的单元格", n+1),
							cell.Row, cell.Col)
					}
				}
			}
		}
		for c := range loop9 {
			for n := range loop9 {
				ex := NewExcluding(s)
				for r := range loop9 {
					ex.Test(r, c, n)
				}
				done, changed2, consistency, cell := ex.Apply()
				if !consistency {
					if !*flagShowOnlyResult {
						fmt.Printf("发生矛盾：第 %d 列没有单元格可以填入 %d\n", c+1, n+1)
					}
					return false
				}
				changed = changed || changed2
				if done {
					if !*flagShowOnlyResult {
						s.Show(fmt.Sprintf("该列唯一可以填入 %d 的单元格", n+1),
							cell.Row, cell.Col)
					}
				}
			}
		}
		for R := range loop3 {
			for C := range loop3 {
				for n := range loop9 {
					ex := NewExcluding(s)
					for rr := range loop3 {
						for cc := range loop3 {
							r := R*3 + rr
							c := C*3 + cc
							ex.Test(r, c, n)
						}
					}
					done, changed2, consistency, cell := ex.Apply()
					if !consistency {
						if !*flagShowOnlyResult {
							fmt.Printf("发生矛盾：区块(%d行,%d列)内没有单元格可以填入 %d\n", R+1, C+1, n+1)
						}
						return false
					}
					changed = changed || changed2
					if done {
						if !*flagShowOnlyResult {
							s.Show(fmt.Sprintf("区块内唯一可以填入 %d 的单元格", n+1),
								cell.Row, cell.Col)
						}
					}
				}
			}
		}
		for r := range loop9 {
			for c := range loop9 {
				ex := NewExcluding(s)
				for n := range loop9 {
					ex.Test(r, c, n)
				}
				done, changed2, consistent, cell := ex.Apply()
				if !consistent {
					if !*flagShowOnlyResult {
						fmt.Printf("发生矛盾：%d 行 %d 列单元格没有可以填入的数字\n", r+1, c+1)
					}
					return false
				}
				changed = changed || changed2
				if done {
					if !*flagShowOnlyResult {
						s.Show("这个单元格唯一可以填的数", cell.Row, cell.Col)
					}
				}
			}
		}
		if !changed {
			break
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
func NewExcluding(s *Situation) *Excluding {
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
