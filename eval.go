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
	var result bool
	if s.branchGeneration < ctx.GensApplyRules {
		result = ctx.logicalEvalWithRules(s, t)
	} else {
		result = ctx.logicalEval(s, t)
	}
	if !result {
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
	// guess := s.ChooseGuessingCell2()
	ctx.branchCount[candidates.Size()]++
	if candidates.Size() == 0 {
		return 0
	}
	var count int
	for _, selected := range candidates.Choices {
		s2 := DuplicateSituation(s)
		t2 := DuplicateTrigger(t)
		s2.branchGeneration++
		s2.Set(t2, selected)
		ctx.evalCount++
		if ctx.ShowProcess {
			s2.Show("在可能的选项里猜一个", int(selected.Row), int(selected.Col))
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
				name = branchName + " " + fmt.Sprintf("<%d>(%d,%d)=%d", s2.Count(), selected.Row+1, selected.Col+1, selected.Num+1)
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
func (ctx *SudokuContext) logicalEval(s *Situation, t *Trigger) bool {
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
		if s.Set(t, rcn) {
			ctx.evalCount++
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
				return false
			}
		}
	}
	if s.Completed() && ctx.ShowProcess {
		fmt.Println("找到了一个解")
	}
	return true
}

// 如果返回false，表示这个局势有矛盾。
func (ctx *SudokuContext) logicalEvalWithRules(s *Situation, t *Trigger) bool {
	for t.confirms.Size() > 0 {
		if !ctx.logicalEval(s, t) {
			return false
		}
		if s.Completed() {
			return true
		}
		changed := s.ApplyExcludeRules(t)
		if ctx.ShowProcess || ctx.ShowBranch {
			fmt.Printf("应用复杂排除规则，新增排除 %d 单元格\n", changed)
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
			return false
		}
	}
	return true
}
