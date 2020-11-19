package main

import (
	"fmt"
	"path"
	"sort"
	"strings"
)

type SudokuContext struct {
	evalCount int
}

func newSudokuContext() *SudokuContext {
	return &SudokuContext{}
}

// recurseEval 开始推断局势 s，并返回所有可能的终局。
// 如果返回nil，表示这个局势有矛盾，不存在正确的解答。
func (ctx *SudokuContext) recurseEval(s *Situation, t *Trigger, branchPath string) []*[9][9]int {
	if *flagShowBranch {
		fmt.Println(branchPath, "开始")
	}
	if !ctx.logicalEval(s, t) {
		if *flagShowBranch {
			fmt.Println(branchPath, fmt.Sprintf("演算到 <%d> 发生矛盾", s.Count()))
		}
		return nil
	}

	completed := s.Completed()
	if completed {
		if *flagShowBranch {
			fmt.Println(branchPath, "到达解")
		}
		cells := s.cells
		return []*[9][9]int{&cells}
	}

	//当前没有找到确定的填充选项，所以获取所有可能选项，然后在所有可能的选项里选一个单元格做尝试。

	//获取所有可能的选项
	choices := s.Choices()
	if len(choices) == 0 {
		return nil
	}
	try := choices[0]
	for _, c := range choices {
		if s.CompareGuestItem(c, try) {
			try = c
		}
	}
	sort.Slice(try.Nums, func(i, j int) bool {
		return s.CompareNumInCell(try.RowCol, try.Nums[i], try.Nums[j])
	})

	result := make([]*[9][9]int, 0)
	for _, n := range try.Nums {
		s2 := s.Copy()
		t = &Trigger{}
		s2.Set(t, RCN(try.Row, try.Col, n))
		ctx.evalCount++
		if !*flagShowOnlyResult {
			s2.Show("在可能的选项里猜一个", try.Row, try.Col)
		}
		subResult := ctx.recurseEval(s2, t, path.Join(branchPath,
			fmt.Sprintf("<%d>(%d,%d)=%d", s2.Count(), try.Row+1, try.Col+1, n+1)))
		result = append(result, subResult...)
		if len(result) > 0 && *flagShowStopAtFirst {
			break
		}
	}
	if *flagShowBranch {
		fmt.Println(branchPath, fmt.Sprintf("找到 %d 个解", len(result)))
	}
	return result
}

// logicalEval 开始推断局势 s，直到没有找到确定的填充选项，不确保全部完成。
// 如果返回false，表示这个局势有矛盾。
func (ctx *SudokuContext) logicalEval(s *Situation, t *Trigger) bool {
	checkConflicts := func(t1 *Trigger) bool {
		if len(t1.Conflicts) > 0 {
			if !*flagShowOnlyResult {
				fmt.Println("发生矛盾：")
				for _, msg := range t1.Conflicts {
					fmt.Println(msg)
				}
			}
			return false
		}
		return true
	}
	if !checkConflicts(t) {
		return false
	}
	for len(t.Confirms) > 0 || len(t.Conflicts) > 0 {
		next := &Trigger{}
		for _, rcn := range t.Confirms {
			cellNumExcludes := s.cellNumExcludes[rcn.Row][rcn.Col]
			rowExcludes := s.rowExcludes[rcn.Row][rcn.Num]
			colExcludes := s.colExcludes[rcn.Col][rcn.Num]
			blockExcludes := s.blockExcludes[rcn.Row/3][rcn.Col/3][rcn.Num]
			if s.Set(next, rcn) {
				ctx.evalCount++
				if !*flagShowOnlyResult {
					title := ""
					if cellNumExcludes == 8 {
						title += fmt.Sprintf("单元格唯一可以填的数\n")
					}
					if rowExcludes == 8 {
						title += fmt.Sprintf("该行唯一可以填 %d 的位置\n", rcn.Num+1)
					}
					if colExcludes == 8 {
						title += fmt.Sprintf("该列唯一可以填 %d 的位置\n", rcn.Num+1)
					}
					if blockExcludes == 8 {
						title += fmt.Sprintf("该区块唯一可以填 %d 的位置\n", rcn.Num+1)
					}
					s.Show(strings.TrimSuffix(title, "\n"), rcn.Row, rcn.Col)
				}
			}
			if !checkConflicts(next) {
				return false
			}
		}
		if len(next.Confirms) == 0 {
			s.ExcludeByRules(next)
		}
		t = next
	}
	if s.Completed() && !*flagShowOnlyResult {
		fmt.Println("找到了一个解")
	}
	return true
}
