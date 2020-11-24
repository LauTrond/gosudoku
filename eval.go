package main

import (
	"fmt"
	"path"
	"strings"
)

type SudokuContext struct {
	evalCount int
	guessesCount int
	results []*[9][9]int8
}

func newSudokuContext() *SudokuContext {
	return &SudokuContext{}
}

func (ctx *SudokuContext) Run(s *Situation, t *Trigger) int {
	if len(t.Conflicts) > 0 {
		if !*flagShowOnlyResult {
			fmt.Println("开局矛盾：")
			for _, msg := range t.Conflicts {
				fmt.Println(msg)
			}
		}
		return 0
	}
	return ctx.recurseEval(s, t, "/")
}

// recurseEval 开始推断局势 s，并返回所有可能的终局。
// 如果返回nil，表示这个局势有矛盾，不存在正确的解答。
func (ctx *SudokuContext) recurseEval(s *Situation, t *Trigger, branchName string) int {
	if *flagShowBranch {
		fmt.Println(branchName, "开始")
	}
	if !ctx.logicalEval(s, t) {
		if *flagShowBranch {
			fmt.Println(branchName, fmt.Sprintf("演算到 <%d> 发生矛盾", s.Count()))
		}
		return 0
	}

	if s.Completed() {
		if *flagShowBranch {
			fmt.Println(branchName, "找到解")
		}
		cells := s.cells
		ctx.results = append(ctx.results, &cells)
		return 1
	}

	//当前没有找到确定的填充选项，所以获取所有可能选项，然后在所有可能的选项里选一个单元格做尝试。

	//获取所有可能的选项
	guess := s.ChooseGuessingCell()

	var count int
	for _, n := range guess.Nums {
		s2 := s.Copy()
		t.Init()
		s2.Set(t, RowColNum{RowCol: guess.RowCol, Num: int(n)})
		ctx.evalCount++
		ctx.guessesCount++
		if !*flagShowOnlyResult {
			s2.Show("在可能的选项里猜一个", int(guess.Row), int(guess.Col))
		}
		if len(t.Conflicts) > 0 {
			if !*flagShowOnlyResult {
				fmt.Println("发生矛盾：")
				for _, msg := range t.Conflicts {
					fmt.Println(msg)
				}
			}
			continue
		}
		name := ""
		if *flagShowBranch {
			name = path.Join(branchName,
				fmt.Sprintf("<%d>(%d,%d)=%d", s2.Count(), guess.Row+1, guess.Col+1, n+1))
		}
		count += ctx.recurseEval(s2, t, name)
		s2.Release()
		if count > 0 && *flagShowStopAtFirst {
			break
		}
	}
	if *flagShowBranch {
		txt := "无解"
		if count > 0 {
			txt = fmt.Sprintf("%d 个解", count)
		}
		fmt.Println(branchName, txt)
	}
	return count
}

// logicalEval 开始推断局势 s，直到没有找到确定的填充选项，不确保全部完成。
// 如果返回false，表示这个局势有矛盾。
func (ctx *SudokuContext) logicalEval(s *Situation, t *Trigger) bool {
	last := t
	t = NewTrigger()
	for len(last.Confirms) > 0 || len(last.Conflicts) > 0 {
		for _, rcn := range last.Confirms {
			cellNumExcludes := s.cellNumExcludes[rcn.Row][rcn.Col]
			rowExcludes := s.rowExcludes[rcn.Num][rcn.Row]
			colExcludes := s.colExcludes[rcn.Num][rcn.Col]
			blockExcludes := s.blockExcludes[rcn.Num][rcn.Row/3][rcn.Col/3]
			if s.Set(t, rcn) {
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
						title += fmt.Sprintf("该宫唯一可以填 %d 的位置\n", rcn.Num+1)
					}
					s.Show(strings.TrimSuffix(title, "\n"), int(rcn.Row), int(rcn.Col))
				}
				if len(t.Conflicts) > 0 {
					if !*flagShowOnlyResult {
						fmt.Println("发生矛盾：")
						for _, msg := range t.Conflicts {
							fmt.Println(msg)
						}
					}
					return false
				}
			}
		}
		//避免重复创建Trigger对象，这里使用两个交替使用
		t, last = last, t
		t.Init()
	}
	if s.Completed() && !*flagShowOnlyResult {
		fmt.Println("找到了一个解")
	}
	return true
}
