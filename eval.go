package main

import (
	"fmt"
	"strings"
)

type SudokuContext struct {
	evalCount    int
	guessesCount int
	solutions    []*[9][9]int8
}

func newSudokuContext() *SudokuContext {
	return &SudokuContext{}
}

func (ctx *SudokuContext) Run(s *Situation, t *Trigger) int {
	if len(t.Conflicts) > 0 {
		if *flagShowProcess {
			fmt.Println("开局矛盾：")
			for _, msg := range t.Conflicts {
				fmt.Println(msg)
			}
		}
		return 0
	}
	return ctx.recurseEval(s, t, fmt.Sprintf("<%d>", s.Count()))
}

// recurseEval 开始推断局势 s，并返回所有可能的终局。
// 如果返回 0，表示这个局势有矛盾，不存在正确的解答。
// t会被释放，不能再使用
func (ctx *SudokuContext) recurseEval(s *Situation, t *Trigger, branchName string) int {
	if *flagShowBranch {
		fmt.Println(branchName, "开始")
	}
	var count int
loopBranch:
	for {
		if !ctx.logicalEval(s, t) {
			if *flagShowBranch {
				fmt.Println(branchName, fmt.Sprintf("演算到 <%d> 矛盾", s.Count()))
			}
			break
		}
		if s.Completed() {
			if *flagShowBranch {
				fmt.Println(branchName, "找到解")
			}
			cells := s.cells
			ctx.solutions = append(ctx.solutions, &cells)
			count++
			break
		}
		//当前没有找到确定的填充选项，所以获取所有可能选项，然后在所有可能的选项里选一个单元格做尝试。

		//选取一个单元格和Num进行尝试
		guess := s.ChooseBranchCell1()
		// guess := s.ChooseGuessingCell2()
		if len(guess) == 0 {
			break
		}
		ctx.guessesCount++
		t = NewTrigger()
		for _, selected := range guess {
			s2 := s.Copy()
			t2 := t.Copy()
			s2.Set(t2, selected)
			ctx.evalCount++
			if *flagShowProcess {
				s2.Show("在可能的选项里猜一个", int(selected.Row), int(selected.Col))
			}
			if len(t2.Conflicts) > 0 {
				if *flagShowProcess {
					fmt.Println("发生矛盾：")
					for _, msg := range t2.Conflicts {
						fmt.Println(msg)
					}
				}
			} else {
				name := ""
				if *flagShowBranch {
					name = branchName + " " + fmt.Sprintf("<%d>(%d,%d)=%d", s2.Count(), selected.Row+1, selected.Col+1, selected.Num+1)
				}
				count += ctx.recurseEval(s2, t2, name)
			}
			s2.Release()
			s.Exclude(t, selected)
			if len(t.Conflicts) > 0 {
				t.Release()
				break loopBranch
			}
			if count > 0 && *flagStopAtFirstSolution {
				t.Release()
				break loopBranch
			}
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
// t会被释放，不能再使用
func (ctx *SudokuContext) logicalEval(s *Situation, t *Trigger) bool {
	last := t
	t = NewTrigger()
	defer t.Release()
	defer last.Release()
	for len(last.Confirms) > 0 {
		for _, rcn := range last.Confirms {
			cellNumExcludes := s.numExcludes[rcn.Row][rcn.Col]
			rowExcludes := s.rowExcludes[rcn.Num][rcn.Row]
			colExcludes := s.colExcludes[rcn.Num][rcn.Col]
			b, _ := rcbp(rcn.Row, rcn.Col)
			blockExcludes := s.blockExcludes[rcn.Num][b]
			if s.Set(t, rcn) {
				ctx.evalCount++
				if *flagShowProcess {
					title := ""
					if cellNumExcludes == 8 {
						title += "单元格唯一可以填的数\n"
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
					if *flagShowProcess {
						fmt.Println("发生矛盾：")
						for _, msg := range t.Conflicts {
							fmt.Println(msg)
						}
					}
					return false
				}
			}
		}
		last, t = t, last
		t.Init()
	}
	if s.Completed() && *flagShowProcess {
		fmt.Println("找到了一个解")
	}
	return true
}
