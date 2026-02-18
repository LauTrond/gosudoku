package main

import (
	"fmt"
	"strings"
)

type SudokuContext struct {
	ShowProcess         bool
	ShowBranch          bool
	StopAtFirstSolution bool
	GensApplyRules      int

	evalCount     int
	rulesDebranch int
	branchCount   [10]int
	solutions     []*[9][9]int8

	//设置一个标准答案，用于检查计算。
	//设置后，跳过选择错误的分支，每次计算后都检查当前局势是否和标准答案矛盾。
	debugAnswer *[9][9]int8
}

func NewSudokuContext() *SudokuContext {
	return &SudokuContext{}
}

func (ctx *SudokuContext) Run(s *Situation, t *Trigger) int {
	if ctx.ShowProcess {
		s.Show("开始", -1, -1)
	}
	if len(t.Conflicts) > 0 {
		if ctx.ShowProcess {
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
func (ctx *SudokuContext) recurseEval(s *Situation, t *Trigger, branchName string) int {
	if ctx.ShowBranch {
		fmt.Println(branchName, "开始")
	}
	var good bool
	if s.branchGeneration < ctx.GensApplyRules {
		_, good = ctx.logicalEvalWithRules(s, t)
	} else {
		_, good = ctx.logicalEval(s, t)
	}
	if !good {
		if ctx.ShowBranch {
			fmt.Println(branchName, fmt.Sprintf("演算到 <%d> 矛盾", s.Count()))
		}
		return 0
	}
	if s.Completed() {
		if ctx.ShowBranch {
			fmt.Println(branchName, "找到解")
		}
		cells := s.cells
		ctx.solutions = append(ctx.solutions, &cells)
		return 1
	}

	//当前没有找到确定的填充选项，所以获取所有可能选项，然后在所有可能的选项里选一个单元格做尝试。

	//选取一个单元格和Num进行尝试
	candidates := s.ChooseBranchCell1()
	// candidates := s.ChooseBranchCellDiffusing()
	ctx.branchCount[candidates.Size()]++
	if candidates.Size() == 0 {
		return 0
	}
	var count int
	for _, selected := range candidates.Choices {
		if ctx.debugAnswer != nil {
			if ctx.debugAnswer[selected.Row][selected.Col] != selected.Num {
				continue
			}
		}
		s2 := DuplicateSituation(s)
		t2 := DuplicateTrigger(t)
		s2.branchGeneration++
		if selected.Val != 0 {
			s2.Set(t2, selected.RowColNum)
		} else {
			s2.Exclude(t2, selected.RowColNum)
		}
		ctx.evalCount++
		if ctx.ShowProcess {
			if selected.Val != 0 {
				s2.Show("在可能的选项里猜一个", int(selected.Row), int(selected.Col))
			} else {
				s2.Show(fmt.Sprintf("在可能的选项里排除 %d", selected.Num), int(selected.Row), int(selected.Col))
			}
		}
		if len(t2.Conflicts) > 0 {
			if ctx.ShowProcess {
				fmt.Println("发生矛盾：")
				for _, c := range t2.Conflicts {
					fmt.Println(c.String())
				}
			}
		} else {
			name := ""
			if ctx.ShowBranch {
				eqstr := "="
				if selected.Val == 0 {
					eqstr = "≠"
				}
				name = branchName + " " + fmt.Sprintf("<%d>(%d,%d)%s%d", s2.Count(), selected.Row+1, selected.Col+1, eqstr, selected.Num+1)
			}
			count += ctx.recurseEval(s2, t2, name)
		}
		ReleaseSituation(s2)
		ReleaseTrigger(t2)
		if len(t.Conflicts) > 0 || count > 0 && ctx.StopAtFirstSolution {
			break
		}
	}
	ReleaseBranchChoices(candidates)

	if ctx.ShowBranch {
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
func (ctx *SudokuContext) logicalEval(s *Situation, t *Trigger) (changed int, good bool) {
	good = true
	for {
		rcn, ok := t.GetConfirm()
		if !ok {
			break
		}
		cellNumExcludes := countTrueBits(s.numExcludeMask[rcn.Row][rcn.Col])
		rowExcludes := countTrueBits(s.rowExcludeMask[rcn.Num][rcn.Row])
		colExcludes := countTrueBits(s.colExcludeMask[rcn.Num][rcn.Col])
		b, _ := rcbp(rcn.Row, rcn.Col)
		blockExcludes := countTrueBits(s.blockExcludeMask[rcn.Num][b])
		if s.Set(t, rcn) > 0 {
			ctx.evalCount++
			changed++
			if ctx.ShowProcess {
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
				if ctx.ShowProcess {
					fmt.Println("发生矛盾：")
					for _, msg := range t.Conflicts {
						fmt.Println(msg)
					}
				}
				good = false
				return
			}
			if ctx.debugAnswer != nil {
				if !ctx.check(s) {
					good = false
					return
				}
			}
		}
	}
	if s.Completed() && ctx.ShowProcess {
		fmt.Println("找到了一个解")
	}
	return
}

// 如果返回false，表示这个局势有矛盾。
func (ctx *SudokuContext) logicalEvalWithRules(s *Situation, t *Trigger) (changed int, good bool) {
	good = true
	for {
		changed2, good2 := ctx.logicalEval(s, t)
		changed += changed2
		if !good2 {
			good = false
			break
		}
		if s.Completed() {
			break
		}
		changed2 = s.ApplyExcludeRules(t)
		if ctx.ShowProcess || ctx.ShowBranch {
			fmt.Printf("应用复杂排除规则，新增排除 %d 单元格\n", changed2)
		}
		if len(t.Conflicts) > 0 || t.confirms.Size() > 0 {
			ctx.rulesDebranch++
		}
		if len(t.Conflicts) > 0 {
			if ctx.ShowProcess {
				fmt.Println("发生矛盾：")
				for _, msg := range t.Conflicts {
					fmt.Println(msg)
				}
			}
			good = false
			break
		}
		if ctx.debugAnswer != nil {
			if !ctx.check(s) {
				good = false
				break
			}
		}
		if changed2 == 0 {
			break
		}
	}
	return
}

func (ctx *SudokuContext) check(s *Situation) (good bool) {
	good = true
	for _, r := range loop9 {
		for _, c := range loop9 {
			if s.cells[r][c] == 0 {
				return
			}
		}
	}
	for _, n := range loop9 {
		for _, r := range loop9 {
			for _, c := range loop9 {
				if s.cellExclude[n][r][c] == 1 && ctx.debugAnswer[r][c] == n {
					if ctx.ShowProcess || ctx.ShowBranch {
						fmt.Printf("错误排除：(%d, %d) %d\n", r+1, c+1, n+1)
					}
					good = false
				}
			}
		}
	}
	return
}
